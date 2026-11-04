package analyzer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatsReader(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want Stats
	}{
		{"empty", "", Stats{}},
		{"no trailing newline", "hello world", Stats{Bytes: 11, Runes: 11, Lines: 1, Words: 2, LongestLn: 11}},
		{"blank lines", "a\n\n  \nbb cc\n", Stats{Bytes: 12, Runes: 12, Lines: 4, BlankLines: 2, Words: 3, LongestLn: 5}},
		{"crlf", "ab\r\ncd\r\n", Stats{Bytes: 8, Runes: 8, Lines: 2, Words: 2, LongestLn: 2}},
		{"utf8", "åäö ü\n", Stats{Bytes: 10, Runes: 6, Lines: 1, Words: 2, LongestLn: 5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StatsReader(strings.NewReader(tt.in))
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDetect(t *testing.T) {
	tests := []struct {
		name     string
		in       []byte
		binary   bool
		encoding string
		lineEnd  string
	}{
		{"empty", nil, false, "empty", ""},
		{"ascii lf", []byte("a\nb\n"), false, "ascii", "LF"},
		{"utf8 crlf", []byte("å\r\nä\r\n"), false, "utf-8", "CRLF"},
		{"mixed", []byte("a\nb\r\n"), false, "ascii", "mixed (LF, CRLF)"},
		{"bom", []byte("\xEF\xBB\xBFhi"), false, "utf-8 (BOM)", "none"},
		{"latin1", []byte("caf\xe9"), false, "unknown (8-bit)", "none"},
		{"binary", []byte{0x7f, 'E', 'L', 'F', 0, 0, 1}, true, "binary", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Detect(tt.in)
			if got.Binary != tt.binary || got.Encoding != tt.encoding || got.LineEnd != tt.lineEnd {
				t.Errorf("got binary=%v encoding=%q lineEnd=%q, want %v %q %q",
					got.Binary, got.Encoding, got.LineEnd, tt.binary, tt.encoding, tt.lineEnd)
			}
		})
	}
}

func TestTrimPartialRune(t *testing.T) {
	// "å" is 0xC3 0xA5; cut after the first byte.
	if got := trimPartialRune([]byte("ab\xc3")); string(got) != "ab" {
		t.Errorf("got %q", got)
	}
	if got := trimPartialRune([]byte("abå")); string(got) != "abå" {
		t.Errorf("got %q", got)
	}
}

func TestHashFile(t *testing.T) {
	p := writeFile(t, t.TempDir(), "f.txt", "abc")
	r, err := HashFile(p, []string{"MD5", "sha256", "md5"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"md5":    "900150983cd24fb0d6963f7d28e17f72",
		"sha256": "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
	}
	if len(r.Digests) != len(want) {
		t.Fatalf("got %d digests, want %d", len(r.Digests), len(want))
	}
	for k, v := range want {
		if r.Digests[k] != v {
			t.Errorf("%s = %s, want %s", k, r.Digests[k], v)
		}
	}
	if _, err := HashFile(p, []string{"crc32"}); err == nil {
		t.Error("expected error for unknown algorithm")
	}
}

func TestScan(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.go", "package a\n")
	writeFile(t, root, "b.GO", "package b\n\n")
	writeFile(t, root, "README", "x")
	writeFile(t, root, "sub/c.txt", "hello")
	writeFile(t, root, "sub/deep/d.txt", "1234567890123")
	writeFile(t, root, ".hidden/e.txt", "secret")

	res, err := Scan(root, ScanOptions{Top: 2})
	if err != nil {
		t.Fatal(err)
	}
	if res.Files != 5 || res.Dirs != 2 {
		t.Errorf("files=%d dirs=%d, want 5 and 2", res.Files, res.Dirs)
	}
	if res.TotalBytes != 10+11+1+5+13 {
		t.Errorf("total=%d", res.TotalBytes)
	}
	if len(res.Largest) != 2 || filepath.Base(res.Largest[0].Path) != "d.txt" {
		t.Errorf("largest=%+v", res.Largest)
	}
	if res.ByExtension[0].Extension != ".go" || res.ByExtension[0].Files != 2 {
		t.Errorf("by ext=%+v", res.ByExtension)
	}

	res, _ = Scan(root, ScanOptions{IncludeHidden: true, Top: 10})
	if res.Files != 6 {
		t.Errorf("with hidden: files=%d, want 6", res.Files)
	}

	res, _ = Scan(root, ScanOptions{MaxDepth: 1, Top: 10})
	if res.Files != 3 {
		t.Errorf("depth 1: files=%d, want 3", res.Files)
	}
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}
