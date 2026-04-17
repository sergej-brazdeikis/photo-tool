# E2E usability and look-and-feel rubric — photo-tool

**Purpose:** Repeatable human evaluation to complement automated tests (CLI black-box, Fyne journeys, `internal/share` HTTP suite). Subjective aesthetics and full visual matrices stay here; automation covers regressions on copy, layout gates, and contracts.

**LLM judge bundle:** From the repo root, run [`scripts/assemble-judge-bundle.sh`](../../scripts/assemble-judge-bundle.sh) to copy this file into a timestamped directory with test logs and `manifest.json`, then point Cursor CLI (or any agent) at that folder using [`judge-prompt-v1.md`](judge-prompt-v1.md). Treat output as advisory; never commit secrets into bundles.

**Related automation:**

| Area | Where |
|------|--------|
| CLI scan/import | [`tests/e2e/cli_test.go`](../../tests/e2e/cli_test.go) |
| Shell nav + cross-tab | [`internal/app/e2e_shell_journeys_test.go`](../../internal/app/e2e_shell_journeys_test.go) |
| Upload initial copy + FR-06 confirm/cancel (seeded paths) | [`internal/app/upload_test.go`](../../internal/app/upload_test.go) (`TestUX_upload_*`), [`internal/app/upload_fr06_flow_test.go`](../../internal/app/upload_fr06_flow_test.go) (`TestUpload_flow_*`) |
| Review / Rejected / Collections | [`internal/app/review_test.go`](../../internal/app/review_test.go), [`rejected_test.go`](../../internal/app/rejected_test.go), [`collections_test.go`](../../internal/app/collections_test.go) |
| Layout / DPI matrix | [`internal/app/nfr01_layout_gate_test.go`](../../internal/app/nfr01_layout_gate_test.go), CI matrix in [`.github/workflows/go.yml`](../../.github/workflows/go.yml) |
| Review bulk actions at NFR-01 | Extend **`nfr01_layout_gate_test`** (or a headless widget test) to assert **Share** and **Delete selected…** bulk controls do not share overlapping layout bounds and primary labels are not truncated at **1024×768**—**`findButtonByText` presence** alone may not match painted overlap seen in screenshots |
| Loupe share-preview (image-first vs metadata) | Prefer a small **builder/layout invariant** test on the mint dialog (e.g. preview band min size vs capped metadata scroll) so regressions fail before `TestUXJourneyCapture`; not yet a substitute for vision on real pixels |
| Journey PNG smoke (flat preview plates) | Optional: after `TestUXJourneyCapture` writes `19_*`–`20_*`, `23_*`, `24_*`, assert the saved PNG’s central region is not a near-uniform slab (cheap variance threshold) to catch async-decode races before the LLM judge runs |
| Share HTTP | [`internal/share/http_test.go`](../../internal/share/http_test.go) |

**LLM judge gap (closed loop):** [`TestUXJourneyCapture`](../../internal/app/ux_journey_capture_test.go) screenshots **1280×800** for most steps, then **1024×768** frames: **`rejected_nfr01_min_window`** and **`upload_empty_nfr01_min_window`** mid-journey, plus **`review_grid_nfr01_min_window`**, **`review_loupe_nfr01_min_window`**, **`review_loupe_share_preview_nfr01_min_window`**, **`collections_album_detail_nfr01_min_window`**, and **`collections_album_list_nfr01_min_window`** (after **Back** from detail) appended after the FR-06 phase so filenames **01–21** stay stable. The vision judge **must** treat every `*_nfr01_min_window*` step as contractual for “no clipped primary chrome” at NFR-01 minimum. What remains **manual**: **theme toggling** mid-session (stale `canvas.Rectangle` fill vs refreshed widgets), OS-specific DPI, and native dialogs.

**Image dominance (LLM + human):** The UX spec subsection **“Normative criteria: image dominance (all primary flows)”** is the cross-cutting bar: **photographic pixels** should **read larger** than any **single** non-image chrome block on primary flows. The vision judge ([`judge-prompt-v2-screenshots.md`](judge-prompt-v2-screenshots.md)) enforces this on **every** bundle PNG, not only min-window shots—**1024×768** frames are **necessary** for clip/density but **not sufficient** (wide captures can still fail image-first if chrome dominates).

---

## Severity scale

