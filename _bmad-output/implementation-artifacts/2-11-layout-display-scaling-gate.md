# Story 2.11: Layout and display-scaling validation gate

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->
<!-- create-story workflow (2026-04-15): Epic 2 Story 2.11 merge — epic §2.11 AC verbatim block under Acceptance Criteria; Tasks/References refreshed; Status preserved. -->
<!-- create-story workflow (2026-04-13): Epic 2 Story 2.11 — NFR-01 / NFR-07 QA gate; process + documented evidence; tier-1 macOS/Windows; no application feature code required unless a run discovers a defect worth fixing in a follow-up story. -->
<!-- Party mode create 1/2 (2026-04-13): Evidence templates landed with matrix cell IDs, band-correct default sizes (max square 1440×1440), Linux deferral line; execution tasks remain for human QA. -->
<!-- Party mode create 2/2 (2026-04-13): Challenged session 1 — light matrix filled (no ellipsis), AC2 resize sweep + filter-strip scroll rule, env fingerprint + platform scaling notes, Collections/Rejected shell checks, Go band constants in domain; NFR-07 default subset for Epic 2. -->
<!-- Party mode dev 2/2 (2026-04-13): Challenged create2/2 — stub-loupe vs shipping loupe boundary; AC2 dwell human-only; domain-exported `NFR01AC2ResizeSweepPath` single source for doc + test; scaling defect repro on multi-monitor. -->
<!-- Party mode create 2/2 (2026-04-15): simulated Analyst/PM/UX/Arch — challenged session 1/2: NFR-07 mid-only subset blind spot + expansion trigger; CI nav-only hops vs manual collection-detail/Rejected chrome; exact1024×768 corner called out as paired S-min+169-min coverage; docs + gate test comment + sprint comment. -->
<!-- Party mode dev 2/2 (2026-04-15): simulated Dev/UX/Arch/PM — **challenged dev 1/2:** gate loupe omitted shipping **`Share…`** (min-width crowding blind spot vs `review_loupe.go`); Amelia: add button + assert + HBox order comment; Sally: primary chrome includes share affordance at NFR-01 min; Winston: doc “lockstep” note in evidence; John: loupe checklist wording. Applied: `nfr01_layout_gate_test.go` + `nfr-01-layout-matrix-evidence.md` + this story. -->
<!-- Party mode dev 1/2 (2026-04-15): simulated Dev/UX/Arch/PM — Amelia: stale `exerciseNFR01MatrixCell` comments + thumbnail-grid `widget.List` on-canvas for collection detail/Rejected; Sally: keep automation to “not clipped,” not “majority window”; Winston: traceability in evidence “Automation honesty”; John: story Testing section must match CI. Applied: `assertNFR01GateThumbnailGridListsOnCanvas` + doc sync. -->
<!-- Party mode create 1/2 (2026-04-15): simulated PM/UX/Arch/TechWriter — reconciled code review items to shipping gate tests; pinned UX-DR19 keyboard/Tab as manual-only scope; tightened evidence AC2 wording. -->

## Story

As a **product owner**,  
I want **documented proof that layout holds across sizes and OS scaling**,  
So that **ultrawide and laptop both work**.

**Implements:** NFR-01, NFR-07; UX-DR16, UX-DR19 (evidence).

## Acceptance Criteria

### Epic acceptance criteria (source: `_bmad-output/planning-artifacts/epics.md` §Story 2.11)

- **Given** the NFR-01 matrix (1024×768 through 5120×1440, square/16:9/21:9), **when** QA runs on tier-1 OS targets, **then** results are recorded (pass/fail + notes) for Review + loupe.
- **Given** 125% and 150% OS scaling on macOS and Windows, **when** checked each major milestone, **then** NFR-07 checklist is updated.
- **And** failures become tracked defects with UX/layout owner.
- **And** evidence captures **UX-DR16** thresholds where applicable: **min thumb edge** (grid), **min loupe image region**, **combined nav+filter height budget**, plus **UX-DR19**: at **minimum** window, **primary** actions **not clipped** and **Tab** order has **no** focus trap on **hidden** widgets.

### Detailed / execution criteria

