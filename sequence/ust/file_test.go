package ust_test

import (
	"testing"

	"github.com/SladkyCitron/gotau/sequence/ust"
	"github.com/SladkyCitron/gotau/umath"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gomidi/midi/v2"
)

func TestSequenceMode1PitchBend(t *testing.T) {
	file := &ust.File{
		Settings: ust.Settings{
			Tempo: 120,
			Mode2: false,
		},
		Notes: []ust.Note{
			{
				Length:  480,
				Lyric:   "a",
				NoteNum: midi.Note(60),
				PitchBend: &ust.PitchBend{
					Start:   umath.XY[float64]{X: -10},
					Pitches: []float64{0, 100, -50},
				},
			},
		},
	}

	seq, err := file.Sequence()
	assert.NoError(t, err)
	assert.Len(t, seq.Notes, 1)
	assert.Len(t, seq.Notes[0].PitchBend, 3)
	assert.InDelta(t, -10, seq.Notes[0].PitchBend[0].X, 1e-12)
	assert.InDelta(t, -10+500.0/96, seq.Notes[0].PitchBend[1].X, 1e-12)
	assert.InDelta(t, -10+2*500.0/96, seq.Notes[0].PitchBend[2].X, 1e-12)
	assert.Equal(t, []float64{0, 100, -50}, []float64{
		seq.Notes[0].PitchBend[0].Y,
		seq.Notes[0].PitchBend[1].Y,
		seq.Notes[0].PitchBend[2].Y,
	})
	assert.Equal(t, []umath.CurveInterpolation{
		umath.CurveInterpolationLinear,
		umath.CurveInterpolationLinear,
		umath.CurveInterpolationLinear,
	}, []umath.CurveInterpolation{
		seq.Notes[0].PitchBend[0].Interp,
		seq.Notes[0].PitchBend[1].Interp,
		seq.Notes[0].PitchBend[2].Interp,
	})
}
