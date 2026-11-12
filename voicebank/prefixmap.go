package voicebank

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"gitlab.com/gomidi/midi/v2"
	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

type NoteFormat uint8

const (
	NoteFormatSharps NoteFormat = iota
	NoteFormatFlats
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

type prefixMapConfig struct {
	encoding  encoding.Encoding
	delimiter rune
	comment   rune
	sortFunc  func(a, b midi.Note) int
	notefmt   NoteFormat
}

// PrefixMapOption represents an option for passing into prefix.map-related functions and methods.
type PrefixMapOption func(*prefixMapConfig)

// PrefixMapWithEncoding specifies the character encoding to use when reading or writing the prefix.map file.
func PrefixMapWithEncoding(enc encoding.Encoding) PrefixMapOption {
	return func(cfg *prefixMapConfig) {
		cfg.encoding = enc
	}
}

// PrefixMapWithDelimiter specifies the delimiter to use when reading or writing the prefix.map file.
func PrefixMapWithDelimiter(delim rune) PrefixMapOption {
	return func(cfg *prefixMapConfig) {
		cfg.delimiter = delim
	}
}

// PrefixMapWithComment specifies the comment character to use when reading the prefix.map file.
// Lines beginning with this character without preceding whitespace will be ignored.
func PrefixMapWithComment(comment rune) PrefixMapOption {
	return func(cfg *prefixMapConfig) {
		cfg.comment = comment
	}
}

// PrefixMapWithSort enables sorting and specifies the sorting function to use when writing
// the prefix.map file.
func PrefixMapWithSort(cmpFunc func(a, b midi.Note) int) PrefixMapOption {
	return func(cfg *prefixMapConfig) {
		cfg.sortFunc = cmpFunc
	}
}

// PrefixMapWithNoteFormat specifies the note format to use when writing notes
// and accidentals in the prefix.map file.
//
// By default, accidentals are written using sharps (e.g. C#, D#).
func PrefixMapWithNoteFormat(notefmt NoteFormat) PrefixMapOption {
	return func(cfg *prefixMapConfig) {
		cfg.notefmt = notefmt
	}
}

// ParsePrefixMap parses and decodes a prefix.map file from the provided reader.
func ParsePrefixMap(data []byte, opts ...PrefixMapOption) (PrefixMap, error) {
	cfg := &prefixMapConfig{
		encoding:  encoding.Nop,
		delimiter: '\t',
		comment:   '#',
	}

	for _, opt := range opts {
		opt(cfg)
	}

	newData, err := cfg.encoding.NewDecoder().Bytes(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode prefix.map: %w", err)
	}

	prefixMap := make(PrefixMap)
	for line := range bytes.SplitSeq(newData, []byte{'\n'}) {
		line = bytes.TrimSuffix(line, []byte(`\r`))
		if cfg.comment != 0 && bytes.HasPrefix(bytes.TrimSpace(line), []byte(string(cfg.comment))) {
			continue
		}

		parts := bytes.SplitN(line, []byte(string(cfg.delimiter)), 3)
		if len(parts) < 3 {
			return nil, fmt.Errorf("voicebank prefix.map: invalid line: %q", line)
		}

		note, err := parseNote(string(bytes.TrimSpace(parts[0])))
		if err != nil {
			return nil, fmt.Errorf("voicebank prefix.map: invalid note %q: %w", parts[0], err)
		}

		prefixMap[note] = Prefix{
			Prefix: string(bytes.TrimSpace(parts[1])),
			Suffix: string(bytes.TrimSpace(parts[2])),
		}
	}
	return prefixMap, nil
}

// we use A-H to support German notation; H means B natural
// https://en.wikipedia.org/wiki/Musical_note#B%E2%99%AD,_B_and_H

var noteRe = regexp.MustCompile(`(?i)([A-H])([#b])?-?(\d+)`)

func parseNote(s string) (midi.Note, error) {
	m := noteRe.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("invalid note format: %q", s)
	}

	base := strings.ToUpper(m[1])
	accidental := strings.ToLower(m[2])
	octave, err := strconv.Atoi(m[3])
	if err != nil {
		return 0, fmt.Errorf("invalid octave: %q", m[3])
	}

	if octave > 10 {
		octave = 10
	}

	var note uint8
	switch base + accidental {
	case "C":
		note = 0
	case "C#", "Db":
		note = 1
	case "D":
		note = 2
	case "D#", "Eb":
		note = 3
	case "E":
		note = 4
	case "F":
		note = 5
	case "F#", "Gb":
		note = 6
	case "G":
		note = 7
	case "G#", "Ab":
		note = 8
	case "A":
		note = 9
	case "A#", "Bb", "Hb": // H = B in German notation
		note = 10
	case "B", "H": // H = B in German notation
		note = 11
	default:
		return 0, fmt.Errorf("invalid base: %q", base+accidental)
	}
	if octave != 0 {
		note += uint8(12 * octave)
		if note > 127 {
			note -= 12
		}
	}
	return midi.Note(note), nil
}

// Encode encodes and writes the prefixes to the provided [io.Writer].
func (pm PrefixMap) Encode(w io.Writer, opts ...PrefixMapOption) error {
	cfg := &prefixMapConfig{
		encoding:  encoding.Nop,
		delimiter: '\t',
		notefmt:   NoteFormatSharps,
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
	}
	var buf bytes.Buffer
	buf.Grow(16) // preallocate some space

	for _, note := range notes {
		prefix := pm[note]

		buf.Reset()

		buf.WriteString(formatNote(note, cfg.notefmt))
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
			return fmt.Errorf("failed to write prefix.map entry for note %s: %w", note.String(), err)
		}
	}

	return nil
}

// Note names. These correspond to [NoteFormat]s.
var noteNames = [...][12]string{
	{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"},
	{"C", "Db", "D", "Eb", "E", "F", "Gb", "G", "Ab", "A", "Bb", "B"},
}

func formatNote(note midi.Note, notefmt NoteFormat) string {
	return noteNames[notefmt][note%12] + strconv.FormatInt(int64(note/12), 10)
}
