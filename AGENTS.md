# Agent guide — photo-tool

Local-first photo library: Fyne desktop app (default), CLI subcommands (`scan`, `import`), loopback HTTP share pages.

## Read order

1. This file
2. [`docs/README.md`](docs/README.md) — focused guides (env, codebase map, share NFR docs)
3. [`_bmad-output/planning-artifacts/architecture.md`](_bmad-output/planning-artifacts/architecture.md) §4–5 — naming, boundaries, agent MUST rules
4. Feature-specific code (see table below)
5. Deep requirements: [`PRD.md`](_bmad-output/planning-artifacts/PRD.md), [`ux-design-specification.md`](_bmad-output/planning-artifacts/ux-design-specification.md)

## Entry points

| Mode | File |
|------|------|
| GUI (no args) | [`main.go`](main.go) → `internal/app` shell |
| CLI (with args) | [`internal/cli/root.go`](internal/cli/root.go) — **no Fyne imports** |

## Package map

```text
internal/app/       Fyne UI (shell, upload, review, collections, rejected)
internal/cli/       Cobra commands (no Fyne imports)
internal/config/    PHOTO_TOOL_* env, library root, share HTTP config
internal/domain/    Shared types: OperationSummary, filters, layout constants
internal/ingest/    Upload/scan/import + dedup (single code path)
internal/exifmeta/  Capture time + camera metadata
internal/filehash/  SHA-256
internal/paths/     Canonical library path layout
internal/store/     SQLite repos + migrations/
internal/share/     Loopback HTTP server, handlers, rate limits
web/share/          Embedded HTML/CSS templates
```

## Hard rules (architecture §4.5)

- **One** dedup implementation for upload, scan, and import (`internal/ingest`)
- **One** `OperationSummary` schema for GUI and CLI (`internal/domain/summary.go`)
- **Never** store raw share tokens in DB — hash only
- **Default queries** exclude `rejected` and `deleted` unless the view explicitly requests them
- **Share HTML** must not emit raw GPS coordinates

## Where to edit

| Task | Start here |
|------|------------|
| Upload / drag-drop | `internal/app/upload.go`, `internal/ingest/` |
| Review / loupe / filters | `internal/app/review.go`, `review_loupe.go`, `review_grid.go`, `internal/store/review_query.go` |
| Collections | `internal/app/collections.go`, `internal/store/collections.go` |
| Share mint + HTTP | `internal/app/share_*.go`, `internal/share/`, `web/share/` |
| CLI parity | `internal/cli/`, `internal/domain/summary.go` |
| Schema change | `internal/store/migrations/`, `migrate.go` |

## Commands

```bash
make build      # → bin/photo-tool
make test       # go test ./...
make test-e2e   # CLI black-box (tests/e2e)
make test-ci    # Fyne software driver (-tags ci)
make gate       # full module gate (tidy, fmt, vet, test, build)
```

See [`Makefile`](Makefile) for `ux-judge-loop`, `judge-bundle`, etc.

## Tests

Layering is documented in [`tests/e2e/README.md`](tests/e2e/README.md).

- **CLI E2E:** `tests/e2e/cli_test.go`
- **HTTP share contract:** `internal/share/http_test.go` — **single source of truth**; do not duplicate scenarios elsewhere
- **Fyne UI:** colocated `internal/app/*_test.go`, `e2e_shell_journeys_test.go`
- **NFR-05 cold load:** `internal/share/nfr05_cold_load_test.go` (methodology: [`docs/share-cold-load-nfr05.md`](docs/share-cold-load-nfr05.md))

Extend the **closest** existing test file when behavior changes.

## Requirements IDs

Comments and tests reference **FR-***, **NFR-***, and **UX-DR*** IDs. Definitions live in:

- [`PRD.md`](_bmad-output/planning-artifacts/PRD.md) — functional and non-functional requirements
- [`ux-design-specification.md`](_bmad-output/planning-artifacts/ux-design-specification.md) — layout and UX decisions

Story history: `_bmad-output/implementation-artifacts/*.md`

## BMAD automation (local only)

Optional scripts: `scripts/bmad-story-workflow.sh`, `make ux-judge-loop`. Require Cursor `agent` CLI on PATH.

**Never add LLM or agent steps to CI** (`.github/workflows/go.yml`).

## Environment variables

Runtime and test overrides: [`docs/env.md`](docs/env.md).
