package scanner

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
)

type Note struct {
	Path        string
	Name        string
	Dir         string
	Size        int64
	ModTime     time.Time
	WordCount   int
	Links       []Link
	Attachments []Attachment
}

type Link struct {
	Text string
}

type Attachment struct {
	Name string
	Type string
}

type ScanResult struct {
	VaultPath string
	Notes     []Note
	Dirs      []string
	TotalSize int64
}

var skipDirs = map[string]bool{
	".obsidian": true,
	".git":      true,
	"node_modules": true,
	".trash":    true,
	"Excildraw": true,
	"Excalidraw": true,
}

var attachExt = map[string]string{
	".png": "image", ".jpg": "image", ".jpeg": "image",
	".gif": "image", ".svg": "image", ".webp": "image", ".bmp": "image",
	".pdf": "pdf",
	".mp3": "audio", ".wav": "audio", ".ogg": "audio", ".flac": "audio",
	".mp4": "video", ".mov": "video", ".avi": "video", ".webm": "video",
	".zip": "archive", ".7z": "archive", ".rar": "archive", ".gz": "archive",
	".docx": "document", ".xlsx": "document", ".pptx": "document",
}

var linkRe = regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`)
var embedRe = regexp.MustCompile(`!\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`)
var urlRe = regexp.MustCompile(`https?://[^\s\)\]\"'<>]+`)

func Scan(vaultPath string) (*ScanResult, error) {
	vaultPath, err := filepath.Abs(vaultPath)
	if err != nil {
		return nil, err
	}

	result := &ScanResult{
		VaultPath: vaultPath,
		Dirs:      []string{},
		Notes:     []Note{},
	}
	seenDirs := map[string]bool{}

	err = filepath.WalkDir(vaultPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if skipDirs[name] {
				return filepath.SkipDir
			}
			if path != vaultPath && !seenDirs[path] {
				seenDirs[path] = true
				rel, _ := filepath.Rel(vaultPath, path)
				result.Dirs = append(result.Dirs, rel)
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".md" {
			note, err := scanNote(path, vaultPath)
			if err == nil {
				result.Notes = append(result.Notes, note)
				result.TotalSize += note.Size
			}
			return nil
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func scanNote(path, vaultPath string) (Note, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return Note{}, err
	}

	relPath, _ := filepath.Rel(vaultPath, path)
	dir := filepath.Dir(relPath)
	if dir == "." {
		dir = "/"
	}

	note := Note{
		Path:    relPath,
		Name:    strings.TrimSuffix(filepath.Base(path), ".md"),
		Dir:     dir,
		Size:    fi.Size(),
		ModTime: fi.ModTime(),
		Links:   []Link{},
		Attachments: []Attachment{},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return note, nil
	}

	content := string(data)

	note.WordCount = countWords(content)

	links := linkRe.FindAllStringSubmatch(content, -1)
	for _, m := range links {
		note.Links = append(note.Links, Link{Text: m[1]})
	}

	embeds := embedRe.FindAllStringSubmatch(content, -1)
	for _, m := range embeds {
		note.Attachments = append(note.Attachments, Attachment{
			Name: m[1],
			Type: inferAttachType(m[1]),
		})
	}

	urls := urlRe.FindAllString(content, -1)
	for _, u := range urls {
		note.Links = append(note.Links, Link{Text: u})
	}

	return note, nil
}

func countWords(s string) int {
	count := 0
	inWord := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if !inWord {
				count++
				inWord = true
			}
		} else if r == '\'' || r == '-' {
			continue
		} else {
			inWord = false
		}
	}
	return count
}

func inferAttachType(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if t, ok := attachExt[ext]; ok {
		return t
	}
	return "other"
}

func (r *ScanResult) NoteCountByDir() map[string]int {
	m := make(map[string]int)
	for _, n := range r.Notes {
		m[n.Dir]++
	}
	return m
}

func (r *ScanResult) LinkCount() int {
	n := 0
	for _, note := range r.Notes {
		n += len(note.Links)
	}
	return n
}

func (r *ScanResult) AttachmentCount() int {
	n := 0
	for _, note := range r.Notes {
		n += len(note.Attachments)
	}
	return n
}
