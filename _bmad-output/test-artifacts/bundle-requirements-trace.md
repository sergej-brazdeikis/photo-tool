# Bundle requirements trace (distillate)

**Purpose:** Shipped in every judge bundle as `context/requirements-trace.md` ([`scripts/assemble-judge-bundle.sh`](../../scripts/assemble-judge-bundle.sh)). Judges and the QA loop agent use it to tie **FRs / stories** to **automated tests**, **UX journey captures** (`ui/steps.json`), and **residual risk**.

**Maintenance:** When epics/stories change, capture steps are added, or major tests land — update this file so it stays aligned with:

- [`_bmad-output/planning-artifacts/epics-v2-ux-aligned-2026-04-14.md`](../planning-artifacts/epics-v2-ux-aligned-2026-04-14.md) (canonical v2 rollup)
- [`_bmad-output/planning-artifacts/epics.md`](../planning-artifacts/epics.md) (sibling rollup)
- [`internal/app/ux_journey_capture_test.go`](../../internal/app/ux_journey_capture_test.go) (`TestUXJourneyCapture` step `id`s)

**Deferred / post-review gaps** are **not** auto-copied into the bundle. See repo [`_bmad-output/implementation-artifacts/deferred-work.md`](../implementation-artifacts/deferred-work.md); judges and QA may cite it under **Gaps** when a story row is **OK** in the matrix but product risk remains.

---

## FR → Epic (compact)

| FR range | Epic | Notes |
|----------|------|--------|
| FR-01–FR-06 | Epic 1 | Ingest, upload, collection confirm |
| FR-07–FR-12, FR-15–FR-25, FR-29–FR-31 | Epic 2 | Review, filters, collections, reject/delete |
| FR-13, FR-14, FR-32 | Epic 3 | Share mint + web viewer |
| FR-26 | Epic 1 + Epic 2 | EXIF placement + UI metadata breadth |
| FR-27–FR-28 | Epic 1 | CLI scan/import |
| FR-33 | Epic 4 | Shareable packages (growth) |
| NFR-01, NFR-07 | Epic 2 | Layout + OS scaling |
| NFR-02 | Epic 1 | Large-tree scan/import |
| NFR-03 | Epic 1 | Dedup integrity |
| NFR-04 | Epic 1 + Epic 2 | OperationSummary / receipts |
| NFR-05, NFR-06 | Epic 3 | Share perf + abuse posture |

---

## Story coverage matrix (loop visibility)

**Columns**

- **Story** — Id as in epics / implementation artifact.
- **FR / UX (short)** — Primary requirement tags (not exhaustive).
- **Artifact** — Implementation story doc under `_bmad-output/implementation-artifacts/`.
- **Automated evidence** — Main test / package anchors (representative, not full file lists).
- **Journey capture** — `ui/steps.json` `id` values from `TestUXJourneyCapture` when the flow is exercised; `—` if not in harness.
- **Loop risk** — **OK** = captures + tests credibly guard; **PARTIAL** = one side weak; **GAP** = known missing or manual-only (see also `deferred-work.md`).

**Bundle log gate (Story 1.5 / FR-06):** Row **1.5** is **OK** in steady state only when the judge bundle’s **primary** full-module logs (typically `logs/go-test.txt` and, if present, `logs/go-test-ci.txt`) show **`ok photo-tool/internal/app`** for the upload / FR-06 flow tests referenced in that row. If those logs are **red** while journey captures still list `upload_*` steps, treat **1.5** as **not loop-closed** for that iteration—the matrix **OK** is **distillate intent**, not proof against a contradictory bundled `go-test` artifact (refresh logs or defer in QA).

### Epic 1 — Ingest

| Story | FR / UX (short) | Artifact | Automated evidence | Journey capture | Loop risk |
|-------|-----------------|----------|----------------------|-----------------|-----------|
| 1.1 | Library root, DB | `1-1-library-foundation.md` | `internal/config`, `internal/store` tests | `—` | PARTIAL (no shell PNG) |
| 1.2 | EXIF / hash | `1-2-capture-time-hash.md` | `internal/exifmeta`, `internal/filehash`, `internal/paths` | `—` | PARTIAL |
| 1.3 | Core ingest | `1-3-core-ingest.md` | `internal/ingest`, `internal/store` | `—` | PARTIAL |
| 1.4 | Collections schema | `1-4-collections-schema.md` | `internal/store` | `—` | PARTIAL |
| 1.5 | Upload + FR-06 | `1-5-upload-confirm-receipt.md` | `internal/app/upload_*_test.go`, ingest | `upload_empty`, `upload_paths_staged`, `upload_fr06_collection_assign`, `upload_after_confirm_idle`, `upload_empty_nfr01_min_window` | OK |
| 1.6 | Scan CLI | `1-6-scan-cli.md` | `internal/cli` | `—` | GAP (CLI not in journey PNGs; e2e/cli partial) |
| 1.7 | Import CLI | `1-7-import-cli.md` | `internal/cli` | `—` | GAP |
| 1.8 | Drag-drop | `1-8-drag-drop-upload.md` | `internal/app` upload tests | `upload_*` shows drop zone; **not** geometry proof | PARTIAL (deferred-work: scroll/hit-test) |

