# goto

Unix tools implemented in Go – ett kommandoradsverktyg för att analysera filer.
Byggt enbart med Go:s standardbibliotek (inga externa beroenden).

## Installation

```sh
go install github.com/srydberg/goto/cmd/goto@latest
# eller lokalt
make build        # -> bin/goto
```

## Användning

```
goto <kommando> [flaggor] [argument]
```

| Kommando | Beskrivning |
|----------|-------------|
| `stats`  | Räknar rader, tomma rader, ord, tecken, byte och längsta rad |
| `type`   | Identifierar MIME-typ, text/binär, teckenkodning och radslut |
| `hash`   | Beräknar checksummor (`md5`, `sha1`, `sha256`, `sha512`) |
| `scan`   | Sammanfattar ett katalogträd per filändelse och storlek |

Alla kommandon tar `-o json` (eller `-format json`) för maskinläsbar utdata.
Flaggor ska stå före filargumenten. `goto help <kommando>` visar detaljer.

### Exempel

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
$ goto scan -hidden -o json ~/projekt | jq '.by_extension[:5]'
```

### Avslutningskoder

| Kod | Betydelse |
|-----|-----------|
| 0   | OK |
| 1   | En eller flera filer kunde inte analyseras (övriga skrivs ändå ut) |
| 2   | Felaktig användning (okänt kommando, ogiltig flagga) |

## Projektstruktur

```
cmd/goto/            main-paket, startpunkt
internal/cli/        argumentparsning, underkommandon och utskrift
internal/analyzer/   analyslogik (stats, typdetektering, hash, scan) – oberoende av CLI
internal/output/     text-/JSON-formattering, tabeller, byte-formatering
```

### Lägga till ett nytt kommando

1. Lägg analyslogiken i `internal/analyzer/` (med tester).
2. Registrera kommandot med `register(&command{...})` i `internal/cli/commands.go`.
   `setup` får en `*flag.FlagSet` för egna flaggor och returnerar körfunktionen.

## Utveckling

```sh
make test    # go test -race ./...
make vet
make cover
```
