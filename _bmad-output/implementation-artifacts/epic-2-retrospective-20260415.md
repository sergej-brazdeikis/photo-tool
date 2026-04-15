# Epic 2 retrospective — photo-tool

**Epic:** Review, filter, organize, and curate (`epics.md`)  
**Mode:** Headless synthesis from `sprint-status.yaml`, `epics.md` (Epic 2), `deferred-work.md`, and story artifact changelog notes. No live facilitation.  
**Date:** 2026-04-15  
**Stance:** Psychological safety; systems, processes, and learning—not individuals.

---

## Epic status (from tracking)

Per `_bmad-output/implementation-artifacts/sprint-status.yaml`:

| Key | Status |
|-----|--------|
| `epic-2` | in-progress |
| `2-1-app-shell-navigation-themes` | done |
| `2-2-filter-strip` | done |
| `2-3-thumbnail-grid-rating-badges` | done |
| `2-4-review-loupe-keyboard-rating` | in-progress |
| `2-5-tags-bulk-review` | done |
| `2-6-reject-undo-hidden-restore` | review |
| `2-7-delete-quarantine` | done |
| `2-8-collections-list-detail` | review |
| `2-9-collection-crud-multi-assign` | in-progress |
| `2-10-quick-collection-assign` | review |
| `2-11-layout-display-scaling-gate` | review |
| `2-12-empty-states-error-tone` | in-progress |
| `epic-2-retrospective` | optional |

**Counts:** 5 stories **done**, 4 **review**, 3 **in-progress** (12 total).

**Interpretation:** This is a **checkpoint retrospective**, not a “green closure” of Epic 2. Substantial UX and Fyne work is merged (shell, filters, grid, tags, delete, and ongoing collections/reject/loupe/layout/error-tone work), but the board still carries open **review** and **in-progress** keys. Work on Epic 3 and Epic 4 is already underway in the same sprint file; that parallelization is a **process signal**, not a judgment on any individual.

---

## Continuity from Epic 1 retrospective (2026-04-13)

| Epic 1 retro theme | Evidence in Epic 2 timeframe |
|--------------------|------------------------------|
| Align “done” across sprint, story status, and review patches | Mixed: many Epic 2 keys remain `review` / `in-progress` while implementation and party-mode notes show active hardening. |
| Board completeness | Epic 2 stories 2.1–2.12 are all tracked; better visibility than early Epic 1 tracking. |
| Collections / SQLite robustness | Collections and review surfaces advanced; `deferred-work.md` still records CRUD, duplicate-name `Select` ambiguity, and filter-refresh gaps to close under process discipline. |

---

## What went well

1. **Specs and traceability tightened with UX alignment** — Epic 2 maps to **UX-DR1–DR5, DR8–DR10, DR13, DR15–DR19** and **NFR-01 / NFR-07** on layout and scaling. Story artifacts and changelog entries show **create-story / party-mode** passes that pinned acceptance criteria, risks, and evidence links (e.g. layout matrix and OS scaling checklists) before and during implementation.

2. **Image-first direction showed up in delivered slices** — Shell and navigation (**2-1**), ordered filter strip with defaults and degradation copy (**2-2**), and paged grid with rating/reject badges and zero-count scroll behavior (**2-3**) are **done** and supported by targeted tests and review notes—consistent with **UX-DR16** and **UX-DR18** coordination called out in `epics.md`.

3. **Automated layout gate started** — Domain layout helpers and `nfr01_layout_gate_test.go` (and related tests) give **repeatable** signals for “primary chrome stays on screen” style constraints, complementing manual NFR-01 / NFR-07 evidence. That reduces reliance on ad hoc “looks fine on my monitor.”

4. **Cross-cutting quality habits** — **`deferred-work.md`** records code-review deferrals with file pointers (filter-strip widget-order assumptions, loupe gate duplication vs production, Review vs Rejected refresh duplication, grid page failure caching). That supports **systems-level** backlog grooming instead of oral tradition.

5. **Curatorial flows progressed** — Tags in bulk review (**2-5**), delete with quarantine (**2-7**), and substantial work on reject/undo, collections, quick assign, empty states, and layout gate (**2-6**–**2-12** in various states) move the product toward **FR-07–FR-12, FR-15–FR-25, FR-29–FR-31** as a coherent epic.

---

## What to improve

1. **Close the Epic 2 funnel before treating the epic as “shipped”** — Multiple stories sit in **review** or **in-progress** while later epics are active. The **system** risk is integration debt: share (**Epic 3**) and packages (**Epic 4**) assume stable review, reject, loupe, and collection semantics. Prefer a **WIP limit** or explicit “Epic 2 exit criteria” on the board over unbounded parallel epic work.

