package voicebank_test

import (
	"bytes"
	"cmp"
	"testing"

	"github.com/SladkyCitron/gotau/voicebank"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gomidi/midi/v2"
	"golang.org/x/text/encoding/japanese"
)

func TestDecodePrefixMap(t *testing.T) {
	b := []byte("C5\tpre\tsuf\nD#5\tpre\t\n# comment\nF#5\t\tsuf\n")

	want := voicebank.PrefixMap{
		midi.Note(midi.C(6)):  voicebank.Prefix{"pre", "suf"},
		midi.Note(midi.Eb(6)): voicebank.Prefix{"pre", ""},
		midi.Note(midi.Gb(6)): voicebank.Prefix{"", "suf"},
	}

	prefixmap, err := voicebank.ParsePrefixMap(b)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, want, prefixmap)
}

func TestDecodePrefixMapWithOpts(t *testing.T) {
	b := []byte("C5 pre suf\nD#5 pre \n; comment\nF#5  suf\n")

	want := voicebank.PrefixMap{
		midi.Note(midi.C(6)):  voicebank.Prefix{"pre", "suf"},
		midi.Note(midi.Eb(6)): voicebank.Prefix{"pre", ""},
		midi.Note(midi.Gb(6)): voicebank.Prefix{"", "suf"},
	}

	prefixmap, err := voicebank.ParsePrefixMap(b,
		voicebank.PrefixMapWithEncoding(japanese.ShiftJIS),
		voicebank.PrefixMapWithComment(';'),
		voicebank.PrefixMapWithDelimiter(' '),
	)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, want, prefixmap)
}

func TestDecodePrefixMapAllNotes(t *testing.T) {
	b := []byte("C5\t\tfoo\n" +
		"C#5\t\tfoo\n" +
		"D5\t\tfoo\n" +
		"D#5\t\tfoo\n" +
		"E5\t\tfoo\n" +
		"F5\t\tfoo\n" +
		"F#5\t\tfoo\n" +
		"G5\t\tfoo\n" +
		"G#5\t\tfoo\n" +
		"A5\t\tfoo\n" +
		"A#5\t\tfoo\n" +
		"B5\t\tfoo\n")

	want := voicebank.PrefixMap{
		midi.Note(midi.C(6)):  voicebank.Prefix{"", "foo"},
		midi.Note(midi.Db(6)): voicebank.Prefix{"", "foo"},
		midi.Note(midi.D(6)):  voicebank.Prefix{"", "foo"},
		midi.Note(midi.Eb(6)): voicebank.Prefix{"", "foo"},
		midi.Note(midi.E(6)):  voicebank.Prefix{"", "foo"},
		midi.Note(midi.F(6)):  voicebank.Prefix{"", "foo"},
		midi.Note(midi.Gb(6)): voicebank.Prefix{"", "foo"},
		midi.Note(midi.G(6)):  voicebank.Prefix{"", "foo"},
		midi.Note(midi.Ab(6)): voicebank.Prefix{"", "foo"},
		midi.Note(midi.A(6)):  voicebank.Prefix{"", "foo"},
		midi.Note(midi.Bb(6)): voicebank.Prefix{"", "foo"},
		midi.Note(midi.B(6)):  voicebank.Prefix{"", "foo"},
	}

	prefixmap, err := voicebank.ParsePrefixMap(b)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, want, prefixmap)
}

func TestDecodePrefixMapInvalidLine(t *testing.T) {
	prefixmap, err := voicebank.ParsePrefixMap([]byte("C#5\tfoo\n"))
	assert.Nil(t, prefixmap)
	assert.ErrorContains(t, err, "voicebank prefix.map: invalid line")
}

func TestDecodePrefixMapInvalidNote(t *testing.T) {
	tests := []struct {
		name    string
		note    string
		wantErr string
	}{
		{name: "empty", note: "", wantErr: "invalid note format"},
		{name: "invalid base", note: "XX", wantErr: "invalid note format"},
		{name: "base only", note: "C", wantErr: "invalid note format"},
		{name: "missing octave", note: "C#", wantErr: "invalid note format"},
		{name: "double accidental", note: "C##5", wantErr: "invalid note format"},
		{name: "trailing data", note: "C5extra", wantErr: "invalid note format"},
		{name: "internal whitespace", note: "C 5", wantErr: "invalid note format"},
		{name: "unsupported base", note: "I5", wantErr: "invalid note format"},
		{name: "unsupported flat", note: "Cb5", wantErr: "invalid base"},
		{name: "unsupported sharp", note: "H#5", wantErr: "invalid base"},
		{name: "negative below range", note: "C-2", wantErr: "note out of MIDI range"},
		{name: "above range", note: "C11", wantErr: "note out of MIDI range"},
		{name: "octave overflow", note: "C999999999999999999999999", wantErr: "invalid octave"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefixmap, err := voicebank.ParsePrefixMap([]byte(tt.note + "\t\tfoo\n"))
			assert.Nil(t, prefixmap)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestPrefixMapRoundTrip(t *testing.T) {
	want := []byte("C5\tpre\tsuf\nD#5\tpre\t\nF#5\t\tsuf\n")

	prefixmap, err := voicebank.ParsePrefixMap(want)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err = prefixmap.Encode(&buf,
		voicebank.PrefixMapWithSort(func(a, b midi.Note) int {
			return cmp.Compare(a, b)
		}),
		voicebank.PrefixMapWithNoteFormat('s'),
	)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, want, buf.Bytes())
}
