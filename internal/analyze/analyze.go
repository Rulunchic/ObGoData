package analyze

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/Rulunchic/ObGoData/internal/scanner"
)

type WordCount struct {
	Word  string
	Count int
}

type CharCount struct {
	Char  string
	Count int
}

type Stats struct {
	TotalNotes   int
	TotalWords   int
	TotalLinks   int
	TotalAttachs int
	TotalSize    int64
	DirCount     int

	NotesByDate    map[string]int
	WordsByDate    map[string]int
	NotesByHour    [24]int
	NotesByWeekday [7]int

	WordFreq    []WordCount
	CharFreq    []CharCount
	LemmaFreq   []WordCount
	WordsPerNote []int

	SizeByDir    map[string]int64
	FilesByDir   map[string]int
	LargestNotes []scanner.Note

	LinkTypes   map[string]int
	AttachTypes map[string]int
}

func Compute(r *scanner.ScanResult) *Stats {
	s := &Stats{
		TotalNotes:   len(r.Notes),
		TotalWords:   0,
		TotalLinks:   0,
		TotalAttachs: 0,
		TotalSize:    r.TotalSize,
		DirCount:     len(r.Dirs),

		NotesByDate:    make(map[string]int),
		WordsByDate:    make(map[string]int),
		SizeByDir:      make(map[string]int64),
		FilesByDir:     make(map[string]int),
		LinkTypes:      make(map[string]int),
		AttachTypes:    make(map[string]int),
	}

	wordFreqRaw := make(map[string]int)
	charFreqRaw := make(map[rune]int)

	for _, note := range r.Notes {
		s.TotalWords += note.WordCount
		s.TotalLinks += len(note.Links)
		s.TotalAttachs += len(note.Attachments)

		s.WordsPerNote = append(s.WordsPerNote, note.WordCount)

		date := note.ModTime.Format("2006-01-02")
		s.NotesByDate[date]++
		s.WordsByDate[date] += note.WordCount

		h := note.ModTime.Hour()
		s.NotesByHour[h]++

		s.NotesByWeekday[note.ModTime.Weekday()]++

		s.SizeByDir[note.Dir] += note.Size
		s.FilesByDir[note.Dir]++

		s.LinkTypes["total"]++
		for _, link := range note.Links {
			if strings.HasPrefix(link.Text, "http://") || strings.HasPrefix(link.Text, "https://") {
				s.LinkTypes["external"]++
			} else {
				s.LinkTypes["wikilink"]++
			}
		}

		for _, att := range note.Attachments {
			s.AttachTypes[att.Type]++
		}
	}

	sort.Slice(s.LargestNotes, func(i, j int) bool {
		return s.LargestNotes[i].Size > s.LargestNotes[j].Size
	})

	loadStopWords()

	for _, note := range r.Notes {
		content := readNoteContent(r.VaultPath, note.Path)
		if content == "" {
			continue
		}

		words := tokenize(content)
		for _, w := range words {
			w = strings.ToLower(w)
			if len(w) < 3 {
				continue
			}
			if !hasLetter(w) {
				continue
			}
			if isStopWord(w) {
				continue
			}
			wordFreqRaw[w]++
		}

		for _, ch := range content {
			if unicode.IsLetter(ch) {
				charFreqRaw[unicode.ToLower(ch)]++
			}
		}
	}

	lemmaRaw := make(map[string]int)
	for w, c := range wordFreqRaw {
		l := lemmatize(w)
		lemmaRaw[l] += c
		s.WordFreq = append(s.WordFreq, WordCount{Word: w, Count: c})
	}
	sort.Slice(s.WordFreq, func(i, j int) bool {
		return s.WordFreq[i].Count > s.WordFreq[j].Count
	})

	for l, c := range lemmaRaw {
		s.LemmaFreq = append(s.LemmaFreq, WordCount{Word: l, Count: c})
	}
	sort.Slice(s.LemmaFreq, func(i, j int) bool {
		return s.LemmaFreq[i].Count > s.LemmaFreq[j].Count
	})

	for ch, c := range charFreqRaw {
		s.CharFreq = append(s.CharFreq, CharCount{Char: string(ch), Count: c})
	}
	sort.Slice(s.CharFreq, func(i, j int) bool {
		return s.CharFreq[i].Count > s.CharFreq[j].Count
	})

	sort.Slice(s.LargestNotes, func(i, j int) bool {
		return s.LargestNotes[i].Size > s.LargestNotes[j].Size
	})
	largest := make([]scanner.Note, len(r.Notes))
	copy(largest, r.Notes)
	sort.Slice(largest, func(i, j int) bool {
		return largest[i].Size > largest[j].Size
	})
	if len(largest) > 15 {
		largest = largest[:15]
	}
	s.LargestNotes = largest

	return s
}