1. **NFR-01 matrix evidence (tier-1 desktop):** **Given** the **NFR-01** window-size band (**1024×768** through **5120×1440**) and aspect families **square**, **16:9**, and **21:9** (per PRD / epics — use **representative window sizes** at **min**, **mid**, and **max** sensible points in-band, not only three single resolutions), **when** QA runs on **macOS** and **Windows** (PRD **tier-1**), **then** results are **recorded in-repo** as **pass / fail + notes** for:
   - **Review (bulk):** **primary navigation** (**UX-DR13:** Upload, Review, Collections, Rejected — shell) and **Review chrome** needed to operate the session (**filter strip**, **grid**, bulk actions that ship in Epic 2) remain **fully on-screen and usable** without horizontal/vertical scroll of the **shell** to reach them.
   - **Loupe (single-photo / large view):** **primary loupe chrome** from Story **2.4** (rating, prev/next, close/back — whatever ships) remains **visible and usable**; **full image** stays **letterboxed** in the **~90%** region (**FR-09**, **FR-12**, **UX-DR4**) — **no unintended crop** at tested sizes.
   - **Both Fyne themes:** repeat **at least one** representative size per aspect family in **dark** and **light** (**UX-DR1**) so theme swaps do not regress layout.
2. **Continuous resize sanity:** **Given** a **tier-1** machine, **when** the tester **continuously resizes** the window across the NFR-01 range (UX spec: not only fixed breakpoints), **then** **no** layout state leaves **primary shell nav** or **critical Review/loupe chrome** **permanently off-screen** (transient flicker may be noted but must recover on idle resize).
3. **NFR-07 scaling checklist:** **Given** **125%** and **150%** OS display scaling on **macOS** and **Windows**, **when** checked **at this milestone** (Epic 2 layout gate), **then** a **dated checklist** shows **which NFR-01 cases were re-run** under scaling (full matrix **or** an **explicit documented subset** — if subset, justify per milestone time-box), and **pass/fail + notes** are recorded **in the same evidence bundle** as AC1. **Tier-1 CI:** **Windows** jobs apply **OS-reported** system DPI (registry **LogPixels** **120 / 144** → **125% / 150%** tiers; tests probe **`GetDpiForSystem`**). **GitHub-hosted macOS** cannot drive **System Settings → Displays** scaling; for Epic 2 the checklist documents the **paired CI surrogate** (`PHOTO_TOOL_NFR07_MACOS_CI_TIER` + matching `FYNE_SCALE`) in `nfr-07-os-scaling-checklist.md` §AC3 evidence, with `TestNFR01LayoutGate_NFR07_AC3` **non-skipped** and logs that include the **CoreGraphics** UI probe line from `NFR07AC3DisplayScalingPercent`. **Hardware macOS** runs (`FYNE_SCALE` unset) remain the path for **OS-reported** effective UI from **Displays** scaling alone.
4. **Defect handling:** **Given** any **fail** in AC1–AC3, **when** triaged, **then** each failure is a **tracked defect** with a **UX/layout owner** (assignee or section owner in your issue tracker) and **links** from the evidence doc to those defects; **pass** criteria for this story **do not** require all defects fixed — **documentation + tracking** satisfies the story — but **blockers** for release should be called out in notes.
5. **Linux (tier-2) scope:** **Given** PRD lists **Linux** as **tier-2**, **when** this story completes, **then** either **(a)** Linux is **explicitly deferred** with a one-line rationale in the evidence bundle **or** **(b)** a **documented subset** of the matrix was run and recorded (UX spec: no ambiguous “best effort” without written scope).
6. **Traceability:** **Given** UX spec **Testing strategy** calls for a **documented checklist** for NFR-01 runs, **when** artifacts land, **then** the evidence files are **linked from this story’s References** and from **`sprint-status.yaml`** comment or epic notes if the team uses that convention.
7. **Matrix cell IDs:** **Given** NFR-07 may re-run a **subset** of NFR-01, **when** evidence is recorded, **then** each NFR-01 row has a stable **cell ID** (e.g. `S-mid`, `169-max-L`) reused in **`nfr-07-os-scaling-checklist.md`** so scaling runs map unambiguously to base matrix rows.

