package pitch_test

import (
	"math/rand/v2"
	"testing"

	"github.com/SladkyCitron/gotau/pitch"
	"github.com/stretchr/testify/assert"
)

func TestFormatNote(t *testing.T) {
	assert.Equal(t, "C#4", pitch.FormatNote(61, 's'))
	assert.Equal(t, "Db4", pitch.FormatNote(61, 'f'))
	assert.Equal(t, "C4", pitch.FormatNote(60, 's'))
	assert.Equal(t, "C4", pitch.FormatNote(60, 'f'))
	assert.Equal(t, "B3", pitch.FormatNote(59, 's'))
	assert.Equal(t, "B3", pitch.FormatNote(59, 'f'))
}

func TestParseNoteInvalid(t *testing.T) {
	tests := []struct {
		name    string
		note    string
		wantErr string
	}{
		{name: "empty", note: "", wantErr: "invalid note format"},
		{name: "invalid base", note: "XX", wantErr: "invalid note format"},
		{name: "base only", note: "C", wantErr: "invalid note format"},
		{name: "missing octave", note: "C#", wantErr: "invalid note format"},
		{name: "double accidental", note: "C##5", wantErr: "invalid note format"},
		{name: "trailing data", note: "C5extra", wantErr: "invalid note format"},
		{name: "internal whitespace", note: "C 5", wantErr: "invalid note format"},
		{name: "unsupported base", note: "I5", wantErr: "invalid note format"},
		{name: "unsupported flat", note: "Cb5", wantErr: "invalid base"},
		{name: "unsupported sharp", note: "H#5", wantErr: "invalid base"},
		{name: "negative below range", note: "C-2", wantErr: "note out of MIDI range"},
		{name: "above range", note: "C11", wantErr: "note out of MIDI range"},
		{name: "octave overflow", note: "C999999999999999999999999", wantErr: "invalid octave"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pitch.ParseNote(tt.note)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func FuzzParseNote(f *testing.F) {
	for _, seed := range []string{
		"C4",
		"C#4",
		"Db3",
		"H4",
		"Hb4",
		"",
		"C",
		"C##5",
		"C5extra",
		"C 5",
		"I5",
		"Cb5",
		"H#5",
		"C-1",
		"C11",
		"C999999999999999999999999",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, s string) {
		note, err := pitch.ParseNote(s)
		if err != nil {
			return
		}
		if note < 12 {
			return
		}

		var fmt byte = 's'
		if s != "" && (s[1] == 'b' || s[1] == 'B') {
			fmt = 'f'
		}
		formatted := pitch.FormatNote(note, fmt)
		roundTrip, err := pitch.ParseNote(formatted)
		if err != nil {
			t.Fatalf("formatted note %q could not be parsed: %v", formatted, err)
		}
		if roundTrip != note {
			t.Fatalf("note %q formatted as %q and parsed as %d, want %d", s, formatted, roundTrip, note)
		}
	})
}

func TestEncodeResamplerPitchBendString(t *testing.T) {
	tests := []struct {
		name string
		x    []float64
		want string
	}{
		{
			name: "empty",
			x:    []float64{},
			want: "AA",
		},
		{
			name: "single point",
			x:    []float64{1},
			want: "AB",
		},
		{
			name: "run length",
			x:    []float64{0, 0, 0, 0},
			want: "AA#3#",
		},
		{
			name: "run length with 2 values",
			x:    []float64{0, 0, 0, 0, 1, 1, 1, 1},
			want: "AA#3#AB#3#",
		},
		{
			name: "max",
			x:    []float64{2047},
			want: "f/",
		},
		{
			name: "min",
			x:    []float64{-2048},
			want: "gA",
		},
		{
			name: "clamping",
			x:    []float64{-3000, 3000},
			want: "gAf/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, pitch.EncodeResamplerPitchBendString(tt.x))
		})
	}
}

func BenchmarkEncodeResamplerPitchBendString(b *testing.B) {
	data := make([]float64, 1000)
	for i := range data {
		data[i] = rand.Float64()*4096 - 2048
	}
	b.ResetTimer()
	for b.Loop() {
		_ = pitch.EncodeResamplerPitchBendString(data)
	}
}
