package ini_test

import (
	"testing"

	"github.com/SladkyCitron/gotau/internal/ini"
	"github.com/stretchr/testify/assert"
)

func TestParseFullDocument(t *testing.T) {
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

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		data string
		opts ini.ParseOptions
		want *ini.Document
	}{
		{
			name: "empty input creates the default section",
			want: &ini.Document{Sections: []ini.Section{{Name: "", Entries: []ini.Entry{}, Raw: ""}}},
		},
		{
			name: "values may be empty or contain equals signs",
			data: "empty=\nexpression=a=b=c",
			want: &ini.Document{
				Sections: []ini.Section{{
					Name: "",
					Entries: []ini.Entry{
						{Key: "empty", Value: ""},
						{Key: "expression", Value: "a=b=c"},
					},
				}},
			},
		},
		{
			name: "whitespace and comments are ignored",
			data: " \t ; comment\r\n\t[ section ] \r\n key \t = \t value \r\n",
			opts: ini.ParseOptions{CommentPrefix: ";"},
			want: &ini.Document{
				Sections: []ini.Section{
					{Name: "", Entries: []ini.Entry{}},
					{Name: "section", Entries: []ini.Entry{{Key: "key", Value: "value"}}},
				},
			},
		},
		{
			name: "comment prefix can be changed",
			data: "# comment\nkey=value",
			opts: ini.ParseOptions{CommentPrefix: "#"},
			want: &ini.Document{
				Sections: []ini.Section{{
					Name:    "",
					Entries: []ini.Entry{{Key: "key", Value: "value"}},
				}},
			},
		},
		{
			name: "quoted values retain escapes when unescaping is disabled",
			data: `value = "a \"quoted\" value"`,
			want: &ini.Document{
				Sections: []ini.Section{{
					Name:    "",
					Entries: []ini.Entry{{Key: "value", Value: `a \"quoted\" value`}},
				}},
			},
		},
		{
			name: "unparseable section preserves non-entry lines",
			data: "[raw]\r\n first line \r\nsecond line\r\n",
			opts: ini.ParseOptions{UnparseableSections: []string{"raw"}},
			want: &ini.Document{
				Sections: []ini.Section{
					{Name: "", Entries: []ini.Entry{}},
					{Name: "raw", Raw: " first line \nsecond line\n"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ini.Parse([]byte(tt.data), tt.opts)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		data string
		opts ini.ParseOptions
		want string
	}{
		{
			name: "invalid entry format",
			data: "not an entry",
			want: "1:1: invalid entry format",
		},
		{
			name: "unterminated section",
			data: "[section",
			want: "1:1: unterminated section declaration",
		},
		{
			name: "empty section name",
			data: "[]",
			want: "1:2: empty section name",
		},
		{
			name: "invalid section name",
			data: "[nested[section]]",
			want: "1:1: invalid section name",
		},
		{
			name: "empty key",
			data: "=value",
			want: "1:1: empty key",
		},
		{
			name: "unterminated quoted value",
			data: `key = "value`,
			want: "1:7: unterminated quoted value",
		},
		{
			name: "invalid quoted string",
			data: `key = "\q"`,
			opts: ini.ParseOptions{UnescapeValueDoubleQuotes: true},
			want: `1:7: invalid quoted string: invalid syntax`,
		},
		{
			name: "line and column are reported after skipped lines",
			data: "; comment\n\nbroken",
			opts: ini.ParseOptions{CommentPrefix: ";"},
			want: "3:1: invalid entry format",
		},
		{
			name: "empty comment prefix disables comments",
			data: "; comment",
			opts: ini.ParseOptions{},
			want: "1:1: invalid entry format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ini.Parse([]byte(tt.data), tt.opts)
			if assert.Error(t, err) {
				assert.Equal(t, tt.want, err.Error())
			}
		})
	}
}

func BenchmarkParse(b *testing.B) {
	data := []byte(`
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
`)
	opts := ini.ParseOptions{
		CommentPrefix:             ";",
		UnparseableSections:       []string{"unparseable"},
		UnescapeValueDoubleQuotes: true,
	}

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for b.Loop() {
		_, err := ini.Parse(data, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}
