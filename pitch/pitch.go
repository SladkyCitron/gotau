package pitch

import (
	"math"
	"strconv"
	"strings"

	"gitlab.com/gomidi/midi/v2"
)

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
	dst = strconv.AppendInt(dst, int64(note/12), 10)
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
