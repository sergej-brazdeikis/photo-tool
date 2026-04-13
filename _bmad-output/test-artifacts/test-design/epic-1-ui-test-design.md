---
stepsCompleted:
  - step-01-detect-mode
  - epic-1-ui-synthesized
lastStep: epic-1-ui-synthesized
lastSaved: '2026-04-13'
mode: epic-level
scope: Epic 1 — UI-facing behavior (primarily Story 1.5; dependencies 1.1–1.4)
---

# Test Design: Epic 1 — UI (ingest + collection confirm)

**Date:** 2026-04-13  
**Author:** Sergej Brazdeikis (Murat / Test Architect)  
**Status:** Draft  

---

## Executive summary

**Scope:** Epic-level test design for **desktop Fyne upload flow** and its contract to **ingest**, **store**, and **OperationSummary** (NFR-04). CLI stories (1.6–1.7) and drag-drop (1.8) are **out of scope** for this document unless noted.

**Risk summary**

| Tier | Count | Notes |
|------|-------|--------|
| Score ≥ 6 | 3 | Wrong persistence on confirm; misleading UX; receipt vs DB drift |
| Score 3–5 | 4 | CI flakiness for Fyne; env library path; dedup semantics; performance of batch |
| Score 1–2 | 2 | Cosmetic copy; theme |

**Coverage summary (indicative effort)**

| Priority | Automated scenarios (target) | Manual / exploratory | Est. build + stabilize |
|----------|------------------------------|------------------------|-------------------------|
| P0 | 8–12 (unit + integration + thin Fyne) | 6 journeys | ~12–18 h |
| P1 | 10–15 | 4 | ~10–14 h |
| P2/P3 | Remaining edge cases | 2 | ~4–6 h |

*Flake budget:* treat any Fyne-driver test above **0.5%** flake on CI as **P0 debt** — prefer unit/integration over widget E2E.

---

## Not in scope

| Item | Reason | Mitigation |
|------|--------|------------|
| Pixel-perfect layout / NFR-01 matrix | Native Fyne; expensive automation | Manual matrix per milestone; screenshot spot-check |
| Scan/import CLI (1.6–1.7) | Not UI | Separate test design or shared **OperationSummary** contract tests only |
| Web share (Epic 3) | Different stack | Playwright when implemented |
| Real HEIC/DNG decode everywhere | Codec / OS variance | Smoke subset + documented “supported sample set” |

---

## Risk assessment

### High priority (score ≥ 6)

| ID | Cat | Description | P | I | Σ | Mitigation | Owner |
|----|-----|-------------|---|---|---|------------|-------|
| R-UI-01 | DATA | **Confirm** persists collection/links when it should not, or **orphan collection** (assign + name but zero ingest success) | 2 | 3 | 6 | Unit tests on guards + `summarizeDoneMessage`; optional DB assert after `test.Tap` | Dev |
| R-UI-02 | BUS | User believes assets are **linked** when they are not (copy / receipt mismatch) | 2 | 3 | 6 | Assert DB `asset_collections` after happy path; test message branches | Dev |
| R-UI-03 | DATA | **Receipt counts** disagree with DB / disk (NFR-04) | 2 | 3 | 6 | Golden-path integration: ingest temp files → query `assets` + compare to `OperationSummary` | Dev |

### Medium (3–5)

| ID | Cat | Description | Σ | Mitigation |
|----|-----|-------------|---|------------|
| R-UI-04 | TECH | Fyne **headless / CI** instability (Linux/Xvfb) | 4 | Tag `fyne` tests; run on main/nightly first; inject fakes for dialog/file |
| R-UI-05 | OPS | Wrong **`PHOTO_TOOL_LIBRARY`** → silent wrong library | 4 | Integration test with env + temp dir; startup smoke |
| R-UI-06 | TECH | **File picker** not automatable in CI | 4 | Inject `addAbsolute` / fake path list for tests |
| R-UI-07 | PERF | Large batch import **UI freeze** (no progress) | 3 | Manual 100+ file smoke; future: async + progress (out of epic scope) |

### Low (1–2)

| R-UI-08 | BUS | Default **Upload YYYYMMDD** wrong timezone edge at midnight | 2 | Unit test `defaultUploadCollectionName` boundary |
| R-UI-09 | OPS | **Cancel** copy confusing | 2 | Copy review + one widget test |

---

## Requirements → coverage (UI slice)

