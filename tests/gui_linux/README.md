# Linux black-box GUI tests (`tests/gui_linux`)

Spawns the **real** `photo-tool` binary (native Fyne/GLX), not the software test driver.

## When tests run

- **`PHOTO_TOOL_GUI_E2E_LINUX=1`** — required for smoke and journey tests
- **`DISPLAY`** or **`WAYLAND_DISPLAY`** — real framebuffer session (use `xvfb-run` headless)

## Commands

```bash
# Headless journey capture smoke
xvfb-run -a env PHOTO_TOOL_GUI_E2E_LINUX=1 \
  go test ./tests/gui_linux/... -run TestLinuxGUIE2E_journeyRealAppCapture -count=1 -timeout=5m
```

Extended matrix uses [`scripts/extended-test-run.sh`](../../scripts/extended-test-run.sh) to build `bin/photo-tool` and write `ui-real/steps.json`.

See also: [`tests/gui_macos/README.md`](../gui_macos/README.md), [`docs/env.md`](../../docs/env.md).
