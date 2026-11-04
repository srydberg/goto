// Package cli implements command-line parsing and dispatch for goto.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/srydberg/goto/internal/output"
)

// Exit codes.
const (
	exitOK    = 0
	exitError = 1 // one or more files could not be analysed
	exitUsage = 2 // invalid invocation
)

// errUsage signals that the command was invoked incorrectly.
var errUsage = errors.New("usage error")

// env carries shared state into each command.
type env struct {
	stdout io.Writer
	stderr io.Writer
	format output.Format
}

// command is a single subcommand.
type command struct {
	name    string
	summary string
	usage   string
	// setup registers command-specific flags and returns the function that runs it.
	setup func(fs *flag.FlagSet) func(e *env, args []string) error
}

var commands = map[string]*command{}

func register(c *command) { commands[c.name] = c }

// Run executes the CLI with args (excluding the program name) and returns the
// process exit code.
func Run(args []string, stdout, stderr io.Writer, version string) int {
	if len(args) == 0 {
		printUsage(stderr)
		return exitUsage
	}

	switch args[0] {
	case "-h", "--help", "help":
		if len(args) > 1 {
			if c, ok := commands[args[1]]; ok {
				newFlagSet(c, stdout).Usage()
				return exitOK
			}
		}
		printUsage(stdout)
		return exitOK
	case "-v", "--version", "version":
		fmt.Fprintf(stdout, "goto %s\n", version)
		return exitOK
	}

	c, ok := commands[args[0]]
	if !ok {
		fmt.Fprintf(stderr, "goto: unknown command %q\n\n", args[0])
		printUsage(stderr)
		return exitUsage
	}

	fs := newFlagSet(c, stderr)
	format := fs.String("format", "text", "output format: text or json")
	fs.StringVar(format, "o", "text", "shorthand for -format")
	run := c.setup(fs)
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitUsage
	}

	f, err := output.ParseFormat(*format)
	if err != nil {
		fmt.Fprintf(stderr, "goto %s: %v\n", c.name, err)
		return exitUsage
	}

	e := &env{stdout: stdout, stderr: stderr, format: f}
	if err := run(e, fs.Args()); err != nil {
		if errors.Is(err, errUsage) {
			fs.Usage()
			return exitUsage
		}
		fmt.Fprintf(stderr, "goto %s: %v\n", c.name, err)
		return exitError
	}
	return exitOK
}

func newFlagSet(c *command, w io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(c.name, flag.ContinueOnError)
	fs.SetOutput(w)
	fs.Usage = func() {
		fmt.Fprintf(w, "Usage: goto %s %s\n\n%s\n\nFlags:\n", c.name, c.usage, c.summary)
		fs.PrintDefaults()
	}
	return fs
}

func printUsage(w io.Writer) {
	names := make([]string, 0, len(commands))
	for n := range commands {
		names = append(names, n)
	}
	sort.Strings(names)

	var b strings.Builder
	b.WriteString("goto - analyze files from the command line\n\n")
	b.WriteString("Usage:\n  goto <command> [flags] [arguments]\n\nCommands:\n")
	t := output.NewTable(&b)
	for _, n := range names {
		t.Row("  "+n, commands[n].summary)
	}
	t.Flush()
	b.WriteString("\nGlobal flags (accepted by every command):\n")
	b.WriteString("  -o, -format string   output format: text or json (default \"text\")\n")
	b.WriteString("\nRun 'goto help <command>' for details on a command.\n")
	fmt.Fprint(w, b.String())
}

// multiError collects per-file failures so a command can keep going and
// report everything at the end.
type multiError struct{ errs []error }

func (m *multiError) add(err error) { m.errs = append(m.errs, err) }

func (m *multiError) report(w io.Writer, cmd string) error {
	for _, err := range m.errs {
		fmt.Fprintf(w, "goto %s: %v\n", cmd, err)
	}
	if len(m.errs) == 0 {
		return nil
	}
	return fmt.Errorf("%d file(s) failed", len(m.errs))
}
