// Command goto is a command-line tool for analyzing files.
package main

import (
	"os"

	"github.com/srydberg/goto/internal/cli"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, version))
}
