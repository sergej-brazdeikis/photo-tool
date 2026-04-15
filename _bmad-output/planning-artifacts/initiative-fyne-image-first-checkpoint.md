# Checkpoint — Fyne image-first initiative (stories **1.5**, **2.1**, **2.2**, **2.3**, **2.11**)

**Mode:** headless YOLO / BMAD checkpoint-preview (no blocking questions — use **Continue** below for the next pass).  
**Parent:** [initiative-fyne-image-first-bmad.md](initiative-fyne-image-first-bmad.md) · **TEA:** [initiative-fyne-image-first-tea-notes.md](initiative-fyne-image-first-tea-notes.md)

---

## 1. Purpose (why this checkpoint)

Validate that the **image-first** slice (upload confirm UX, compact shell, filter strip, thumbnail grid, NFR layout evidence) hangs together before calling initiative stories “done.” Code + CI moved; **human smoke** still owns qualitative layout, OS scaling, and continuous-resize feel.

---

## 2. What changed in the repo (executive summary)

| Area | Change |
|------|--------|
| **Upload (1.5)** | Bounded **batch preview** strip (cap + “+ N more” copy), **collapsible receipt** (`Accordion`), import **close intercept** while ingest or collection step pending, **async ingest** with `fyne.Do` in production vs **`SynchronousIngest`** / other `UploadViewOptions` for headless tests. |
| **Shell (2.1)** | Production shell omits non-shipping **semantic style preview** rail so NFR-01 width budget matches shipped UX; `newMainShell(..., omitSemanticStylePreview, ...)` and nav transition ordering when switching panels. |
| **Review filter (2.2)** | Exported **filter segment label order** for regression tests; strip behavior covered in `review_test.go` / domain filter tests. |
| **Review grid (2.3)** | **Zero-row list** when there are no matches (empty state not undermined), user-safe grid error strings, paging/cell/bind logic + tests; **Rejected** grid path aligned (hide scroll when appropriate, shared filter vocabulary). |
| **NFR layout / scaling (2.11)** | `internal/domain/nfr_layout.go` holds **matrix cell IDs**, **NFR-07 default subset**, **AC2 resize sweep path**; `nfr01_layout_gate_test.go` expanded; evidence/checklist markdown updated. |
| **CI / tooling** | `.github/workflows/go.yml`: **NFR-07** matrix (macOS `FYNE_SCALE` + tier env, Windows `LogPixels`), `bash -n` on helper scripts, optional **UX journey capture** job (`TestUXJourneyCapture`) + artifact upload on one macOS leg. `Makefile` / `scripts/assemble-judge-bundle.sh` support local judge bundles / `ux-judge-loop`. |
| **Specs / sprint** | Story markdown and `sprint-status.yaml` refreshed; **1.5** and **2.11** remain **`review`** at last YAML snapshot — confirm before marking done. |

**Uncommitted / untracked (workspace):** git status may show additional UX capture hooks/tests (for example `internal/app/ux_capture_grid_hook.go`, `ux_journey_capture_test.go`, `ux_layout_invariants_test.go`) not yet in the tracked diff — reconcile before merge.

---

## 3. Where to look (files — read these first)

Use these as the review spine (paths are repo-relative for IDE links).

**Upload flow & FR-06**

- `internal/app/upload.go:26` — preview strip constants (`uploadPreviewStripMaxItems`, `uploadPreviewThumbMin`).
- `internal/app/upload.go:35` — `uploadImportCloseBlocked` policy for `SetCloseIntercept`.
- `internal/app/upload.go:49` — `UploadViewOptions` (`SynchronousIngest`, `SkipCompletionDialogs`, etc.).
- `internal/app/upload.go:120` — receipt accordion + batch preview block wiring.
- `internal/app/upload_fr06_flow_test.go` — headless flow / preview strip / UX-DR17 test posture.

**Shell & themes**

- `internal/app/shell.go:13` — primary nav order (`primaryNavItems`).
- `internal/app/shell.go:53` — `newMainShell` + `omitSemanticStylePreview` rationale.
- `internal/app/shell_test.go`, `internal/app/theme_test.go` — regression coverage.

**Review: filters, grid, loupe chrome**

- `internal/app/review.go:25` — filter sentinels; `review.go:35` — `ReviewFilterStripSegmentLabels`.
- `internal/app/review_grid.go:26` — paging/columns; `review_grid.go:35` — `reviewGridListRowCount` empty behavior.
- `internal/app/review_grid_test.go`, `internal/app/review_test.go` — strip + grid invariants.
- `internal/app/review_loupe.go` — small ordering/chrome tweaks if loupe shares layout budget with grid.

**Rejected (grid parity with Review)**

- `internal/app/rejected.go:22` — filter suffix alignment with Review; bulk/grid behavior.

**NFR-01 / 2.11**

- `internal/domain/nfr_layout.go:5` — window band constants.
- `internal/domain/nfr_layout.go:21` — `NFR01Epic2MatrixCells` (cell IDs).
- `internal/domain/nfr_layout.go:53` — `NFR01AC2ResizeSweepPath`.
- `internal/app/nfr01_layout_gate_test.go` — structural layout gate vs matrix.

**CI**

- `.github/workflows/go.yml:10` — OS/matrix and NFR-07 env.
- `.github/workflows/go.yml:51` — UX journey capture smoke (macOS, one leg).

**Evidence (manual matrix source of truth)**

- `_bmad-output/implementation-artifacts/nfr-01-layout-matrix-evidence.md`
- `_bmad-output/implementation-artifacts/nfr-07-os-scaling-checklist.md`

