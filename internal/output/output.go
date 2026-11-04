// Package output provides helpers for rendering results as text or JSON.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// Format is an output format.
type Format string

const (
	Text Format = "text"
	JSON Format = "json"
)

// ParseFormat validates a format name.
func ParseFormat(s string) (Format, error) {
	switch f := Format(strings.ToLower(s)); f {
	case Text, JSON:
		return f, nil
	default:
		return "", fmt.Errorf("unknown format %q (use text or json)", s)
	}
}

// WriteJSON writes v as indented JSON followed by a newline.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Table renders aligned columns.
type Table struct {
	tw *tabwriter.Writer
}

// NewTable creates a table that writes to w. If headers are given they are
// written as the first row.
func NewTable(w io.Writer, headers ...string) *Table {
	t := &Table{tw: tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)}
	if len(headers) > 0 {
		t.Row(headers...)
	}
	return t
}

// Row appends a row.
func (t *Table) Row(cols ...string) {
	fmt.Fprintln(t.tw, strings.Join(cols, "\t"))
}

// Flush writes the buffered table.
func (t *Table) Flush() error { return t.tw.Flush() }

// HumanBytes formats n using binary units (KiB, MiB, ...).
func HumanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