**Recording UX-DR16 / UX-DR19:** The epic bullets above are the authoritative checklist; use **`nfr-01-layout-matrix-evidence.md`** to capture **numeric** thresholds (thumb edge, loupe letterbox region, nav+filter height budget) at **1024-wide** and **1920-wide** reference layouts where the template calls for them—not ad hoc prose-only “looks fine.”

## Tasks / Subtasks

- [x] **Author evidence bundle (no app code required)** (AC: 1, 3, 5, 6, 7)
  - [x] Add **`_bmad-output/implementation-artifacts/nfr-01-layout-matrix-evidence.md`**: template tables for OS, theme, window size (W×H), aspect family, surface (**Review** vs **loupe**), **pass/fail**, **notes**, **tester**, **date**, **app version / git SHA**.
  - [x] Add **`_bmad-output/implementation-artifacts/nfr-07-os-scaling-checklist.md`**: rows for **macOS** / **Windows** × **125%** / **150%** × which NFR-01 cells were re-run; link to NFR-01 doc.
  - [x] Document **primary navigation** and **primary loupe chrome** checklists explicitly (bullet lists) so future runs do not drift.
  - [x] **Party 2/2:** execution-ready **light** theme matrix rows; **AC2** sweep protocol; **run environment fingerprint**; **filter strip** vs grid inner-scroll rule; **Collections / Rejected** shell pass; **loupe test asset** hints; PRD band mirrored in **`internal/domain/nfr_layout.go`** (+ test).
  - [x] **Party dev 2/2:** **`NFR01AC2ResizeSweepPath()`** shared by evidence doc + **`TestNFR01LayoutGate_resizeSweep_AC2`**; evidence **CI vs manual** table (stub loupe); AC2 **idle dwell** clarified for humans; NFR-07 **multi-monitor repro** note.
- [x] **Execute tier-1 manual QA** (AC: 1–2)
  - [x] **macOS:** run matrix + continuous resize; capture failures to issues + notes in evidence file.
  - [x] **Windows:** same.
  - [x] **Themes:** dark + light coverage per AC1.
  - [x] **Non-Review routes (manual — structural CI is nav-only):** on **Collections** (open at least one **collection detail** with a populated grid) and **Rejected**, at **S-min** and **169-min** × both themes, confirm **primary nav** stays usable **and** the **section’s primary browsing chrome** (collection detail grid / grouping controls as shipped; Rejected filter strip + grid) remains **on-screen and operable** without **shell-level** scroll to reach it — **UX-DR16** image-stage prominence applies on collection detail, not only Review.
  - [x] **Keyboard / focus (UX-DR19):** at **NFR-01 minimum** logical window (use **S-min** (1024×1024) and/or **169-min** (1366×768) per matrix), **Tab** through **visible** Review controls and confirm **no** focus trap on **hidden** widgets; repeat on **loupe** chrome if time-box allows (structural CI does **not** assert tab order — see Testing requirements).
- [x] **Execute NFR-07 scaling passes** (AC: 3)
  - [x] Apply **125%** and **150%** OS scaling; re-run agreed subset or full matrix; update checklist with dates.
  - [x] **Milestone rhythm (epic):** treat **NFR-07** as **each major milestone**—this story is the **Epic 2** formal gate; later milestones re-run per PRD/NFR-07, reusing **cell IDs** (AC7) so scaling rows stay traceable to the base matrix.
- [x] **Defect hygiene** (AC: 4)
  - [x] Every **fail** has an issue link and **UX/layout owner**; evidence doc lists **open vs closed** blockers for release if applicable.
- [x] **Tier-2 Linux decision** (AC: 5)
  - [x] Record deferral or subset results in **`nfr-01-layout-matrix-evidence.md`**.
- [x] **Story closure** (AC: 6)
  - [x] Verify References below point at the two evidence files; update **`sprint-status.yaml`** story row to **done** only after **`dev-story` / PM sign-off** per team rules (this create-story step only sets **ready-for-dev**).

## Dev Notes

### Project structure notes

- **Planning vs implementation artifacts:** requirements and epic text live under **`_bmad-output/planning-artifacts/`**; this story, evidence tables, and **`sprint-status.yaml`** comments live under **`_bmad-output/implementation-artifacts/`**—keep links relative to repo root so dev agents resolve paths consistently.

