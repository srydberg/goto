package analyzer

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// ScanOptions controls a directory scan.
type ScanOptions struct {
	// IncludeHidden includes files and directories whose name starts with ".".
	IncludeHidden bool
	// MaxDepth limits recursion depth; 0 means unlimited. The root is depth 0.
	MaxDepth int
	// Top is the number of largest files to report.
	Top int
}

// ExtSummary aggregates files sharing an extension.
type ExtSummary struct {
	Extension string `json:"extension"`
	Files     int    `json:"files"`
	Bytes     int64  `json:"bytes"`
}

// FileEntry is a single file with its size.
type FileEntry struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

// ScanResult summarises a directory tree.
type ScanResult struct {
	Root        string       `json:"root"`
	Files       int          `json:"files"`
	Dirs        int          `json:"dirs"`
	TotalBytes  int64        `json:"total_bytes"`
	ByExtension []ExtSummary `json:"by_extension"`
	Largest     []FileEntry  `json:"largest"`
	Errors      []string     `json:"errors,omitempty"`
}

// Scan walks root and aggregates file counts and sizes. Unreadable entries
// are recorded in ScanResult.Errors rather than aborting the scan.
func Scan(root string, opt ScanOptions) (ScanResult, error) {
	res := ScanResult{Root: root}
	byExt := map[string]*ExtSummary{}
	var files []FileEntry

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == root {
				return err
			}
			res.Errors = append(res.Errors, err.Error())
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if path != root && !opt.IncludeHidden && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			if path != root {
				res.Dirs++
				if opt.MaxDepth > 0 && depth(root, path) >= opt.MaxDepth {
					return fs.SkipDir
				}
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			res.Errors = append(res.Errors, err.Error())
			return nil
		}
		size := info.Size()
		res.Files++
		res.TotalBytes += size
		files = append(files, FileEntry{Path: path, Bytes: size})

		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext == "" {
			ext = "(none)"
		}
		e, ok := byExt[ext]
		if !ok {
			e = &ExtSummary{Extension: ext}
			byExt[ext] = e
		}
		e.Files++
		e.Bytes += size
		return nil
	})
	if err != nil {
		return res, err
	}

	for _, e := range byExt {
		res.ByExtension = append(res.ByExtension, *e)
	}
	sort.Slice(res.ByExtension, func(i, j int) bool {
		a, b := res.ByExtension[i], res.ByExtension[j]
		if a.Bytes != b.Bytes {
			return a.Bytes > b.Bytes
		}
		return a.Extension < b.Extension
	})

	sort.Slice(files, func(i, j int) bool {
		if files[i].Bytes != files[j].Bytes {
			return files[i].Bytes > files[j].Bytes
		}
		return files[i].Path < files[j].Path
	})
	if opt.Top >= 0 && len(files) > opt.Top {
		files = files[:opt.Top]
	}
	res.Largest = files
	return res, nil
}

func depth(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	return strings.Count(rel, string(filepath.Separator)) + 1
}
