package ini_test

import (
	"testing"

	"github.com/SladkyCitron/gotau/internal/ini"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	data := `
; This is a comment
  foo=42

[section]
bar  =39
baz=  hello world
quotes = "some \"quotes\""

[ unparseable ]
Some unparseable INI data
; Another comment
Without key-value pairs
`
	want := &ini.Document{
		Sections: []ini.Section{
			{Name: "", Entries: []ini.Entry{
				{Key: "foo", Value: "42"},
			}, Raw: ""},
			{Name: "section", Entries: []ini.Entry{
				{Key: "bar", Value: "39"},
				{Key: "baz", Value: "hello world"},
				{Key: "quotes", Value: `some "quotes"`},
			}, Raw: ""},
			{Name: "unparseable", Raw: `Some unparseable INI data
Without key-value pairs
`},
		},
	}

	got, err := ini.Parse([]byte(data), ini.ParseOptions{
		CommentPrefix:             ";",
		UnparseableSections:       []string{"unparseable"},
		UnescapeValueDoubleQuotes: true,
	})
	assert.NoError(t, err)

	assert.Equal(t, want, got)
}
