package pitch

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"gitlab.com/gomidi/midi/v2"
)

// we use A-H to support German notation; H means B natural
// https://en.wikipedia.org/wiki/Musical_note#B%E2%99%AD,_B_and_H

var noteRe = regexp.MustCompile(`(?i)^([A-H])([#b])?(-?\d+)$`)

// ParseNote parses a note string (e.g., "C#4", "Db3", "A5") into a MIDI note number.
// It supports, sharp (#) and flat (b) notation, German notation (H for B natural),
// as well as octave numbers.
// The octave number is expected to be an integer, where C4 is middle C (MIDI note 60).
func ParseNote(s string) (midi.Note, error) {
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

	var semitone int

	switch base + accidental {
	case "C":
		semitone = 0
	case "C#", "Db":
		semitone = 1
	case "D":
		semitone = 2
	case "D#", "Eb":
		semitone = 3
	case "E":
		semitone = 4
	case "F":
		semitone = 5
	case "F#", "Gb":
		semitone = 6
	case "G":
		semitone = 7
	case "G#", "Ab":
		semitone = 8
	case "A":
		semitone = 9
	case "A#", "Bb":
		semitone = 10
	case "B", "H":
		semitone = 11
	case "Hb":
		semitone = 10
	default:
		return 0, fmt.Errorf("invalid base: %q", base+accidental)
	}

	midiNote := 12*(octave+1) + semitone

	if midiNote < 0 || midiNote > 127 {
		return 0, fmt.Errorf("note out of MIDI range: %q", s)
	}

	return midi.Note(midiNote), nil
}

// Note names for the supported sharp and flat formats.
var noteNames = map[byte][12]string{
	's': {"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"},
	'f': {"C", "Db", "D", "Eb", "E", "F", "Gb", "G", "Ab", "A", "Bb", "B"},
}

// AppendNote appends the MIDI note to the provided byte slice, according to the format fmt.
//
// The format fmt is one of:
//
//   - 's' for sharps (e.g. C#4)
//   - 'f' for flats (e.g. Db4)
func AppendNote(dst []byte, note midi.Note, fmt byte) []byte {
	if fmt != 's' && fmt != 'f' {
		fmt = 's' // default to sharps if invalid format is provided
	}
	dst = append(dst, noteNames[fmt][note%12]...)
	dst = strconv.AppendInt(dst, int64((note/12)-1), 10)
	return dst
}

// FormatNote converts the MIDI note to a string, according to the format fmt.
//
// The format fmt is one of:
//
//   - 's' for sharps (e.g. C#4)
//   - 'f' for flats (e.g. Db4)
func FormatNote(note midi.Note, fmt byte) string {
	return string(AppendNote(make([]byte, 0, 4), note, fmt))
}

// EncodeResamplerPitchBendString encodes a pitch bend curve into the UTAU resampler pitch bend string format.
func EncodeResamplerPitchBendString(x []float64) string {
	if len(x) == 0 {
		return "AA"
	}

	last := int16(math.MinInt16)
	run := 0 // run length

	var buf strings.Builder
	buf.Grow(len(x) * 2)         // allocate some space
	runTmp := make([]byte, 0, 8) // scratch buffer for run length

	for i := range x {
		num := int16(math.Round(x[i]))
		if num == last {
			run++
			continue
		}

		// flush run
		if run > 0 {
			buf.WriteByte('#')
			buf.Write(strconv.AppendInt(runTmp[:0], int64(run), 10))
			buf.WriteByte('#')
			run = 0
		}

		writeInt12(&buf, num)
		last = num
	}

	// flush remaining run
	if run > 0 {
		buf.WriteByte('#')
		buf.Write(strconv.AppendInt(runTmp[:0], int64(run), 10))
		buf.WriteByte('#')
	}

	return buf.String()
}

const b64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

func writeInt12(buf *strings.Builder, v int16) {
	if v == 0 {
		if _, err := buf.WriteString("AA"); err != nil {
			panic(err)
		}
		return
	}
	if v < -2048 {
		v = -2048
	}
	if v > 2047 {
		v = 2047
	}
	if v < 0 {
		v += 4096
	}

	hi := (v >> 6) & 0x3f
	lo := v & 0x3f

	if err := buf.WriteByte(b64[hi]); err != nil {
		panic(err)
	}
	if err := buf.WriteByte(b64[lo]); err != nil {
		panic(err)
	}
}