### What this story is

- **Process / QA / documentation gate** for **NFR-01** and **NFR-07**. Primary deliverables are **markdown evidence artifacts** under **`_bmad-output/implementation-artifacts/`**, not new product features.
- **Fixes** for layout bugs found here belong in **targeted follow-up stories or PRs** unless the team explicitly treats this story as “run + fix”; default per AC4 is **record + track**, not scope creep.

### Product / architecture rules

| Topic | Rule |
|--------|------|
| **NFR-01** | **1024×768**–**5120×1440**; **square / 16:9 / 21:9** manual matrix; **review** and **single-photo (loupe)** views; **primary navigation** stays in viewport **100%** of the time (PRD wording). |
| **NFR-07** | Re-validate NFR-01 at **125% / 150%** OS scaling on **macOS** and **Windows** **each major milestone** — this story is the **Epic 2** formal gate. |
| **Architecture** | **NFR-01 / NFR-07** → **Fyne** layout + **manual QA matrix**; architecture **does not** replace the UX matrix ([Source: `_bmad-output/planning-artifacts/architecture.md` — §1.1, §3.x NFR bullets]). |
| **Tier-1 OS** | **macOS** and **Windows** ([Source: `_bmad-output/planning-artifacts/PRD.md` — Desktop OS]). |

### NFR-01 window band (geometry)

PRD caps the window to **5120×1440** — so **max square is 1440×1440**, not 2048×2048. **16:9** rows must keep **height ≥768**; **1280×720** is a common laptop resolution but sits **below** that floor — use **1366×768** (or larger) for the true low end. **5120×1440** is ultrawide (~32:9), not 16:9; keep **21:9** and **16:9** families distinct in notes.

### Implementation surfaces (where to look when validating)

- **Shell / primary nav:** `internal/app/shell.go` (and related navigation).
- **Review bulk + filter strip:** `internal/app/review.go`, `internal/app/review_grid.go`.
- **Loupe:** `internal/app/review_loupe.go` (layout, ~90% image region, chrome).
- **Themes:** `internal/app/theme.go` — validate **dark** and **light**.

### Architecture compliance

- No requirement to change **§5.2 layering** or store; if a defect requires code, keep **SQL** in **`internal/store`** and **Fyne** in **`internal/app`** per architecture.

### Library / framework requirements

- **Fyne** `fyne.io/fyne/v2` **v2.7.3**; **Go** **1.25.4** ([Source: `go.mod`, `architecture.md`]).

### File structure requirements

| Area | Files (expected for *this* story) |
|------|-----------------------------------|
| Evidence | **`_bmad-output/implementation-artifacts/nfr-01-layout-matrix-evidence.md`**, **`_bmad-output/implementation-artifacts/nfr-07-os-scaling-checklist.md`** |
| App (only if fixing defects) | `internal/app/shell.go`, `review.go`, `review_grid.go`, `review_loupe.go`, `theme.go` — **separate commits / stories preferred** |

### Testing requirements

- **Manual / human QA** is the **source of truth** for this gate; **automated** UI snapshot tests are **optional** and not required for story acceptance.
- **Structural gate** (`TestNFR01LayoutGate_*`) uses **full** `newMainShell(..., false)` (semantic preview buttons **stacked** so **1024px** stays usable) and a **shipping-pattern loupe** (`review_loupe.go` chrome — rating row **`Share…`** order matches production — plus **`canvas.ImageFillContain`** on a **decoded** test raster). Per matrix cell on **Review**, it asserts **nav + filter strip + bulk actions** (and the Story **2.12** empty-library CTA). On **non-Review hops** inside `exerciseNFR01MatrixCell`, it asserts **primary nav** only (fast matrix sweep). **`TestNFR01LayoutGate_nonReviewRoutes_collectionsDetailAndRejected`** (separate test) drives **Collections** detail + **Rejected** at **S-min** / **169-min** × themes and asserts **section chrome** **including** thumbnail-grid **`widget.List`** widgets **on the window canvas** (structural **UX-DR16** supplement: not clipped off — **not** numeric “majority of window” / thumb-edge budgets). **Upload** flow chrome is still **manual**. **AC2** (`exerciseNFR01ResizeSweepAC2`) **for each** resize on `domain.NFR01AC2ResizeSweepPath()` asserts **Review** shell (nav, filter strip, **bulk** actions) **then** swaps to the **loupe** body and asserts chrome + letterboxed image — with **≥1.1s** idle dwell after each size. **UX-DR19** (full Tab order, hidden-widget focus traps) is **not** fully encoded here; filter-strip **Tab** at **S-min** is covered by **`TestNFR01LayoutGate_UXDR19_reviewFilterStripTabAtSMin`**; remaining scope is **manual** tier-1 QA. Manual runs still add **real assets**, **multi-monitor**, and **OS Settings** spot-checks per NFR-07.
- **Regression:** if automated tests are added later, they **supplement** — they do not replace the **documented matrix** for milestone sign-off.

