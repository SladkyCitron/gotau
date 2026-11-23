package ust_test

import (
	"testing"

	"github.com/SladkyCitron/gotau/sequence/ust"
	"github.com/stretchr/testify/assert"
)

func TestParsePitchBend(t *testing.T) {
	pb, err := ust.ParsePitchBend(
		"5",       // type
		"10;2",    // start
		"",        // pbs (ignored because start is set)
		"30,40",   // pbw
		"0.5,1.0", // pby
		"l,s",     // pbm
		"0,100,-50",
	)
	assert.NoError(t, err)
	assert.NotNil(t, pb)

	assert.Equal(t, 5, pb.Type)
	assert.Equal(t, float64(10.0), pb.Start.X)
	assert.Equal(t, float64(2.0), pb.Start.Y)
	assert.Equal(t, []float64{0, 100, -50}, pb.Pitches)
	assert.Equal(t, []float64{30, 40}, pb.Widths)
	assert.Equal(t, []float64{0.5, 1.0}, pb.Ys)
	assert.Equal(t, []ust.PitchBendMode{"l", "s"}, pb.Modes)
}

func TestParsePitchBendInvalidType(t *testing.T) {
	_, err := ust.ParsePitchBend("not-a-number", "", "", "", "", "")
	assert.Error(t, err)
}

func TestParsePitchBendEmptyFields(t *testing.T) {
	pb, err := ust.ParsePitchBend("", "", "0", "10", "0", "")
	assert.NoError(t, err)
	assert.Equal(t, 5, pb.Type)
	assert.Equal(t, float64(0.0), pb.Start.X)
	assert.Equal(t, float64(0.0), pb.Start.Y)
}