| Ref | Behavior | P0 test level | Artifact |
|-----|----------|---------------|----------|
| FR-06 | No collection/links until **Confirm** | Integration + Fyne tap | DB query zero rows before confirm |
| FR-04–FR-05 | Default name, rename, confirm links batch | Unit (name) + integration | `CreateCollection` + `LinkAssetsToCollection` |
| FR-03 / NFR-04 | Receipt: added / duplicate / failed (+ updated) | Unit (`summarize…`) + ingest integration | Match `domain.OperationSummary` |
| UX-DR6 | Receipt visible after import | Manual + optional Fyne assert label text | Screenshot checklist |
| 1.1 | Library root / DB open | Integration (existing store tests) + smoke `main` | `internal/store`, `config` |

---

## Test levels (recommended pyramid)

1. **Unit (largest):** pure functions — receipt strings, name trimming, “link only if `len(assetIDs)>0`”, `OperationSummary` JSON tags. **Fast, CI-safe.**

2. **Integration (medium):** `sql.Open` sqlite file or memory + `ingest` + `store` — **no Fyne**. Proves FR-01–FR-03 and collection APIs. Already strong in `internal/ingest`, `internal/store`.

3. **Fyne widget (small):** `fyne.io/fyne/v2/test` — build `NewUploadView` with **injected** path list + fake notifier if you refactor; use `test.Tap` on Confirm/Cancel. **Cap at 3–5 scenarios** to control flake.

4. **Manual (mandatory for Epic 1 UI):** multi-file accumulation, duplicate file twice, cancel path, rename collection, failure-only batch (bad paths). **Document in story Dev Agent Record** (you already reference this pattern).

---

## P0 scenarios (run every commit / PR)

| ID | Scenario | Level | Risk |
|----|----------|-------|------|
| P0-1 | Import 2 new files → receipt **Added=2**, DB rows=2 | Integration | R-UI-03 |
| P0-2 | Import same file twice → **Skipped duplicate**, disk single copy | Integration | R-UI-03 |
| P0-3 | Assign + name + **all ingest failures** → **no** `collections` row | Unit + integration | R-UI-01 |
| P0-4 | Skip collection + Confirm → no new collection | Integration / widget | R-UI-01 |
| P0-5 | Assign + confirm with 1+ success → collection exists + links | Integration | R-UI-02 |
| P0-6 | `summarizeDoneMessage` branches (skip / linked / wanted-but-none) | Unit | R-UI-02 |
| P0-7 | Cancel after import → no collection writes (if cancel only skips collection step) | Widget or integration | FR-06 |
| P0-8 | `go test ./...` green on Linux + macOS matrix (CI) | Gate | R-UI-04 |

---

## P1 scenarios (PR to main)

| ID | Scenario | Level |
|----|----------|-------|
| P1-1 | Rename collection field → persisted name matches trimmed input | Integration |
| P1-2 | Default `Upload YYYYMMDD` matches local calendar rule | Unit |
| P1-3 | `LinkAssetsToCollection` idempotent (double confirm blocked by UX, but API safe) | Store test (exists) |
| P1-4 | Library env override path used | Integration / small test |
| P1-5 | Receipt hides **Updated** when zero | Widget or unit binding |

---

## Entry / exit criteria

**Entry**

- [ ] Story 1.5 AC agreed; ingest + store APIs stable enough to freeze contracts.
- [ ] Temp library + sqlite reproducible in tests (pattern: `t.TempDir()`).

**Exit**

- [ ] All **P0** automated scenarios pass on CI (or documented skip for Fyne with issue link).
- [ ] **Manual checklist** executed once per release candidate for Epic 1 UI.
- [ ] No open **R-UI-01..03** defects without waiver.

---

## CI strategy (Murat’s call)

- **Default pipeline:** `go test ./...`, `go vet`, optional `staticcheck` — **no** Fyne window on PR if unstable.
- **Tier 2 (nightly / main):** Fyne tests with **software driver** or **Xvfb** on Linux; macOS runner optional.
- **Do not** block merge on native file-picker automation — inject paths.

---

## Handoff — next skills

| Next | When |
|------|------|
| **TA** (`bmad-testarch-automate`) | Generate concrete test cases + file stubs from P0/P1 tables |
| **RV** (`bmad-testarch-test-review`) | After first PR of Fyne tests |
| **CI** (`bmad-testarch-ci`) | When you split fast vs nightly jobs |

---

*Murat — Epic 1 UI test design is risk-first: most proof lives under **unit + integration**; Fyne is a thin, capped layer; manual fills the honesty gap for perception and layout.*
