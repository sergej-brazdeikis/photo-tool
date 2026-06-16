# Photo Tool

Local-first photo library for macOS, Windows, and Linux. Import and organize photos, review with ratings and tags, manage collections, and mint read-only share links from the desktop app.

**Stack:** Go · Fyne · SQLite · embedded share HTML

## Prerequisites

- Go **1.26+** ([`go.mod`](go.mod))
- Fyne system dependencies for the GUI ([Fyne docs](https://docs.fyne.io/started/))

## Quick start

```bash
make build
make run          # launches the GUI
# or
./bin/photo-tool
```

Library data defaults to `~/.config/photo-tool/library` (Linux). Override with:

```bash
export PHOTO_TOOL_LIBRARY=/path/to/library
./bin/photo-tool
```

## CLI

With arguments, the binary runs CLI subcommands (no GUI):

```bash
./bin/photo-tool scan --dir ~/Pictures
./bin/photo-tool scan --dir ~/Pictures --recursive --dry-run
./bin/photo-tool import --dir /path/under/library/root
```

## Development

```bash
make test         # unit + integration tests
make test-e2e     # CLI black-box tests
make test-ci      # Fyne CI driver tests
make gate         # full module gate
```

## Documentation

| Doc | Audience |
|-----|----------|
| [**AGENTS.md**](AGENTS.md) | AI assistants and contributors — start here for code navigation |
| [**docs/**](docs/README.md) | Environment variables, codebase map, share NFR methodology |
| [**tests/e2e/README.md**](tests/e2e/README.md) | Test layering and where to extend coverage |
| [**_bmad-output/planning-artifacts/**](_bmad-output/planning-artifacts/) | PRD, architecture, UX specification |

**Contributing with AI?** Read [AGENTS.md](AGENTS.md) first.
