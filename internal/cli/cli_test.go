package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func run(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	code = Run(args, &out, &errb, "test")
	return code, out.String(), errb.String()
}

func TestUsageAndVersion(t *testing.T) {
	if code, _, errOut := run(t); code != exitUsage || !strings.Contains(errOut, "Commands:") {
		t.Errorf("no args: code=%d stderr=%q", code, errOut)
	}
	if code, out, _ := run(t, "--version"); code != exitOK || out != "goto test\n" {
		t.Errorf("version: code=%d out=%q", code, out)
	}
	if code, _, _ := run(t, "nope"); code != exitUsage {
		t.Errorf("unknown command: code=%d", code)
	}
	if code, out, _ := run(t, "help", "hash"); code != exitOK || !strings.Contains(out, "goto hash") {
		t.Errorf("help hash: code=%d out=%q", code, out)
	}
}

func TestStatsCommand(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f.txt")
	os.WriteFile(p, []byte("one two\nthree\n"), 0o644)

	code, out, _ := run(t, "stats", "-o", "json", p)
	if code != exitOK {
		t.Fatalf("code=%d", code)
	}
	var res []map[string]any
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0]["lines"].(float64) != 2 || res[0]["words"].(float64) != 3 {
		t.Errorf("got %v", res)
	}

	code, _, errOut := run(t, "stats", p, filepath.Join(t.TempDir(), "missing"))
	if code != exitError || !strings.Contains(errOut, "missing") {
		t.Errorf("missing file: code=%d stderr=%q", code, errOut)
	}
}

func TestHashCommand(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f.txt")
	os.WriteFile(p, []byte("abc"), 0o644)

	code, out, _ := run(t, "hash", "-a", "md5", p)
	if code != exitOK || out != "900150983cd24fb0d6963f7d28e17f72  "+p+"\n" {
		t.Errorf("code=%d out=%q", code, out)
	}
	if code, _, _ := run(t, "hash", "-a", "bogus", p); code != exitError {
		t.Errorf("bogus algo: code=%d", code)
	}
}

func TestInvalidFormat(t *testing.T) {
	if code, _, _ := run(t, "stats", "-o", "xml", "x"); code != exitUsage {
		t.Errorf("code=%d", code)
	}
}

func TestScanCommand(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi"), 0o644)
	code, out, _ := run(t, "scan", dir)
	if code != exitOK || !strings.Contains(out, "Files: 1") {
		t.Errorf("code=%d out=%q", code, out)
	}
}
