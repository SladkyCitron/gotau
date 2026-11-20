package ust

import (
	"bytes"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/SladkyCitron/gotau/internal/ini"
	"gitlab.com/gomidi/midi/v2"
	"golang.org/x/net/html/charset"
)

var noteRe = regexp.MustCompile(`^#\d+$`)
var charsetRe = regexp.MustCompile(`(?im)^[\t ]*Charset[\t ]*=[\t ]*([^\r\n]+)`)

func sectionValue(sec ini.Section, key string) string {
	if entry := sec.Get(key); entry != nil {
		return entry.Value
	}
	return ""
}

func optionalFloat(sec ini.Section, key string) (*float64, error) {
	value := sectionValue(sec, key)
	if value == "" {
		return nil, nil
	}

	v, err := parseFiniteFloat(value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", key, err)
	}

	return &v, nil
}

func parseFiniteFloat(value string) (float64, error) {
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}

	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("value must be finite")
	}

	return v, nil
}

// Parse parses a UST file.
func Parse(data []byte) (file *File, _ error) {
	// Trim BOM if present, as it can interfere with charset detection
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})

	// Detect encoding. An explicit declaration takes precedence over automatic
	// detection; Shift-JIS is the fallback for invalid UTF-8.
	const sniffLen = 4096
	sniffData := data
	if len(sniffData) > sniffLen {
		sniffData = sniffData[:sniffLen]
	}
	charsetName := ""
	if match := charsetRe.FindSubmatch(sniffData); match != nil {
		charsetName = strings.TrimSpace(string(match[1]))
	} else if utf8.Valid(data) {
		charsetName = "utf-8"
	} else {
		charsetName = "shift_jis"
	}

	lc := strings.ToLower(charsetName)
	if lc != "" && lc != "utf-8" && lc != "utf8" {
		enc, _ := charset.Lookup(lc)
		if enc == nil {
			return nil, fmt.Errorf("unsupported charset: %s", charsetName)
		}
		data, err := enc.NewDecoder().Bytes(data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode UST file: %w", err)
		}
		return parse(data)
	}

	return parse(data)
}

