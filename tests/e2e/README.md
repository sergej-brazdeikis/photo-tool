# End-to-end tests (`tests/e2e`)

## CLI black-box

[`cli_test.go`](cli_test.go) builds the real `photo-tool` binary (unless `PHOTO_TOOL_E2E_BIN` points at a prebuilt one), sets `PHOTO_TOOL_LIBRARY` to a temp library, and runs `phototool scan` / `phototool import` with assertions on exit codes, stdout, and database state.

Run from the module root:

```bash
go test ./tests/e2e/... -count=1
```

## Desktop GUI (Fyne)

Multi-step shell journeys and widget-level UI tests live in the main module under [`internal/app`](../../internal/app) (for example `e2e_shell_journeys_test.go`, `review_test.go`, `nfr01_layout_gate_test.go`). They use `fyne.io/fyne/v2/test`, not a browser driver.

## HTTP share (loopback)

Canonical automated coverage for share links (mint token → GET handler → HTML / image / errors) is in [`internal/share/http_test.go`](../../internal/share/http_test.go). That file is the single source of truth for HTTP contract E2E; avoid copying scenarios here so fixes stay in one place.

## Manual UX / visual evaluation

See [`_bmad-output/test-artifacts/e2e-usability-rubric.md`](../../_bmad-output/test-artifacts/e2e-usability-rubric.md) for heuristic checklists, task scenarios, and release sign-off. Use that doc for picker-heavy flows and subjective look-and-feel; re-run tasks before releases.

## Maintaining Tier 1 automation

When behavior changes, extend the closest layer (avoid duplicating HTTP scenarios outside [`internal/share/http_test.go`](../../internal/share/http_test.go)):

| Change touches | Extend |
|----------------|--------|
| CLI flags / exit codes / scan-import semantics | [`cli_test.go`](cli_test.go) |
| Share HTTP routes or response shape | [`internal/share/http_test.go`](../../internal/share/http_test.go) |
| Primary nav or cross-panel shell | [`internal/app/e2e_shell_journeys_test.go`](../../internal/app/e2e_shell_journeys_test.go) |
| Upload receipt or collection confirm/cancel | [`internal/app/upload.go`](../../internal/app/upload.go) + [`upload_fr06_flow_test.go`](../../internal/app/upload_fr06_flow_test.go) |

## Optional: LLM judge bundle

[`scripts/assemble-judge-bundle.sh`](../../scripts/assemble-judge-bundle.sh) writes `_bmad-output/test-artifacts/judge-bundles/<stamp>/` with logs, copied rubric, and `manifest.json`. Pair with [`_bmad-output/test-artifacts/judge-prompt-v1.md`](../../_bmad-output/test-artifacts/judge-prompt-v1.md) in Cursor CLI for qualitative review (advisory only).