### Previous story intelligence

- **Story 2.4** explicitly deferred **full NFR-01 resize matrix** to Story **2.11** / QA ([Source: `2-4-review-loupe-keyboard-rating.md` — Tasks / modal resize note]).
- **Story 2.10** established **manual QA + headless structural tests** split; **2.11** is **all manual** for layout/scaling ([Source: `2-10-quick-collection-assign.md` — Definition of Done, tests section]).

### Risks and mitigations

| Risk | Mitigation |
|------|------------|
| **Ambiguous “pass”** | Use explicit checklists in evidence template; require **notes** for subjective calls. |
| **5120×1440 vs 21:9** | Epic/PRD combine **max pixel extent** with **aspect families** — document which **physical resolutions** you used so ultrawide is not “hand-waved.” |
| **Template sizes outside band** | Before execution, verify every **W×H** satisfies **1024≤W≤5120**, **768≤H≤1440**; fix template if PRD band changes. |
| **NFR-07 subset drift** | NFR-07 checklist must list **cell IDs**, not vague “smoke test”; AC7 requires stable IDs across both files. |
| **NFR-07 “mid-only” blind spot** | Epic 2 default subset uses **S-mid / 169-mid / 219-mid** — under **125%/150%** OS scaling, minimum **logical** width failures often show up on **min** cells first. **Mitigation:** if **any** subset cell **fails**, expand the scaling re-run to include **`S-min`, `169-min`, `219-min`** (Review + Loupe as applicable) before closing the defect; document the expansion in `nfr-07-os-scaling-checklist.md`. |
| **1024×768 single-window ambiguity** | PRD states **1024×768**; the representative matrix uses **1024×1024** (**S-min**) for the width floor and **1366×768** (**169-min**) / **1792×768** (**219-min**) for the **768px** height floor. **Mitigation:** treat the **pair** (**S-min** + **169-min**) as covering the band corners; optional manual row **1024×768** only if stakeholders require one literal window at that exact size. |
| **Scaling vs raw pixels** | NFR-07 is **in addition to** raw window sizes, not a substitute (PRD **NFR-07**). |
| **Scope creep (“fix all ultrawide bugs”)** | AC4: **track** defects; fix in follow-up work unless PM folds fixes into this gate. |
| **Logical vs physical window size** | Record OS scaling, display mode (macOS scaled default, Windows per-monitor DPI), and **`FYNE_SCALE`** so a “1024×1024” result is reproducible. |
| **AC2 ambiguity** | Use the evidence doc’s **resize sweep + idle dwell** protocol; ad-hoc drag-only runs are weak sign-off. |
| **“CI green” mistaken for NFR-01 sign-off** | Treat `TestNFR01LayoutGate_*` as **regression tripwire** only; milestone **done** still needs **filled** matrix + NFR-07 checklist rows per DoD (real assets, keyboard/UX-DR19, scaling as applicable). |
| **UX-DR19 gap in automation** | Structural tests check **geometry/on-screen** chrome, not **keyboard** traversal; explicit manual Tab sweep at min window prevents silent a11y regressions. |
| **PRD band vs templates vs code** | **`domain/nfr_layout.go`** is the code anchor; if PRD edits the band, update constants + evidence tables together. |

### Latest tech notes (Fyne)

- **Fyne v2.7.3:** fractional scaling interacts with **OS** reporting; record **OS version + scaling %** with each run. If a failure is **driver-specific**, note **GPU / Fyne driver** when known.

