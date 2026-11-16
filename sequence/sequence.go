package sequence

import (
	"time"

	"github.com/SladkyCitron/gotau/internal/timeutil"
	"github.com/SladkyCitron/gotau/umath"
	"gitlab.com/gomidi/midi/v2"
)

// Sequence represents a voice synth note sequence. It holds metadata and a list of notes.
type Sequence struct {
	// Metadata is the metadata.
	Metadata Metadata

	// Notes is the list of notes. It should be sorted by position (ascending).
	Notes []Note
}

// Metadata represents the metadata of a sequence.
type Metadata struct {
	// Name is the human-readable name of the sequence (e.g. project name, song name).
	Name string

	// VoicebankPath is the path to the voicebank used by the sequence.
	VoicebankPath string

	// OutputPath is the path to the final output file (e.g. wav file).
	OutputPath string

	// Resolution is the number of MIDI ticks per quarter note (TPQN).
	Resolution int

	// Tempo is the tempo of the sequence in beats per minute (BPM).
	Tempo float64
}

// time signature is always 4/4 (for now)

// Note represents a single musical note in the sequence.
type Note struct {
	// Position is the position of the note in MIDI ticks.
	Position int

	// Duration is the duration of the note in MIDI ticks.
	Duration int

	// Lyric is the lyric of the note. It can be either latin characters or kana.
	Lyric string

	// Note is the MIDI note number.
	Note midi.Note

	// Intensity is the loudness or intensity of the note (0.0 to 1.0).
	Intensity float64

	// Velocity affects timing (smaller = more rushed; rarely used).
	Velocity float64

	// Modulation is the modulation depth, mostly used for vibrato.
	Modulation float64

	// Preutterance is the duration (in milliseconds) before note to start playback (in oto). If it's omitted, falls back to oto defaults.
	Preutterance *float64

	// PreutteranceOverride is the explicit auto-adjusted pre-utterance override.
	PreutteranceOverride *float64

	// VoiceOverlap is the amount of overlap into the previous note. If it's omitted, falls back to oto defaults.
	VoiceOverlap *float64

	// StartPoint is the time where to begin sampling inside the audio file (in milliseconds).
	StartPoint float64

	// VoiceOverlapOverride is the explicit auto-adjusted overlap override.
	VoiceOverlapOverride *float64

	// StartPointOverride is the explicit auto-adjusted start-point override.
	StartPointOverride *float64

	// Envelope is the volume envelope curve.
	Envelope umath.Envelope

	// PitchBend is the pitch bend curve. It is a curve that maps time (in milliseconds) to pitch (in cents).
	PitchBend umath.Curve

	// Flags are the flags for passing to the resampler. These can be resampler-specific.
	Flags Flags
}

// Sequencer is the interface for something that can produce a [Sequence].
type Sequencer interface {
	// Sequence produces a [Sequence].
	Sequence() (Sequence, error)
}

// Len returns the sequence's length in MIDI ticks.
func (s Sequence) Len() int {
	len := 0
	for _, note := range s.Notes {
		len += note.Duration
	}
	return len
}

// Duration returns the sequence's length as a [time.Duration].
func (s Sequence) Duration() time.Duration {
	return timeutil.TicksToDuration(s.Len(), s.Metadata.Resolution, s.Metadata.Tempo)
}
