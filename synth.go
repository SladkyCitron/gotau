package gotau

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math"
	"path/filepath"
	"slices"
	"strings"

	"github.com/SladkyCitron/gotau/cache"
	"github.com/SladkyCitron/gotau/concat"
	"github.com/SladkyCitron/gotau/phonemizer"
	"github.com/SladkyCitron/gotau/resampler"
	"github.com/SladkyCitron/gotau/sequence"
	"github.com/SladkyCitron/gotau/umath"
	"github.com/SladkyCitron/gotau/voicebank"
	"github.com/SladkyCitron/gotau/voicebank/otoini"
	"github.com/SladkyCitron/gotau/voicebank/prefixmap"
	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/aio"
	"github.com/SladkyCitron/resona/codec"
	_ "github.com/SladkyCitron/resona/codec/au"
	_ "github.com/SladkyCitron/resona/codec/qoa"
	"github.com/SladkyCitron/resona/codec/wav"
	"github.com/SladkyCitron/resona/freq"
)

// Synth is the main singing voice synthsizer that renders notes into audio samples.
type Synth struct {
	vb        *voicebank.Voicebank
	ph        phonemizer.Phonemizer
	res       resampler.Resampler
	cat       concat.Concatenator
	resCache  cache.Cache
	sched     *scheduler
	sr        int
	renderBuf []float32
	readPos   int
	prevNote  *sequence.Note
}

// New creates a new [Synth] with the given sample rate, voicebank, resampler, and concatenator.
func New(sr int, vb *voicebank.Voicebank, res resampler.Resampler, cat concat.Concatenator) *Synth {
	s := &Synth{
		vb:        vb,
		ph:        &phonemizer.Default{},
		res:       res,
		cat:       cat,
		resCache:  &cache.NopCache{},
		sched:     &scheduler{},
		sr:        sr,
		renderBuf: make([]float32, 0, sr), // 1 second buffer by default
	}
	return s
}

// Buffer controls memory allocation by the Synth.
// It sets the internal buffer to use when rendering notes.
// The contents of the buffer are ignored.
func (s *Synth) Buffer(buf []float32) {
	s.renderBuf = buf[0:cap(buf)]
}

// SetPhonemizer sets the phonemizer.
func (s *Synth) SetPhonemizer(ph phonemizer.Phonemizer) {
	s.ph = ph
}

// SetResamplerCache sets the cache for storing resampled notes.
func (s *Synth) SetResamplerCache(c cache.Cache) {
	s.resCache = c
}

// SetResolution sets the timing resolution in ticks per quarter note (TPQN).
//
// Higher values increase timing precision but may result in more scheduling
// overhead.
func (s *Synth) SetResolution(resolution int) {
	s.sched.tpqn = resolution
}

// SetTempo sets the playback tempo in beats per minute (BPM).
func (s *Synth) SetTempo(tempo float64) {
	s.sched.bpm = tempo
}

// Enqueue adds notes to the synthesis queue.
//
// Notes are scheduled according to their tick position and will be rendered
// in order during subsequent ReadSamples calls.
func (s *Synth) Enqueue(notes ...sequence.Note) {
	s.sched.enqueue(notes...)
}

// EnqueueSequence adds all notes from the given sequence to the synthesis
// queue and updates the synthesizer's timing parameters.
//
// The sequence's resolution and tempo override the current scheduler settings.
func (s *Synth) EnqueueSequence(seq sequence.Sequence) {
	s.SetResolution(seq.Metadata.Resolution)
	s.SetTempo(seq.Metadata.Tempo)
	s.Enqueue(seq.Notes...)
}

