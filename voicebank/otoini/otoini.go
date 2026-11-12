// Package otoini provides functionality for reading and writing oto.ini files.
package otoini

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"

	"github.com/SladkyCitron/gotau/internal/ini"
	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

// Entry represents a single entry in an oto.ini file.
type Entry struct {
	// Filename is the name of the audio file associated with this oto entry.
	Filename string

	// Directory is the path to the directory where the oto.ini file and audio files are located.
	Directory string

	// Alias is the phoneme or alias associated with this oto entry.
	Alias string

	// Offset is the offset time in milliseconds.
	Offset float64

	// Consonant is the consonant duration in milliseconds.
	Consonant float64

	// Cutoff is the cutoff time in milliseconds.
	Cutoff float64

	// Preutterance is the preutterance time in milliseconds.
	Preutterance float64

	// Overlap is the overlap time in milliseconds.
	Overlap float64
}

// FilePath returns the file path to the audio file.
func (e Entry) FilePath() string {
	// using path and not filepath because fs.FS uses forward slashes
	return path.Join(e.Directory, e.Filename)
}

// Oto represents the oto.ini configuration in an UTAU voicebank.
// It holds a list of phonemes, aliases, and their associated parameters.
type Oto []Entry

type config struct {
	encoding       encoding.Encoding
	comment        rune
	floatPrecision int
	dir            string
}

// Option represents an option for passing into oto.ini-related functions and methods.
type Option func(*config)

// WithEncoding specifies the character encoding to use when reading or writing the oto.ini file.
func WithEncoding(encoding encoding.Encoding) Option {
	return func(cfg *config) {
		cfg.encoding = encoding
	}
}

// WithComment specifies the comment character to use when reading the oto.ini file.
// Lines beginning with this character without preceding whitespace will be ignored.
func WithComment(comment rune) Option {
	return func(cfg *config) {
		cfg.comment = comment
	}
}

// WithFloatPrecision specifies the float precision to use when writing float values in the oto.ini file.
func WithFloatPrecision(prec int) Option {
	if prec < 0 {
		panic("float precision cannot be negative")
	}

	return func(cfg *config) {
		cfg.floatPrecision = prec
	}
}

// WithDirectory specifies the path to the directory where the oto.ini file is to put
// into the [Entry.Directory] field when reading the oto.ini file.
func WithDirectory(dir string) Option {
	return func(cfg *config) {
		cfg.dir = dir
	}
}

// Parse parses and decodes an oto.ini file.
func Parse(data []byte, opts ...Option) (Oto, error) {
	cfg := &config{
		encoding: encoding.Nop,
		comment:  '#',
	}

	for _, opt := range opts {
		opt(cfg)
	}

	newData, err := cfg.encoding.NewDecoder().Bytes(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode oto.ini: %w", err)
	}

	iniDoc, err := ini.Parse(newData, ini.ParseOptions{
		CommentPrefix: string(cfg.comment),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse oto.ini: %w", err)
	}

	var oto Oto
	for _, section := range iniDoc.Sections {
		for _, entry := range section.Entries {
			filename := entry.Key
			values := strings.Split(entry.Value, ",")

			if len(values) != 6 {
				return nil, fmt.Errorf("invalid oto entry for %s, expected 6 values, got %d", strconv.Quote(filename), len(values))
			}

			alias := values[0]

			if values[1] == "" {
				values[1] = "0"
			}
			offset, err := strconv.ParseFloat(values[1], 64)
			if err != nil {
				return oto, fmt.Errorf("failed to parse offset value for %s: %w", strconv.Quote(filename), err)
			}

			if values[2] == "" {
				values[2] = "0"
			}
			consonant, err := strconv.ParseFloat(values[2], 64)
			if err != nil {
				return oto, fmt.Errorf("failed to parse consonant value for %s: %w", strconv.Quote(filename), err)
			}

			if values[3] == "" {
				values[3] = "0"
			}
			cutoff, err := strconv.ParseFloat(values[3], 64)
			if err != nil {
				return oto, fmt.Errorf("failed to parse cutoff value for %s: %w", strconv.Quote(filename), err)
			}

			if values[4] == "" {
				values[4] = "0"
			}
			preutter, err := strconv.ParseFloat(values[4], 64)
			if err != nil {
				return oto, fmt.Errorf("failed to parse preutterance value for %s: %w", strconv.Quote(filename), err)
			}

			if values[5] == "" {
				values[5] = "0"
			}
			overlap, err := strconv.ParseFloat(values[5], 64)
			if err != nil {
				return oto, fmt.Errorf("failed to parse overlap value for %s: %w", strconv.Quote(filename), err)
			}

			oto = append(oto, Entry{
				Filename:     filename,
				Directory:    cfg.dir,
				Alias:        alias,
				Offset:       offset,
				Consonant:    consonant,
				Cutoff:       cutoff,
				Preutterance: preutter,
				Overlap:      overlap,
			})
		}
	}
	return oto, nil
}

// Get retrieves an [Entry] by its phoneme alias.
func (o Oto) Get(alias string) (_ Entry, ok bool) {
	for i := range o {
		if o[i].Alias == alias {
			return o[i], true
		}
	}
	return Entry{}, false
}

// Encode encodes and writes the oto.ini entries to the provided [io.Writer].
func (o Oto) Encode(w io.Writer, opts ...Option) error {
	cfg := &config{
		encoding:       encoding.Nop,
		floatPrecision: -1,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	newWriter := w
	if cfg.encoding != encoding.Nop {
		newWriter = transform.NewWriter(w, cfg.encoding.NewEncoder())
	}
	var buf bytes.Buffer
	buf.Grow(256)                   // preallocate some space
	floatTmp := make([]byte, 0, 64) // scratch buffer for floats

	for _, entry := range o {
		// filename=alias,offset,consonant,cutoff,preutter,overlap
		// filename and alias are strings, the rest are floats
		buf.Reset()

		buf.WriteString(entry.Filename)
		buf.WriteByte('=')
		buf.WriteString(entry.Alias)
		buf.WriteByte(',')
		buf.Write(strconv.AppendFloat(floatTmp[:0], entry.Offset, 'f', cfg.floatPrecision, 64))
		buf.WriteByte(',')
		buf.Write(strconv.AppendFloat(floatTmp[:0], entry.Consonant, 'f', cfg.floatPrecision, 64))
		buf.WriteByte(',')
		buf.Write(strconv.AppendFloat(floatTmp[:0], entry.Cutoff, 'f', cfg.floatPrecision, 64))
		buf.WriteByte(',')
		buf.Write(strconv.AppendFloat(floatTmp[:0], entry.Preutterance, 'f', cfg.floatPrecision, 64))
		buf.WriteByte(',')
		buf.Write(strconv.AppendFloat(floatTmp[:0], entry.Overlap, 'f', cfg.floatPrecision, 64))
		buf.WriteByte(10) // newline

		if _, err := buf.WriteTo(newWriter); err != nil {
			return fmt.Errorf("failed to write oto entry for %s: %w", strconv.Quote(entry.Filename), err)
		}
	}
	return nil
}
