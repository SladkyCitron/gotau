// Package ini provides functionality for parsing and representing INI files.
package ini

// Document represents an INI document with sections and entries.
type Document struct {
	Sections []Section
}

// Section represents a section in an INI document.
type Section struct {
	Name    string
	Entries []Entry
	Raw     string
}

// Entry represents a key-value pair in an INI section.
type Entry struct {
	Key   string
	Value string
}
