package ust

import (
	"gitlab.com/gomidi/midi/v2"
)

// Note represents a note in an UST file.
type Note struct {
	Length               int        // Length is the duration in ticks.
	Lyric                string     // Lyric is the lyric or phoneme to be sung.
	NoteNum              midi.Note  // NoteNum is the MIDI note number.
	Intensity            float64    // Intensity is the loudness or intensity of the note (default = 100).
	Velocity             float64    // Velocity affects timing (smaller = more rushed; rarely used).
	Modulation           float64    // Modulation is the modulation depth, mostly used for vibrato (default = 0).
	Preutterance         *float64   // PreUtterance is the duration (in milliseconds) before note to start playback (in OTO). If it's omitted, falls back to OTO defaults.
	PreutteranceOverride *float64   // PreutteranceOverride is the explicit auto-adjusted pre-utterance override.
	VoiceOverlap         *float64   // VoiceOverlap is the amount of overlap into the previous note. If it's omitted, falls back to OTO defaults.
	StartPoint           float64    // StartPoint is the time where to begin sampling inside the audio file (in milliseconds).
	VoiceOverlapOverride *float64   // VoiceOverlapOverride is the explicit auto-adjusted overlap override.
	StartPointOverride   *float64   // StartPointOverride is the explicit auto-adjusted start-point override.
	Envelope             *Envelope  // Envelope is the volume envelope.
	PitchBend            *PitchBend // PitchBend is the pitch bend data.
	Flags                string     // Flags is a string of flags for passing to the resampler. These can be resampler-specific.
}

// IsLyricRest checks whether the lyrics is a rest / pause (e.g. "-", "R").
func IsLyricRest(s string) bool {
	return s == "-" || s == "R"
}
