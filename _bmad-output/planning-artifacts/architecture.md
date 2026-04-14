---
workflowType: architecture
workflowCompleted: '2026-04-12'
stepsCompleted:
  - 1
  - 2
  - 3
  - 4
  - 5
  - 6
  - 7
  - 8
inputDocuments:
  - _bmad-output/planning-artifacts/PRD.md
  - _bmad-output/planning-artifacts/ux-design-specification.md
  - docs/input/initial-idea.md
project_name: photo-tool
author: Sergej Brazdeikis
stack:
  go: '1.25.4'
  fyne: v2.7.3
---

# Solution architecture — photo-tool

This document records **solution design decisions** so implementation (Fyne desktop, CLI, share web) stays consistent. It aligns with the PRD and UX specification dated **2026-04-12**.

---

## 1. Project context analysis

### 1.1 Requirements overview (architectural lens)

**Functional clusters (PRD):**

| Cluster | FRs | Architectural need |
|--------|-----|-------------------|
| Import / storage | FR-01–FR-06 | Canonical path layout, hashing, dedup, transactional DB + filesystem writes, receipts |
| Review / share | FR-07–FR-14, FR-29–FR-32 | Filter/query model, loupe layout, in-app share minting, snapshot share records |
| Reject / delete | FR-29–FR-31 | Flags vs tombstones, undo semantics, delete persistence, query defaults |
| Collections | FR-15–FR-25 | Relational model, full-page navigation state, grouping queries |
| Metadata | FR-26 | EXIF/TIFF pipeline, optional sidecar or DB columns |
| Scan / import CLI | FR-27–FR-28 | Same domain services as GUI, streaming walks, dry-run |
| Growth packages | FR-33 | Multi-asset manifest tables/API (deferred schema detail until epic) |

**Non-functional drivers:**

- **NFR-01 / NFR-07:** Layout and fractional OS scaling → Fyne container layout contracts, manual QA matrix (no new backend risk).
- **NFR-02:** Large batch scan/import → streaming walk, bounded memory, chunked DB commits.
- **NFR-03:** Deterministic dedup → single hash algorithm, single code path for upload/scan/import.
- **NFR-04 / CLI parity:** Shared **operation result** type and stable **category names** for GUI + CLI summaries (including reject when applicable).
- **NFR-05:** Share page cold load → minimal HTML/CSS, sized renditions, avoid WASM for MVP default path.
- **NFR-06:** Unguessable tokens, basic abuse posture (rate limit, localhost default bind).

**UX-driven constraints:**

- Preview-before-mint, snapshot shares, Rejected/Hidden surface, dual Fyne themes (dark default + light peer), safe chrome / ultrawide (feeds NFR-01).
- Share page: WCAG 2.1 Level A, no raw GPS on web, semantic HTML favored.

### 1.2 Scale and complexity

- **Domain:** Consumer media, local-first DAM, moderate complexity.
- **Tier-1 platforms:** macOS, Windows desktop; Linux tier-2.
- **Multi-tenancy / accounts:** Out of scope MVP (single operator library).
- **Cross-cutting:** One ingestion pipeline, one library DB, shared services for GUI and CLI.

---

## 2. Starter and foundation

There is **no third-party app generator**. The foundation is:

