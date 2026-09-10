// Package resample provides the Resampler interface and related types.
//
// Resamplers are responsible for rendering a note given an input voice sample and
// resampling parameters (pitch, velocity, etc.). It has nothing to do with sample rate conversion;
// it purely refers to the UTAU resampler and the process of rendering a note from a base voice sample.
//
// Some resamplers are also capable of analysis and using per-voice sample analysis sidecar files
// (e.g. .frq) for caching F0/spectral features and improving resampling speed. These resamplers implement the [Analyzer] interface.
package resampler

import (
	"io"

	"github.com/SladkyCitron/gotau/sequence"
	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/aio"
	"gitlab.com/gomidi/midi/v2"
)

// Resampler resamples an input voice sample into a rendered note waveform.
type Resampler interface {
	// ID returns a unique identifier for this resampler. It is used for hashing and caching purposes.
	ID() string

	// Resample renders a note from the given input sample using the provided
	// resampling configuration (pitch, velocity, oto settings, pitch bend, etc.).
	Resample(in aio.SampleReader, cfg ResampleConfig) (aio.SampleReader, error)
}

// Analyzer is the interface for resamplers that are capable of analysis
// and using per-sample analysis sidecar files (e.g. .frq, .llsm, .pmk) for F0/spectral features.
type Analyzer interface {
	Resampler

	// ResampleWithAnalysis renders a note from the given input sample using the provided
	// resampling configuration (pitch, velocity, oto settings, pitch bend, etc.) and
	// analysis sidecar file. If analysis is nil, the resampler can generate a new one.
	ResampleWithAnalysis(in aio.SampleReader, analysis io.Reader, cfg ResampleConfig) (aio.SampleReader, error)

	// Analyze analyzes the given input sample and generates an analysis sidecar file.
	// The file format and extension is determined by [Analyzer.AnalysisExt].
	// Closing is the caller's responsibility.
	Analyze(in aio.SampleReader, format afmt.Format) (io.ReadCloser, error)

	// AnalysisExt returns the file extension of the analysis sidecar file format
	// used by this resampler (e.g. ".frq").
	AnalysisExt() string
}

// ResampleConfig represents the configuration for passing into [Resampler.Resample].
type ResampleConfig struct {
	// Pitch is the MIDI note number to resample to.
	Pitch midi.Note

	// Velocity is the velocity. It is a value between 0 and 1, where 1 is the maximum velocity.
	Velocity float64

	// Flags is a string of flags to pass to the resampler. These can be resampler-specific.
	Flags sequence.Flags

	// Offset is the offset time in milliseconds (from oto).
	Offset float64

	// Length is the desired length of the final resampled note in milliseconds.
	Length float64

	// Consonant is the consonant duration in milliseconds (from oto).
	Consonant float64

	// Cutoff is the cutoff time in milliseconds (from oto).
	Cutoff float64

	// Intensity is the volume of the note.
	Intensity float64

	// Modulation is the modulation (vibrato) depth.
	Modulation float64

	// Tempo is the tempo in BPM.
	Tempo float64

	// PitchBend is the pitch bend curve. It is a slice of pitches in cents, sampled at regular intervals (every 5 ticks).
	PitchBend []float64

	// AudioFormat is the audio format of the input and output audio data.
	AudioFormat afmt.Format
}
