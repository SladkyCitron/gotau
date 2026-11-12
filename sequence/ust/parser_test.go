package ust_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SladkyCitron/gotau/sequence/ust"
	"github.com/SladkyCitron/gotau/umath"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gomidi/midi/v2"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		name         string
		expectedFile *ust.File
		expectErr    bool
		errContains  string
	}{
		{
			name: "Full",
			expectedFile: &ust.File{
				Version: ust.Version2_0,
				Settings: ust.Settings{
					Tempo:       120,
					ProjectName: "Full",
					Project:     "_testdata/Full.ust",
					VoiceDir:    "path/to/voicebank",
					OutFile:     "path/to/output.wav",
					CacheDir:    "path/to/cache",
					Tool1:       "wavtool",
					Tool2:       "resampler",
					Mode2:       true,
				},
				Notes: []ust.Note{
					{
						Length:       720,
						Lyric:        "a",
						NoteNum:      midi.Note(69),
						Intensity:    100,
						Velocity:     100,
						Modulation:   0,
						Preutterance: float64Ptr(42),
						VoiceOverlap: float64Ptr(42),
						StartPoint:   42,
						Envelope: &ust.Envelope{
							P1:         0,
							P2:         5,
							P3:         35,
							V1:         0,
							V2:         100,
							V3:         100,
							V4:         0,
							P4:         0,
							P5:         10,
							V5:         100,
							HasRelease: true,
						},
						PitchBend: &ust.PitchBend{
							Type:   5,
							Start:  umath.XY[float64]{X: -40, Y: 0},
							Widths: []float64{65, 69},
							Ys:     []float64{0, 42},
							Modes:  []ust.PitchBendMode{"l", "s"},
						},
						Flags: "g0",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "EmptyOptionalValues",
			expectedFile: &ust.File{
				Version: ust.Version1_2,
				Settings: ust.Settings{
					Tempo:       120,
					ProjectName: "EmptyOptionalValues",
					VoiceDir:    "path/to/voicebank",
					OutFile:     "path/to/output.wav",
					Tool1:       "wavtool",
					Tool2:       "resampler",
					Mode2:       true,
				},
				Notes: []ust.Note{
					{
						Length:     720,
						Lyric:      "a",
						NoteNum:    midi.Note(69),
						Intensity:  100,
						Modulation: 0,
					},
				},
			},
			expectErr: false,
		},
		{
			name: "OpenUtau_UTF-8",
			expectedFile: &ust.File{
				Version: ust.Version1_2,
				Settings: ust.Settings{
					Tempo:    120,
					Project:  "C:\\Users\\matus\\Desktop\\test.ust",
					VoiceDir: "C:\\Users\\matus\\Documents\\OpenUtau\\Singers\\重音テト OU用日本語統合ライブラリー",
					CacheDir: "C:\\Users\\matus\\Documents\\OpenUtau\\Cache",
					Mode2:    true,
				},
				Notes: []ust.Note{
					{
						Length:     720,
						Lyric:      "a",
						NoteNum:    midi.Note(69),
						Velocity:   100,
						Intensity:  100,
						Modulation: 0,
						PitchBend: &ust.PitchBend{
							Type:   5,
							Start:  umath.XY[float64]{X: -40, Y: 0},
							Widths: []float64{65},
							Ys:     []float64{0},
							Modes:  []ust.PitchBendMode{"", ""},
						},
						Flags: "g0B0H0P86",
					},
				},
			},
		},
		{
			name: "OpenUtau_ShiftJIS",
			expectedFile: &ust.File{
				Version: ust.Version1_2,
				Settings: ust.Settings{
					Tempo:    120,
					Project:  "C:\\Users\\matus\\Desktop\\test.ust",
					VoiceDir: "C:\\Users\\matus\\Documents\\OpenUtau\\Singers\\重音テト OU用日本語統合ライブラリー",
					CacheDir: "C:\\Users\\matus\\Documents\\OpenUtau\\Cache",
					Mode2:    true,
				},
				Notes: []ust.Note{
					{
						Length:     720,
						Lyric:      "a",
						NoteNum:    midi.Note(69),
						Velocity:   100,
						Intensity:  100,
						Modulation: 0,
						PitchBend: &ust.PitchBend{
							Type:   5,
							Start:  umath.XY[float64]{X: -40, Y: 0},
							Widths: []float64{65},
							Ys:     []float64{0},
							Modes:  []ust.PitchBendMode{"", ""},
						},
						Flags: "g0B0H0P86",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join("testdata", test.name+".ust"))
			if err != nil {
				t.Fatal(err)
			}

			file, err := ust.Parse(b)
			if test.expectErr {
				assert.Error(t, err)
				assert.Nil(t, file)
				if test.errContains != "" {
					assert.Contains(t, err.Error(), test.errContains)
				}
			} else {
				assert.NoError(t, err)
				fileIsEqual(t, test.expectedFile, file)
			}
		})
	}
}

func fileIsEqual(t *testing.T, expect, actual *ust.File) {
	assert.Equal(t, expect.Version, actual.Version)
	assert.Equal(t, expect.Settings, actual.Settings)
	assert.Equal(t, expect.Notes, actual.Notes)
}

func float64Ptr(v float64) *float64 {
	return &v
}
