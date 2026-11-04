package cli

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/srydberg/goto/internal/analyzer"
	"github.com/srydberg/goto/internal/output"
)

func init() {
	register(&command{
		name:    "stats",
		summary: "Count lines, words, characters and bytes",
		usage:   "[flags] <file>...",
		setup:   setupStats,
	})
	register(&command{
		name:    "type",
		summary: "Detect MIME type, encoding and line endings",
		usage:   "[flags] <file>...",
		setup:   setupType,
	})
	register(&command{
		name:    "hash",
		summary: "Compute checksums (md5, sha1, sha256, sha512)",
		usage:   "[flags] <file>...",
		setup:   setupHash,
	})
	register(&command{
		name:    "scan",
		summary: "Summarize a directory tree by extension and size",
		usage:   "[flags] [dir]",
		setup:   setupScan,
	})
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// --- stats ---------------------------------------------------------------

func setupStats(fs *flag.FlagSet) func(*env, []string) error {
	return func(e *env, args []string) error {
		if len(args) == 0 {
			return errUsage
		}
		var (
			results = []analyzer.Stats{}
			total   = analyzer.Stats{Path: "total"}
			errs    multiError
		)
		for _, p := range args {
			s, err := analyzer.StatsFile(p)
			if err != nil {
				errs.add(err)
				continue
			}
			results = append(results, s)
			total.Bytes += s.Bytes
			total.Lines += s.Lines
			total.BlankLines += s.BlankLines
			total.Words += s.Words
			total.Runes += s.Runes
			total.LongestLn = max(total.LongestLn, s.LongestLn)
		}

		if e.format == output.JSON {
			if err := output.WriteJSON(e.stdout, results); err != nil {
				return err
			}
		} else if len(results) > 0 {
			t := output.NewTable(e.stdout, "LINES", "BLANK", "WORDS", "CHARS", "BYTES", "LONGEST", "FILE")
			row := func(s analyzer.Stats) {
				t.Row(itoa(s.Lines), itoa(s.BlankLines), itoa(s.Words), itoa(s.Runes),
					itoa(s.Bytes), itoa(s.LongestLn), s.Path)
			}
			for _, s := range results {
				row(s)
			}
			if len(results) > 1 {
				row(total)
			}
			t.Flush()
		}
		return errs.report(e.stderr, "stats")
	}
}

// --- type ----------------------------------------------------------------

func setupType(fs *flag.FlagSet) func(*env, []string) error {
	return func(e *env, args []string) error {
		if len(args) == 0 {
			return errUsage
		}
		var (
			results = []analyzer.TypeInfo{}
			errs    multiError
		)
		for _, p := range args {
			info, err := analyzer.DetectFile(p)
			if err != nil {
				errs.add(err)
				continue
			}
			results = append(results, info)
		}

		if e.format == output.JSON {
			if err := output.WriteJSON(e.stdout, results); err != nil {
				return err
			}
		} else if len(results) > 0 {
			t := output.NewTable(e.stdout, "FILE", "MIME", "ENCODING", "LINE ENDINGS")
			for _, r := range results {
				le := r.LineEnd
				if le == "" {
					le = "-"
				}
				t.Row(r.Path, r.MIME, r.Encoding, le)
			}
			t.Flush()
		}
		return errs.report(e.stderr, "type")
	}
}

// --- hash ----------------------------------------------------------------

func setupHash(fs *flag.FlagSet) func(*env, []string) error {
	algos := fs.String("a", "sha256",
		"comma-separated algorithms: "+strings.Join(analyzer.HashAlgorithms(), ", "))
	return func(e *env, args []string) error {
		if len(args) == 0 {
			return errUsage
		}
		list, err := analyzer.ParseAlgorithms(*algos)
		if err != nil {
			return err
		}
		var (
			results = []analyzer.HashResult{}
			errs    multiError
		)
		for _, p := range args {
			r, err := analyzer.HashFile(p, list)
			if err != nil {
				errs.add(err)
				continue
			}
			results = append(results, r)
		}

		if e.format == output.JSON {
			if err := output.WriteJSON(e.stdout, results); err != nil {
				return err
			}
		} else {
			// Same layout as sha256sum & friends: "<digest>  <file>", one line
			// per algorithm, in the order requested.
			for _, r := range results {
				for _, a := range list {
					prefix := ""
					if len(r.Digests) > 1 {
						prefix = a + ":"
					}
					fmt.Fprintf(e.stdout, "%s%s  %s\n", prefix, r.Digests[a], r.Path)
				}
			}
		}
		return errs.report(e.stderr, "hash")
	}
}

// --- scan ----------------------------------------------------------------

func setupScan(fs *flag.FlagSet) func(*env, []string) error {
	hidden := fs.Bool("hidden", false, "include hidden files and directories")
	maxDepth := fs.Int("depth", 0, "maximum directory depth (0 = unlimited)")
	top := fs.Int("top", 10, "number of largest files to list")
	return func(e *env, args []string) error {
		if len(args) > 1 {
			return errUsage
		}
		root := "."
		if len(args) == 1 {
			root = args[0]
		}
		if st, err := os.Stat(root); err != nil {
			return err
		} else if !st.IsDir() {
			return fmt.Errorf("%s: not a directory", root)
		}

		res, err := analyzer.Scan(root, analyzer.ScanOptions{
			IncludeHidden: *hidden,
			MaxDepth:      *maxDepth,
			Top:           *top,
		})
		if err != nil {
			return err
		}

		if e.format == output.JSON {
			return output.WriteJSON(e.stdout, res)
		}

		w := e.stdout
		fmt.Fprintf(w, "Root:  %s\nFiles: %d\nDirs:  %d\nSize:  %s (%d bytes)\n",
			res.Root, res.Files, res.Dirs, output.HumanBytes(res.TotalBytes), res.TotalBytes)

		if len(res.ByExtension) > 0 {
			fmt.Fprintln(w, "\nBy extension:")
			t := output.NewTable(w, "  EXT", "FILES", "SIZE", "SHARE")
			for _, x := range res.ByExtension {
				share := 0.0
				if res.TotalBytes > 0 {
					share = float64(x.Bytes) * 100 / float64(res.TotalBytes)
				}
				t.Row("  "+x.Extension, strconv.Itoa(x.Files), output.HumanBytes(x.Bytes),
					fmt.Sprintf("%.1f%%", share))
			}
			t.Flush()
		}
		if len(res.Largest) > 0 {
			fmt.Fprintln(w, "\nLargest files:")
			t := output.NewTable(w)
			for _, f := range res.Largest {
				t.Row("  "+output.HumanBytes(f.Bytes), f.Path)
			}
			t.Flush()
		}
		for _, msg := range res.Errors {
			fmt.Fprintf(e.stderr, "goto scan: %s\n", msg)
		}
		return nil
	}
}
