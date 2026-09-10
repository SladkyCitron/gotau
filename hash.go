package gotau

import (
	"encoding/binary"
	"io"
	"io/fs"

	"github.com/SladkyCitron/gotau/cache"
	"github.com/SladkyCitron/gotau/resampler"
)

const keyVersion uint64 = 1

func (s *Synth) getKeyFunc(cfg resampler.ResampleConfig, path string, fileinfo fs.FileInfo) cache.KeyFunc {
	mtime, err := fileinfo.ModTime().MarshalBinary()
	if err != nil {
		mtime = []byte{}
	}

	flags, err := cfg.Flags.MarshalBinary()
	if err != nil {
		flags = []byte{}
	}

	return func(w io.Writer) {
		_, _ = w.Write([]byte("gotau-resampler"))
		_ = binary.Write(w, binary.LittleEndian, keyVersion)
		_, _ = w.Write([]byte(s.res.ID()))
		_, _ = w.Write([]byte(path))
		_ = binary.Write(w, binary.LittleEndian, fileinfo.Size())
		_, _ = w.Write(mtime)
		_, _ = w.Write([]byte{byte(cfg.Pitch)})
		_ = binary.Write(w, binary.LittleEndian, cfg.Velocity)
		_, _ = w.Write(flags)
		_ = binary.Write(w, binary.LittleEndian, cfg.Offset)
		_ = binary.Write(w, binary.LittleEndian, cfg.Length)
		_ = binary.Write(w, binary.LittleEndian, cfg.Consonant)
		_ = binary.Write(w, binary.LittleEndian, cfg.Cutoff)
		_ = binary.Write(w, binary.LittleEndian, cfg.Intensity)
		_ = binary.Write(w, binary.LittleEndian, cfg.Modulation)
		_ = binary.Write(w, binary.LittleEndian, cfg.Tempo)
		_ = binary.Write(w, binary.LittleEndian, uint64(len(cfg.PitchBend)))
		_ = binary.Write(w, binary.LittleEndian, cfg.PitchBend)
	}
}
