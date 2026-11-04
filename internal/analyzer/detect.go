package analyzer

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// sniffLen is the number of bytes inspected to detect content type.
const sniffLen = 512

// TypeInfo describes the detected type of a file.
type TypeInfo struct {
	Path      string `json:"path"`
	Extension string `json:"extension"`
	MIME      string `json:"mime"`
	Binary    bool   `json:"binary"`
	Encoding  string `json:"encoding"`
	LineEnd   string `json:"line_endings,omitempty"`
}

// DetectFile inspects the start of the file at path and reports its type.
func DetectFile(path string) (TypeInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return TypeInfo{}, err
	}
	defer f.Close()

	buf := make([]byte, 64*1024)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return TypeInfo{}, err
	}
	sample := buf[:n]
	if n == len(buf) {
		// The file is larger than the buffer, so the sample may end mid-rune.
		sample = trimPartialRune(sample)
	}
	info := Detect(sample)
	info.Path = path
	info.Extension = strings.ToLower(filepath.Ext(path))
	return info, nil
}

// Detect classifies a content sample.
func Detect(sample []byte) TypeInfo {
	head := sample
	if len(head) > sniffLen {
		head = head[:sniffLen]
	}
	info := TypeInfo{
		MIME:   http.DetectContentType(head),
		Binary: IsBinary(sample),
	}
	switch {
	case len(sample) == 0:
		info.Encoding = "empty"
	case info.Binary:
		info.Encoding = "binary"
	case bytes.HasPrefix(sample, []byte{0xEF, 0xBB, 0xBF}):
		info.Encoding = "utf-8 (BOM)"
	case utf8.Valid(sample):
		if isASCII(sample) {
			info.Encoding = "ascii"
		} else {
			info.Encoding = "utf-8"
		}
	default:
		info.Encoding = "unknown (8-bit)"
	}
	if !info.Binary && len(sample) > 0 {
		info.LineEnd = lineEndings(sample)
	}
	return info
}

// IsBinary reports whether sample looks like binary data. It uses the same
// heuristic as many tools (e.g. git, grep): presence of a NUL byte, or a high
// proportion of non-text control characters.
func IsBinary(sample []byte) bool {
	if len(sample) == 0 {
		return false
	}
	if bytes.IndexByte(sample, 0) >= 0 {
		return true
	}
	var ctrl int
	for _, b := range sample {
		if b < 0x20 && b != '\n' && b != '\r' && b != '\t' && b != '\f' && b != '\b' && b != 0x1b {
			ctrl++
		}
	}
	return ctrl*100/len(sample) > 10
}

func isASCII(b []byte) bool {
	for _, c := range b {
		if c >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// trimPartialRune drops an incomplete UTF-8 sequence at the end of a sample
// that was cut off mid-rune.
func trimPartialRune(b []byte) []byte {
	for i := 1; i < utf8.UTFMax && i <= len(b); i++ {
		if utf8.RuneStart(b[len(b)-i]) {
			if !utf8.FullRune(b[len(b)-i:]) {
				return b[:len(b)-i]
			}
			break
		}
	}
	return b
}

func lineEndings(b []byte) string {
	crlf := bytes.Count(b, []byte("\r\n"))
	lf := bytes.Count(b, []byte("\n")) - crlf
	cr := bytes.Count(b, []byte("\r")) - crlf
	var kinds []string
	if lf > 0 {
		kinds = append(kinds, "LF")
	}
	if crlf > 0 {
		kinds = append(kinds, "CRLF")
	}
	if cr > 0 {
		kinds = append(kinds, "CR")
	}
	switch len(kinds) {
	case 0:
		return "none"
	case 1:
		return kinds[0]
	default:
		return "mixed (" + strings.Join(kinds, ", ") + ")"
	}
}
