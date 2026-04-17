# macOS black-box GUI smoke (`tests/gui_macos`)

Optional **out-of-process** automation for the real Fyne binary (not Playwright/Selenium). Sources are tagged `//go:build darwin` plus a `!darwin` skip stub so **`go test ./...` on Linux never compiles Robotgo**.

## When tests run

- **Default:** `go test ./tests/gui_macos/...` **skips** the smoke test unless **`PHOTO_TOOL_GUI_E2E_MACOS=1`**.
- **Entrypoint:** [`scripts/run-gui-e2e-macos.sh`](../../scripts/run-gui-e2e-macos.sh) sets that env, **`CGO_ENABLED=1`**, and (when **`BUNDLE`** is set) tees output to **`$BUNDLE/logs/gui-e2e-macos.txt`** and enables failure PNGs under **`logs/gui-e2e-failure-*.png`**.

## Requirements (macOS)

1. **CGO + Xcode Command Line Tools** — Robotgo uses CGO (`go test` must not force `CGO_ENABLED=0`).
2. **Accessibility** — System Events drives window geometry, frontmost, optional AX button scan, and global clicks. Under *System Settings → Privacy & Security → Accessibility*, enable the **exact process that runs `go test`** (often **Cursor**, **Terminal**, or **iTerm**). If you use the integrated terminal and clicks still do nothing, also enable **Cursor Helper (Renderer)** or related **Cursor Helper** entries when macOS lists them.
3. **Screen Recording** — The smoke test uses **`screencapture -R …`** on the **whole window** and compares an FNV hash of the PNG **file bytes** before/after navigation. Allow **Screen Recording** for the same host as above (again, **Cursor Helper** may appear separately from **Cursor** — enable both if unsure).
4. **Graphical session** — Run on a logged-in desktop (not SSH-only without a console session).
5. **Isolated library** — The test sets **`PHOTO_TOOL_LIBRARY`** to a temp directory (via [`internal/config`](../../internal/config/library.go)); the spawned binary does not touch your default library.

## Flakiness and tuning

- **Retina / scaling:** If taps miss, set **`PHOTO_TOOL_GUI_E2E_REVIEW_OFFSET_X`** (default `84`) and **`PHOTO_TOOL_GUI_E2E_REVIEW_OFFSET_Y`** (default `0`; vertical positions are swept automatically).
- **Window chrome:** Assumptions match the default **800×560** window from [`main.go`](../../main.go). Nav taps sweep the left rail; **`PHOTO_TOOL_GUI_E2E_REVIEW_OFFSET_X`** nudges the horizontal aim (default `84`).
- **Timing:** Slow machines may need a longer `-timeout` (the script uses **15m**).

## Stack (attempt order)

1. **AppleScript AX** — Clicks a **Review** control when the accessibility tree exposes it (often missing for Fyne/GLFW).
2. **[Robotgo](https://github.com/go-vgo/robotgo)** — `ActivePid`, optional `GetBounds`, and **`Move` + `Click`** with `Scale=true` on Retina; this is usually what makes the test pass when AX is sparse.
3. **Process-scoped `click at`** — `tell process … click at` after `frontmost`.
4. **Bare global `click at`** — Legacy System Events form.
5. **`screencapture`** — Full-window PNG; pass when the raw file hash changes after a tap (or when AX strings for Review appear).

## UX judge loop

Set **`UX_LOOP_GUI_E2E_MACOS=1`** on Darwin when running [`scripts/ux-judge-loop.sh`](../../scripts/ux-judge-loop.sh). On GUI failure, **`UX_LOOP_GUI_E2E_FIX=1`** (default) invokes the implementer contract [`ux-fix-from-gui-e2e-macos-prompt-v1.md`](../../_bmad-output/test-artifacts/ux-fix-from-gui-e2e-macos-prompt-v1.md). This step is **not** part of Linux [`assemble-judge-bundle.sh`](../../scripts/assemble-judge-bundle.sh).

## See also

- Headless Fyne / CLI E2E: [`tests/e2e/README.md`](../e2e/README.md).
