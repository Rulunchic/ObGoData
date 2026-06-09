package analyze

import "github.com/kljensen/snowball"

func lemmatize(w string) string {
	s, err := snowball.Stem(w, "russian", true)
	if err != nil {
		return w
	}
	return s
}
