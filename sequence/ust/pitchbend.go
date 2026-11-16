package ust

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/SladkyCitron/gotau/umath"
	"github.com/SladkyCitron/slicestrconv"
)

// PitchBendMode represents a pitch bend segment interpolation mode.
type PitchBendMode string

// PitchBend represents the pitch bend data. Mode1 uses cents, Mode2 uses deci-semitones.
type PitchBend struct {
	Type    int               // Type is the pitch bend type (0 = no bend, 5 = default).
	Start   umath.XY[float64] // Start is the starting point in milliseconds (X-axis) and initial pitch offset (Y-axis).
	Pitches []float64         // Pitches are Mode 1 pitch samples in cents, recorded every 5 ticks.
	Widths  []float64         // Widths are the widths in milliseconds for each pitch segment.
	Ys      []float64         // Ys are the pitch offsets for each segment.
	Modes   []PitchBendMode   // Modes are the interpolation modes for each segment.
}

func ParsePitchBend(typ, start, pbs, pbw, pby, pbm string, pitches ...string) (pb *PitchBend, err error) {
	pb = &PitchBend{}

	// PBType
	pb.Type = 5
	if typ != "" {
		pb.Type, err = strconv.Atoi(typ)
		if err != nil {
			return nil, fmt.Errorf("failed to parse pitch bend type: %w", err)
		}
	}

	// PBStart / PBS
	if start != "" {
		if err := pb.parseStart(start); err != nil {
			return nil, fmt.Errorf("failed to parse pitch bend start: %w", err)
		}
	} else if pbs != "" {
		if err := pb.parseStart(pbs); err != nil {
			return nil, fmt.Errorf("failed to parse pitch bend start: %w", err)
		}
	}

	slicestrconv.OpeningBracket = ""
	slicestrconv.ClosingBracket = ""

	// PBW
	pb.Widths, err = slicestrconv.ParseFloat64Slice(pbw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pitch bend widths: %w", err)
	}

	// PBY
	pb.Ys, err = slicestrconv.ParseFloat64Slice(pby)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pitch bend ys: %w", err)
	}

	// PBM
	for m := range strings.SplitSeq(pbm, ",") {
		pb.Modes = append(pb.Modes, PitchBendMode(m))
	}

	if len(pitches) > 0 {
		pb.Pitches, err = parsePitchSamples(pitches[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse Mode 1 pitches: %w", err)
		}
	}

	return pb, nil
}

func parsePitchSamples(s string) ([]float64, error) {
	if s == "" {
		return nil, nil
	}

	parts := strings.Split(s, ",")
	pitches := make([]float64, len(parts))
	for i, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		pitch, err := parseFiniteFloat(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		pitches[i] = pitch
	}
	return pitches, nil
}

func (pb *PitchBend) parseStart(s string) error {
	pbsParts := strings.Split(s, ";")
	x, err := strconv.ParseFloat(pbsParts[0], 64)
	if err != nil {
		return err
	}
	y := 0.0
	if len(pbsParts) > 1 {
		y, err = strconv.ParseFloat(pbsParts[1], 64)
		if err != nil {
			return err
		}
	}
	pb.Start = umath.XY[float64]{X: x, Y: y}
	return nil
}
