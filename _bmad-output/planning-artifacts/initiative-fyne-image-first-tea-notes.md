# Initiative test strategy — Fyne image-first (TEA / Murat)

**Parent:** [initiative-fyne-image-first-bmad.md](initiative-fyne-image-first-bmad.md)  
**Scope:** Stories **1.5**, **2.1**, **2.2**, **2.3**, **2.11** (upload → shell → filter → grid → NFR evidence), plus cross-cutting **NFR-01**, **NFR-07**, **UX-DR17**.

**Principle (test levels):** Prefer fast, deterministic checks at the lowest level that still mirrors real usage. Reserve manual matrix work for invariants the Fyne test driver and CI surrogates cannot honestly assert (continuous resize feel, true OS scaling, qualitative thumb readability).

---

## 1. Risk register (initiative slice)

| ID | Risk | Likelihood × impact (qual.) | Mitigation |
|----|------|------------------------------|------------|
| R-TEA-01 | **NFR-01 false confidence** — `TestNFR01LayoutGate_*` green while tier-1 manual matrix fails (chrome, scaling, WM quirks). | High × High | Treat CI layout gate as **regression tripwire** only; keep [nfr-01-layout-matrix-evidence.md](../implementation-artifacts/nfr-01-layout-matrix-evidence.md) and [nfr-07-os-scaling-checklist.md](../implementation-artifacts/nfr-07-os-scaling-checklist.md) filled per Story **2.11** DoD. |
| R-TEA-02 | **UX-DR17 violations** — post-async UI mutation off main thread or racy shared state between worker and Fyne widgets. | Med × High | Worker + `fyne.Do` discipline in upload/grid; **`go test -race`** on touched packages before merge when goroutines or callbacks change. |
| R-TEA-03 | **Vertical budget collapse** — shell + filter strip + bulk row compete at **NFR-01** minimum logical width. | Med × High | Stress `internal/app` NFR-01 gate + manual cells **S-min** / **169-min**; watch Story **2.2** filter overflow behavior. |
| R-TEA-04 | **Grid list recycle / stale async** — wrong thumbnail or state after scroll when decode or ingest races row bind. | Med × High | `review_grid_test.go` invariants + race smoke on grid/upload paths; align with architecture §3.8.1 epoch/cancel notes. |
| R-TEA-05 | **Upload driver gap** — `SetCloseIntercept` not exercised the same way in headless `Window.Close()` vs real user. | Low × Med | Policy tests (e.g. `TestUpload_importCloseBlocked_policy`) + explicit manual row for close during collection step. |
| R-TEA-06 | **Evidence / traceability debt** — Story **2.11** numeric thumb thresholds or NFR-07 cell IDs missing. | Med × Med | AC5 readability notes in evidence doc; stable **cell IDs** across NFR-01 and NFR-07 rows. |

---

## 2. AC-level checks (automation posture)

Map each story’s AC to **H** (headless Fyne / `go test`), **U** (unit / domain), **M** (manual / visual / matrix), **E** (black-box `tests/e2e`). Use **Primary** where that level owns the acceptance proof.

### Story 1.5 — Upload confirm & receipt

| AC | Summary | H | U | M | E |
|----|---------|---|---|---|---|
| 1 | Receipt counts match `OperationSummary` / NFR-04 (`upload_fr06_flow_test`, deep label walks) | Primary | Secondary (`domain/summary`, CLI parity) | Stress very large batches | Optional `tests/e2e` import parity |
| 2 | No collection/links before confirm | Primary | Secondary (`internal/store` contracts) | — | — |
| 3–4 | Create collection + persisted name matches input (incl. rename) | Primary | — | — | — |
| 5 | Large previews, collapsible receipt, batch preview strip | Primary (`TestUpload_flow_confirmStep_showsBatchPreviewStrip`, overflow cap) | — | “Large” on 4K, pathological decode | — |
| 6 | **UX-DR17** — UI updates on main thread after async ingest | Primary (`SynchronousIngest` only in tests) | — | — | — |

### Story 2.1 — App shell & themes

- **H — Primary:** `shell_test.go`, `theme_test.go` (nav construction, theme tokens, focus vs warning/error, separator vs background).
- **H — Secondary:** `nfr01_layout_gate_test.go` uses full `newMainShell` for Review rows (stacked preview strip at min width).
- **M:** Keyboard / focus baseline at NFR-01 min logical sizes (Story **2.11** / UX-DR19 checklist — not structural tab-order CI).

### Story 2.2 — Filter strip

- **H — Primary:** `review_test.go` (strip layout, overflow / “more” behavior, label ordering); `TestReviewFilterStrip_defaultSentinels_matchStory22` for default-string regression.
- **U — Secondary:** `domain/review_filter_test.go` for SQL / filter semantics.
- **M:** Ultrawide and min-width clipping vs shell chrome.

### Story 2.3 — Thumbnail grid