func (s *Synth) ReadSamples(p []float32) (int, error) {
	n := 0
	// fill the buffer and output as much as possible
	for n < len(p) {
		// if the whole buffer has been read, render more notes
		for s.readPos >= len(s.renderBuf) {
			if len(s.sched.queue) == 0 {
				if n == 0 {
					return 0, io.EOF
				}
				return n, nil
			}

			note, _ := s.sched.pop()
			var prev *sequence.Note
			if s.prevNote != nil {
				prev = s.prevNote
			}
			var next *sequence.Note
			if peek, ok := s.sched.peek(); ok {
				next = &peek
			}

			if err := s.renderNotes(note, prev, next); err != nil {
				return n, fmt.Errorf("gotau Synth: failed to render note %q: %w", note.Lyric, err)
			}
			s.prevNote = &note
		}

		copied := copy(p[n:], s.renderBuf[s.readPos:])
		s.readPos += copied
		n += copied
	}
	return n, nil
}

func (s *Synth) renderNotes(note sequence.Note, prev *sequence.Note, next *sequence.Note) error {
	s.debugLog("note", note)

	notes := []sequence.Note{note}

	phNotes := make([]phonemizer.Note, len(notes))
	for i := range notes {
		phNotes[i] = phonemizer.Note{
			Position: notes[i].Position,
			Duration: notes[i].Duration,
			Lyric:    notes[i].Lyric,
			Note:     notes[i].Note,
		}
	}

	var phPrev *phonemizer.Note
	if prev != nil {
		phPrev = &phonemizer.Note{
			Position: prev.Position,
			Duration: prev.Duration,
			Lyric:    prev.Lyric,
			Note:     prev.Note,
		}
	}

	var phNext *phonemizer.Note
	if next != nil {
		phNext = &phonemizer.Note{
			Position: next.Position,
			Duration: next.Duration,
			Lyric:    next.Lyric,
			Note:     next.Note,
		}
	}

	// get preutterance and overlap of next note
	var nextPreutter float64
	var nextOverlap float64
	if next != nil {
		nextPhonemes := slices.Collect(s.ph.Phonemize([]phonemizer.Note{*phNext}, &phNotes[0], nil))
		if len(nextPhonemes) > 0 {
			var nextPrefix prefixmap.Prefix
			if s.vb.PrefixMap != nil {
				if entry, ok := s.vb.PrefixMap[phNext.Note]; ok {
					nextPrefix = entry
				}
			}
			if nextOtoEntry, ok := s.resolvePhoneme(nextPhonemes[0], nextPrefix); ok {
				nextPreutter = s.getPreutter(nextOtoEntry, *next)
				nextOverlap = s.getOverlap(nextOtoEntry, *next)
			}
		}
	}

	for ph := range s.ph.Phonemize(phNotes, phPrev, phNext) {
		targetNote := phNotes[ph.Index]

		var prefix prefixmap.Prefix
		if s.vb.PrefixMap != nil {
			if entry, ok := s.vb.PrefixMap[targetNote.Note]; ok {
				prefix = entry
			}
		}
		otoEntry, ok := s.resolvePhoneme(ph, prefix)

		if !ok {
			s.debugLog("fallback silence", note)
			silenceSec := s.sched.ticksToSeconds(note.Duration) - nextPreutter/1000
			silenceSec = math.Max(0, silenceSec) // guard to prevent runtime panics
			buf := make([]float32, int(silenceSec*float64(s.sr)))
			s.renderBuf = append(s.renderBuf, buf...)
			s.sched.tickPos = note.Position + note.Duration
			return nil
		}

		if err := s.renderSingleNote(notes[ph.Index], otoEntry, nextPreutter, nextOverlap, next); err != nil {
			return fmt.Errorf("failed to render phoneme %q: %w", otoEntry.Alias, err)
		}
	}
	s.sched.tickPos = note.Position + note.Duration
	return nil
}