func readNoteContent(vaultPath, relPath string) string {
	full := filepath.Join(vaultPath, relPath)
	data, err := os.ReadFile(full)
	if err != nil {
		return ""
	}
	return string(data)
}

func tokenize(s string) []string {
	var words []string
	var buf strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || r == '-' || r == '\'' {
			buf.WriteRune(r)
		} else {
			if buf.Len() > 0 {
				words = append(words, buf.String())
				buf.Reset()
			}
		}
	}
	if buf.Len() > 0 {
		words = append(words, buf.String())
	}
	return words
}

var stopWordsSet map[string]bool

func loadStopWords() {
	if stopWordsSet != nil {
		return
	}
	list := []string{
		"the", "a", "an", "and", "or", "but", "in", "on", "at", "to", "for",
		"of", "with", "by", "from", "as", "is", "was", "are", "were", "be",
		"been", "being", "have", "has", "had", "do", "does", "did", "will",
		"would", "could", "should", "may", "might", "shall", "can", "need",
		"not", "no", "nor", "so", "if", "then", "than", "that", "this",
		"these", "those", "it", "its", "he", "she", "they", "them", "their",
		"his", "her", "my", "your", "our", "we", "you", "i", "me", "us",
		"all", "each", "every", "both", "few", "some", "any", "many", "much",
		"more", "most", "other", "another", "such", "what", "which", "who",
		"whom", "when", "where", "why", "how", "about", "up", "out", "off",
		"over", "under", "again", "further", "once", "here", "there",
		"into", "during", "before", "after", "above", "below", "between",
		"through", "because", "until", "while", "just", "also", "very",
		"too", "only", "own", "same", "like",

		"и", "в", "во", "не", "что", "он", "на", "я", "с", "со", "как",
		"а", "то", "все", "она", "так", "его", "но", "да", "ты", "к",
		"у", "же", "вы", "за", "бы", "по", "из", "им", "от", "о", "об",
		"для", "или", "если", "ещё", "уже", "быть", "есть", "был", "была",
		"было", "были", "будет", "будут", "нет", "их", "его", "её", "вас",
		"вам", "нас", "нам", "них", "ним", "это", "этот", "эта", "эти",
		"того", "тому", "том", "этом", "который", "которая", "которые",
		"чтобы", "можно", "надо", "нужно", "при", "про", "без", "до",
		"над", "под", "после", "через", "между", "перед", "среди",
		"только", "даже", "ведь", "вот", "вон", "ещё", "уже", "теперь",
		"потом", "всегда", "никогда", "опять", "снова", "здесь", "там",
		"тут", "куда", "откуда", "почему", "зачем", "потому", "поэтому",
		"весь", "вся", "всё", "сами", "сам", "сама", "само", "самые",
		"другой", "другая", "другие", "иной", "иная", "иные", "каждый",
		"каждая", "каждые", "любой", "любая", "любые",

		"obsidian", "note", "notes", "file", "link", "tag",
	}
	stopWordsSet = make(map[string]bool, len(list))
	for _, w := range list {
		stopWordsSet[w] = true
	}
}

func isStopWord(w string) bool {
	return stopWordsSet[w]
}

func hasLetter(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}
