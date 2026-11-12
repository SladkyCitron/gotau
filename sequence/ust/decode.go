package ust

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/SladkyCitron/gotau/internal/ini"
	"gitlab.com/gomidi/midi/v2"
	"golang.org/x/net/html/charset"
)

var noteRe *regexp.Regexp = regexp.MustCompile(`#\d+`)

func sectionValue(sec ini.Section, key string) string {
	if entry := sec.Get(key); entry != nil {
		return entry.Value
	}
	return ""
}

// Decode decodes a UST file.
func Decode(r io.Reader) (file *File, _ error) {
	// detect encoding, step 1: sniff
	const sniffLen = 256
	buf := make([]byte, sniffLen)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read UST file for encoding detection: %w", err)
	}

	// step 2: find charset in the sniff
	_charset := "shift_jis" // default
	for line := range strings.SplitSeq(string(buf[:n]), "\n") {
		if strings.Contains(line, "Charset=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				_charset = strings.TrimSpace(parts[1])
			}
			break
		}
	}

	concatReader := io.MultiReader(bytes.NewReader(buf[:n]), r)

	// step 3: transcode to UTF-8
	newReader := concatReader
	lc := strings.ToLower(strings.TrimSpace(_charset))
	if lc != "" && lc != "utf-8" && lc != "utf8" {
		dec, err := charset.NewReaderLabel(lc, concatReader)
		if err != nil {
			return nil, fmt.Errorf("failed to create decoder for charset %s: %w", strconv.Quote(_charset), err)
		}
		newReader = dec
	}

	data, err := io.ReadAll(newReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read UST file: %w", err)
	}

	iniDoc, err := ini.Parse(data, ini.ParseOptions{
		UnparseableSections: []string{"#VERSION"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse UST file: %w", err)
	}

	file = &File{iniDoc: iniDoc}

	for _, sec := range file.iniDoc.Sections {
		switch sec.Name {
		case "DEFAULT":
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
		default:
			if noteRe.MatchString(sec.Name) {
				if err := file.parseNote(sec); err != nil {
					return nil, fmt.Errorf("failed to parse note %s: %w", sec.Name, err)
				}
			} else {
				slog.Warn("invalid section, skipping", "name", sec.Name)
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

	tempo, err := strconv.ParseFloat(sectionValue(sec, "Tempo"), 64) // Tempo
	if err != nil {
		return fmt.Errorf("failed to parse tempo: %w", err)
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
	f.Settings.Mode2, err = strconv.ParseBool(sectionValue(sec, "Mode2"))
	if err != nil {
		return fmt.Errorf("failed to parse Mode2: %w", err)
	}

	return nil
}

func (f *File) parseNote(sec ini.Section) (err error) {
	note := Note{}

	// Length
	note.Length, err = strconv.Atoi(sectionValue(sec, "Length"))
	if err != nil {
		return fmt.Errorf("failed to parse length: %w", err)
	}

	// Lyric
	note.Lyric = sectionValue(sec, "Lyric")

	// NoteNum
	noteNum, err := strconv.ParseUint(sectionValue(sec, "NoteNum"), 10, 8)
	if err != nil {
		return fmt.Errorf("failed to parse note number: %w", err)
	}
	note.NoteNum = midi.Note(noteNum)

	// Intensity
	note.Intensity = 100
	if key := sec.Get("Intensity"); key != nil && key.Value != "" {
		intensity, err := strconv.ParseFloat(key.Value, 64)
		if err != nil {
			return fmt.Errorf("failed to parse intensity: %w", err)
		}
		note.Intensity = intensity
	}

	// Velocity
	note.Velocity = 0
	if key := sec.Get("Velocity"); key != nil && key.Value != "" {
		note.Velocity, err = strconv.ParseFloat(key.Value, 64)
		if err != nil {
			return fmt.Errorf("failed to parse velocity: %w", err)
		}
	}

	// Modulation
	note.Modulation = 0
	if key := sec.Get("Modulation"); key != nil && key.Value != "" {
		modulation, err := strconv.ParseFloat(key.Value, 64)
		if err != nil {
			return fmt.Errorf("failed to parse modulation: %w", err)
		}
		note.Modulation = modulation
	}

	// PreUtterance
	note.Preutterance = nil
	if key := sec.Get("PreUtterance"); key != nil && key.Value != "" {
		preUtterance, err := strconv.ParseFloat(key.Value, 64)
		if err != nil {
			return fmt.Errorf("failed to parse pre-utterance: %w", err)
		}
		note.Preutterance = &preUtterance
	}

	// VoiceOverlap
	note.VoiceOverlap = nil
	if key := sec.Get("VoiceOverlap"); key != nil && key.Value != "" {
		voiceOverlap, err := strconv.ParseFloat(key.Value, 64)
		if err != nil {
			return fmt.Errorf("failed to parse voice overlap: %w", err)
		}
		note.VoiceOverlap = &voiceOverlap
	}

	// StartPoint
	note.StartPoint = 0
	if key := sec.Get("StartPoint"); key != nil && key.Value != "" {
		note.StartPoint, err = strconv.ParseFloat(key.Value, 64)
		if err != nil {
			return fmt.Errorf("failed to parse start point: %w", err)
		}
	}

	// Envelope
	if key := sec.Get("Envelope"); key != nil && key.Value != "" {
		_envelope, err := ParseEnvelope(key.Value)
		if err != nil {
			return fmt.Errorf("failed to parse envelope: %w", err)
		}
		note.Envelope = _envelope
	}

	// Pitch bend (PB*)
	note.PitchBend = nil
	if sec.Has("PBS") || sec.Has("PBStart") {
		_pb, err := ParsePitchBend(
			sectionValue(sec, "PBType"),
			sectionValue(sec, "PBStart"),
			sectionValue(sec, "PBS"),
			sectionValue(sec, "PBW"),
			sectionValue(sec, "PBY"),
			sectionValue(sec, "PBM"),
		)
		if err != nil {
			return fmt.Errorf("failed to parse pitch bend data: %w", err)
		}
		note.PitchBend = _pb
	}

	// Flags
	note.Flags = sectionValue(sec, "Flags")

	f.Notes = append(f.Notes, note)

	return nil
}