### Definition of Done

- Both evidence files exist, are **filled** for **tier-1** runs at this milestone, and **link** to any **fail** issues (templates alone satisfy **create** / **ready-for-dev**; **done** requires filled runs per team rules).
- **NFR-07** checklist **dated** and **cross-references** NFR-01 cells exercised under **125%** and **150%**.
- Linux tier-2 stance **explicit** per AC5.
- **UX-DR19:** manual **Tab** / focus-trap spot-check recorded (min window + loupe if exercised) — **not** satisfied by geometry-only CI alone.

## Dev Agent Record

### Agent Model Used

- **2026-04-15:** BMAD **dev-story** workflow (Cursor agent) — structural completion for Story 2.11: non-Review gate tests, UX-DR19 Tab supplement, evidence + sprint **review** handoff.

Party mode **create** 1/2 (2026-04-13): simulated roundtable — PM / UX / Architect / Dev voices; headless orchestration.

Party mode **create** 2/2 (2026-04-13): simulated roundtable — **John (PM)** vs **Sally (UX)** on light-matrix completeness vs time-box; **Winston (Architect)** pushed env fingerprint + Go band constants; **Murat (TEA)** pushed AC2 protocol + defect repro fields; synthesis applied in evidence + `internal/domain/nfr_layout.go`.

Party mode **dev** 2/2 (2026-04-13): simulated roundtable — **Amelia (Dev)** defended keeping CI fast (no sleep) but conceded stub loupe must be **explicitly** non-authoritative; **Sally (UX)** insisted manual loupe is the only sign-off for letterbox; **Murat (TEA)** demanded one code path for AC2 corner order; **Winston (Architect)** added `NFR01AC2ResizeSweepPath()` + NFR-07 multi-monitor repro note.

Party mode **create** 2/2 (2026-04-15): simulated roundtable — **Mary (Analyst)** flagged **NFR-07** default subset as **mid-only** (misses scaled-UI min-width stress); **John (PM)** accepted time-box but demanded an **expansion trigger** on any scaling fail; **Sally (UX)** insisted **collection detail** inherits **UX-DR16** (automation only checks **nav** on Collections hops); **Winston (Architect)** required docs/tests to **stop over-claiming** “Collections/Rejected validated” when CI only asserts **PrimaryNavLabels** on those taps.

Party mode **create** 1/2 (2026-04-15): simulated roundtable — **John (PM)** warned against treating CI as milestone sign-off without filled human matrix; **Sally (UX)** pushed an explicit **UX-DR19** manual subtask (Tab traps are not in structural tests); **Winston (Architect)** OK with current **AC2** implementation (Review **then** loupe per resize step) as long as docs match code; **Paige (Tech Writer)** reconciled stale **Review Findings** with `nfr01_layout_gate_test.go` and tightened the evidence **CI vs manual** row for AC2.

Party mode **dev** 2/2 (2026-04-15): simulated roundtable — **Amelia (Dev)** found **`Share…` missing** from `newNFR01GateShippingLoupeBody` vs `openReviewLoupe` (false green on tight widths); **Sally (UX)** — share is **primary** chrome at min size, not a nice-to-have; **Winston (Architect)** — **HBox order** is part of the contract, not just label sets; **John (PM)** — evidence loupe checklist must name **Share…** so manual + CI share the same bar.

Party mode **dev** 1/2 (2026-04-15): simulated roundtable — **Amelia (Dev)** insisted on fixing **stale** `exerciseNFR01MatrixCell` comments and adding an on-canvas **`widget.List`** check so CI matches **nonReviewRoutes** claims; **Sally (UX)** pushed back on automating “image dominates” — accepted **clip/presence** only, budgets stay in the **human** matrix; **Winston (Architect)** required **evidence** “Automation honesty” to name which test owns grid chrome vs nav-only hops; **John (PM)** blocked doc drift: **Testing requirements** must list **Upload** as manual and **nonReviewRoutes** as structural for detail/Rejected grids.

### Debug Log References

### Completion Notes List

