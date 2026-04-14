# NFR-01 layout matrix — evidence (tier-1 desktop)

**Purpose:** Manual QA record for **NFR-01** (1024×768–5120×1440 window band; square / 16:9 / 21:9 families).  
**Story:** 2.11 — Layout and display-scaling validation gate.  
**PRD refs:** NFR-01; UX responsive / ultrawide chrome.

**Epic 2 recorded gate (automation):** Matrix rows below document **pass** results from `go test ./...` on **tier-1 CI** (**macOS** and **Windows** runners in [`.github/workflows/go.yml`](../../.github/workflows/go.yml)) with tests `TestNFR01LayoutGate_matrixCells`, `TestNFR01LayoutGate_resizeSweep_AC2`, and `TestNFR01LayoutGate_NFR07FYNEProxy`. **Git SHA** column is the commit for which tests were executed and this bundle was updated; refresh it after layout-affecting changes. **Human** runs still add **multi-monitor**, **GPU/driver** quirks, and **real library assets** in the shipping modal loupe. **NFR-07 (Story 2.11 AC3)** is documented in [`nfr-07-os-scaling-checklist.md`](./nfr-07-os-scaling-checklist.md): **`TestNFR01LayoutGate_NFR07_AC3`** on **Windows** LogPixels matrix jobs (**GetDpiForSystem**, FYNE_SCALE unset) and on **two** **macOS** GitHub Actions jobs (**`PHOTO_TOOL_NFR07_MACOS_CI_TIER`** + matching **`FYNE_SCALE`** surrogate; see workflow). **Hardware macOS** may still re-run with the CoreGraphics probe for driver / display quirks. **`TestNFR01LayoutGate_NFR07FYNEProxy`** remains an extra **FYNE_SCALE** structural supplement across platforms.

## How to use this doc

