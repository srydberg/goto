// Package analyzer contains the core file analysis logic. It is independent
// of the CLI so it can be reused and tested in isolation.
package analyzer

import (
	"bufio"
	"io"
	"os"
	"unicode"
	"unicode/utf8"
)

// Stats holds content statistics for a single file.
type Stats struct {
	Path       string `json:"path"`
	Bytes      int64  `json:"bytes"`
	Lines      int64  `json:"lines"`
	BlankLines int64  `json:"blank_lines"`
	Words      int64  `json:"words"`
	Runes      int64  `json:"chars"`
	LongestLn  int64  `json:"longest_line"`
}

// StatsFile computes statistics for the file at path.
func StatsFile(path string) (Stats, error) {
	f, err := os.Open(path)
	if err != nil {
		return Stats{}, err
	}
	defer f.Close()

	s, err := StatsReader(f)
	s.Path = path
	return s, err
}

// StatsReader computes statistics for everything read from r.
// A final line without a trailing newline is still counted as a line.
func StatsReader(r io.Reader) (Stats, error) {
	var (
		s        Stats
		br       = bufio.NewReader(r)
		inWord   bool
		lineLen  int64
		lineHas  bool // current line contains non-space characters
		lineOpen bool // current line has at least one rune
	)

	endLine := func() {
		s.Lines++
		if !lineHas {
			s.BlankLines++
		}
		if lineLen > s.LongestLn {
			s.LongestLn = lineLen
		}
		lineLen, lineHas, lineOpen = 0, false, false
	}

	for {
		r, size, err := br.ReadRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			return s, err
		}
		s.Bytes += int64(size)
		s.Runes++

		if r == '\n' {
			inWord = false
			endLine()
			continue
		}
		lineOpen = true
		if r != '\r' {
			lineLen++
		}

		if unicode.IsSpace(r) {
			inWord = false
			continue
		}
		lineHas = true
		if !inWord && r != utf8.RuneError {
			s.Words++
			inWord = true
		}
	}
	if lineOpen {
		endLine()
	}
	return s, nil
}
