// Package external implements an external resampler.
package external

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/SladkyCitron/gotau/pitch"
	"github.com/SladkyCitron/gotau/resampler"
	"github.com/SladkyCitron/resona/afmt"
	"github.com/SladkyCitron/resona/aio"
	"github.com/SladkyCitron/resona/codec/wav"
	"github.com/zeebo/xxh3"
)

var _ resampler.Analyzer = (*Resampler)(nil)

// Resampler is a resampler that uses an external command-line UTAU resampler program to perform resampling.
type Resampler struct {
	// ConfigureCmd is an optional hook that allows configuring the exec.Cmd before running it.
	ConfigureCmd func(cmd *exec.Cmd)

	cmdName     string
	sampleFmt   afmt.SampleFormat
	analysisExt string
}

// New creates a new [Resampler] with the given program name and
// sample format for encoding temporary WAV files for passing into the resampler.
//
// The program should be a command-line UTAU resampler (e.g. resampler, moresampler, straycat, etc.)
// that accepts input and output WAV file paths, analysis sidecar files (optional), and
// other parameters as arguments and processes the input WAV file accordingly.
func New(name string, analysisExt string, sampleFmt afmt.SampleFormat) *Resampler {
	return &Resampler{cmdName: name, sampleFmt: sampleFmt, analysisExt: analysisExt}
}

func (r *Resampler) ID() string {
	return fmt.Sprintf("external:%s:%s", r.cmdName, r.analysisExt)
}

func (r *Resampler) Resample(in aio.SampleReader, cfg resampler.ResampleConfig) (aio.SampleReader, error) {
	input, err := r.createTempWav(in, cfg)
	if err != nil {
		return nil, fmt.Errorf("external: failed to create temporary wav file: %w", err)
	}

	output := input[:len(input)-len(filepath.Ext(input))] + "-out.wav"

	if err := r.runCmd(input, output, cfg); err != nil {
		return nil, err
	}

	if err := os.Remove(input); err != nil {
		return nil, fmt.Errorf("external: failed to remove temporary wav file: %w", err)
	}

	out, err := r.decodeOutFile(output)
	if err != nil {
		return nil, fmt.Errorf("external: failed to decode output wav file: %w", err)
	}

	return out, nil
}

func (r *Resampler) ResampleWithAnalysis(in aio.SampleReader, analysis io.Reader, cfg resampler.ResampleConfig) (aio.SampleReader, error) {
	if analysis == nil {
		return r.Resample(in, cfg)
	}

	input, err := r.createTempWav(in, cfg)
	if err != nil {
		return nil, fmt.Errorf("external: failed to create temporary wav file: %w", err)
	}

	analysisPath, err := r.createTempAnalysis(analysis, input)
	if err != nil {
		return nil, fmt.Errorf("external: failed to create temporary analysis sidecar file: %w", err)
	}

	output := input[:len(input)-len(filepath.Ext(input))] + "-out.wav"

	if err := r.runCmd(input, output, cfg); err != nil {
		return nil, err
	}

	if err := os.Remove(input); err != nil {
		return nil, fmt.Errorf("external: failed to remove temporary wav file: %w", err)
	}

	if err := os.Remove(analysisPath); err != nil {
		return nil, fmt.Errorf("external: failed to remove temporary wav file: %w", err)
	}

	out, err := r.decodeOutFile(output)
	if err != nil {
		return nil, fmt.Errorf("external: failed to decode output wav file: %w", err)
	}

	return out, nil
}

func (r *Resampler) Analyze(in aio.SampleReader, format afmt.Format) (io.ReadCloser, error) {
	input, err := os.CreateTemp("", "gotau-externalresampler-analysis-*.wav")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.Remove(input.Name()) }() // clean up

	var wavFormat uint16
	switch r.sampleFmt.Encoding {
	case afmt.SampleEncodingInt, afmt.SampleEncodingUint:
		wavFormat = wav.FormatInt
	case afmt.SampleEncodingFloat:
		wavFormat = wav.FormatFloat
	default:
		_ = input.Close()
		return nil, fmt.Errorf("invalid sample format: %s", r.sampleFmt.String())
	}
	enc, err := wav.NewEncoder(input, format, r.sampleFmt, wavFormat)
	if err != nil {
		_ = input.Close()
		return nil, err
	}

	if _, err := aio.Copy(enc, in); err != nil {
		_ = input.Close()
		return nil, err
	}

	if err := enc.Close(); err != nil {
		_ = input.Close()
		return nil, err
	}

	if err := input.Close(); err != nil {
		return nil, err
	}

	dummyOutput := input.Name()[:len(input.Name())-len(filepath.Ext(input.Name()))] + "-out.wav"

	cmd := exec.Command(r.cmdName, input.Name(), dummyOutput, "0", "0", "GN")
	if r.ConfigureCmd != nil {
		r.ConfigureCmd(cmd)
	}
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("external: failed to run resampler command: %w", err)
	}

	if err := os.Remove(dummyOutput); err != nil {
		return nil, fmt.Errorf("external: failed to remove dummy output file: %w", err)
	}

	// most resamplers accept e.g. something_wav.frq
	ext := filepath.Ext(input.Name())
	extless := input.Name()[:len(input.Name())-len(ext)]
	analysisPath := extless + strings.ReplaceAll(ext, ".", "_") + r.analysisExt

	return openTemp(analysisPath)
}

