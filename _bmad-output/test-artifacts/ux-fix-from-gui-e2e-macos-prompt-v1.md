# UX loop — fix from macOS GUI E2E (implementer, local / Cursor `agent` only)

**Prerequisites:** A judge bundle `BUNDLE` where `scripts/run-gui-e2e-macos.sh` exited non-zero for this iteration, so `BUNDLE/logs/gui-e2e-macos.txt` exists (and optionally `BUNDLE/logs/gui-e2e-failure-*.png` from the test harness).

## Role

You are the **implementer** for a **failing black-box macOS GUI smoke test** (`tests/gui_macos` under `//go:build darwin`). This is **orthogonal** to the vision judge and the QA text agent: focus on **repro from logs**, **coordinate / timing / permission** brittleness, and **minimal** product or test changes so the scripted journey passes again.

## Inputs

- `BUNDLE/logs/gui-e2e-macos.txt` — full `go test` output for `./tests/gui_macos/...`
- `BUNDLE/logs/gui-e2e-failure-*.png` — optional full-screen captures written on failure (when `PHOTO_TOOL_GUI_E2E_BUNDLE` was set)
- `tests/gui_macos/README.md` — env vars (`PHOTO_TOOL_GUI_E2E_REVIEW_OFFSET_X/Y`, `FYNE_SCALE`, etc.)
- Primary code paths: [`main.go`](../../main.go) (window title/size), [`internal/app/shell.go`](../../internal/app/shell.go) (primary nav layout)

## Requirements

1. Reproduce locally on macOS with `PHOTO_TOOL_GUI_E2E_MACOS=1` and `CGO_ENABLED=1` (see README). Prefer **adjusting offsets / waits** in the smoke test or documenting **Accessibility** steps before changing shipping UI.
2. If the product must change, keep edits **minimal** and aligned with existing UX specs (nav order, window defaults).
3. From repo root (on macOS for the GUI test itself; Linux CI still runs `go test ./...` without compiling darwin-only sources):

   ```bash
   CGO_ENABLED=1 PHOTO_TOOL_GUI_E2E_MACOS=1 go test ./tests/gui_macos/... -count=1
   go test ./...
   go test ./tests/e2e/... -count=1
   go build .
   ```

4. **Do not** re-run the vision judge, QA agent, or GUI harness in this task; [`scripts/ux-judge-loop.sh`](../../scripts/ux-judge-loop.sh) will run the next iteration.

## Output

- Code and/or tests updated; failure screenshots should become unnecessary after the fix.
