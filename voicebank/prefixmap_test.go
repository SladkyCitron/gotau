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
		midi.Note(midi.C(5)):  voicebank.Prefix{"pre", "suf"},
		midi.Note(midi.Eb(5)): voicebank.Prefix{"pre", ""},
		midi.Note(midi.Gb(5)): voicebank.Prefix{"", "suf"},
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
		midi.Note(midi.C(5)):  voicebank.Prefix{"pre", "suf"},
		midi.Note(midi.Eb(5)): voicebank.Prefix{"pre", ""},
		midi.Note(midi.Gb(5)): voicebank.Prefix{"", "suf"},
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
		"Db5\t\tfoo\n" +
		"D5\t\tfoo\n" +
		"D#5\t\tfoo\n" +
		"Eb5\t\tfoo\n" +
		"E5\t\tfoo\n" +
		"F5\t\tfoo\n" +
		"F#5\t\tfoo\n" +
		"Gb5\t\tfoo\n" +
		"G5\t\tfoo\n" +
		"G#5\t\tfoo\n" +
		"Ab5\t\tfoo\n" +
		"A5\t\tfoo\n" +
		"A#5\t\tfoo\n" +
		"Bb5\t\tfoo\n" +
		"Hb5\t\tfoo\n" +
		"B5\t\tfoo\n" +
		"H5\t\tfoo\n")

	want := voicebank.PrefixMap{
		midi.Note(midi.C(5)):  voicebank.Prefix{"", "foo"},
		midi.Note(midi.Db(5)): voicebank.Prefix{"", "foo"},
		//midi.Note(midi.Db(5)):  voicebank.Prefix{"", "foo"},
		midi.Note(midi.D(5)):  voicebank.Prefix{"", "foo"},
		midi.Note(midi.Eb(5)): voicebank.Prefix{"", "foo"},
		//midi.Note(midi.Eb(5)):  voicebank.Prefix{"", "foo"},
		midi.Note(midi.E(5)):  voicebank.Prefix{"", "foo"},
		midi.Note(midi.F(5)):  voicebank.Prefix{"", "foo"},
		midi.Note(midi.Gb(5)): voicebank.Prefix{"", "foo"},
		//midi.Note(midi.Gb(5)):  voicebank.Prefix{"", "foo"},
		midi.Note(midi.G(5)):  voicebank.Prefix{"", "foo"},
		midi.Note(midi.Ab(5)): voicebank.Prefix{"", "foo"},
		//midi.Note(midi.Ab(5)):  voicebank.Prefix{"", "foo"},
		midi.Note(midi.A(5)):  voicebank.Prefix{"", "foo"},
		midi.Note(midi.Bb(5)): voicebank.Prefix{"", "foo"},
		//midi.Note(midi.Bb(5)):  voicebank.Prefix{"", "foo"},
		midi.Note(midi.B(5)): voicebank.Prefix{"", "foo"},
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
	prefixmap, err := voicebank.ParsePrefixMap([]byte("XX\t\tfoo\n"))
	assert.Nil(t, prefixmap)
	assert.ErrorContains(t, err, "invalid note format")
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
		voicebank.PrefixMapWithNoteFormat(voicebank.NoteFormatSharps),
	)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, want, buf.Bytes())
}