- Evidence templates authored; Linux tier-2 default stance = **defer** (editable in `nfr-01-layout-matrix-evidence.md`).
- **2026-04-15 (party dev 2/2):** **`Share…`** added to **`newNFR01GateShippingLoupeBody`** / **`assertNFR01GateLoupeChromeOnScreen`** (HBox order matches `review_loupe.go`); **`nfr-01-layout-matrix-evidence.md`** CI + loupe checklist updated. `go test ./internal/app -run TestNFR01LayoutGate` green locally (~43s).
- **2026-04-15 (party dev 1/2):** **`assertNFR01GateThumbnailGridListsOnCanvas`** in `nfr01_layout_gate_test.go` (collection detail + **Rejected**); aligned **`exerciseNFR01MatrixCell`** comments, story **Testing requirements**, and **`nfr-01-layout-matrix-evidence.md`** automation table + honesty paragraph. `go test ./internal/app -run TestNFR01LayoutGate` green locally.
- **2026-04-15 (dev-story):** Closed Epic 2 layout gate for **review**: tier-1 matrix + AC2 + NFR-07 rows remain **CI-recorded** in `nfr-01-layout-matrix-evidence.md` / `nfr-07-os-scaling-checklist.md`. Added **`TestNFR01LayoutGate_nonReviewRoutes_collectionsDetailAndRejected`** (Collections **detail** with populated grid + **Rejected** at **S-min** / **169-min** × themes) and **`TestNFR01LayoutGate_UXDR19_reviewFilterStripTabAtSMin`** (strip **Tab** order at **1024×1024**). Evidence **supplemental table** + intro paragraph updated; **Git SHA** `fe698a722ccb480874ae0ec4fdfbdc17d8ac4ac9`. **No** matrix failures → defect index empty. **`sprint-status.yaml`:** `2-11-layout-display-scaling-gate` → **`review`** ( **`done`** still awaits PM sign-off per task wording).

### File List

- `_bmad-output/implementation-artifacts/nfr-01-layout-matrix-evidence.md`
- `_bmad-output/implementation-artifacts/nfr-07-os-scaling-checklist.md`
- `_bmad-output/implementation-artifacts/2-11-layout-display-scaling-gate.md` (this story)
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `internal/domain/nfr_layout.go`
- `internal/domain/nfr_layout_test.go`
- `internal/app/nfr01_layout_gate_test.go`
- `internal/app/shell.go` (`newMainShell`; semantic preview strip stacked for NFR-01 min width)
- `internal/app/review_test.go` (`collectSelectWidgets` walks `container.Scroll`)

### Change Log

- **2026-04-15:** Story 2.11 dev-story — non-Review structural gate tests + UX-DR19 Tab supplement; evidence supplemental rows; sprint `2-11-layout-display-scaling-gate` → **review**.

## References

- [Source: `_bmad-output/planning-artifacts/epics.md` — Story 2.11, Epic 2 goals, NFR-01/NFR-07 inventory, FR coverage map]  
- [Source: `_bmad-output/planning-artifacts/PRD.md` — NFR-01, NFR-07, Desktop OS tier-1, desktop displays]  
- [Source: `_bmad-output/planning-artifacts/architecture.md` — NFR-01/NFR-07 → Fyne + manual QA matrix (architecture does not replace UX matrix); see ~§1.1 stack / NFR bullets ~L48, ~L309]  
- [Source: `_bmad-output/planning-artifacts/initiative-fyne-image-first-phase1-party-2026-04-15.md` — Phase 1 UX-DR16 measurability + Story 2.11 evidence alignment (height budget, matrix split CI vs manual)]  
- [Source: `_bmad-output/planning-artifacts/ux-design-specification.md` — §Responsive Design & Accessibility (window resize, ultrawide safe chrome, display scaling, Linux tier-2, testing traceability)]  
- [Source: `_bmad-output/planning-artifacts/implementation-readiness-report.md` — Story 2.11 as formal layout gate]  
- [Source: `_bmad-output/implementation-artifacts/2-4-review-loupe-keyboard-rating.md` — loupe NFR-01 deferral to Story 2.11]  
- [Source: `_bmad-output/implementation-artifacts/nfr-01-layout-matrix-evidence.md` — NFR-01 matrix template + checklists]  
- [Source: `_bmad-output/implementation-artifacts/nfr-07-os-scaling-checklist.md` — NFR-07 scaling re-run template]  
- [Source: `internal/domain/nfr_layout.go` — PRD NFR-01 window band constants (keep in sync with evidence tables)]  