### Epic 2 — Review / organize

| Story | FR / UX (short) | Artifact | Automated evidence | Journey capture | Loop risk |
|-------|-----------------|----------|----------------------|-----------------|-----------|
| 2.1 | Shell, nav, themes | `2-1-app-shell-navigation-themes.md` | `internal/app` shell tests, `e2e_shell_journeys_test.go` | All flows (nav) | OK |
| 2.2 | Filter strip FR-15/16 | `2-2-filter-strip.md` | `internal/app/review_*`, NFR tests | `review_grid_*`, `review_filter_*`, `review_filters_fr16_reset`, `*_nfr01_min_window` review | OK |
| 2.3 | Grid badges | `2-3-thumbnail-grid-rating-badges.md` | `internal/app/review_*` | `review_grid_default_filters`, loupe | PARTIAL (pixel/badge asserts limited) |
| 2.4 | Loupe rating keys | `2-4-review-loupe-keyboard-rating.md` | `internal/app` loupe tests | `review_loupe`, `review_loupe_nfr01_min_window` | PARTIAL (keyboard persistence thin in capture) |
| 2.5 | Tags bulk | `2-5-tags-bulk-review.md` | `internal/app/review_test.go` | `review_filter_tag_uxcaptag` | PARTIAL |
| 2.6 | Reject / undo / restore | `2-6-reject-undo-hidden-restore.md` | `internal/app/rejected_*` | `rejected_*` | OK |
| 2.7 | Delete / trash | `2-7-delete-quarantine.md` | `internal/app`, `internal/store` | **Not** in journey omissions | GAP |
| 2.8 | Collections list/detail | `2-8-collections-list-detail.md` | `internal/app/collections_*` | `collections_*`, `collections_album_list_nfr01_min_window` | OK |
| 2.9 | Collection CRUD / multi-assign | `2-9-collection-crud-multi-assign.md` | `internal/app`, `internal/store` | `collections_new_album_form` (dialog only) | PARTIAL |
| 2.10 | Quick collection assign FR-08 | `2-10-quick-collection-assign.md` | `internal/app` | **Not** explicit hover menu in capture | GAP |
| 2.11 | NFR-01 / NFR-07 layout | `2-11-layout-display-scaling-gate.md` | `internal/app/nfr01_*`, `nfr07_*` | `*_nfr01_min_window` across upload/review/collections/rejected | OK |
| 2.12 | Empty states | `2-12-empty-states-error-tone.md` | `internal/app` | Empty/filter steps partially | PARTIAL |

### Epic 3 — Share

| Story | FR / UX (short) | Artifact | Automated evidence | Journey capture | Loop risk |
|-------|-----------------|----------|----------------------|-----------------|-----------|
| 3.1 | Share preview mint | `3-1-share-preview-snapshot-mint.md` | `internal/app/share_*` | `review_loupe_share_preview`, `*_nfr01_min_window` | PARTIAL (no real browser) |
| 3.2 | Loopback HTTP | `3-2-loopback-http-token.md` | `internal/share/http_test.go` | `—` | PARTIAL (HTTP not PNG) |
| 3.3 | Share HTML | `3-3-share-html-readonly.md` | `internal/share/http_test.go` | `—` | PARTIAL |
| 3.4 | Privacy WCAG | `3-4-share-privacy-wcag.md` | `internal/share` | `—` | GAP (WCAG manual / tooling) |
| 3.5 | Perf / abuse NFR-05/06 | `3-5-share-performance-abuse.md` | `internal/share/nfr05_*`, ratelimit tests | `—` | PARTIAL |

### Epic 4 — Packages (growth)

| Story | FR / UX (short) | Artifact | Automated evidence | Journey capture | Loop risk |
|-------|-----------------|----------|----------------------|-----------------|-----------|
| 4.1 | FR-33 packages | `4-1-multi-asset-snapshot-packages.md` | `internal/share`, `internal/app` share flows | **Not** in default journey list | GAP |

### NFR evidence docs (supporting)

| Id | Artifact | Role |
|----|----------|------|
| NFR-01 matrix | `nfr-01-layout-matrix-evidence.md` | Human/layout evidence; complements `*_nfr01_min_window` PNGs |
| NFR-07 | `nfr-07-os-scaling-checklist.md` | OS scaling checklist; not all in CI |

---

## Journey harness omissions (explicit)

Aligned with `steps.json` `omissions` in [`ux_journey_capture_test.go`](../../internal/app/ux_journey_capture_test.go): native file picker, real OS DPI, **CLI scan/import URL open**, **library trash / delete-confirm**, etc. Judges should list these under **Gaps** when verdict touches those FRs.

---

## How agents use this file

1. **Vision judge** — Cross-check **PARTIAL** / **GAP** rows against PNGs and logs; name story IDs in **Gaps** when evidence cannot close the row.
2. **QA loop** — Map **Feature behavior matrix** rows to **Story / FR** here; fail the pass when a **release-critical** row is **GAP** with no deferral sentence.
3. **QA implementer** — Prefer new tests or capture steps that upgrade **GAP** → **PARTIAL** → **OK**.
