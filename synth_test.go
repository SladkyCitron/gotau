package gotau_test

import (
	"testing"

	"github.com/SladkyCitron/gotau"
	"github.com/SladkyCitron/gotau/sequence"
	"github.com/SladkyCitron/gotau/voicebank/otoini"
	"github.com/stretchr/testify/assert"
)

func TestCalculateResamplerTiming(t *testing.T) {
	mo := otoini.Entry{
		Offset:       18,
		Consonant:    80,
		Cutoff:       49,
		Preutterance: 30,
		Overlap:      14,
	}
	a := otoini.Entry{
		Offset:       24,
		Consonant:    56,
		Cutoff:       73,
		Preutterance: 5,
		Overlap:      20,
	}

	nextA := sequence.Note{Position: 240, Duration: 240, Velocity: 100}
	moTiming := gotau.CalculateResamplerTiming(mo, sequence.Note{Position: 0, Duration: 240, Velocity: 100}, 170.45454545454547, 30, &nextA, 5, 20)
	assert.Equal(t, 18.0, moTiming.Offset)
	assert.Equal(t, 250.0, moTiming.Length)
	assert.Equal(t, 80.0, moTiming.Consonant)
	assert.Equal(t, 49.0, moTiming.Cutoff)

	moWithStartPoint := gotau.CalculateResamplerTiming(mo, sequence.Note{Position: 0, Duration: 240, Velocity: 100, StartPoint: 20}, 170.45454545454547, 30, &nextA, 5, 20)
	assert.Equal(t, 300.0, moWithStartPoint.Length)

	nextMo := sequence.Note{Position: 480, Duration: 240, Velocity: 100}
	aTiming := gotau.CalculateResamplerTiming(a, sequence.Note{Position: 240, Duration: 240, Velocity: 100}, 170.45454545454547, 5, &nextMo, 30, 14)
	assert.Equal(t, 24.0, aTiming.Offset)
	assert.Equal(t, 200.0, aTiming.Length)
	assert.Equal(t, 56.0, aTiming.Consonant)
	assert.Equal(t, 73.0, aTiming.Cutoff)
}
