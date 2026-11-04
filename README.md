# goto

Unix tools implemented in Go – a command-line tool for analyzing files.
Built with the Go standard library only (no external dependencies).

## Installation

```sh
go install github.com/srydberg/goto/cmd/goto@latest
# or locally
make build        # -> bin/goto
```

## Usage

```
goto <command> [flags] [arguments]
```

| Command | Description |
|---------|-------------|
| `stats` | Counts lines, blank lines, words, characters, bytes and the longest line |
| `type`  | Detects MIME type, text/binary, character encoding and line endings |
| `hash`  | Computes checksums (`md5`, `sha1`, `sha256`, `sha512`) |
| `scan`  | Summarizes a directory tree by file extension and size |

Every command accepts `-o json` (or `-format json`) for machine-readable output.
Flags must come before file arguments. `goto help <command>` shows details.

### Examples

```sh
$ goto stats LICENSE README.md
LINES  BLANK  WORDS  CHARS  BYTES  LONGEST  FILE
21     4      169    1074   1074   78       LICENSE
2      0      7      36     36     28       README.md
23     4      176    1110   1110   78       total

$ goto type LICENSE bin/goto
FILE      MIME                       ENCODING  LINE ENDINGS
LICENSE   text/plain; charset=utf-8  ascii     LF
bin/goto  application/octet-stream   binary    -

$ goto hash -a md5,sha256 LICENSE
md5:75014920b7423d77a81595bbabf4d158  LICENSE
sha256:a8b4b3dd…  LICENSE

$ goto scan -top 3 -depth 2 .
$ goto scan -hidden -o json ~/projects | jq '.by_extension[:5]'
```

### Exit codes

| Code | Meaning |
|------|---------|
| 0    | Success |
| 1    | One or more files could not be analyzed (the rest are still printed) |
| 2    | Invalid usage (unknown command, invalid flag) |

## Project layout

```
cmd/goto/            main package, entry point
internal/cli/        argument parsing, subcommands and printing
internal/analyzer/   analysis logic (stats, type detection, hash, scan) – independent of the CLI
internal/output/     text/JSON formatting, tables, byte formatting
```

### Adding a new command

1. Put the analysis logic in `internal/analyzer/` (with tests).
2. Register the command with `register(&command{...})` in `internal/cli/commands.go`.
   `setup` receives a `*flag.FlagSet` for its own flags and returns the run function.

## Development

```sh
make test    # go test -race ./...
make vet
make cover
```