**Sprint truth**

- `_bmad-output/implementation-artifacts/sprint-status.yaml:154` — `1-5-upload-confirm-receipt: review`
- `_bmad-output/implementation-artifacts/sprint-status.yaml:160` — `2-1-app-shell-navigation-themes: done`
- `_bmad-output/implementation-artifacts/sprint-status.yaml:161` — `2-2-filter-strip: done`
- `_bmad-output/implementation-artifacts/sprint-status.yaml:162` — `2-3-thumbnail-grid-rating-badges: done`
- `_bmad-output/implementation-artifacts/sprint-status.yaml:170` — `2-11-layout-display-scaling-gate: review`

---

## 4. Manual smoke — Upload

Run the real app (`make run` or your usual binary) on at least one **tier-1** window size from the NFR matrix (for example **S-min** 1024×1024 or **169-min** 1366×768).

1. **Upload** → add **several** images (mixed count: fewer than preview cap and **more than** `uploadPreviewStripMaxItems` — see `internal/app/upload.go:28`).
2. Confirm **batch preview** appears with large thumbs; horizontal scroll if needed; verify **overflow copy** (“+ N more…”) matches expectation.
3. **Import** and confirm **receipt** counts look correct; **collapse/expand** the receipt accordion; ensure secondary controls do not drown the preview.
4. Complete or cancel the **collection** step; verify you **cannot** close the window during import or pending collection step (dialog path from close intercept).
5. Optional stress: **large** JPEGs — watch for UI hitching (known tradeoff: previews load on UI goroutine per comments in `upload.go`).

---

## 5. Manual smoke — Review

Same session, same library (or seed data):

1. **Review** → confirm **filter strip** is a single horizontal band: **Collection**, **Minimum rating**, **Tags** order (`internal/app/review.go:35`); at **minimum width**, confirm overflow / “more filters” behavior still matches spec (see `review_test.go` for expected semantics).
2. Change filters and confirm **live count** updates; with **zero** matches, confirm **empty state** is visible (grid should not show misleading blank chrome — `reviewGridListRowCount` in `internal/app/review_grid.go:38`).
3. With matches, scroll the grid; spot-check **rating / hidden badges** and that thumbnails match assets (stale-thumb risk called out in TEA notes).
4. Open **loupe** from a cell; sanity-check **Share…** / chrome width against NFR gate assumptions (`nfr01_layout_gate_test.go` / loupe checklist in evidence doc).
5. **Rejected** tab: confirm filter strip feels consistent with Review; empty vs populated scroll visibility.

---

## 6. Open risks (honest gaps)

Aligned with TEA **R-TEA-01–06** in [initiative-fyne-image-first-tea-notes.md](initiative-fyne-image-first-tea-notes.md):

| ID | Risk |
|----|------|
| **R-TEA-01** | CI **NFR-01 layout gate** is a **regression tripwire**, not proof of tier-1 **manual** matrix pass — false confidence if evidence rows are stale or empty. |
| **R-TEA-02** | **UX-DR17:** any new async/bind path can regress threading; run **`go test -race`** on `./internal/app` (and `./internal/ingest` if touched) before merge when goroutines or callbacks change. |
| **R-TEA-03** | **Vertical budget** at min logical width: shell + filter + bulk row + loupe still compete — stress **S-min** / **169-min** manually. |
| **R-TEA-04** | **Grid recycle / stale async** after scroll — watch wrong thumbnail or state; rely on tests + manual spot checks after grid changes. |
| **R-TEA-05** | **Close intercept** behavior may differ from headless `Window.Close()` — policy tests exist; still manually try closing during collection once. |
| **R-TEA-06** | **Story 2.11** evidence: numeric readability thresholds and **stable cell IDs** must stay aligned with `internal/domain/nfr_layout.go` and markdown tables. |
| **CI vs local** | **UX journey capture** and judge loop are **partially** automated in CI (one macOS leg) vs full **local** `make ux-judge-loop` — do not equate green CI with signed-off visual quality. |

**Sprint:** **`1-5`** and **`2-11`** in **`review`** — close explicitly after smoke + evidence update; do not assume “done” from peer stories alone.

---

## 7. Quick automation reminders (before sign-off)

- `go test ./...` (matches default CI).
- `make test-ci` if preferences / `ci` tag paths change.
- After threading edits: `go test -race ./internal/app ./internal/ingest -count=1` (or `GO_TEST_RACE=1` via `scripts/bmad-story-workflow.sh --phase=gate`).

---

## 8. Continue (next checkpoint — no chat required)

**Continue — pass A (close1.5 + 2.11):** Re-read `_bmad-output/implementation-artifacts/1-5-upload-confirm-receipt.md` and `2-11-layout-display-scaling-gate.md` against the manual smokes above; update `nfr-01-layout-matrix-evidence.md` / `nfr-07-os-scaling-checklist.md` for any new findings; then flip sprint keys to **`done`** only with evidence filled.

**Continue — pass B (cross-story regression):** Run **Collections** and **Rejected** nav transitions from Review (shell prelude in `internal/app/shell.go:73`); confirm no stale undo state or wrong panel after rapid nav switching.

**Continue — pass C (merge hygiene):** Ensure untracked UX capture files are either committed with CI/docs pointers or dropped; verify `.github/workflows/go.yml` matrix still matches `NFR07Epic2DefaultSubsetCellIDs` in `internal/domain/nfr_layout.go:45`.

---

_Last written: 2026-04-15 — checkpoint-preview headless YOLO._
