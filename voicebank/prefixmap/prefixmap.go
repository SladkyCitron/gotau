package prefixmap

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"slices"

	"github.com/SladkyCitron/gotau/pitch"
	"gitlab.com/gomidi/midi/v2"
	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

// PrefixMap represents the prefix.map configuration in an UTAU voicebank.
// It maps notes to their corresponding [Prefix] (prefix and suffix).
type PrefixMap map[midi.Note]Prefix

// Prefix represents a single entry in the prefix.map file.
// It holds the prefix and suffix.
type Prefix struct {
	Prefix string
	Suffix string
}

type config struct {
	encoding  encoding.Encoding
	delimiter rune
	comment   rune
	sortFunc  func(a, b midi.Note) int
	notefmt   byte
}

// Option represents an option for passing into prefix.map-related functions and methods.
type Option func(*config)

// WithEncoding specifies the character encoding to use when reading or writing the prefix.map file.
func WithEncoding(enc encoding.Encoding) Option {
	return func(cfg *config) {
		cfg.encoding = enc
	}
}

// WithDelimiter specifies the delimiter to use when reading or writing the prefix.map file.
func WithDelimiter(delim rune) Option {
	return func(cfg *config) {
		cfg.delimiter = delim
	}
}

// WithComment specifies the comment character to use when reading the prefix.map file.
// Lines beginning with this character with or without preceding whitespace will be ignored.
func WithComment(comment rune) Option {
	return func(cfg *config) {
		cfg.comment = comment
	}
}

// WithSort enables sorting and specifies the sorting function to use when writing
// the prefix.map file.
func WithSort(cmpFunc func(a, b midi.Note) int) Option {
	return func(cfg *config) {
		cfg.sortFunc = cmpFunc
	}
}

// WithNoteFormat specifies the note format to use when writing notes
// and accidentals in the prefix.map file.
//
// By default, accidentals are written using sharps (e.g. C#, D#).
func WithNoteFormat(notefmt byte) Option {
	return func(cfg *config) {
		cfg.notefmt = notefmt
	}
}

// Parse parses and decodes a prefix.map file.
func Parse(data []byte, opts ...Option) (PrefixMap, error) {
	cfg := &config{
		encoding:  encoding.Nop,
		delimiter: '\t',
		comment:   '#',
	}

	for _, opt := range opts {
		opt(cfg)
	}

	newData, err := cfg.encoding.NewDecoder().Bytes(data)
	if err != nil {
		return nil, fmt.Errorf("prefixmap Parse: failed to decode prefix.map: %w", err)
	}

	prefixMap := make(PrefixMap)
	for line := range bytes.SplitSeq(newData, []byte{'\n'}) {
		line = bytes.TrimSuffix(line, []byte(`\r`))
		if len(line) == 0 || cfg.comment != 0 && bytes.HasPrefix(bytes.TrimSpace(line), []byte(string(cfg.comment))) {
			continue
		}

		parts := bytes.SplitN(line, []byte(string(cfg.delimiter)), 3)
		if len(parts) < 3 {
			return nil, fmt.Errorf("prefixmap Parse: invalid line: %q", line)
		}

		note, err := pitch.ParseNote(string(bytes.TrimSpace(parts[0])))
		if err != nil {
			return nil, fmt.Errorf("prefixmap Parse: invalid note %q: %w", parts[0], err)
		}

		if _, exists := prefixMap[note]; exists {
			return nil, fmt.Errorf("prefixmap Parse: duplicate note %q", parts[0])
		}

		prefixMap[note] = Prefix{
			Prefix: string(bytes.TrimSpace(parts[1])),
			Suffix: string(bytes.TrimSpace(parts[2])),
		}
	}
	return prefixMap, nil
}

// Encode encodes and writes the prefixes to the provided [io.Writer].
func (pm PrefixMap) Encode(w io.Writer, opts ...Option) error {
	cfg := &config{
		encoding:  encoding.Nop,
		delimiter: '\t',
		notefmt:   's',
	}

	for _, opt := range opts {
		opt(cfg)
	}

	notes := slices.Collect(maps.Keys(pm))
	if cfg.sortFunc != nil {
		slices.SortFunc(notes, cfg.sortFunc)
	}

	newWriter := w
	if cfg.encoding != encoding.Nop {
		newWriter = transform.NewWriter(w, cfg.encoding.NewEncoder())
		defer func() {
			// *transform.Writer implements io.Closer so this is safe to do without the ok check
			closer := newWriter.(io.Closer)
			if err := closer.Close(); err != nil {
				panic(fmt.Errorf("prefixmap Encode: failed to close writer: %w", err))
			}
		}()
	}
	var buf bytes.Buffer
	buf.Grow(64)                  // preallocate some space
	noteTmp := make([]byte, 0, 4) // scratch buffer for note string

	for _, note := range notes {
		prefix := pm[note]

		buf.Reset()

		buf.Write(pitch.AppendNote(noteTmp[:0], note, cfg.notefmt))
		buf.WriteRune(cfg.delimiter)
		if prefix.Prefix != "" {
			buf.WriteString(prefix.Prefix)
		}
		buf.WriteRune(cfg.delimiter)
		if prefix.Suffix != "" {
			buf.WriteString(prefix.Suffix)
		}
		buf.WriteByte(10) // newline

		if _, err := buf.WriteTo(newWriter); err != nil {
			return fmt.Errorf("prefixmap Encode: failed to write entry for note %s: %w", note.String(), err)
		}
	}

	return nil
}