1. Record **`git SHA`** (e.g. `git rev-parse HEAD`) and **tester** for each run session.  
2. Use **matrix cell IDs** (e.g. `S-min`) consistently so **NFR-07** checklist can reference the same rows.  
3. **Pass** = all bullets in the [Primary navigation checklist](#primary-navigation-checklist) and [Loupe checklist](#loupe-single-photo--large-view-checklist) hold for that cell **without** scrolling the **shell** to reach them. **Inner scroll:** thumbnail **grid body** may scroll vertically; the **filter strip** must **not** require horizontal scroll to reach Collection / rating / tags controls at tested widths — if it does, record **fail** (or **pass with notes** only if UX explicitly documents strip overflow behavior).  
4. **Fail** → open a tracked defect; paste issue URL in **Notes** and list it in [Defect index](#defect-index). Each defect row should include **short repro** (cell ID, theme, route, asset class portrait/landscape).

### Tier-2 Linux (AC5)

**Stance at Epic 2 gate:** _Defer tier-2 Linux for this milestone — manual matrix is time-boxed to tier-1 (macOS, Windows); Linux subset scheduled post–tier-1 sign-off._

_(Edit the line above if you instead run a documented subset on Linux.)_

---

## Default representative sizes (within NFR-01 band)

Unless substituting with documented justification, use these **logical window sizes** (Fyne/client area as reported by the OS at **100%** scaling for the NFR-01-only runs; NFR-07 re-runs use scaling noted in `nfr-07-os-scaling-checklist.md`).

| Aspect family | Min (W×H) | Mid (W×H) | Max (W×H) | Notes |
|---------------|-----------|-----------|-----------|--------|
| **Square** | 1024×1024 | 1280×1280 | 1440×1440 | Band caps **height at 1440** — largest square is **1440×1440** (not 2048×2048). |
| **16:9** | 1366×768 | 1920×1080 | 2560×1440 | **1280×720** is common but **below min height 768**; use **1366×768** (or larger) for the true low end. **5120×1440** is ~32:9 ultrawide, not 16:9. |
| **21:9** | 1792×768 | 2560×1080 | 5120×1440 | Min row keeps **H=768** and ~21:9; max uses PRD **extent** 5120×1440. |

**Continuous resize (AC2):** Use the [Resize sweep protocol](#continuous-resize-sweep-protocol-ac2) (not ad-hoc dragging only). The same corner order is exported for structural tests as `domain.NFR01AC2ResizeSweepPath()` — keep this doc and that function in sync.

**Code cross-check:** PRD band bounds are mirrored in Go as `internal/domain/nfr_layout.go` (`NFR01Window*`); update that file if the PRD band ever changes.

---

## Run environment fingerprint (minimum)

Record once per session (or per OS scaling change); copy into matrix **Notes** or a session preamble.

| Field | Example | Why |
|-------|---------|-----|
| OS + build | macOS 15.x; Windows 11 23H2 | Driver/toolkit differences |
| Displays | 1× built-in; 2× mixed DPI | Multi-monitor moves chrome |
| OS scaling % | 100% (NFR-01 base) / 125% / 150% (NFR-07) | Logical vs physical pixels |
| Fyne / app | `go.mod` fyne version; `FYNE_SCALE` if set | Toolkit scaling layer |
| Renderer (if known) | e.g. `metal`, `opengl`, `software` | Headless CI vs local |

---

## Continuous resize sweep protocol (AC2)

1. Start from a **passing** fixed cell (e.g. **169-mid**, dark, Review).  
2. Resize along a path that visits **band corners**: toward **S-min** area → toward **219-max** → back through **169-min** → end at **219-mid** (order flexible; hit **both** small-width and ultrawide extremes).  
3. After each major size change, **pause ≥1s** (idle) before judging — **transient** mis-layout during drag is OK if it **recovers** after idle.  
4. If any **primary nav**, **filter strip** (no horizontal scroll rule above), or **critical loupe chrome** stays unusable after idle, **fail** AC2 and file a defect referencing this protocol.

**CI note:** `TestNFR01LayoutGate_resizeSweep_AC2` (and the AC2 subtests inside `TestNFR01LayoutGate_NFR07FYNEProxy`) **sleep ≥1.1s** after each resize step so layout can settle, matching the human **idle dwell** rule below. If a failure only appears with slow drags, record it as an AC2 defect anyway.

---

## CI structural regression (`TestNFR01LayoutGate_*`) — not a manual substitute

The Go tests under `internal/app/nfr01_layout_gate_test.go` exercise **Fyne test driver** layout at matrix sizes. **Review** rows use **full** `newMainShell(..., false)` (Story **2.1** semantic preview strip — buttons **stacked vertically** so the NFR-01 **1024px** floor still fits the filter strip). **Loupe** rows use the **same** widget tree shape as `review_loupe.go` (rating row, tags row, albums strip, **`loupeImageLayout`** ~90% band) with a **decoded in-memory raster** and **`canvas.ImageFillContain`** (letterboxing path — not a solid placeholder).

| What CI covers | What still requires human matrix |
|----------------|----------------------------------|
| Primary nav + filter strip on-screen for **Review** at each **logical** window size × theme | **Real** assets from disk in the **modal** loupe (decode latency, corrupt files, portrait/landscape library shots) |
| **Shipping-pattern** loupe chrome + **ImageFillContain** geometry | **End-to-end** DB-backed tag/album interactions, keyboard shortcuts, focus |
| Resize path aligned with `domain.NFR01AC2ResizeSweepPath()` **plus ≥1.1s idle** after each step | Slow drags, **OS Settings** display scale without `FYNE_SCALE` (see NFR-07), multi-monitor, GPU/driver glitches |

If CI passes but manual loupe fails, **manual wins** — file a defect and treat CI as a **regression tripwire** only.

**Tier-1 OS coverage for recorded passes:** The **same** test binaries run on **macOS-latest** and **windows-latest** in CI; matrix **OS** column is **`macOS+Windows (CI)`** to reflect both runners. Local `go test` on a single workstation validates the same code path for that host only.

---

## Primary navigation checklist (Review bulk — shell)

Applies to **Upload**, **Review**, **Collections**, **Rejected** — order/labels per **UX-DR13**.

- [ ] All four nav targets visible and **activatable** without horizontal scroll of the **shell**.
- [ ] Current route/read state is visually clear at this size.
- [ ] **Filter strip** (Collection → min rating → tags): controls visible; keyboard focus can reach strip and operate (per UX-DR15).
- [ ] **Thumbnail grid** and bulk actions that ship in Epic 2: primary actions reachable; grid may scroll internally.

### Collections / Rejected routes (same shell)

When the active panel is **Collections** (list or detail) or **Rejected**, re-apply the **first two bullets** above (all nav targets + route state). Section-specific toolbars/lists may use inner scroll; **shell** must not hide nav.

## Loupe (single-photo / large view) checklist

Per **FR-09**, **FR-12**, **UX-DR4** (~90% image region, letterboxed, no unintended crop).

- [ ] **Primary loupe chrome** (rating, prev/next, close/back — as implemented in Story 2.4) **visible and usable** without shell-level scroll.
- [ ] **Full image** visible in the loupe **letterboxed** inside the ~90% region — **no unintended crop** at this window size.
- [ ] Safe chrome for aspect from **1:1 through 21:9** asset content (test at least one portrait and one landscape asset per theme if possible).

### Suggested loupe test assets (document what you used)

| Asset | Purpose |
|-------|---------|
| One **portrait** (tall) RAW/JPEG | Letterbox + vertical chrome |
| One **landscape** wide (≥3:2) | Horizontal letterbox |
| One **square** or near-square | 1:1 family |

---

## Results matrix

### Theme: Dark

| Cell ID | OS | Window (W×H) | Aspect family | Surface | Theme | Pass/Fail | Tester | Date | Git SHA | Notes |
|---------|----|--------------|---------------|---------|-------|-----------|--------|------|---------|-------|
| S-min | macOS+Windows (CI) | 1024×1024 | square | Review | dark | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Structural Review shell; `TestNFR01LayoutGate_matrixCells`; full `newMainShell(..., false)` (stacked semantic preview). |
| S-mid | macOS+Windows (CI) | 1280×1280 | square | Review | dark | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Structural Review shell; `TestNFR01LayoutGate_matrixCells`; full `newMainShell(..., false)` (stacked semantic preview). |
| S-max | macOS+Windows (CI) | 1440×1440 | square | Review | dark | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Structural Review shell; `TestNFR01LayoutGate_matrixCells`; full `newMainShell(..., false)` (stacked semantic preview). |
| S-min-L | macOS+Windows (CI) | 1024×1024 | square | Loupe | dark | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Shipping loupe chrome + `loupeImageLayout`; decoded raster + `ImageFillContain`; `TestNFR01LayoutGate_matrixCells`. |
| S-mid-L | macOS+Windows (CI) | 1280×1280 | square | Loupe | dark | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Shipping loupe chrome + `loupeImageLayout`; decoded raster + `ImageFillContain`; `TestNFR01LayoutGate_matrixCells`. |
| S-max-L | macOS+Windows (CI) | 1440×1440 | square | Loupe | dark | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Shipping loupe chrome + `loupeImageLayout`; decoded raster + `ImageFillContain`; `TestNFR01LayoutGate_matrixCells`. |
| 169-min | macOS+Windows (CI) | 1366×768 | 16:9 | Review | dark | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Structural Review shell; `TestNFR01LayoutGate_matrixCells`; full `newMainShell(..., false)` (stacked semantic preview). |
| 169-mid | macOS+Windows (CI) | 1920×1080 | 16:9 | Review | dark | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Structural Review shell; `TestNFR01LayoutGate_matrixCells`; full `newMainShell(..., false)` (stacked semantic preview). |
| 169-max | macOS+Windows (CI) | 2560×1440 | 16:9 | Review | dark | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Structural Review shell; `TestNFR01LayoutGate_matrixCells`; full `newMainShell(..., false)` (stacked semantic preview). |
| 169-min-L | macOS+Windows (CI) | 1366×768 | 16:9 | Loupe | dark | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Shipping loupe chrome + `loupeImageLayout`; decoded raster + `ImageFillContain`; `TestNFR01LayoutGate_matrixCells`. |
| 169-mid-L | macOS+Windows (CI) | 1920×1080 | 16:9 | Loupe | dark | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Shipping loupe chrome + `loupeImageLayout`; decoded raster + `ImageFillContain`; `TestNFR01LayoutGate_matrixCells`. |
| 169-max-L | macOS+Windows (CI) | 2560×1440 | 16:9 | Loupe | dark | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Shipping loupe chrome + `loupeImageLayout`; decoded raster + `ImageFillContain`; `TestNFR01LayoutGate_matrixCells`. |
| 219-min | macOS+Windows (CI) | 1792×768 | 21:9 | Review | dark | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Structural Review shell; `TestNFR01LayoutGate_matrixCells`; full `newMainShell(..., false)` (stacked semantic preview). |
| 219-mid | macOS+Windows (CI) | 2560×1080 | 21:9 | Review | dark | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Structural Review shell; `TestNFR01LayoutGate_matrixCells`; full `newMainShell(..., false)` (stacked semantic preview). |
| 219-max | macOS+Windows (CI) | 5120×1440 | 21:9 | Review | dark | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Structural Review shell; `TestNFR01LayoutGate_matrixCells`; full `newMainShell(..., false)` (stacked semantic preview). |
| 219-min-L | macOS+Windows (CI) | 1792×768 | 21:9 | Loupe | dark | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Shipping loupe chrome + `loupeImageLayout`; decoded raster + `ImageFillContain`; `TestNFR01LayoutGate_matrixCells`. |
| 219-mid-L | macOS+Windows (CI) | 2560×1080 | 21:9 | Loupe | dark | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Shipping loupe chrome + `loupeImageLayout`; decoded raster + `ImageFillContain`; `TestNFR01LayoutGate_matrixCells`. |
| 219-max-L | macOS+Windows (CI) | 5120×1440 | 21:9 | Loupe | dark | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Shipping loupe chrome + `loupeImageLayout`; decoded raster + `ImageFillContain`; `TestNFR01LayoutGate_matrixCells`. |

### Theme: Light

| Cell ID | OS | Window (W×H) | Aspect family | Surface | Theme | Pass/Fail | Tester | Date | Git SHA | Notes |
|---------|----|--------------|---------------|---------|-------|-----------|--------|------|---------|-------|
| S-min | macOS+Windows (CI) | 1024×1024 | square | Review | light | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Structural Review shell; `TestNFR01LayoutGate_matrixCells`; full `newMainShell(..., false)` (stacked semantic preview). |
| S-mid | macOS+Windows (CI) | 1280×1280 | square | Review | light | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Structural Review shell; `TestNFR01LayoutGate_matrixCells`; full `newMainShell(..., false)` (stacked semantic preview). |
| S-max | macOS+Windows (CI) | 1440×1440 | square | Review | light | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Structural Review shell; `TestNFR01LayoutGate_matrixCells`; full `newMainShell(..., false)` (stacked semantic preview). |
| S-min-L | macOS+Windows (CI) | 1024×1024 | square | Loupe | light | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Shipping loupe chrome + `loupeImageLayout`; decoded raster + `ImageFillContain`; `TestNFR01LayoutGate_matrixCells`. |
| S-mid-L | macOS+Windows (CI) | 1280×1280 | square | Loupe | light | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Shipping loupe chrome + `loupeImageLayout`; decoded raster + `ImageFillContain`; `TestNFR01LayoutGate_matrixCells`. |
| S-max-L | macOS+Windows (CI) | 1440×1440 | square | Loupe | light | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Shipping loupe chrome + `loupeImageLayout`; decoded raster + `ImageFillContain`; `TestNFR01LayoutGate_matrixCells`. |
| 169-min | macOS+Windows (CI) | 1366×768 | 16:9 | Review | light | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Structural Review shell; `TestNFR01LayoutGate_matrixCells`; full `newMainShell(..., false)` (stacked semantic preview). |
| 169-mid | macOS+Windows (CI) | 1920×1080 | 16:9 | Review | light | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Structural Review shell; `TestNFR01LayoutGate_matrixCells`; full `newMainShell(..., false)` (stacked semantic preview). |
| 169-max | macOS+Windows (CI) | 2560×1440 | 16:9 | Review | light | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Structural Review shell; `TestNFR01LayoutGate_matrixCells`; full `newMainShell(..., false)` (stacked semantic preview). |
| 169-min-L | macOS+Windows (CI) | 1366×768 | 16:9 | Loupe | light | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Shipping loupe chrome + `loupeImageLayout`; decoded raster + `ImageFillContain`; `TestNFR01LayoutGate_matrixCells`. |
| 169-mid-L | macOS+Windows (CI) | 1920×1080 | 16:9 | Loupe | light | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Shipping loupe chrome + `loupeImageLayout`; decoded raster + `ImageFillContain`; `TestNFR01LayoutGate_matrixCells`. |
| 169-max-L | macOS+Windows (CI) | 2560×1440 | 16:9 | Loupe | light | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Shipping loupe chrome + `loupeImageLayout`; decoded raster + `ImageFillContain`; `TestNFR01LayoutGate_matrixCells`. |
| 219-min | macOS+Windows (CI) | 1792×768 | 21:9 | Review | light | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Structural Review shell; `TestNFR01LayoutGate_matrixCells`; full `newMainShell(..., false)` (stacked semantic preview). |
| 219-mid | macOS+Windows (CI) | 2560×1080 | 21:9 | Review | light | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Structural Review shell; `TestNFR01LayoutGate_matrixCells`; full `newMainShell(..., false)` (stacked semantic preview). |
| 219-max | macOS+Windows (CI) | 5120×1440 | 21:9 | Review | light | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Structural Review shell; `TestNFR01LayoutGate_matrixCells`; full `newMainShell(..., false)` (stacked semantic preview). |
| 219-min-L | macOS+Windows (CI) | 1792×768 | 21:9 | Loupe | light | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Shipping loupe chrome + `loupeImageLayout`; decoded raster + `ImageFillContain`; `TestNFR01LayoutGate_matrixCells`. |
| 219-mid-L | macOS+Windows (CI) | 2560×1080 | 21:9 | Loupe | light | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Shipping loupe chrome + `loupeImageLayout`; decoded raster + `ImageFillContain`; `TestNFR01LayoutGate_matrixCells`. |
| 219-max-L | macOS+Windows (CI) | 5120×1440 | 21:9 | Loupe | light | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Shipping loupe chrome + `loupeImageLayout`; decoded raster + `ImageFillContain`; `TestNFR01LayoutGate_matrixCells`. |

**AC1 note:** Story AC1 requires **at least one** representative size per aspect family in **each** theme — that is a **minimum**. Full rows above are **execution-ready**; if time-boxed, fill the minimum cells first (e.g. S-mid, 169-mid, 219-mid × Review + Loupe), then expand; list any skipped rows under **Scope / subset justification**. Optional extra ultrawide point **3440×1440** may be added as **219-mid2** if hardware supports it.

### Continuous resize (AC2)

| OS | Theme | Pass/Fail | Tester | Date | Git SHA | Notes (off-screen elements after idle resize?) |
|----|-------|-----------|--------|------|---------|-----------------------------------------------|
| macOS (CI) | dark | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | `TestNFR01LayoutGate_resizeSweep_AC2`; **≥1.1s idle** after each resize; full `newMainShell`; Fyne test driver. |
| macOS (CI) | light | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Same as dark row; theme variant subtest. |
| Windows (CI) | dark | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Same structural path on `windows-latest`. |
| Windows (CI) | light | Pass | go test / GitHub Actions | 2026-04-13 | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | Same structural path on `windows-latest`. |

---

## Defect index

| Cell ID / area | Issue URL | UX/layout owner | Open/closed | Release blocker? (Y/N) |
|----------------|-----------|-----------------|-------------|-------------------------|
|  |  |  |  |  |

---

## Scope / subset justification (if not full matrix)

_Time-box or milestone subset explanation (required when any default cell is skipped):_

-
