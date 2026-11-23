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

func TestParse(t *testing.T) {
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
						Velocity:   100,
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

func TestParseUST20Operations(t *testing.T) {
	data := []byte(`[#VERSION]
UST Version2.0
[#SETTING]
Tempo=120
Mode2=true
[#0000]
Length=480
Lyric=a
NoteNum=60
[#INSERT]
Length=240
Lyric=i
NoteNum=62
[#0002]
Length=960
Lyric=u
NoteNum=64
[#DELETE]
[#TRACKEND]
`)
	file, err := ust.Parse(data)
	assert.NoError(t, err)
	assert.Equal(t, ust.Version2_0, file.Version)
	assert.Equal(t, []ust.Note{
		{
			Length:    480,
			Lyric:     "a",
			NoteNum:   midi.Note(60),
			Intensity: 100,
			Velocity:  100,
		},
		{
			Length:    240,
			Lyric:     "i",
			NoteNum:   midi.Note(62),
			Intensity: 100,
			Velocity:  100,
		},
	}, file.Notes)
}

func TestParseUTF8BOM(t *testing.T) {
	data := append([]byte{0xef, 0xbb, 0xbf}, []byte(`[#VERSION]
UST Version2.0
[#SETTING]
Tempo=120
Mode2=true
[#0000]
Length=480
Lyric=a
NoteNum=60
[#TRACKEND]
`)...)

	file, err := ust.Parse(data)
	assert.NoError(t, err)
	assert.Equal(t, ust.Version2_0, file.Version)
	assert.Len(t, file.Notes, 1)
	assert.Equal(t, "a", file.Notes[0].Lyric)
}

func TestParseUTF8WithoutCharset(t *testing.T) {
	data := []byte(`[#SETTING]
Tempo=120
Mode2=true
[#0000]
Length=480
Lyric=あ
NoteNum=60
[#TRACKEND]
`)

	file, err := ust.Parse(data)
	assert.NoError(t, err)
	assert.Equal(t, "あ", file.Notes[0].Lyric)
}

func TestParseMode1PitchBend(t *testing.T) {
	data := []byte(`[#SETTING]
Tempo=120
Mode2=false
[#0000]
Length=480
Lyric=a
NoteNum=60
PBType=5
PBStart=-10
PitchBend=0,100,-50
[#TRACKEND]
`)

	file, err := ust.Parse(data)
	assert.NoError(t, err)
	assert.Equal(t, []float64{0, 100, -50}, file.Notes[0].PitchBend.Pitches)
	assert.Equal(t, float64(-10), file.Notes[0].PitchBend.Start.X)
}

func TestParseUST20Overrides(t *testing.T) {
	data := []byte(`[#VERSION]
UST Version2.0
[#SETTING]
Tempo=120
Mode2=false
[#0000]
Length=480
Lyric=a
NoteNum=60
PreUtterance=8
VoiceOverlap=8
StartPoint=3
@preuttr=6.25
@overlap=4.5
@stpoint=12.25
[#TRACKEND]
`)

	file, err := ust.Parse(data)
	assert.NoError(t, err)
	assert.Equal(t, float64Ptr(6.25), file.Notes[0].PreutteranceOverride)
	assert.Equal(t, float64Ptr(4.5), file.Notes[0].VoiceOverlapOverride)
	assert.Equal(t, float64Ptr(12.25), file.Notes[0].StartPointOverride)
}

func TestParseShiftJISWithoutCharset(t *testing.T) {
	lyric := string([]byte{0x82, 0xa0}) // あ in Shift-JIS
	data := []byte(`[#SETTING]
Tempo=120
Mode2=true
[#0000]
Length=480
Lyric=` + lyric + `
NoteNum=60
[#TRACKEND]
`)

	file, err := ust.Parse(data)
	assert.NoError(t, err)
	assert.Equal(t, "あ", file.Notes[0].Lyric)
}

func TestParseExplicitCharsetTakesPrecedence(t *testing.T) {
	data := []byte(`Charset=not-a-real-charset
[#SETTING]
Tempo=120
Mode2=true
[#0000]
Length=480
Lyric=a
NoteNum=60
[#TRACKEND]
`)

	file, err := ust.Parse(data)
	assert.Nil(t, file)
	assert.ErrorContains(t, err, "unsupported charset: not-a-real-charset")
}

func TestParseInvalidTempo(t *testing.T) {
	data := []byte(`[#SETTING]
Tempo=-1
`)
	file, err := ust.Parse(data)
	assert.Nil(t, file)
	assert.ErrorContains(t, err, "invalid tempo: -1")
}

func TestParseInvalidNoteNum(t *testing.T) {
	data := []byte(`[#0000]
Length=480
Lyric=a
NoteNum=128
[#TRACKEND]
`)
	file, err := ust.Parse(data)
	assert.Nil(t, file)
	assert.ErrorContains(t, err, "invalid MIDI note number: 128")
}

func TestParseInvalidLength(t *testing.T) {
	data := []byte(`[#0000]
Length=-1
Lyric=a
NoteNum=60
[#TRACKEND]
`)
	file, err := ust.Parse(data)
	assert.Nil(t, file)
	assert.ErrorContains(t, err, "invalid note length: -1")
}

func FuzzParse(f *testing.F) {
	f.Add([]byte(`[#VERSION]
UST Version2.0
[#SETTING]
Tempo=120
Mode2=true
[#0000]
Length=480
Lyric=a
NoteNum=60
[#TRACKEND]
`))

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ust.Parse(data)
	})
}

func fileIsEqual(t *testing.T, expect, actual *ust.File) {
	assert.Equal(t, expect.Version, actual.Version)
	assert.Equal(t, expect.Settings, actual.Settings)
	assert.Equal(t, expect.Notes, actual.Notes)
}

func float64Ptr(v float64) *float64 {
	return &v
}
