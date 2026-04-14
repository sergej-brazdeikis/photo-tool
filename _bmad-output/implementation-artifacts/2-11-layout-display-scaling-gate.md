# Story 2.11: Layout and display-scaling validation gate

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->
<!-- create-story workflow (2026-04-13): Epic 2 Story 2.11 — NFR-01 / NFR-07 QA gate; process + documented evidence; tier-1 macOS/Windows; no application feature code required unless a run discovers a defect worth fixing in a follow-up story. -->
<!-- Party mode create 1/2 (2026-04-13): Evidence templates landed with matrix cell IDs, band-correct default sizes (max square 1440×1440), Linux deferral line; execution tasks remain for human QA. -->
<!-- Party mode create 2/2 (2026-04-13): Challenged session 1 — light matrix filled (no ellipsis), AC2 resize sweep + filter-strip scroll rule, env fingerprint + platform scaling notes, Collections/Rejected shell checks, Go band constants in domain; NFR-07 default subset for Epic 2. -->
<!-- Party mode dev 2/2 (2026-04-13): Challenged create2/2 — stub-loupe vs shipping loupe boundary; AC2 dwell human-only; domain-exported `NFR01AC2ResizeSweepPath` single source for doc + test; scaling defect repro on multi-monitor. -->

## Story

As a **product owner**,  
I want **documented proof that layout holds across sizes and OS scaling**,  
So that **ultrawide and laptop both work**.

**Implements:** NFR-01, NFR-07.

## Acceptance Criteria

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

## Tasks / Subtasks

- [x] **Author evidence bundle (no app code required)** (AC: 1, 3, 5, 6, 7)
  - [x] Add **`_bmad-output/implementation-artifacts/nfr-01-layout-matrix-evidence.md`**: template tables for OS, theme, window size (W×H), aspect family, surface (**Review** vs **loupe**), **pass/fail**, **notes**, **tester**, **date**, **app version / git SHA**.
  - [x] Add **`_bmad-output/implementation-artifacts/nfr-07-os-scaling-checklist.md`**: rows for **macOS** / **Windows** × **125%** / **150%** × which NFR-01 cells were re-run; link to NFR-01 doc.
  - [x] Document **primary navigation** and **primary loupe chrome** checklists explicitly (bullet lists) so future runs do not drift.
  - [x] **Party 2/2:** execution-ready **light** theme matrix rows; **AC2** sweep protocol; **run environment fingerprint**; **filter strip** vs grid inner-scroll rule; **Collections / Rejected** shell pass; **loupe test asset** hints; PRD band mirrored in **`internal/domain/nfr_layout.go`** (+ test).
  - [x] **Party dev 2/2:** **`NFR01AC2ResizeSweepPath()`** shared by evidence doc + **`TestNFR01LayoutGate_resizeSweep_AC2`**; evidence **CI vs manual** table (stub loupe); AC2 **idle dwell** clarified for humans; NFR-07 **multi-monitor repro** note.
- [ ] **Execute tier-1 manual QA** (AC: 1–2)
  - [ ] **macOS:** run matrix + continuous resize; capture failures to issues + notes in evidence file.
  - [ ] **Windows:** same.
  - [ ] **Themes:** dark + light coverage per AC1.
- [ ] **Execute NFR-07 scaling passes** (AC: 3)
  - [ ] Apply **125%** and **150%** OS scaling; re-run agreed subset or full matrix; update checklist with dates.
- [ ] **Defect hygiene** (AC: 4)
  - [ ] Every **fail** has an issue link and **UX/layout owner**; evidence doc lists **open vs closed** blockers for release if applicable.
- [ ] **Tier-2 Linux decision** (AC: 5)
  - [ ] Record deferral or subset results in **`nfr-01-layout-matrix-evidence.md`**.
- [ ] **Story closure** (AC: 6)
  - [ ] Verify References below point at the two evidence files; update **`sprint-status.yaml`** story row to **done** only after **`dev-story` / PM sign-off** per team rules (this create-story step only sets **ready-for-dev**).

