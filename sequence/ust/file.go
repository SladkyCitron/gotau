package ust

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/SladkyCitron/gotau/internal/ini"
	"github.com/SladkyCitron/gotau/internal/timeutil"
	"github.com/SladkyCitron/gotau/sequence"
	"github.com/SladkyCitron/gotau/umath"
)

// 0,5,35,0,100,100,0,%,0,10,100
// p1,p2,p3,v1,v2,v3,v4,%,p4,p5,v5

var defaultEnvelope = umath.Envelope{
	{X: 0, Y: 0},
	{X: 5, Y: 1},
	{X: 35, Y: 1},
	{X: 0, Y: 0},
	{X: 10, Y: 1},
}

var _ sequence.Sequencer = (*File)(nil)

// File represents a parsed UST file.
type File struct {
	Version  Version  // Version is the UST file format version.
	Settings Settings // Settings represents the settings of the UST file.
	Notes    []Note   // Notes hold the notes.

	iniDoc *ini.Document // iniDoc holds the raw parsed INI file structure (used internally).
}

func (f *File) Sequence() (sequence.Sequence, error) {
	seq := sequence.Sequence{
		Metadata: sequence.Metadata{
			Name:          f.Settings.ProjectName,
			VoicebankPath: f.Settings.VoiceDir,
			OutputPath:    f.Settings.OutFile,
			Resolution:    480, // MIDI sequencing default
			Tempo:         f.Settings.Tempo,
		},
	}

	var position int
	for i, note := range f.Notes {
		if IsLyricRest(note.Lyric) {
			position += note.Length
			continue
		}

		flags, err := sequence.ParseFlags(note.Flags)
		if err != nil {
			return sequence.Sequence{}, fmt.Errorf("ust Sequence: failed to parse flags for note #%04d: %w", i+1, err)
		}

		noteDurMs := timeutil.TicksToSeconds(note.Length, seq.Metadata.Resolution, seq.Metadata.Tempo) * 1000
		seq.Notes = append(seq.Notes, sequence.Note{
			Position:     position,
			Duration:     note.Length,
			Lyric:        note.Lyric,
			Note:         note.NoteNum,
			Intensity:    note.Intensity / 100,
			Velocity:     note.Velocity,
			Modulation:   note.Modulation,
			Preutterance: note.Preutterance,
			VoiceOverlap: note.VoiceOverlap,
			StartPoint:   note.StartPoint,
			Envelope:     envelopeToCurve(note.Envelope),
			PitchBend:    pitchBendToCurve(note.PitchBend, f.Settings.Mode2, noteDurMs),
			Flags:        flags,
		})
		position += note.Length
	}
	return seq, nil
}

func envelopeToCurve(env *Envelope) umath.Envelope {
	points := make([]umath.XY[float64], 0, 5)

	if env == nil {
		return defaultEnvelope
	}

	points = append(points, umath.XY[float64]{X: env.P1, Y: env.V1 / 100})
	points = append(points, umath.XY[float64]{X: env.P2, Y: env.V2 / 100})
	points = append(points, umath.XY[float64]{X: env.P3, Y: env.V3 / 100})
	points = append(points, umath.XY[float64]{X: env.P4, Y: env.V3 / 100})
	if env.HasRelease {
		points[3].Y = env.V4 / 100
		points = append(points, umath.XY[float64]{X: env.P5, Y: env.V5 / 100})
	}

	return points
}

func pitchBendToCurve(pb *PitchBend, mode2 bool, noteDurMs float64) umath.Curve {
	if pb == nil {
		return umath.Curve{}
	}
	if len(pb.Widths) == 0 {
		return umath.Curve{}
	}

	for len(pb.Ys) < len(pb.Widths) {
		pb.Ys = append(pb.Ys, 0)
	}

	for len(pb.Modes) < len(pb.Widths) {
		pb.Modes = append(pb.Modes, "")
	}

	points := make(umath.Curve, 0, len(pb.Widths)+1)

	// Mode1
	if !mode2 {
		x := pb.Start.X
		y := pb.Start.Y
		for i := range pb.Widths {
			points = append(points, umath.CurvePoint{
				X:      x,
				Y:      y,
				Interp: convertPBM(pb.Modes[i]),
			})

			x += pb.Widths[i]
			y = pb.Ys[i]
		}

		// final point
		points = append(points, umath.CurvePoint{
			X: x,
			Y: y,
		})
		return points
	}

	// Mode2
	x := pb.Start.X
	y := pb.Start.Y
	var firstMode PitchBendMode
	if len(pb.Modes) > 0 {
		firstMode = pb.Modes[0]
	}
	points = append(points, umath.CurvePoint{
		X:      x,
		Y:      y * 10, // convert deci-semitones to cents
		Interp: convertPBM(firstMode),
	})
	for i := range pb.Widths {
		x += pb.Widths[i]
		if i < len(pb.Ys) {
			y = pb.Ys[i]
		}
		var interpMode PitchBendMode
		if i+1 < len(pb.Modes) {
			interpMode = pb.Modes[i+1]
		}
		points = append(points, umath.CurvePoint{
			X:      x,
			Y:      y * 10, // convert deci-semitones to cents
			Interp: convertPBM(interpMode),
		})
	}

	// final point
	points = append(points, umath.CurvePoint{
		X: pb.Start.X + noteDurMs,
		Y: y * 10, // convert deci-semitones to cents
	})

	slices.SortFunc(points, func(a, b umath.CurvePoint) int {
		return cmp.Compare(a.X, b.X)
	})

	return points
}

func convertPBM(mode PitchBendMode) umath.CurveInterpolation {
	switch mode {
	case "r":
		return umath.CurveInterpolationSineOut
	case "s":
		return umath.CurveInterpolationLinear
	case "j":
		return umath.CurveInterpolationSineIn
	default:
		return umath.CurveInterpolationSineInOut
	}
}
