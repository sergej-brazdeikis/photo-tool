# End-to-end tests (`tests/e2e`)

## CLI black-box

[`cli_test.go`](cli_test.go) builds the real `photo-tool` binary (unless `PHOTO_TOOL_E2E_BIN` points at a prebuilt one), sets `PHOTO_TOOL_LIBRARY` to a temp library, and runs `phototool scan` / `phototool import` with assertions on exit codes, stdout, and database state.

Run from the module root:

```bash
go test ./tests/e2e/... -count=1
```

## Desktop GUI (Fyne)

Multi-step shell journeys and widget-level UI tests live in the main module under [`internal/app`](../../internal/app) (for example `e2e_shell_journeys_test.go`, `review_test.go`, `nfr01_layout_gate_test.go`). They use `fyne.io/fyne/v2/test`, not a browser driver.

**Light (white) theme for tests:** set `PHOTO_TOOL_TEST_THEME=light` or `white` when running `go test ./internal/app/...` so tests that default to dark use [`applyTestPhotoToolTheme`](../../internal/app/test_theme_env_test.go) with `theme.VariantLight` (UX capture screenshots, layout invariants, etc.). Matrix tests that sweep both variants (e.g. NFR-01) are unchanged.

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

## CI vs local UX gates

- **CI** runs `go test` and shell syntax checks only — **no** Cursor `agent`, **no** LLM. Deterministic UX checks live in the main module (for example `internal/app/ux_layout_invariants_test.go` for the shipping shell).
- **Local:** [`scripts/assemble-judge-bundle.sh`](../../scripts/assemble-judge-bundle.sh) produces `_bmad-output/test-artifacts/judge-bundles/<stamp>/` with `logs/go-test*.txt`, `context/rubric.md`, `ui/*.png` + `ui/steps.json` + `ui/README.md`, and `manifest.json` (including `steps` + `capture_manifest` from `steps.json`). Capture uses `PHOTO_TOOL_UX_JOURNEY_TEST=1` and `PHOTO_TOOL_UX_CAPTURE_DIR=<bundle>/ui`. [`TestUXJourneyCapture`](../../internal/app/ux_journey_capture_test.go) walks **all primary shell flows** in order (Upload empty → Review grid/loupe/Share preview/filters → Collections list/new-album dialog/detail/groupings/back → Rejected + narrow filter → second shell pass: Upload staged + FR-06 + post-confirm). Each step has a `flow` field in `steps.json` for the judge.
- **CI (optional smoke):** one macOS matrix leg runs `TestUXJourneyCapture` with those env vars and uploads PNGs as a workflow artifact — still **no** LLM/`agent`.
- **Closed loop (local only):** [`scripts/ux-judge-loop.sh`](../../scripts/ux-judge-loop.sh) or `make ux-judge-loop` — re-bundles each round, runs `agent` as judge, then implementer, then a **non-interactive** post-fix code review (BMAD `bmad-code-review`–style, writes `verdict/code-review-post-fix.md` in the bundle) until the verdict’s last line is `UX_JUDGE_RESULT=pass` or `UX_LOOP_MAX` (default `5`). Set `UX_LOOP_POST_FIX_REVIEW=0` to skip the review step. Set `UX_AGENT` or `CURSOR_AGENT` if your CLI binary is not named `agent`. To run **until a local wall-clock time** (for example 6pm), use `UX_LOOP_UNTIL_LOCAL=18:00` with `UX_LOOP_NO_MAX=1` (or combine with a high `UX_LOOP_MAX` as a safety cap). **Do not add this to CI.**
