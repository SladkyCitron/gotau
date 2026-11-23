package prefixmap_test

import (
	"bytes"
	"cmp"
	"testing"

	"github.com/SladkyCitron/gotau/voicebank/prefixmap"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gomidi/midi/v2"
	"golang.org/x/text/encoding/japanese"
)

func TestDecodePrefixMap(t *testing.T) {
	b := []byte("C5\tpre\tsuf\nD#5\tpre\t\n# comment\nF#5\t\tsuf\n")

	want := prefixmap.PrefixMap{
		midi.Note(midi.C(6)):  prefixmap.Prefix{"pre", "suf"},
		midi.Note(midi.Eb(6)): prefixmap.Prefix{"pre", ""},
		midi.Note(midi.Gb(6)): prefixmap.Prefix{"", "suf"},
	}

	prefixmap, err := prefixmap.Parse(b)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, want, prefixmap)
}

func TestDecodePrefixMapWithOpts(t *testing.T) {
	b := []byte("C5 pre suf\nD#5 pre \n; comment\nF#5  suf\n")

	want := prefixmap.PrefixMap{
		midi.Note(midi.C(6)):  prefixmap.Prefix{"pre", "suf"},
		midi.Note(midi.Eb(6)): prefixmap.Prefix{"pre", ""},
		midi.Note(midi.Gb(6)): prefixmap.Prefix{"", "suf"},
	}

	prefixmap, err := prefixmap.Parse(b,
		prefixmap.WithEncoding(japanese.ShiftJIS),
		prefixmap.WithComment(';'),
		prefixmap.WithDelimiter(' '),
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

	want := prefixmap.PrefixMap{
		midi.Note(midi.C(6)):  prefixmap.Prefix{"", "foo"},
		midi.Note(midi.Db(6)): prefixmap.Prefix{"", "foo"},
		midi.Note(midi.D(6)):  prefixmap.Prefix{"", "foo"},
		midi.Note(midi.Eb(6)): prefixmap.Prefix{"", "foo"},
		midi.Note(midi.E(6)):  prefixmap.Prefix{"", "foo"},
		midi.Note(midi.F(6)):  prefixmap.Prefix{"", "foo"},
		midi.Note(midi.Gb(6)): prefixmap.Prefix{"", "foo"},
		midi.Note(midi.G(6)):  prefixmap.Prefix{"", "foo"},
		midi.Note(midi.Ab(6)): prefixmap.Prefix{"", "foo"},
		midi.Note(midi.A(6)):  prefixmap.Prefix{"", "foo"},
		midi.Note(midi.Bb(6)): prefixmap.Prefix{"", "foo"},
		midi.Note(midi.B(6)):  prefixmap.Prefix{"", "foo"},
	}

	prefixmap, err := prefixmap.Parse(b)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, want, prefixmap)
}

func TestDecodePrefixMapInvalidLine(t *testing.T) {
	prefixmap, err := prefixmap.Parse([]byte("C#5\tfoo\n"))
	assert.Nil(t, prefixmap)
	assert.ErrorContains(t, err, "prefixmap Parse: invalid line")
}

func TestPrefixMapRoundTrip(t *testing.T) {
	want := []byte("C5\tpre\tsuf\nD#5\tpre\t\nF#5\t\tsuf\n")

	pm, err := prefixmap.Parse(want)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err = pm.Encode(&buf,
		prefixmap.WithSort(func(a, b midi.Note) int {
			return cmp.Compare(a, b)
		}),
		prefixmap.WithNoteFormat('s'),
	)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, want, buf.Bytes())
}
