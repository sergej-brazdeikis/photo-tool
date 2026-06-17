# Environment variables

Consolidated reference for `PHOTO_TOOL_*` and related variables. See source files for edge-case behavior.

## Runtime

| Variable | Default | Purpose |
|----------|---------|---------|
| `PHOTO_TOOL_LIBRARY` | `{UserConfigDir}/photo-tool/library` | Absolute library root. ASCII whitespace trimmed; Unicode spaces (e.g. NBSP) are not. |
| `PHOTO_TOOL_SHARE_HTTP_HOST` | `127.0.0.1` | Share HTTP listen host. Use `::1` for IPv6 loopback. |
| `PHOTO_TOOL_SHARE_HTTP_PORT` | `8765` | Preferred TCP port; up to 10 successors tried on `EADDRINUSE`. |
| `PHOTO_TOOL_SHARE_HTTP_BIND_ALL` | unset | Set to `1`/`true`/`yes` to bind `0.0.0.0` (LAN exposure — security warning in code). |

Source: [`internal/config/library.go`](../internal/config/library.go), [`internal/config/share_http.go`](../internal/config/share_http.go).

## Tests — general

| Variable | When to set |
|----------|-------------|
| `PHOTO_TOOL_E2E_BIN` | Point CLI E2E at a prebuilt binary instead of building in test temp dir. |
| `PHOTO_TOOL_TEST_THEME` | `light`/`white` or `dark` — override theme in Fyne tests and UX capture. |
| `FYNE_SCALE` | Fyne logical scaling (e.g. `1.25`, `1.5`) for HiDPI layout tests and CI matrix. |

## Tests — UX journey capture

Used by `TestUXJourneyCapture` and judge bundles (`scripts/assemble-judge-bundle.sh`).

| Variable | When to set |
|----------|-------------|
| `PHOTO_TOOL_UX_JOURNEY_TEST` | Must be `1` to enable journey capture hooks. |
| `PHOTO_TOOL_UX_CAPTURE_DIR` | Writable directory for judge-bundle PNG output. |
| `PHOTO_TOOL_UX_UPLOAD_SEED_PATHS` | Newline-separated absolute paths to seed Upload panel (phase 2 of journey test). |

## Tests — NFR-07 display scaling

| Variable | When to set |
|----------|-------------|
| `PHOTO_TOOL_NFR07_MACOS_CI_TIER` | `125` or `150` — macOS CI surrogate when System Settings scaling cannot be driven. Must pair with matching `FYNE_SCALE` (`1.25` / `1.5`). |

## Tests — macOS GUI E2E (optional, local only)

| Variable | When to set |
|----------|-------------|
| `PHOTO_TOOL_GUI_E2E_MACOS` | Must be `1` to run `tests/gui_macos/` (display, Accessibility, CGO). |
| `PHOTO_TOOL_GUI_E2E_BUNDLE` | Directory for failure screenshots. |
| `PHOTO_TOOL_GUI_E2E_REVIEW_OFFSET_X` | Click offset for Review nav (default `84`). |
| `PHOTO_TOOL_GUI_E2E_REVIEW_OFFSET_Y` | Vertical click offset (default `0`). |

See [`tests/gui_macos/README.md`](../tests/gui_macos/README.md).

## Scripts — UX judge loop (local only, not CI)

| Variable | Default | Purpose |
|----------|---------|---------|
| `UX_AGENT` / `CURSOR_AGENT` | `agent` | Cursor agent CLI binary name. |
| `UX_LOOP_MAX` | `5` | Max judge/fix rounds. |
| `UX_LOOP_NO_MAX` | unset | If `1`, ignore `UX_LOOP_MAX`. |
| `UX_LOOP_UNTIL_LOCAL` | unset | Stop at local time `HH:MM` (24h). |
| `UX_LOOP_POST_FIX_REVIEW` | `1` | Run post-fix code review agent. |
| `UX_LOOP_SELF_IMPROVE` | `1` | Run self-improve agent. |
| `UX_LOOP_POST_IMPL_TEST` | `0` | If `1`, run full test suite after implementer. |
| `UX_LOOP_QA_AGENT` | `0` | If `1`, run QA agent after post-fix review. |
| `UX_LOOP_QA_FIX` | `1` | If `0`, skip QA implementer on `QA_LOOP_RESULT=fail`. |
| `UX_LOOP_GUI_E2E_MACOS` | `0` | If `1` on Darwin, run GUI E2E after QA block. |
| `UX_LOOP_GUI_E2E_FIX` | `1` | If `0`, skip GUI implementer on failure. |
| `UX_BUNDLE_ALLOW_E2E_FAIL` | unset | If `1`, assemble judge bundle even when E2E tests fail. |
| `UX_BUNDLE_EMIT_PATH_ONLY` | unset | If `1`, `assemble-judge-bundle.sh` prints bundle path only. |

See [`scripts/ux-judge-loop.sh`](../scripts/ux-judge-loop.sh) and [`tests/e2e/README.md`](../tests/e2e/README.md).

## Tests — extended matrix loop (Linux, local only)

Hands-free extended testing: functional + real-app UX capture + step/flow judges + parallel fix agents. **Not for CI.**

| Variable | When to set |
|----------|-------------|
| `PHOTO_TOOL_GUI_E2E_LINUX` | Must be `1` for real-binary journey in `main.go` and `tests/gui_linux/`. |
| `PHOTO_TOOL_UX_CAPTURE_APP_MODE` | `real_binary` for authoritative `step_ux` / `flow_ux` PNGs (Tier B). |
| `PHOTO_TOOL_UX_CAPTURE_FLOWS` | Comma-separated flow filter (`upload,review`, `scale_spot`, `edge`, `layout`, …). |
| `PHOTO_TOOL_UX_FIXTURE_SCALE` | Scale tier preset `S0`–`S8` for library seed (`S1` default, `S4`/`S5` for scale UX). |
| `PHOTO_TOOL_SCALE_TIER` | Tier for `extended-test-run.sh --layer=scale_ux` (default `S4`). |
| `PHOTO_TOOL_UX_CAPTURE_SOFTWARE_SUBDIR` | Set to `ui-software` when running Tier A software-driver capture via extended runner. |
| `EXTENDED_LOOP_MAX` | Max rounds for `extended-test-loop.sh` (default `20`). |
| `EXTENDED_PARALLEL_FIX` | Concurrent fix agents (default `4`). |
| `EXTENDED_USE_XVFB` | Set by setup when `DISPLAY` unset and `xvfb-run` is available. |

Scripts: [`scripts/extended-test-setup.sh`](../scripts/extended-test-setup.sh), [`scripts/extended-test-run.sh`](../scripts/extended-test-run.sh), [`scripts/extended-test-loop.sh`](../scripts/extended-test-loop.sh). Matrix package: [`tests/extended/`](../tests/extended/). Scale fixture CLI: `go run ./tests/extended/cmd/seed-library -out LIB -tier S4`.

### Scale tiers (volume / UX limits)

| Tier | Assets | Use |
|------|--------|-----|
| S0 | 0 | Empty-library UX |
| S1 | 3 | Default UX journey |
| S4 | 96 | Grid paging (2×48) |
| S5 | 500 | Package cap boundary; tag bulk |
| S5R | 500 rejected | Rejected grid at volume |
| S6 | 501 | Over package cap |
| S8 | 10k files | NFR-02 CLI scan tree (filesystem only) |

Make targets: `make scale-test` (unit, `-short`), `make extended-test-scale` (functional only), `make extended-test-scale-ux` (real-binary capture + `scale-report.html`).