| Level | Meaning |
|-------|---------|
| **Blocker** | Wrong data, data loss risk, crash, or security issue |
| **Major** | Task failure, misleading copy, broken control, severe contrast/readability |
| **Minor** | Friction, inconsistent wording, small layout oddity with workaround |
| **Cosmetic** | Polish only; no impact on task success |

---

## Nielsen heuristics (quick pass)

Rate each **screen or flow** (Upload, Review, Collections, Rejected, loupe, share in browser). Mark **OK / Issue** and note severity.

1. **Visibility of system status** — Receipt counts, matching counts, errors explain what happened and what to do next.  
2. **Match between system and real world** — Familiar terms (albums, ratings, tags); no unexplained jargon.  
3. **User control and freedom** — Cancel on collection step, reset filters, back navigation, undo reject where promised.  
4. **Consistency and standards** — Primary nav order, filter strip labels, button importance (e.g. destructive actions).  
5. **Error prevention** — Dry-run scan/import; confirm destructive actions.  
6. **Recognition rather than recall** — Collection/tag options visible; share URLs copyable.  
7. **Flexibility and efficiency** — Keyboard/modifiers for multi-select where documented; bulk actions.  
8. **Aesthetic and minimalist design** — No redundant chrome; empty states guide next step. **Photo-tool:** on Upload / Review / Collections / Rejected / loupe / share preview, treat **image dominance** (UX spec **Normative criteria: image dominance**) as in scope here: non-image chrome should not **visually outweigh** the aggregate photo region.  
9. **Help users recognize, diagnose, recover from errors** — Honest library/DB messages; next-step hints.  
10. **Help and documentation** — In-app copy sufficient for primary tasks (manual README for power users is optional).

---

## Task-based scenarios (5–10 minutes each)

Complete each task **without** reading source. Record pass/fail, time, and issues.

| ID | Task | Pass criteria |
|----|------|----------------|
| T1 | Add **10** images via picker + Import; leave collection as “Skip”; confirm receipt matches expectations | Receipt shows added/skipped/failed; library contains expected files |
| T2 | Same batch with **Assign to collection**, rename album, **Confirm** | New album exists; assets linked; no orphan collection on all-fail (compare to automated FR-06 cases) |
| T3 | Open **Review**; filter by collection and minimum rating; open loupe; rate and tag one asset | Filters and grid stay consistent; no stale counts |
| T4 | **Reject** one photo from Review; open **Rejected**; confirm it appears; use **Back to Review** | Navigation and counts coherent |
| T5 | **Collections**: create/open album; empty album shows **Back to albums** and **Go to Review** | CTAs work as labeled |
| T6 | **Share**: mint link for one asset (or package if implemented); open in browser | Page loads; image/HTML matches expectation; token errors handled |
| T7 | **CLI**: `phototool scan --dir …` and `import --dir …` against a temp library | Matches GUI semantics for a small folder (cross-check with README) |

---

## Look-and-feel matrix (manual)

Run on **at least** one macOS and one Windows row when changing UI chrome. CI runs DPI/scale variants for layout gates; this matrix adds subjective judgment.

| Theme | Window size | Platform / scale | Notes |
|-------|-------------|------------------|--------|
| Light | Default (e.g. 1120×600) | macOS | Nav readable; scrollbars acceptable |
| Dark | Default | macOS | Contrast on drop zone + lists |
| Light | Minimum reasonable | Windows | No clipped primary nav (see NFR-01 cells) |
| Dark | Minimum reasonable | Windows | |
| Light | FYNE_SCALE 1.25 / 1.5 (macOS CI tiers) | macOS | Optional spot-check after layout changes |
| Windows | LogPixels 120 / 144 (CI) | Windows | Touch targets and filter strip |

---

## Sign-off (release / milestone)

| Date | Version / branch | Evaluator | Automated suite green? | Manual tasks T1–T7 | Matrix complete? | Blockers |
|------|------------------|-----------|----------------------|--------------------|--------------------|----------|
| | | | ☐ | ☐ | ☐ | |

**Sign-off:** Name: _______________  Date: _______________

---

## When to re-run

- Primary navigation, upload/receipt, review filters, or share HTTP contract changes  
- New empty states or error strings  
- Fyne/theme upgrades  
- Before external demo or release candidate