## Dev Notes

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
- **Structural gate** (`TestNFR01LayoutGate_*`) uses **full** `newMainShell(..., false)` (semantic preview buttons **stacked** so **1024px** stays usable) and a **shipping-pattern loupe** (`review_loupe.go` chrome + **`canvas.ImageFillContain`** on a **decoded** test raster). **AC2** sweep sleeps **≥1.1s** after each resize. Manual runs still add **real assets**, **multi-monitor**, and **OS Settings** spot-checks per NFR-07.
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
| **Scaling vs raw pixels** | NFR-07 is **in addition to** raw window sizes, not a substitute (PRD **NFR-07**). |
| **Scope creep (“fix all ultrawide bugs”)** | AC4: **track** defects; fix in follow-up work unless PM folds fixes into this gate. |
| **Logical vs physical window size** | Record OS scaling, display mode (macOS scaled default, Windows per-monitor DPI), and **`FYNE_SCALE`** so a “1024×1024” result is reproducible. |
| **AC2 ambiguity** | Use the evidence doc’s **resize sweep + idle dwell** protocol; ad-hoc drag-only runs are weak sign-off. |
| **PRD band vs templates vs code** | **`domain/nfr_layout.go`** is the code anchor; if PRD edits the band, update constants + evidence tables together. |

### Latest tech notes (Fyne)

- **Fyne v2.7.3:** fractional scaling interacts with **OS** reporting; record **OS version + scaling %** with each run. If a failure is **driver-specific**, note **GPU / Fyne driver** when known.

### Definition of Done

- Both evidence files exist, are **filled** for **tier-1** runs at this milestone, and **link** to any **fail** issues (templates alone satisfy **create** / **ready-for-dev**; **done** requires filled runs per team rules).
- **NFR-07** checklist **dated** and **cross-references** NFR-01 cells exercised under **125%** and **150%**.
- Linux tier-2 stance **explicit** per AC5.

## Dev Agent Record

### Agent Model Used

Party mode **create** 1/2 (2026-04-13): simulated roundtable — PM / UX / Architect / Dev voices; headless orchestration.

Party mode **create** 2/2 (2026-04-13): simulated roundtable — **John (PM)** vs **Sally (UX)** on light-matrix completeness vs time-box; **Winston (Architect)** pushed env fingerprint + Go band constants; **Murat (TEA)** pushed AC2 protocol + defect repro fields; synthesis applied in evidence + `internal/domain/nfr_layout.go`.

Party mode **dev** 2/2 (2026-04-13): simulated roundtable — **Amelia (Dev)** defended keeping CI fast (no sleep) but conceded stub loupe must be **explicitly** non-authoritative; **Sally (UX)** insisted manual loupe is the only sign-off for letterbox; **Murat (TEA)** demanded one code path for AC2 corner order; **Winston (Architect)** added `NFR01AC2ResizeSweepPath()` + NFR-07 multi-monitor repro note.

### Debug Log References

### Completion Notes List

- Evidence templates authored; Linux tier-2 default stance = **defer** (editable in `nfr-01-layout-matrix-evidence.md`).

### File List

- `_bmad-output/implementation-artifacts/nfr-01-layout-matrix-evidence.md`
- `_bmad-output/implementation-artifacts/nfr-07-os-scaling-checklist.md`
- `_bmad-output/implementation-artifacts/2-11-layout-display-scaling-gate.md` (this story)
- `internal/domain/nfr_layout.go`
- `internal/domain/nfr_layout_test.go`
- `internal/app/nfr01_layout_gate_test.go`
- `internal/app/shell.go` (`newMainShell`; semantic preview strip stacked for NFR-01 min width)
- `internal/app/review_test.go` (`collectSelectWidgets` walks `container.Scroll`)

## References

- [Source: `_bmad-output/planning-artifacts/epics.md` — Story 2.11, Epic 2 goals, NFR-01/NFR-07 inventory, FR coverage map]  
- [Source: `_bmad-output/planning-artifacts/PRD.md` — NFR-01, NFR-07, Desktop OS tier-1, desktop displays]  
- [Source: `_bmad-output/planning-artifacts/architecture.md` — NFR-01/NFR-07 → Fyne + manual QA; stack versions]  
- [Source: `_bmad-output/planning-artifacts/ux-design-specification.md` — §Responsive Design & Accessibility (window resize, ultrawide safe chrome, display scaling, Linux tier-2, testing traceability)]  
- [Source: `_bmad-output/planning-artifacts/implementation-readiness-report.md` — Story 2.11 as formal layout gate]  
- [Source: `_bmad-output/implementation-artifacts/2-4-review-loupe-keyboard-rating.md` — loupe NFR-01 deferral to Story 2.11]  
- [Source: `_bmad-output/implementation-artifacts/nfr-01-layout-matrix-evidence.md` — NFR-01 matrix template + checklists]  
- [Source: `_bmad-output/implementation-artifacts/nfr-07-os-scaling-checklist.md` — NFR-07 scaling re-run template]  
- [Source: `internal/domain/nfr_layout.go` — PRD NFR-01 window band constants (keep in sync with evidence tables)]  