- **H — Primary:** `review_grid_test.go` (row count, badges, sanitized errors, list invariants); broader paging/async behavior in `review_test.go` as implemented.
- **M:** Story **2.11** AC5 — recorded minimum readability at reference layouts (numeric notes in evidence doc).
- **UX-DR17:** Treat async bind/decode changes like Story 1.5 — add **`-race`** smoke when goroutines or cross-thread callbacks move.

### Story 2.11 — Layout / scaling gate

- **H — Secondary / tripwire:** `nfr01_layout_gate_test.go`, `nfr07_ac3_darwin_ci_surrogate_test.go`, CI matrix with `FYNE_SCALE` / Windows DPI; `internal/domain/nfr_layout.go` constants stay aligned with evidence tables.
- **M — Primary:** Tier-1 manual NFR-01 matrix, NFR-07 scaling checklist, continuous resize AC2, traceability AC6–7 (cell IDs).

---

## 3. `go test` packages to stress

Run full suite: `go test ./...` (CI default). For **focused** regression after touching initiative code:

| Package | Why stress it | Example focus |
|---------|----------------|---------------|
| `./internal/app` | Fyne: upload, shell, review, grid, loupe, NFR-01 gate, UX capture | `-run 'Test(NFR01|Upload_flow|Review|PhotoToolTheme|UXLayout|UXJourney)'` |
| `./internal/domain` | NFR-01 geometry, filter SQL fragments | `nfr_layout_test`, `review_filter_test` |
| `./internal/ingest` | Batch ingest, hashing, concurrency with store | Full package + `-race` when changing ingest |
| `./internal/store` | Collections, links, review queries | After FR-04/05/06 or grid data paths |
| `./tests/e2e` | CLI / binary parity with upload semantics | `make test-e2e` |

**CI tags:** `make test-ci` runs `go test -tags ci ./...` (preferences / CI-only paths) — keep green when shell startup or IDs change.

**Burn-in (optional):** `-count=5` or `-count=10` on `./internal/app` after flaky timing fixes (aligns with “flakiness is debt”).

---

## 4. Headless Fyne vs manual matrix (NFR-01 / NFR-07)

Classify by **user-visible invariant** first (per Phase 1 initiative note), then assign automation.

| Invariant | Headless / CI | Manual or hybrid |
|-----------|----------------|------------------|
| Nav + filter + bulk + loupe chrome **on canvas** at discrete matrix cells | `TestNFR01LayoutGate_matrixCells`, non-Review routes | Real WM, multi-monitor |
| Loupe widget tree **lockstep** with `review_loupe.go` (e.g. **Share…** width) | Gate fixture + assertions | Visual sanity |
| **NFR-07** OS scaling | Windows **LogPixels** jobs; macOS **CI surrogate** (`PHOTO_TOOL_NFR07_MACOS_CI_TIER`, `FYNE_SCALE`) documented in checklist | System Settings scaling on developer machines |
| **Continuous resize** — chrome does not stay permanently off-screen | Not fully simulated | Story **2.11** AC2 |
| **Thumb minimum readability**, letterbox ratios, chrome % budgets | Largely **not** encoded (deferred in evidence) | Matrix notes + future rubric |
| **UX-DR19** Tab / focus | Structural CI does not assert tab order | Manual checklist at S-min / 169-min |
| UX journey PNG smoke | `TestUXJourneyCapture` (macOS CI leg) | LLM judge loop is **local only** (not CI) |

**Honesty rule:** If an AC is marked **pass** on CI only, it must not contradict the manual matrix row for the same cell ID.

---

## 5. `-race` and **UX-DR17** (ingest / grid)

**When to run:** Any change that adds or rewires **goroutines**, **callbacks**, or **shared mutable state** between `ingest`, thumbnail decode, DB refresh, and Fyne widgets.

**How (local / gate):**

```bash
GO_TEST_RACE=1 ./scripts/bmad-story-workflow.sh --phase=gate
```

That enables `go test -race ./...` in the workflow script. For a faster loop while iterating:

```bash
go test -race ./internal/app ./internal/ingest -count=1
```

**Priority targets after edits:**

- **Upload async path:** `upload.go` worker + `fyne.Do`; tests in `upload_fr06_flow_test.go`, `upload_test.go`.
- **Grid async / list bind:** `review_grid.go` and friends; `review_grid_test.go` plus broader `review_test.go` flows.
- **Ingest + store:** `internal/ingest` together with app-level flows that call `IngestWithAssetIDs`.

**Interpretation:** A race failure is a **release blocker** for stories claiming **UX-DR17** until root-caused (either fix threading contract or shrink the claim in the story).

---

## 6. Quality gate checklist (initiative exit)

- [ ] `go test ./...` green on **macOS + Windows** matrix (see [.github/workflows/go.yml](../../.github/workflows/go.yml)).
- [ ] Story **2.11** evidence files updated; NFR-07 rows reference stable **cell IDs**.
- [ ] After async/threading changes: **`GO_TEST_RACE=1`** gate or equivalent `-race` run documented in PR.
- [ ] Manual matrix completed for tier-1 OS targets where AC requires it (not replaced by CI alone).

_Last updated: 2026-04-15 — Murat / `bmad-tea` headless YOLO._
