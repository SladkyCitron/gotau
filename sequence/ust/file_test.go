package ust_test

import (
	"testing"

	"github.com/SladkyCitron/gotau/sequence"
	"github.com/SladkyCitron/gotau/sequence/ust"
	"github.com/SladkyCitron/gotau/umath"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gomidi/midi/v2"
)

func TestFile_SequenceMode1PitchBend(t *testing.T) {
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

func TestFile_Sequence(t *testing.T) {
	preutterance := 12.5
	preutteranceOverride := 6.25
	voiceOverlap := 8.0
	voiceOverlapOverride := 4.5
	startPointOverride := 12.25
	file := &ust.File{
		Settings: ust.Settings{
			Tempo:       120,
			ProjectName: "test",
			VoiceDir:    "voicebank",
			OutFile:     "output.wav",
		},
		Notes: []ust.Note{
			{Length: 240, Lyric: "R", NoteNum: midi.Note(60)},
			{
				Length:               480,
				Lyric:                "a",
				NoteNum:              midi.Note(69),
				Intensity:            80,
				Velocity:             110,
				Modulation:           25,
				Preutterance:         &preutterance,
				PreutteranceOverride: &preutteranceOverride,
				VoiceOverlap:         &voiceOverlap,
				StartPoint:           3,
				VoiceOverlapOverride: &voiceOverlapOverride,
				StartPointOverride:   &startPointOverride,
				Envelope: &ust.Envelope{
					P1: 1, P2: 2, P3: 3,
					V1: 0, V2: 80, V3: 100, V4: 50,
				},
				Flags: "g-5B50",
			},
			{
				Length:  240,
				Lyric:   "i",
				NoteNum: midi.Note(71),
				Flags:   "",
			},
		},
	}

	got, err := file.Sequence()
	assert.NoError(t, err)
	assert.Equal(t, sequence.Metadata{
		Name:          "test",
		VoicebankPath: "voicebank",
		OutputPath:    "output.wav",
		Resolution:    480,
		Tempo:         120,
	}, got.Metadata)
	assert.Len(t, got.Notes, 2)

	assert.Equal(t, 240, got.Notes[0].Position)
	assert.Equal(t, 480, got.Notes[0].Duration)
	assert.Equal(t, "a", got.Notes[0].Lyric)
	assert.Equal(t, midi.Note(69), got.Notes[0].Note)
	assert.Equal(t, 0.8, got.Notes[0].Intensity)
	assert.Equal(t, 110.0, got.Notes[0].Velocity)
	assert.Equal(t, 25.0, got.Notes[0].Modulation)
	assert.Equal(t, &preutterance, got.Notes[0].Preutterance)
	assert.Equal(t, &preutteranceOverride, got.Notes[0].PreutteranceOverride)
	assert.Equal(t, &voiceOverlap, got.Notes[0].VoiceOverlap)
	assert.Equal(t, 3.0, got.Notes[0].StartPoint)
	assert.Equal(t, &voiceOverlapOverride, got.Notes[0].VoiceOverlapOverride)
	assert.Equal(t, &startPointOverride, got.Notes[0].StartPointOverride)
	assert.Equal(t, sequence.Flags{"g": -5, "B": 50}, got.Notes[0].Flags)
	assert.Equal(t, umath.Envelope{
		{X: 1, Y: 0},
		{X: 2, Y: 0.8},
		{X: 3, Y: 1},
		{X: 0, Y: 1},
	}, got.Notes[0].Envelope)

	assert.Equal(t, 720, got.Notes[1].Position)
	assert.Equal(t, 240, got.Notes[1].Duration)
	assert.Equal(t, "i", got.Notes[1].Lyric)
	assert.Equal(t, umath.Envelope{
		{X: 0, Y: 0},
		{X: 5, Y: 1},
		{X: 35, Y: 1},
		{X: 0, Y: 0},
		{X: 10, Y: 1},
	}, got.Notes[1].Envelope)
}

func TestFile_SequenceInvalidFlags(t *testing.T) {
	file := &ust.File{
		Settings: ust.Settings{Tempo: 120},
		Notes: []ust.Note{
			{Length: 480, Lyric: "a", NoteNum: midi.Note(60), Flags: "!"},
		},
	}

	got, err := file.Sequence()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse flags for note #0001")
	assert.Equal(t, sequence.Sequence{}, got)
}

func TestFile_SequenceMode2PitchBend(t *testing.T) {
	file := &ust.File{
		Settings: ust.Settings{
			Tempo: 120,
			Mode2: true,
		},
		Notes: []ust.Note{
			{
				Length:  480,
				Lyric:   "a",
				NoteNum: midi.Note(60),
				PitchBend: &ust.PitchBend{
					Start:  umath.XY[float64]{X: -40, Y: -2},
					Widths: []float64{65, 69},
					Ys:     []float64{0, 42},
					Modes:  []ust.PitchBendMode{"l", "s"},
				},
			},
		},
	}

	got, err := file.Sequence()
	assert.NoError(t, err)
	assert.Equal(t, umath.Curve{
		{X: -40, Y: -20, Interp: umath.CurveInterpolationSineInOut},
		{X: 25, Y: 0, Interp: umath.CurveInterpolationLinear},
		{X: 94, Y: 420, Interp: umath.CurveInterpolationSineInOut},
		{X: 460, Y: 420},
	}, got.Notes[0].PitchBend)
}
