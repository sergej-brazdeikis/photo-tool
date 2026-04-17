# End-to-end tests (`tests/e2e`)

## CLI black-box

[`cli_test.go`](cli_test.go) builds the real `photo-tool` binary (unless `PHOTO_TOOL_E2E_BIN` points at a prebuilt one), sets `PHOTO_TOOL_LIBRARY` to a temp library, and runs `phototool scan` / `phototool import` with assertions on exit codes, stdout, and database state.

Run from the module root:

```bash
go test ./tests/e2e/... -count=1
```

## Desktop GUI (Fyne)

Multi-step shell journeys and widget-level UI tests live in the main module under [`internal/app`](../../internal/app) (for example `e2e_shell_journeys_test.go`, `review_test.go`, `nfr01_layout_gate_test.go`). They use `fyne.io/fyne/v2/test`, not a browser driver. Shared **minimum thumbnail / loupe / upload preview** sizes (image-forward UX bar) live in [`internal/app/ux_image_dims.go`](../../internal/app/ux_image_dims.go); align changes with [`ux-design-specification.md`](../../_bmad-output/planning-artifacts/ux-design-specification.md) and run NFR-01 + journey capture after tuning.

For **macOS-only** black-box automation of the real GUI binary (Robotgo + `screencapture`, optional in the UX judge loop), see [`tests/gui_macos/README.md`](../gui_macos/README.md).

**Theme:** [`TestUXJourneyCapture`](../../internal/app/ux_journey_capture_test.go) defaults to **light** for judge-bundle PNGs. Use `PHOTO_TOOL_TEST_THEME=dark` when you want dark captures. Other Fyne tests that call [`applyTestPhotoToolTheme`](../../internal/app/test_theme_env_test.go) with `theme.VariantDark` switch to light when `PHOTO_TOOL_TEST_THEME` is `light` or `white`. Matrix tests that sweep both variants (e.g. NFR-01) are unchanged.

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
- **Local:** [`scripts/assemble-judge-bundle.sh`](../../scripts/assemble-judge-bundle.sh) produces `_bmad-output/test-artifacts/judge-bundles/<stamp>/` with `logs/go-test.txt`, `logs/go-test-ci.txt`, **`logs/go-test-e2e.txt`** (`go test ./tests/e2e/...` — this package), `logs/ux-capture.txt`, `context/rubric.md`, **`context/requirements-trace.md`**, `ui/*.png` + `ui/steps.json` + `ui/capture-files.txt` (PNG manifest for vision tools) + `ui/README.md`, and `manifest.json` (including `exit_codes.go_test_e2e`, `steps`, `capture_manifest` from `steps.json`, and **`requirements_trace`**). By default a **non-zero E2E exit code fails the assemble step** so the UX judge loop does not run on a red CLI surface; set **`UX_BUNDLE_ALLOW_E2E_FAIL=1`** when assembling to record the failure in `manifest.json` `omissions` only. If `go test` reports UX capture exit 0 but any `steps[].file` PNG is missing or tiny, the script **exits non-zero** after writing `manifest.json` so the judge loop does not run on an empty gallery. Capture uses `PHOTO_TOOL_UX_JOURNEY_TEST=1` and `PHOTO_TOOL_UX_CAPTURE_DIR=<bundle>/ui` (screenshots use **light** theme unless `PHOTO_TOOL_TEST_THEME=dark`). [`TestUXJourneyCapture`](../../internal/app/ux_journey_capture_test.go) walks **all primary shell flows** in order (Upload empty → Review grid/loupe/Share preview/filters → Collections list/new-album dialog/detail/groupings/back → Rejected + narrow filter → second shell pass: Upload staged + FR-06 + post-confirm). Each step has a `flow` field in `steps.json` for the judge.
- **CI (optional smoke):** one macOS matrix leg runs `TestUXJourneyCapture` with those env vars and uploads PNGs as a workflow artifact — still **no** LLM/`agent`.
- **Closed loop (local only):** [`scripts/ux-judge-loop.sh`](../../scripts/ux-judge-loop.sh) or `make ux-judge-loop` — re-bundles each round, runs `agent` as judge, then UX implementer, optional **`UX_LOOP_POST_IMPL_TEST=1`** (`go test ./... ./tests/e2e/...` after implementer; on failure skips post-fix / QA / self-improve for that iteration), then optional post-fix code review (`verdict/code-review-post-fix.md`), optional **QA agent** **`UX_LOOP_QA_AGENT=1`** ([`ux-qa-post-fix-loop.md`](../../_bmad-output/test-artifacts/ux-qa-post-fix-loop.md) → `verdict/qa-loop-output.md`), then optional **QA implementer** when the last line is **`QA_LOOP_RESULT=fail`** (default **`UX_LOOP_QA_FIX=1`**, [`ux-fix-from-qa-prompt-v1.md`](../../_bmad-output/test-artifacts/ux-fix-from-qa-prompt-v1.md)), then on **macOS only** optional **`UX_LOOP_GUI_E2E_MACOS=1`** ([`run-gui-e2e-macos.sh`](../../scripts/run-gui-e2e-macos.sh) → `logs/gui-e2e-macos.txt`) and optional **`UX_LOOP_GUI_E2E_FIX=1`** ([`ux-fix-from-gui-e2e-macos-prompt-v1.md`](../../_bmad-output/test-artifacts/ux-fix-from-gui-e2e-macos-prompt-v1.md)) after a failing GUI step, then optional self-improve — until the vision verdict’s last line is `UX_JUDGE_RESULT=pass` or `UX_LOOP_MAX` (default `5`). Each bundle includes **`context/requirements-trace.md`** (from [`bundle-requirements-trace.md`](../../_bmad-output/test-artifacts/bundle-requirements-trace.md)); when you add flows or tests, update that distillate so judge and QA stay aligned with epics. Set `UX_LOOP_POST_FIX_REVIEW=0` to skip the code-review step; set `UX_LOOP_QA_FIX=0` to skip the QA implementer even after `QA_LOOP_RESULT=fail`. Set `UX_AGENT` if your CLI binary is not named `agent`. To run **until a local wall-clock time** (for example 6pm), use `UX_LOOP_UNTIL_LOCAL=18:00` with `UX_LOOP_NO_MAX=1` (or combine with a high `UX_LOOP_MAX` as a safety cap). **Do not add this to CI.**