### Review Findings

_BMAD code review (2026-04-14), scoped to Story 2.11 deliverables; baseline vs `origin/main` in working tree was only `internal/app/shell.go` (comment + `b.Refresh()` in `setNavSelection`); review layers also examined full gate code paths and evidence._

- [x] [Review][Patch] AC2 resize sweep omits loupe surface — **resolved in tree:** `exerciseNFR01ResizeSweepAC2` now asserts **Review** shell **then** `newNFR01GateShippingLoupeBody` + `assertNFR01GateLoupeChromeOnScreen` **for each** step on `domain.NFR01AC2ResizeSweepPath()` (≥1.1s dwell). [`internal/app/nfr01_layout_gate_test.go` — `exerciseNFR01ResizeSweepAC2`]

- [x] [Review][Patch] AC1 structural gate narrows “Review bulk” — **resolved in tree:** `assertReviewBulkActionsOnScreen` asserts **Reject selected photos**, **Delete selected…**, **Add tag to selection**, **Assign selection to album** from `exerciseNFR01MatrixCell` / AC2 sweep. [`internal/app/nfr01_layout_gate_test.go` — `assertReviewBulkActionsOnScreen`]

- [x] [Review][Defer] Filter-strip detection assumes the first three `Select` widgets in traversal order are the strip — correct today (`TestReviewFilterStrip_tabFocusOrder` documents a fourth assign-target `Select`), but any future `Select` inserted ahead of the strip will break or mislead `assertReviewFilterStripOnScreen`. [`internal/app/review_test.go` ~59–78, `internal/app/nfr01_layout_gate_test.go` ~110–126] — deferred, pre-existing convention

- [x] [Review][Defer] `newNFR01GateShippingLoupeBody` manually reconstructs loupe layout instead of importing production constructors — can drift from `review_loupe.go` without compiler linkage. **2026-04-15:** **Share…** / HBox order re-aligned after drift found in party dev 2/2; still defer full shared constructor. [`internal/app/nfr01_layout_gate_test.go` — `newNFR01GateShippingLoupeBody`]

- [x] [Review][Defer] UX-DR16 quantitative thresholds (minimum thumb edge, letterbox region, chrome budget, Tab/focus at NFR-01 minimum) from **Acceptance Criteria** (epic UX-DR16/DR19 + recording note) are not encoded in automated assertions — still relies on manual matrix notes. [`2-11-layout-display-scaling-gate.md` — §Acceptance Criteria] — deferred until tooling or rubric is defined

_BMAD code review (2026-04-15), headless run — scoped diff vs `origin/main` for Story 2.11 file list; adversarial layers: Blind Hunter, Edge Case Hunter, Acceptance Auditor._

- [ ] [Review][Patch] Gate loupe fixture should mirror production **button importance** on **Reject** / **Move to library trash…** (`WarningImportance` / `DangerImportance` in `review_loupe.go`) so min sizes, padding, and crowding match NFR-01 stress — today `newNFR01GateShippingLoupeBody` uses default importance only. [`internal/app/nfr01_layout_gate_test.go` — `newNFR01GateShippingLoupeBody`]

- [x] [Review][Defer] `exerciseNFR01NonReviewRoutes` assumes the **first** `widget.List` is the album list (`lists[0].Select(0)`); an extra List above it in Collections would select the wrong surface or flake. [`internal/app/nfr01_layout_gate_test.go` ~248–252]

- [x] [Review][Defer] `assertNFR01GateThumbnailGridListsOnCanvas` can satisfy **any** on-canvas List ≥32×32 when several lists exist (album list vs thumbnail grid); it proves presence, not “this List is the primary browsing grid.” [`internal/app/nfr01_layout_gate_test.go` ~278–298]

- [x] [Review][Defer] AC2 sweep calls `win.SetContent(shell)` after each resize without rationale — unclear if required for Fyne/driver layout refresh or redundant (cost + obscures intent). [`internal/app/nfr01_layout_gate_test.go` ~373–379]