func (s *Synth) renderSingleNote(note sequence.Note, otoEntry otoini.Entry, nextPreutterMs float64, nextOverlap float64, next *sequence.Note) error {
	// get preutterance of current note
	preutterMs := s.getPreutter(otoEntry, note)
	preutterSec := preutterMs / 1000

	// emit possible silence before note
	if startTick := note.Position - s.sched.secondsToTicks(preutterSec); startTick > s.sched.tickPos {
		s.debugLog("silence", note)
		silenceSec := s.sched.ticksToSeconds(note.Position-s.sched.tickPos) - preutterSec
		silenceSec = math.Max(0, silenceSec) // guard to prevent runtime panics
		buf := make([]float32, int(silenceSec*float64(s.sr)))
		var err error
		s.renderBuf, err = s.cat.Concatenate(s.renderBuf, buf, concat.ConcatenateConfig{
			Duration:    startTick - s.sched.tickPos,
			Tempo:       s.sched.bpm,
			Resolution:  s.sched.tpqn,
			AudioFormat: afmt.Format{SampleRate: freq.Frequency(s.sr) * freq.Hertz, NumChannels: 1},
			Envelope: umath.Envelope{
				{X: 0, Y: 0},
				{X: 0, Y: 0},
				{X: 0, Y: 0},
				{X: 0, Y: 0},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to concatenate silence: %w", err)
		}
		s.sched.tickPos = startTick
	}

	durationMs := s.sched.ticksToSeconds(note.Duration) * 1000
	timing := CalculateResamplerTiming(otoEntry, note, durationMs, preutterMs, next, nextPreutterMs, nextOverlap)
	pitchLeadingMs := timing.Leading

	// generate pitch bend curve
	// the timing math is probably wrong, thus the NaNs
	// also we'll probably need to lerp the pitches with the previous ones and also crossfade them with the prev note's ones
	const pitchIntervalTicks = 5
	positionMs := s.sched.ticksToSeconds(note.Position) * 1000
	pitchCountMs := (positionMs + timing.Length) - (positionMs - pitchLeadingMs)
	pitchCount := int(math.Ceil(float64(s.sched.secondsToTicks(pitchCountMs/1000)) / 5))
	pitchCount = max(0, pitchCount)
	pitchBend := make([]float64, pitchCount)
	pitchSampleStartMs := positionMs - pitchLeadingMs
	pitchIntervalMs := s.sched.ticksToSeconds(pitchIntervalTicks) * 1000
	for i := range pitchBend {
		samplePosMs := pitchSampleStartMs + float64(i)*pitchIntervalMs
		pitch := note.PitchBend.AtClamped(samplePosMs - positionMs)
		if math.IsNaN(pitch) {
			pitch = 0
		}
		pitchBend[i] = pitch
	}

	resampleCfg := resampler.ResampleConfig{
		Pitch:       note.Note,
		Velocity:    note.Velocity,
		Flags:       note.Flags,
		Offset:      timing.Offset,
		Length:      timing.Length,
		Consonant:   timing.Consonant,
		Cutoff:      timing.Cutoff,
		Intensity:   note.Intensity,
		Modulation:  note.Modulation,
		Tempo:       s.sched.bpm,
		PitchBend:   pitchBend,
		AudioFormat: afmt.Format{SampleRate: freq.Frequency(s.sr) * freq.Hertz, NumChannels: 1},
	}

	fileinfo, err := fs.Stat(s.vb.FS(), otoEntry.FilePath())
	if err != nil {
		return fmt.Errorf("failed to stat voicebank audio file: %w", err)
	}

	var resampled aio.SampleReader
	var doCache bool
	key := s.getKeyFunc(resampleCfg, otoEntry.FilePath(), fileinfo)
	ctx := context.Background()
	if rc, err := s.resCache.Open(ctx, key); err == nil {
		resampled, err = wav.NewDecoder(rc)
		if err != nil {
			return err
		}
	} else {
		f, err := s.vb.FS().Open(otoEntry.FilePath())
		if err != nil {
			return err
		}
		defer f.Close()

		deco, _, err := codec.Decode(f)
		if err != nil {
			return err
		}
		if sr := int(deco.Format().SampleRate.Hertz()); sr != s.sr {
			return fmt.Errorf("voicebank (%d Hz) and synth (%d Hz) sample rates do not match", sr, s.sr)
		}

		if analyzer, ok := s.res.(resampler.Analyzer); ok {
			// check if there's the analysis sidecar file available
			ext := filepath.Ext(otoEntry.FilePath())
			name := otoEntry.FilePath()[:len(otoEntry.FilePath())-len(ext)]
			analysisPath := name + strings.ReplaceAll(ext, ".", "_") + analyzer.AnalysisExt()
			analysisFile, err := s.vb.FS().Open(analysisPath)
			if err == nil {
				resampled, err = analyzer.ResampleWithAnalysis(deco, analysisFile, resampleCfg)
				if err != nil {
					return fmt.Errorf("failed to resample: %w", err)
				}
				if err := analysisFile.Close(); err != nil {
					return fmt.Errorf("failed to close analysis sidecar file: %w", err)
				}
			} else {
				// nope
				resampled, err = s.res.Resample(deco, resampleCfg)
				if err != nil {
					return fmt.Errorf("failed to resample: %w", err)
				}
			}
		} else {
			resampled, err = s.res.Resample(deco, resampleCfg)
			if err != nil {
				return fmt.Errorf("failed to resample: %w", err)
			}
		}

		doCache = true
	}

	noteBuf, err := aio.ReadAll(resampled)
	if err != nil {
		return fmt.Errorf("failed to read resampled audio: %w", err)
	}

	var doCacheDone chan error
	if doCache {
		doCacheDone = make(chan error, 1)
		// cache the resampled audio
		go func() {
			var cacheErr error
			defer func() {
				doCacheDone <- cacheErr
			}()

			f, err := s.resCache.Create(ctx, key)
			if err != nil {
				_ = f.Abort()
				cacheErr = fmt.Errorf("failed to create cache entry: %w", err)
				return
			}

			enc, err := wav.NewEncoder(
				f,
				resampleCfg.AudioFormat,
				afmt.SampleFormat{BitDepth: 32, Encoding: afmt.SampleEncodingFloat, Endian: binary.LittleEndian},
				wav.FormatFloat,
			)
			if err != nil {
				_ = f.Abort()
				cacheErr = fmt.Errorf("failed to create wav encoder for caching: %w", err)
				return
			}

			if _, err := enc.WriteSamples(noteBuf); err != nil {
				_ = f.Abort()
				cacheErr = fmt.Errorf("failed to cache resampled audio: %w", err)
				return
			}

			if err := enc.Close(); err != nil {
				_ = f.Abort()
				cacheErr = fmt.Errorf("failed to close wav encoder for caching: %w", err)
				return
			}

			if err := f.Close(); err != nil {
				_ = f.Abort()
				cacheErr = fmt.Errorf("failed to close cache entry: %w", err)
				return
			}
		}()
	}

	correction := preutterMs - nextPreutterMs + nextOverlap
	concatCfg := concat.ConcatenateConfig{
		Offset:      s.getStartPoint(note),
		Duration:    note.Duration,
		Tempo:       s.sched.bpm,
		Resolution:  s.sched.tpqn,
		LengthDelta: correction,
		Overlap:     s.getOverlap(otoEntry, note),
		Envelope:    note.Envelope,
		AudioFormat: afmt.Format{SampleRate: freq.Frequency(s.sr) * freq.Hertz, NumChannels: 1},
	}
	s.renderBuf, err = s.cat.Concatenate(s.renderBuf, noteBuf, concatCfg)
	if err != nil {
		return fmt.Errorf("failed to concatenate: %w", err)
	}

	// wait for caching to finish and check error
	if doCache {
		if err := <-doCacheDone; err != nil {
			// log error instead???
			// it's non-critical (kinda) since it only affects caching
			return err
		}
	}

	return nil
}

func (s *Synth) resolvePhoneme(ph phonemizer.Phoneme, prefix prefixmap.Prefix) (e otoini.Entry, ok bool) {
	if ph.Error != nil {
		// log error??
		return otoini.Entry{}, false
	}

	for _, alias := range ph.Candidates {
		if e, ok = s.vb.Oto.Get(prefix.Prefix + alias + prefix.Suffix); ok {
			return e, true
		}
	}
	return otoini.Entry{}, false
}

func (s *Synth) getPreutter(otoEntry otoini.Entry, note sequence.Note) float64 {
	if note.PreutteranceOverride != nil {
		return *note.PreutteranceOverride
	}
	if note.Preutterance != nil {
		return *note.Preutterance
	}
	return otoEntry.Preutterance
}

func (s *Synth) getStartPoint(note sequence.Note) float64 {
	return effectiveStartPoint(note)
}

func effectiveStartPoint(note sequence.Note) float64 {
	if note.StartPointOverride != nil {
		return *note.StartPointOverride
	}
	return note.StartPoint * math.Pow(2, 1-note.Velocity/100)
}

func (s *Synth) getOverlap(otoEntry otoini.Entry, note sequence.Note) float64 {
	if note.VoiceOverlapOverride != nil {
		return *note.VoiceOverlapOverride
	}
	if note.VoiceOverlap != nil {
		return *note.VoiceOverlap
	}
	return otoEntry.Overlap
}

// ResamplerTiming contains the timing parameters for the resampler.
//
// It's used internally by [Synth] to determine how to render the note with the
// correct timing and pitch, but it's also exported for testing purposes.
type ResamplerTiming struct {
	Offset    float64
	Length    float64
	Consonant float64
	Cutoff    float64
	Leading   float64
	SkipOver  float64
}

// CalculateResamplerTiming calculates the timing parameters for the resampler
// based on the given note and its oto entry.
//
// It's used internally by [Synth] to determine how to render the note with the
// correct timing and pitch, but it's also exported for testing purposes.
func CalculateResamplerTiming(
	otoEntry otoini.Entry,
	note sequence.Note,
	durationMs float64,
	preutterMs float64,
	next *sequence.Note,
	nextPreutterMs float64,
	nextOverlapMs float64,
) ResamplerTiming {
	stretchRatio := math.Pow(2, 1-note.Velocity/100)
	pulledMs := math.Round((preutterMs/stretchRatio-otoEntry.Preutterance)*1000) / 1000
	pulledMs = math.Max(0, math.Min(pulledMs, math.Max(0, otoEntry.Offset)))

	offset := otoEntry.Offset - pulledMs
	leading := (otoEntry.Preutterance + pulledMs) * stretchRatio
	skipOver := leading - preutterMs
	consonant := otoEntry.Consonant + pulledMs
	cutoff := otoEntry.Cutoff
	if cutoff < 0 {
		cutoff -= pulledMs
	}

	durCorrection := preutterMs
	if next != nil && next.Position == note.Position+note.Duration {
		tailIntrude := math.Max(nextPreutterMs-nextOverlapMs, 0)
		tailOverlap := math.Max(nextOverlapMs, 0)
		durCorrection = preutterMs - tailIntrude + tailOverlap
	}

	length := durationMs + durCorrection + skipOver + effectiveStartPoint(note)
	length = math.Max(length, consonant)
	length = math.Ceil(length/50+0.5) * 50
	length = math.Max(0, length)

	return ResamplerTiming{
		Offset:    offset,
		Length:    length,
		Consonant: consonant,
		Cutoff:    cutoff,
		Leading:   leading,
		SkipOver:  skipOver,
	}
}

func (s *Synth) debugLog(msg string, note sequence.Note) {
	log.Printf("at %v -> %s: %v", s.sched.tickPos, msg, note)
}
