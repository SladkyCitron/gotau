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

func (d *Document) GetSection(name string) *Section {
	for i := range d.Sections {
		if d.Sections[i].Name == name {
			return &d.Sections[i]
		}
	}
	return nil
}

func (d *Document) GetDefaultSection() *Section {
	return &d.Sections[0]
}

func (s *Section) Get(key string) *Entry {
	for i := range s.Entries {
		if s.Entries[i].Key == key {
			return &s.Entries[i]
		}
	}
	return nil
}