func (r *Resampler) AnalysisExt() string {
	return r.analysisExt
}

func (r *Resampler) runCmd(input, output string, cfg resampler.ResampleConfig) error {
	flags := "?"
	if len(cfg.Flags) > 0 {
		flags = cfg.Flags.String()
	}

	cmd := exec.Command(
		r.cmdName,
		input,
		output,
		strconv.FormatInt(int64(cfg.Pitch), 10),
		strconv.FormatInt(int64(cfg.Velocity*100), 10),
		flags,
		strconv.FormatFloat(cfg.Offset, 'f', -1, 64),
		strconv.FormatFloat(cfg.Length, 'f', -1, 64),
		strconv.FormatFloat(cfg.Consonant, 'f', -1, 64),
		strconv.FormatFloat(cfg.Cutoff, 'f', -1, 64),
		strconv.FormatInt(int64(cfg.Intensity*100), 10),
		strconv.FormatInt(int64(cfg.Modulation*100), 10),
		"!"+strconv.FormatFloat(cfg.Tempo, 'f', -1, 64), // apparently the tempo starts with "!" and not "T"???
		pitch.EncodeResamplerPitchBendString(cfg.PitchBend),
	)
	if r.ConfigureCmd != nil {
		r.ConfigureCmd(cmd)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("external: failed to run resampler command: %w", err)
	}

	return nil
}

func (r *Resampler) createTempWav(in aio.SampleReader, cfg resampler.ResampleConfig) (string, error) {
	// create filename
	h := xxh3.New()
	_, _ = h.WriteString(r.cmdName)
	_, _ = h.Write([]byte{byte(cfg.Pitch)})
	_ = binary.Write(h, binary.LittleEndian, cfg.Velocity)
	flags, _ := cfg.Flags.MarshalBinary()
	_, _ = h.Write(flags)
	_ = binary.Write(h, binary.LittleEndian, cfg.Offset)
	_ = binary.Write(h, binary.LittleEndian, cfg.Length)
	_ = binary.Write(h, binary.LittleEndian, cfg.Consonant)
	_ = binary.Write(h, binary.LittleEndian, cfg.Cutoff)
	_ = binary.Write(h, binary.LittleEndian, cfg.Intensity)
	_ = binary.Write(h, binary.LittleEndian, cfg.Modulation)
	_ = binary.Write(h, binary.LittleEndian, cfg.Tempo)
	_ = binary.Write(h, binary.LittleEndian, uint64(len(cfg.PitchBend)))
	_ = binary.Write(h, binary.LittleEndian, cfg.PitchBend)
	path := filepath.Join(os.TempDir(), fmt.Sprintf("gotau-externalresampler-%016x.wav", h.Sum64()))

	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var wavFormat uint16
	switch r.sampleFmt.Encoding {
	case afmt.SampleEncodingInt, afmt.SampleEncodingUint:
		wavFormat = wav.FormatInt
	case afmt.SampleEncodingFloat:
		wavFormat = wav.FormatFloat
	default:
		return "", fmt.Errorf("invalid sample format: %s", r.sampleFmt.String())
	}
	enc, err := wav.NewEncoder(f, cfg.AudioFormat, r.sampleFmt, wavFormat)
	if err != nil {
		return "", err
	}

	if _, err := aio.Copy(enc, in); err != nil {
		return "", err
	}

	if err := enc.Close(); err != nil {
		return "", err
	}

	return path, nil
}

func (r *Resampler) createTempAnalysis(analysis io.Reader, wavPath string) (string, error) {
	// most resamplers accept e.g. something_wav.frq
	ext := filepath.Ext(wavPath)
	extless := wavPath[:len(wavPath)-len(ext)]
	newPath := extless + strings.ReplaceAll(ext, ".", "_") + r.analysisExt

	f, err := os.Create(newPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, analysis); err != nil {
		return "", err
	}

	return newPath, nil
}

func (r *Resampler) decodeOutFile(path string) (aio.SampleReader, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	deco, err := wav.NewDecoder(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	if err := os.Remove(path); err != nil {
		return nil, err
	}

	return deco, nil
}