- **Language:** Go **1.25.4** (`go.mod`).
- **Desktop UI:** [Fyne](https://fyne.io/) **v2.7.3** (`fyne.io/fyne/v2`).
- **Entrypoint today:** repository root `main.go` (minimal shell).

**Direction for implementation:** evolve toward a **single binary** whose default mode launches the Fyne UI, with **CLI subcommands** (e.g. scan/import) reusing the same internal packages—see §6.

---

## 3. Core architectural decisions

### 3.1 Decision priority

**Critical (block consistent implementation):**

1. Library storage layout + hash algorithm (dedup).
2. Local database engine and migration strategy.
3. Asset reject / delete persistence and default query semantics.
4. Share token model and MVP web delivery stack.
5. Reject **undo** rule (product-visible).

**Important:**

6. EXIF extraction approach (pure Go vs external tool).
7. HTTP server placement and bind policy.
8. Thumbnail / grid loading strategy (bounded memory).

**Deferred / Growth:**

9. Sharable **packages** (FR-33): extend `share_links` (or sibling table) with manifest JSON and multi-asset resolution; same preview-then-mint flow.
10. **Fyne WASM** share UI: optional spike after MVP HTML path meets NFR-05 and WCAG checks.

### 3.2 Library storage and deduplication

- **Library root:** Configurable absolute path (user setting + sensible default per OS). All managed files live under this root unless explicitly “register in place” import rules say otherwise (import tool semantics per PRD FR-28).
- **Canonical path:** `{libraryRoot}/{YYYY}/{MM}/{DD}/` using **capture datetime** from metadata when present; fallback order **documented in code** (e.g. EXIF → TIFF → embedded → file mtime) and aligned with PRD **Provenance**.
- **Filename:** `{capture-time-local-or-utc-documented}_{content-hash-short}.{ext}` — full hash stored in DB; PRD FR-02 requires traceable uniqueness; use **SHA-256** hex (or base32) with a **fixed-length prefix** in the filename to avoid huge names; store **full hash** in DB.
- **Dedup (FR-03, NFR-03):** **Size + SHA-256** of file bytes. Same bytes → same outcome in upload, scan, and import paths via one shared function in `internal/ingest` (or `internal/library`).

### 3.3 Data architecture

- **Engine:** **SQLite** for local single-user library.
- **Driver:** Prefer **`modernc.org/sqlite`** (pure Go, simpler cross-platform builds) unless profiling forces CGO `mattn/go-sqlite3`.
- **Migrations:** SQL files under `internal/store/migrations/` with numeric prefixes; applied at startup via a small migrator (e.g. **golang-migrate** or minimal custom runner). Schema changes are **forward-only**; document rollback policy as “restore backup” for MVP.
- **Core entities (logical):**
  - **assets:** id, content_hash, file_path relative to library root, capture_time, width/height, mime, **rejected** bool, **rejected_at**, **deleted_at** nullable, raw metadata fields as needed for FR-26.
  - **collections:** id, name, display_date, created_at.
  - **asset_collections:** many-to-many.
  - **tags** / **asset_tags** if not folded into a single JSON column initially—prefer normalized tables if filtering FR-15 requires it.
  - **share_links:** id, **token_hash** (never store raw token), asset_id (MVP single-asset), created_at, optional **payload** JSON for snapshot metadata (rating at mint, rendition paths); Growth adds package manifest reference.

**Indexes:** content_hash, capture_time, rejected, collection membership for filter queries.

### 3.4 Reject, undo, and delete

**Reject (FR-29):**

- Persist **`rejected = true`** on asset; default **all** browse/filter/collection/share selection queries use **`rejected = false`**.
- Dedicated **Rejected/Hidden** view queries **`rejected = true`**.
- Share mint path **rejects** (fails validation) if asset is rejected unless explicitly overridden by a future product decision (default: **cannot share rejected**).

**Undo (FR-30, UX):**

- **MVP rule (locked for implementation):** **(A)** **Session undo stack:** last reject(s) in the **current review context** can be undone until the user **navigates away** from that context (e.g. leaves bulk review for another primary area) or **closes the app**; **(B)** **Rejected/Hidden** always allows **restore** (clear reject flag). Optional **30s** toast undo can be added as UX polish but is not required for parity with (A)+(B).
- Implementation sketch: in-memory LIFO of `asset_id` scoped to a `review_session_id` or view route; **Restore** uses DB update.

**Delete (FR-31):**

- **MVP behavior (locked):** **Soft-delete + quarantine files:** set **`deleted_at`**, remove asset from default queries, **move** underlying file(s) to **`{libraryRoot}/.trash/{asset_id}/`** (or hash-prefixed folder) so recovery is possible without defining OS trash integration. **Hard erase** (secure delete) is **out of MVP** unless explicitly added later.
- UI: distinct from Reject; **confirmation** required before delete pipeline runs.

Document in user-facing copy that delete is **stronger** than reject and that **purge trash** may be a later setting.

### 3.5 Share service (MVP)

- **Transport:** **HTTP** served from the **desktop process** (in-process `net/http` server) on **loopback** by default (`127.0.0.1:{port}`), configurable port; **LAN exposure** only with explicit opt-in (document security warning).
- **Token:** Generate **32 random bytes**, URL-safe encoding; store **SHA-256(token)** in DB; compare constant-time on lookup.
- **Routes (MVP):** e.g. `GET /s/{token}` → resolve hash → load snapshot row → render HTML. Optional `GET /assets/...` for image bytes with same session cookie or signed query (prefer **opaque token-bound URLs** to avoid guessable paths).
- **Semantics:** **Snapshot** at mint (PRD FR-32 default): persist **asset id** + **rendition path or content hash** + **rating at mint** if shown on web per FR-14.
- **Privacy:** Strip **raw GPS** and restricted fields in **share template**; desktop may show full EXIF.
- **NFR-05:** Pre-render or cache **web-sized** JPEG/WEBP; avoid on-the-fly full RAW decode on share path. **Measurement:** `docs/share-cold-load-nfr05.md` and `internal/share/nfr05_cold_load_test.go` (`TestNFR05_ShareColdLoadMedian`).
- **NFR-06:** Loopback bind + token entropy + simple **per-IP rate limit** (in-memory) on share routes with a **bounded visitor map** (eviction when distinct IP keys exceed cap); document need for reverse-proxy limits if later deployed remotely. **Posture:** `docs/share-abuse-posture.md` (canonical); defaults in `internal/share/ratelimit.go`.

### 3.6 MVP web stack vs Fyne WASM

- **MVP default:** **`html/template`** + **small static CSS** under `web/share/` (or `internal/share/templates/`), served by `net/http`. Supports WCAG tooling (axe), small cold load, clear separation from Fyne.
- **Fyne WASM:** **Spike after MVP** if code reuse outweighs bundle size and a11y validation; UX spec allows either—this architecture **defaults to HTML** until metrics say otherwise.

### 3.7 EXIF and metadata

- **Primary approach:** Pure Go **dsoprea/go-exif** with format-specific structure packages (JPEG/PNG/TIFF) for breadth without bundling ExifTool.
- **Fallback (optional later):** Document **ExifTool** wrapper (e.g. barasher/go-exiftool) behind build tag or config if RAW/TIFF edge cases exceed Go parser coverage—out of MVP unless a story requires it.

### 3.8 Desktop UI architecture (Fyne)

- **Themes:** Custom `fyne.Theme` implementing UX **semantic roles** (dark default + light peer, distinct primary / destructive / reject-caution).
- **Structure:** Compose standard widgets; **custom layout** only for **loupe** and **grid cell** as needed (UX component strategy).
- **State:** Prefer **explicit view models** (small structs per screen) fed by **repository interfaces** in `internal/store`—keeps GUI testable and mirrors CLI usage.
- **Grid performance:** **Paged queries** from DB + **thumbnail cache** on disk under `{libraryRoot}/.cache/thumbnails/`; avoid loading full pixmap sets for entire library.

### 3.9 CLI / GUI parity (NFR-04)

- Define a single **`OperationSummary`** (names stable): `added`, `skipped_duplicate`, `updated`, `failed`, and when applicable **`rejected`** count for operations that touch reject semantics.
- GUI shows receipts; CLI prints the **same fields** (machine-friendly optional `--json` later).

### 3.10 Authentication and security (MVP)

- **No auth** for local app; share relies on **secret URL** + loopback default.
- **Secrets:** No API keys in MVP share path; if cloud hosting added later, redesign threat model.

### 3.11 Infrastructure

- **Distribution:** GoReleaser or manual builds per OS—not blocking architecture; CI can run `go test ./...` and staticcheck later.

### 3.12 Decision impact / implementation order

1. Config + library root + path/hash helpers.
2. SQLite schema + migrations + repositories.
3. Ingest pipeline (upload / scan / import) writing DB + files + **OperationSummary**.
4. Fyne shell + themes + navigation placeholders.
5. Review queries (filters FR-15–FR-16) + reject/restore/delete.
6. Share mint + HTTP server + HTML template.
7. Hardening: NFR-02 streaming, NFR-01 layout QA, share a11y checks.

---

## 4. Implementation patterns and consistency rules

### 4.1 Naming (Go)

- **Exported types:** `CamelCase` (e.g. `Asset`, `Collection`).
- **JSON (if used):** `snake_case` field tags for CLI `--json` stability and share API consistency.
- **DB columns:** `snake_case`.
- **Package names:** short, single lowercase word (`store`, `ingest`, `share`).

### 4.2 Errors

- Use **`fmt.Errorf` with `%w`**; domain errors as **`errors.New` / sentinel vars** in `internal/domain` where needed.
- User-visible strings: separate from error wrapping (Fyne dialogs vs logs).

### 4.3 Logging

- **`log/slog`** default structured logger; levels INFO/WARN/ERROR; debug behind `-verbose` or env.

### 4.4 Tests

- **Table-driven** unit tests for ingest, dedup, paths, share token validation.
- **Integration tests:** temp dir + SQLite file for store and ingest.

### 4.5 Agent MUST rules

- **One** dedup implementation used by upload, scan, import.
- **One** `OperationSummary` schema for GUI and CLI.
- **Never** store raw share tokens in DB (hash only).
- **Default queries** exclude `rejected` and `deleted` unless view explicitly requests them.
- **Share HTML** must not emit raw GPS (PRD domain requirements).

---

## 5. Project structure and boundaries

### 5.1 Target directory layout

```text
photo-tool/
├── go.mod
├── go.sum
├── main.go                    # interim: launches GUI; migrate to cmd/ when subcommands land
├── cmd/
│   └── phototool/             # optional: future canonical main (GUI + cobra root)
├── internal/
│   ├── app/                   # Fyne application, navigation, theme wiring
│   ├── domain/                # Asset, Collection, filters, invariants
│   ├── ingest/                # upload, scan, import; dedup; receipts
│   ├── exifmeta/              # EXIF extraction facade
│   ├── store/                 # sqlite repo, migrations
│   ├── share/                 # http server, handlers, template binding
│   └── cli/                   # cobra commands calling ingest/store
├── web/
│   └── share/                 # HTML templates, static CSS for share pages
├── assets/                    # optional: embedded icons for Fyne
└── _bmad-output/planning-artifacts/
```

### 5.2 Boundaries

| Boundary | Rule |
|----------|------|
| **domain → store** | domain types only; no SQL in Fyne packages |
| **ingest → store, exifmeta, filesystem** | orchestration; no Fyne import |
| **share → store** | read-only share resolution + template data DTOs |
| **app (Fyne) → domain, store, ingest, share** | UI triggers use cases; no SQL in widgets |

### 5.3 Requirements → location (summary)

- FR-01–FR-06, FR-27–FR-28 → `internal/ingest`, `internal/exifmeta`, `internal/store`
- FR-07–FR-14, FR-29–FR-32, FR-15–FR-25 → `internal/app`, `internal/domain`, `internal/store`
- FR-26 → `internal/exifmeta` + persistence in `internal/store`
- FR-33 (Growth) → `internal/share`, schema migration
- NFR-05/06 → `internal/share`, `web/share`

---

## 6. CLI and binary layout

- **Target:** Single module binary **`phototool`** (name TBD; can remain `photo-tool` executable from module root).
- **CLI:** [cobra](https://github.com/spf13/cobra) root: default **no subcommand** → start Fyne UI; **`scan`**, **`import`** subcommands call `internal/ingest` with identical summaries as GUI.
- **Migration path:** Move `main.go` to `cmd/phototool/main.go` when cobra is introduced; keep root `main.go` as thin wrapper only if needed for `go run .` during transition.

---

## 7. Validation (architecture readiness)

### 7.1 Coherence

- SQLite + single library root + content-addressed dedup align with PRD storage and NFR-03.
- Reject/delete/share rules are consistent with FR-29–FR-32 and UX preview-before-mint.
- HTML share path aligns with NFR-05 and WCAG tooling; WASM explicitly deferred.

### 7.2 Requirements coverage (spot check)

- **FR-01–FR-06:** Path layout, hash naming, dedup, collection confirm gate → ingest + DB.
- **FR-07–FR-14, FR-29–FR-32:** Filters, review, share mint → app + store + share.
- **FR-15–FR-25:** Collections and grouping → SQL + app views.
- **FR-27–FR-28:** CLI + dry-run → ingest + cobra.
- **NFR-01/07:** Fyne layout + QA process (architecture does not replace UX matrix).
- **NFR-02:** Streaming ingest design explicit.
- **NFR-04:** OperationSummary contract explicit.

### 7.3 Open follow-ups (non-blocking)

- Exact **filename** pattern (UTC vs local, separator characters) — document in `internal/ingest` README when implemented.
- **RAW** display list and decoder dependencies — architecture/test artifact per PRD.
- **Revocation** of share links — out of MVP; add if product requires.

---

## 8. Traceability

| PRD / UX theme | Architecture section |
|----------------|----------------------|
| Storage / dedup | §3.2, §3.3 |
| Reject / delete / undo | §3.4 |
| Share snapshot / privacy | §3.5, §3.6 |
| CLI parity | §3.9, §4.2, §6 |
| NFR-02 / NFR-03 | §3.2, §3.3, §3.8 |
| NFR-05 / NFR-06 | §3.5, §3.6 |
| UX themes / dual theme | §3.8 |
| Growth packages FR-33 | §3.1, §3.3, §5.3 |

---

_End of solution architecture document._