func parse(data []byte) (*File, error) {
	iniDoc, err := ini.Parse(data, ini.ParseOptions{
		UnparseableSections: []string{"#VERSION"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse UST file: %w", err)
	}

	file := &File{iniDoc: iniDoc}
	noteIndex := 0

	for _, sec := range file.iniDoc.Sections {
		switch sec.Name {
		case "":
			continue
		case "#VERSION":
			if err := file.parseVersion(sec); err != nil {
				return nil, fmt.Errorf("failed to parse version section: %w", err)
			}
		case "#SETTING":
			if err := file.parseSetting(sec); err != nil {
				return nil, fmt.Errorf("failed to parse settings: %w", err)
			}
		case "#TRACKEND":
			return file, nil
		case "#INSERT":
			note, err := file.parseNote(sec)
			if err != nil {
				return nil, fmt.Errorf("failed to parse inserted note: %w", err)
			}
			file.Notes = slices.Insert(file.Notes, noteIndex, note)
			noteIndex++
		case "#DELETE":
			if noteIndex > 0 {
				noteIndex--
				file.Notes = slices.Delete(file.Notes, noteIndex, noteIndex+1)
			}
		default:
			if noteRe.MatchString(sec.Name) {
				note, err := file.parseNote(sec)
				if err != nil {
					return nil, fmt.Errorf("failed to parse note %s: %w", sec.Name, err)
				}
				if noteIndex < len(file.Notes) {
					file.Notes[noteIndex] = note
				} else {
					file.Notes = append(file.Notes, note)
				}
				noteIndex++
			} else {
				//slog.Warn("invalid section, skipping", "name", sec.Name)
				continue
			}
		}
	}

	return file, nil
}

func (f *File) parseVersion(sec ini.Section) error {
	f.Version = Version1_2
	for line := range strings.SplitSeq(sec.Raw, "\n") {
		if strings.Contains(line, "UST Version") {
			ver, err := ParseVersion(strings.TrimSpace(line))
			if err != nil {
				return err
			}
			f.Version = ver
		}
	}
	return nil
}

func (f *File) parseSetting(sec ini.Section) (err error) {
	f.Settings = Settings{}

	tempo, err := parseFiniteFloat(sectionValue(sec, "Tempo")) // Tempo
	if err != nil {
		return fmt.Errorf("failed to parse tempo: %w", err)
	}
	if tempo <= 0 {
		return fmt.Errorf("invalid tempo: %f", tempo)
	}
	f.Settings.Tempo = tempo

	f.Settings.ProjectName = sectionValue(sec, "ProjectName") // ProjectName
	f.Settings.Project = sectionValue(sec, "Project")         // Project (OpenUtau)
	f.Settings.VoiceDir = sectionValue(sec, "VoiceDir")       // VoiceDir
	f.Settings.OutFile = sectionValue(sec, "OutFile")         // OutFile
	f.Settings.CacheDir = sectionValue(sec, "CacheDir")       // CacheDir
	f.Settings.Tool1 = sectionValue(sec, "Tool1")             // Tool1
	f.Settings.Tool2 = sectionValue(sec, "Tool2")             // Tool2

	// Mode2
	if value := sectionValue(sec, "Mode2"); value != "" {
		f.Settings.Mode2, err = strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("failed to parse Mode2: %w", err)
		}
	}

	return nil
}

func (f *File) parseNote(sec ini.Section) (Note, error) {
	note := Note{}
	var err error

	// Length
	note.Length, err = strconv.Atoi(sectionValue(sec, "Length"))
	if err != nil {
		return Note{}, fmt.Errorf("failed to parse length: %w", err)
	}
	if note.Length <= 0 {
		return Note{}, fmt.Errorf("invalid note length: %d", note.Length)
	}

	// Lyric
	note.Lyric = sectionValue(sec, "Lyric")

	// NoteNum
	noteNum, err := strconv.ParseUint(sectionValue(sec, "NoteNum"), 10, 8)
	if err != nil {
		return Note{}, fmt.Errorf("failed to parse note number: %w", err)
	}
	if noteNum > 127 {
		return Note{}, fmt.Errorf("invalid MIDI note number: %d", noteNum)
	}
	note.NoteNum = midi.Note(noteNum)

	// Intensity
	note.Intensity = 100
	if intensity, err := optionalFloat(sec, "Intensity"); err != nil {
		return Note{}, fmt.Errorf("failed to parse intensity: %w", err)
	} else if intensity != nil {
		note.Intensity = *intensity
	}

	// Velocity
	note.Velocity = 100
	if velocity, err := optionalFloat(sec, "Velocity"); err != nil {
		return Note{}, fmt.Errorf("failed to parse velocity: %w", err)
	} else if velocity != nil {
		note.Velocity = *velocity
	}

	// Modulation
	note.Modulation = 0
	if modulation, err := optionalFloat(sec, "Modulation"); err != nil {
		return Note{}, fmt.Errorf("failed to parse modulation: %w", err)
	} else if modulation != nil {
		note.Modulation = *modulation
	}

	// PreUtterance
	if note.Preutterance, err = optionalFloat(sec, "PreUtterance"); err != nil {
		return Note{}, fmt.Errorf("failed to parse pre-utterance: %w", err)
	}

	if note.PreutteranceOverride, err = optionalFloat(sec, "@preuttr"); err != nil {
		return Note{}, fmt.Errorf("failed to parse pre-utterance override: %w", err)
	}

	// VoiceOverlap
	if note.VoiceOverlap, err = optionalFloat(sec, "VoiceOverlap"); err != nil {
		return Note{}, fmt.Errorf("failed to parse voice overlap: %w", err)
	}

	// StartPoint
	note.StartPoint = 0
	if startPoint, err := optionalFloat(sec, "StartPoint"); err != nil {
		return Note{}, fmt.Errorf("failed to parse start point: %w", err)
	} else if startPoint != nil {
		note.StartPoint = *startPoint
	}

	// UST 2.0 overrides
	if note.VoiceOverlapOverride, err = optionalFloat(sec, "@overlap"); err != nil {
		return Note{}, fmt.Errorf("failed to parse overlap override: %w", err)
	}
	if note.StartPointOverride, err = optionalFloat(sec, "@stpoint"); err != nil {
		return Note{}, fmt.Errorf("failed to parse start-point override: %w", err)
	}

	// Envelope
	if key := sec.Get("Envelope"); key != nil && key.Value != "" {
		_envelope, err := ParseEnvelope(key.Value)
		if err != nil {
			return Note{}, fmt.Errorf("failed to parse envelope: %w", err)
		}
		note.Envelope = _envelope
	}

	// Pitch bend (PB*)
	note.PitchBend = nil
	mode1Pitches := sectionValue(sec, "PitchBend")
	if mode1Pitches == "" {
		mode1Pitches = sectionValue(sec, "Pitches")
	}
	if mode1Pitches == "" {
		mode1Pitches = sectionValue(sec, "Piches")
	}
	if sec.Has("PBS") || sec.Has("PBStart") || sec.Has("PBW") || sec.Has("PBY") ||
		sec.Has("PBM") || sec.Has("PBType") || sec.Has("PitchBend") ||
		sec.Has("Pitches") || sec.Has("Piches") {
		pitchBend, err := ParsePitchBend(
			sectionValue(sec, "PBType"),
			sectionValue(sec, "PBStart"),
			sectionValue(sec, "PBS"),
			sectionValue(sec, "PBW"),
			sectionValue(sec, "PBY"),
			sectionValue(sec, "PBM"),
			mode1Pitches,
		)
		if err != nil {
			return Note{}, fmt.Errorf("failed to parse pitch bend data: %w", err)
		}
		note.PitchBend = pitchBend
	}

	// Flags
	note.Flags = sectionValue(sec, "Flags")

	return note, nil
}