2. **Definition of done: one source of truth** — Align `sprint-status.yaml`, story `Status`, open **Review** / **Patch** items, and BMAD code-review outcomes. Stories that stay in **review** with green tests still consume attention and hide regressions when **Epic 3** changes touch the same packages.

3. **Brittle headless UI tests** — Deferred items note **widget tree order** assumptions (first `Select` triple, first `List` in collections gate, loupe stub mirroring production). The process should treat these as **known coupling**: when UI structure changes, gate tests should fail loudly or be refactored with stable test IDs / narrower assertions.

4. **Shared grid behavior under error** — Documented pattern: **transient query failures** can leave a grid page in a failed state until filters or navigation force refresh (**2-4** / **2-6** deferrals). From a **systems** perspective, decide whether automatic retry, visible “retry” affordance, or explicit user messaging is the standard—then apply consistently on Review and Rejected.

5. **Duplication hotspots** — `refreshReviewData` vs `refreshRejectedData` and loupe gate vs `review_loupe.go` create **drift risk**. A small **shared helper** or shared layout builder (where safe) reduces parity bugs between modes.

6. **Epic 1 open items still interact with Epic 2** — Upload / drag-drop (**1-5**, **1-8**) remain in **review** while review surfaces depend on ingest and receipts. Track **cross-epic** readiness explicitly so “Review feels done” is not undermined by upload edge cases.

---

## Risks for the next epic (Epic 3: Share a single photo)

1. **Reject and eligibility rules** — Share flows depend on **FR-29 / FR-32** (rejected assets excluded, preview before mint). Incomplete or unstable **2-6** / **2-4** behavior can leak into “wrong asset” or “blocked share” confusion in **3-1**–**3-3**.

2. **Loupe and chrome stability** — Mint and preview UX assume a trustworthy large view and keyboard/star behavior. **2-4** still **in-progress**; layout gate (**2-11**) still **review**: any mismatch between **CI stub loupe** and **manual loupe** can hide real regressions until late.

3. **Collection and filter consistency** — **2-9** / **2-8** / **2-10** in non-done states risk stale lists, ambiguous collection labels (duplicate names), or filter-vs-CRUD refresh gaps surfacing when share copy references “current” library state.

4. **NFR-07 vs CI** — Manual OS scaling sign-off may lag **nav-only** automation. The **risk** is signing Epic 2 layout without exercising collection detail / Rejected at scaled DPI, then discovering share-adjacent chrome issues under **Epic 3** manual QA.

5. **Error tone and empty states** — **2-12** **in-progress**; share stories need **clear failure** copy (mint, loopback, HTML). Weak **UX-DR9 / UX-DR18** on Review can set the wrong pattern for **UX-DR7** in Epic 3.

---

## Action items (concrete)

1. **Publish Epic 2 exit checklist on the board** — **Owner:** Product / project lead. **Success:** A short ordered list (e.g. loupe + reject + layout evidence + collections CRUD + empty/error matrix) is **explicitly** satisfied or consciously deferred with a recorded reason; `epic-2` is not treated as complete until that list is green.

2. **Drive all Epic 2 keys to terminal states** — **Owner:** Developer + reviewer. **Success:** Each `2-*` key ends as **done** or **backlog** with an issue link; no long-lived **review** without an open patch list or merge decision within one planning cycle.

3. **Harden layout gate tests against structural drift** — **Owner:** Developer. **Success:** Replace or supplement “first N widgets” assumptions with assertions tied to **stable object names**, exported test hooks, or narrower component contracts; update `deferred-work.md` when resolved.

4. **Extract shared Review/Rejected refresh helper** — **Owner:** Developer. **Success:** One code path (or documented shared function) for collection/tag strip refresh and count-line error suffixes so Review and Rejected cannot diverge silently (`deferred-work.md` item).

5. **Grid query failure UX standard** — **Owner:** Developer + UX. **Success:** Document and implement one approach (retry, banner + action, or automatic invalidation) for failed page loads on **Review** and **Rejected** grids; close related deferrals in **2-4** / **2-6** story records.

---

## Significant discoveries (systems level)

- **Parallel epics on one sprint file** — Epic 2, 3, and 4 all show **in-progress** work simultaneously. That can improve throughput but **obscures** epic-level completion and retro honesty; the tracking system should make **epic completion** visible separately from **story churn**.

- **Evidence split (CI vs manual)** — Layout and scaling stories correctly call out **automation vs human sign-off**. The gap between them is a **process risk**, not a personal one: treat manual matrix rows as first-class deliverables with owners.

---

## Next steps (non-binding)

- Re-run or extend this retrospective when **all** `2-*` stories reach **done** and `epic-2-retrospective` is toggled to match team ritual.
- Optionally update `sprint-status.yaml` `epic-2-retrospective` only when the team agrees this checkpoint satisfies their “retro done” policy.

---

_End of Epic 2 checkpoint retrospective._
