package ini

import (
	"bytes"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// ParseOptions represents options for parsing INI files.
type ParseOptions struct {
	// CommentPrefix specifies the prefix used for comments in the INI file. Default is ";".
	// If set to an empty string, comments will not be recognized.
	CommentPrefix string

	// UnparseableSections specifies the names of sections that cannot not be parsed correctly.
	// If a section name is included in this list, its content will be stored as raw text
	// in the [Section.Raw] field instead of being parsed into key-value pairs.
	UnparseableSections []string

	// UnescapeValueDoubleQuotes indicates whether to unescape double quotes inside value
	// to regular format when value is surrounded by double quotes.
	UnescapeValueDoubleQuotes bool
}

type ParseError struct {
	Line   int
	Column int
	Err    string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Line, e.Column, e.Err)
}

// Parse parses the given INI data and returns a [Document] representation.
func Parse(data []byte, opts ...ParseOptions) (*Document, error) {
	if len(opts) > 1 {
		panic("ini.Parse: only one ParseOptions argument is allowed")
	}

	if len(opts) > 0 {
		// Use the provided options if available
		return parseWithOptions(data, opts[0])
	}
	return parseWithOptions(data, ParseOptions{
		CommentPrefix: ";",
	})
}

func parseWithOptions(data []byte, opts ParseOptions) (*Document, error) {
	opts.CommentPrefix = strings.TrimSpace(opts.CommentPrefix)

	doc := &Document{
		Sections: []Section{
			{Name: "", Entries: []Entry{}},
		},
	}

	lineNum := 1
	curSection := &doc.Sections[0]
	var rawBuf bytes.Buffer
	rawBuf.Grow(len(data))
	for line := range bytes.SplitSeq(data, []byte{'\n'}) {
		origLine := bytes.TrimSuffix(line, []byte{'\r'})

		line = bytes.TrimSpace(origLine)

		if len(line) == 0 || bytes.HasPrefix(line, []byte(opts.CommentPrefix)) && opts.CommentPrefix != "" {
			lineNum++
			continue // Skip empty lines and comments
		}

		if line[0] == '[' {
			if line[len(line)-1] != ']' {
				return nil, &ParseError{
					Line:   lineNum,
					Column: 1,
					Err:    "unterminated section declaration",
				}
			}

			sectionName := bytes.TrimSpace(line[1 : len(line)-1])
			if len(sectionName) == 0 {
				return nil, &ParseError{
					Line:   lineNum,
					Column: 2,
					Err:    "empty section name",
				}
			}

			if bytes.Contains(sectionName, []byte{'['}) || bytes.Contains(sectionName, []byte{']'}) {
				return nil, &ParseError{
					Line:   lineNum,
					Column: 1,
					Err:    "invalid section name",
				}
			}

			// Dump raw content of the previous section if it was unparseable
			if slices.Contains(opts.UnparseableSections, curSection.Name) {
				curSection.Raw = rawBuf.String()
				rawBuf.Reset()
			}

			doc.Sections = append(doc.Sections, Section{
				Name: string(sectionName),
			})
			curSection = &doc.Sections[len(doc.Sections)-1]
			lineNum++
			continue
		}

		if slices.Contains(opts.UnparseableSections, curSection.Name) {
			rawBuf.Write(origLine)
			rawBuf.WriteByte('\n')
			lineNum++
			continue
		}

		key, value, ok := bytes.Cut(line, []byte{'='})
		if !ok {
			return nil, &ParseError{Line: lineNum, Column: 1, Err: "invalid entry format"}
		}

		key = bytes.TrimSpace(key)
		value = bytes.TrimSpace(value)

		if len(key) == 0 {
			return nil, &ParseError{
				Line:   lineNum,
				Column: 1,
				Err:    "empty key",
			}
		}

		if len(value) >= 2 && value[0] == '"' {
			if len(value) < 2 || value[len(value)-1] != '"' {
				return nil, &ParseError{
					Line:   lineNum,
					Column: len(origLine) - len(value) + 1,
					Err:    "unterminated quoted value",
				}
			}

			if opts.UnescapeValueDoubleQuotes {
				_value, err := strconv.Unquote(string(value))
				if err != nil {
					return nil, &ParseError{
						Line:   lineNum,
						Column: len(origLine) - len(value) + 1,
						Err:    fmt.Errorf("invalid quoted string: %w", err).Error(),
					}
				}
				value = []byte(_value)
			} else {
				value = bytes.TrimPrefix(value, []byte{'"'})
				value = bytes.TrimSuffix(value, []byte{'"'})
			}
		}

		curSection.Entries = append(curSection.Entries, Entry{Key: string(key), Value: string(value)})

		lineNum++
	}
	// Dump raw content of the previous section if it was unparseable one last time
	if slices.Contains(opts.UnparseableSections, curSection.Name) {
		curSection.Raw = rawBuf.String()
	}
	return doc, nil
}
