---
workflowType: epics-stories
epicsVersion: 2
documentTitle: photo-tool epics (UX-aligned v2)
completedDate: '2026-04-14'
derivedFrom:
  - _bmad-output/planning-artifacts/epics.md
notes: >-
  Canonical epic rollup for UX spec revision 2026-04-14 (image-first, UX-DR16–19).
  Use this file when authoring new story specs; epics.md is the pre-v2 sibling unless merged later.
stepsCompleted:
  - step-01-validate-prerequisites
  - step-02-design-epics
  - step-03-create-stories
  - step-04-final-validation
inputDocuments:
  - _bmad-output/planning-artifacts/PRD.md
  - _bmad-output/planning-artifacts/architecture.md
  - _bmad-output/planning-artifacts/ux-design-specification.md
  - _bmad-output/planning-artifacts/ux-design-directions.html
author: Sergej Brazdeikis
---

# photo-tool — Epic breakdown (v2, UX-aligned 2026-04-14)

This document decomposes the PRD, solution architecture, and UX design specification into user-value epics and implementable stories with acceptance criteria. **Version 2** folds in the **2026-04-14** UX specification revision: **image-first** layout, **UX-DR16–UX-DR19**, tightened filter/grid rules, and extended acceptance criteria on the stories below.

**Inputs used:** [PRD.md](PRD.md), [architecture.md](architecture.md), [ux-design-specification.md](ux-design-specification.md) (full revision **2026-04-14**, **Party mode follow-ups**, **UX consistency patterns**), [ux-design-directions.html](ux-design-directions.html) (wireframes **Direction A** default). No additional documents were excluded.

---

## Backlog alignment (UX spec revision 2026-04-14)

The UX specification now treats **image-first** layout as non-negotiable: the **thumbnail / loupe stage** dominates the window; **filter strip + cell chrome** are the main **control-soup** risks. This epic rollup adds **UX-DR16–UX-DR19**, tightens **UX-DR2** / **UX-DR3** wording, and extends **acceptance criteria** on key stories so Fyne work traces to the same bar.

Canonical detail: [ux-design-specification.md](ux-design-specification.md) (**Party mode follow-ups (2026-04-14)**, **Image stage and density**, **Form patterns** for filter overflow). Layout wireframes: [ux-design-directions.html](ux-design-directions.html).

---

## Overview

Photo Tool is a local-first photo library product (Go + Fyne desktop, minimal HTML share). Epics are ordered so each delivers **standalone user value**; later epics build on earlier ones without requiring future work to function.

---

## Requirements inventory

### Functional requirements

- **FR-01:** Users can upload multiple images in one action; system places each new file under `{Year}/{Month}/{Day}/` using **EXIF capture datetime** when present.
- **FR-02:** System names each stored file using capture time plus a **content hash** (algorithm fixed in architecture) so names remain unique and traceable.
- **FR-03:** System detects duplicates by **file size and content checksum**; retains one copy; reports count of skipped duplicates for the operation.
- **FR-04:** Users can assign all images from an upload batch to one or more collections before or immediately after upload completes.
- **FR-05:** Default collection name for that flow is `Upload YYYYMMDD` (calendar date of upload batch initiation or documented rule); user can clear or rename before confirming.
- **FR-06:** System creates no collection and assigns no links until the user **explicitly confirms**.
- **FR-07:** Users can apply **tags** and **ratings 1–5** in bulk review.
- **FR-08:** Users can assign a collection from a **hover** or equivalent quick action on a thumbnail without opening full view.
- **FR-09:** Users can open an image in a large view using up to **90%** of the available viewport.
- **FR-10:** Users can set rating by clicking **1–5** on keyboard or by clicking stars; change saves **without** extra confirmation.
- **FR-11:** In large review view, **layout adapts** so controls remain visible from **1:1** through **21:9** aspect ratios.
- **FR-12:** Large review view shows the **entire image** (letterboxed as needed) within the 90% region for both portrait and landscape assets.
- **FR-13:** Users can obtain a **shareable URL** that opens the **same photo** in review context in a browser.
- **FR-14:** MVP: valid share URL opens the **same photo** in **read-only** layout (image fitted; **current star rating visible**); browser rating edit out of scope.
- **FR-15:** Filter panel order is **Collection**, then **minimum rating**, then **tags**.
- **FR-16:** Default filter selections are **No assigned collection** and **Any rating**.
- **FR-17:** Users can assign selected photos to a collection from the filter workflow.
- **FR-18:** Users can create, rename/edit, and delete collections (name required; display date optional with default = collection creation date).
- **FR-19:** Users can assign one photo to **multiple collections** from single-photo view and create a new collection there.
- **FR-20:** Deleting a collection removes all photo–collection relations, then deletes the collection record.
- **FR-21:** Collections list navigates to a **dedicated full page** (not a popup) for one collection’s photos.
- **FR-22:** Collection detail sorts photos by **capture time** (EXIF-first).
- **FR-23:** Default grouping is by **star rating** descending; empty rating groups omitted; within group sort by capture time.
- **FR-24:** Users can switch grouping to **by day** or **by camera name**.
- **FR-25:** Single-photo view: up to **90%** viewport, full image, rating via keyboard/stars, prev/next (arrows mid-height, keys, swipe on touch).
- **FR-26:** System extracts and stores/displays listed metadata fields when present (camera, capture datetime, lens, exposure, focal length, GPS, resolution, orientation, flash, metering, white balance).
- **FR-27:** Scan tool: `--dir`, `--recursive`, `--dry-run`; discover images; EXIF (min capture time, camera, lens); dedup; copy non-duplicates to canonical layout; update DB; no writes when dry-run.
- **FR-28:** Import tool: walk configurable uploads path; register missing files / backfill EXIF per rules; `--dry-run` summary only.
- **FR-29:** **Reject** soft-hides from default surfaces and from share/package selection by default; dedicated **Rejected/Hidden** view.
- **FR-30:** **Rejected/Hidden** lists rejected items; **restore** clears reject; optional session undo per architecture.
- **FR-31:** **Delete** distinct from Reject; guarded confirmation; destructive styling; persistence per architecture (MVP: soft-delete + quarantine under library `.trash`).
- **FR-32:** Before share URL finalized, user **confirms** after **preview** of asset; default **snapshot** semantics; rejected not shareable via default flow.
- **FR-33 (Growth):** **Sharable packages**: multi-asset snapshot; preview manifest before mint; optional audience presets; rejected excluded by default.

### Non-functional requirements

- **NFR-01 (Layout):** Between **1024×768** and **5120×1440**, review and single-photo views keep primary navigation within viewport **100%** of the time (manual matrix: square, 16:9, 21:9).
- **NFR-02 (Performance):** Import/scan shows progress or batch logs; **10,000-file** dry-run without unbounded memory (streaming/chunked).
- **NFR-03 (Integrity):** Duplicate decisions deterministic across upload, scan, import.
- **NFR-04 (Observability):** Each import-like operation emits **added**, **skipped duplicate**, **updated metadata**, **failed** (codes); reject-related counts consistent **GUI vs CLI** when applicable.
- **NFR-05 (Browser share):** Shared URL cold load **under 3 seconds** on broadband (staging/CI measurement, excluding user network).
- **NFR-06 (Security):** Non-guessable tokens; rate-limit/abuse posture documented before public deployment.
- **NFR-07 (Display scaling):** Re-validate NFR-01 at **125% / 150%** OS scaling on macOS/Windows each major milestone.

### Additional requirements (architecture)

- Single Go module; Fyne desktop primary; share via **in-process `net/http`**, **loopback** default.
- SQLite (**modernc.org/sqlite**); migrations under `internal/store/migrations/`; DB at `{library}/.phototool/library.sqlite`.
- **SHA-256** content hash; canonical day dirs `{root}/{YYYY}/{MM}/{DD}/`; suggested filename `{UTC time}_{hash12}{ext}`.
- **`OperationSummary`** (or equivalent) shared by GUI and CLI with stable category names.
- Share tokens stored as **hash only**; snapshot rows in `share_links` (conceptually).
- Reject undo: **session/navigation-bound** undo plus **Rejected/Hidden** restore.
- Structured logging via **`log/slog`**; errors wrapped with `%w`.
- Web share MVP: **`html/template`** + static CSS; Fyne WASM deferred spike.
- EXIF: **dsoprea/go-exif** (+ image-structure helpers) as primary; ExifTool optional later.

### UX design requirements (actionable)

- **UX-DR1:** Implement **dual Fyne themes** (dark default + light peer) from one semantic role table (background, surface, primary, destructive, reject/caution, focus, text primary/secondary).
- **UX-DR2:** **Filter strip** component: order **Collection → minimum rating → tags**; defaults **no collection**, **any rating**; keyboard traversal and visible focus; **one row** by default—**overflow** advanced filters to a **sheet/drawer** so the strip does **not** read as a **second nav bar**.
- **UX-DR3:** **Thumbnail grid cell**: rating badge, reject indicator, non-hover duplicates for key actions; pending/failed decode states; at **default density**, **image** remains the **dominant** read in the cell—**defer** nonessential chrome to **hover/focus** where feasible (**comfortable** tier per UX spec).
- **UX-DR4:** **Review loupe**: image up to **~90%** footprint, letterboxed full image, prev/next affordances; safe chrome for **1:1–21:9** (NFR-01).
- **UX-DR5:** **Reject** control/key **not** adjacent to rating keys **1–5**; Reject uses **caution** styling; **Delete** uses **destructive** styling and confirm.
- **UX-DR6:** **Operation receipt** pattern after batch import/scan: added / duplicate / failed (+ consistent CLI).
- **UX-DR7:** **Share preview sheet**: confirm before mint; no final URL until successful mint; **Copy** control for URL (auto-copy opt-in only).
- **UX-DR8:** **Transient feedback** for undo-reject (capped stack); distinct from batch receipts.
- **UX-DR9:** **Empty states** with one primary CTA (library empty, no results, empty Rejected).
- **UX-DR10:** **Collections list + full-page detail** shell (not modal stack for main browsing).
- **UX-DR11:** Share page **WCAG 2.1 Level A**: focus, labels, contrast; **no raw GPS** on web; neutral **alt** policy (e.g. “Shared photo”) without leaking EXIF/filename into alt without review.
- **UX-DR12:** Share page **200% zoom** primary path usable; **`prefers-reduced-motion`** for non-essential motion.
- **UX-DR13:** Primary nav areas: **Upload**, **Review**, **Collections**, **Rejected** (consistent order/labels).
- **UX-DR14:** **Drag-and-drop** target for upload alongside file picker (same pipeline).
- **UX-DR15:** Document **focus order** filter strip → grid → loupe for desktop QA.
- **UX-DR16 (Image stage prominence):** On **Review** and **collection detail**, the **thumbnail grid stage** occupies the **majority** of the window at default layout; **combined** top chrome (nav + filter strip) stays within a **documented height budget**; **minimum thumbnail edge** and **minimum loupe image region** are **validated** in Story **2.11** evidence (numeric thresholds in layout matrix / NFR-01 notes—not ad hoc per screen).
- **UX-DR17 (AC-UI-THREAD):** After **async** work (ingest, decode, DB), **Fyne** mutations run on the **main thread**; **`-race`** smoke on ingest + grid paths remains green.
- **UX-DR18 (AC-LIST-STATES):** Primary asset **grid** shows defined UX for **empty**, **loading**, **error**, and **populated** (copy + disabled actions per UX spec matrix).
- **UX-DR19 (AC-RESIZE):** At **NFR-01 minimum** window size, **primary** actions are **not clipped**; **Tab** reaches all **visible** controls **without** focus traps on **hidden** widgets.

---

### FR coverage map

| FR | Epic | Notes |
|----|------|--------|
| FR-01–FR-06 | Epic 1 | Ingest + upload collection confirm |
| FR-07–FR-12, FR-15–FR-25, FR-29–FR-31 | Epic 2 | Review, filters, collections, reject/delete |
| FR-13, FR-14, FR-32 | Epic 3 | Share mint + web viewer |
| FR-26 | Epic 1 (core extraction) + Epic 2 (display/persistence breadth) | Split: capture time for ingest; full panel in UI |
| FR-27–FR-28 | Epic 1 | CLI scan/import |
| FR-33 | Epic 4 (Growth) | Packages |
| NFR-01, NFR-07 | Epic 2 | Layout + scaling QA story |
| NFR-02 | Epic 1 | Streaming scan/import |
| NFR-03 | Epic 1 | Single dedup path |
| NFR-04 | Epic 1 + Epic 2 | Summary type + GUI receipts |
| NFR-05, NFR-06 | Epic 3 | Share perf + security posture |
| UX-DR1, UX-DR13, UX-DR15–UX-DR19 | Epic 2 | Shell, themes, layout QA, thread + list states |
| UX-DR2–UX-DR5, UX-DR8–UX-DR10, UX-DR16 | Epic 2 | Components, flows, image prominence |
| UX-DR17 | Epic 1 + Epic 2 | Async → UI thread (ingest + grid) |
| UX-DR6, UX-DR14 | Epic 1 | Receipts + DnD |
| UX-DR7 | Epic 3 | Share preview |
| UX-DR11–UX-DR12 | Epic 3 | Web share a11y |

---

## Epic list

### Epic 1: Ingest photos into a trustworthy local library

**Goal:** The user can add images via GUI or CLI, get **honest receipts**, store files under the canonical layout with **dedup**, and optionally attach an upload batch to collections **only after explicit confirm**.

**FRs covered:** FR-01–FR-06, FR-26 (minimum for placement + progressive completeness), FR-27–FR-28  
**NFRs addressed:** NFR-02, NFR-03, NFR-04  
**UX / arch:** UX-DR6, UX-DR14, **UX-DR17** (async ingest → UI thread); architecture ingest path, `OperationSummary`, SQLite/migrations as needed.

### Epic 2: Review, filter, organize, and curate

**Goal:** The user can browse **image-first**: **filters** and **nav** stay compact while the **grid/loupe** dominates; rate/tag, **loupe** with resilient layout, **collections** on full pages, and **reject** / **delete** with recovery paths—using **dual themes** and navigation that match the UX spec.

**FRs covered:** FR-07–FR-12, FR-15–FR-25, FR-29–FR-31  
**NFRs addressed:** NFR-01, NFR-04 (GUI), NFR-07  
**UX:** UX-DR1–UX-DR5, UX-DR8–UX-DR10, UX-DR13, UX-DR15–UX-DR19.

### Epic 3: Share a single photo for browser review

**Goal:** The user can **preview and confirm**, mint a **snapshot** link, and recipients get a **read-only** page that meets **privacy** and **WCAG A** expectations.

**FRs covered:** FR-13, FR-14, FR-32 (and FR-29 exclusion rules)  
**NFRs addressed:** NFR-05, NFR-06  
**UX:** UX-DR7, UX-DR11–UX-DR12.

### Epic 4 (Growth): Sharable packages

**Goal:** Curate **multi-asset snapshot** links with **manifest preview** and optional **audience presets**.

**FRs covered:** FR-33

---

## Epic 1: Ingest photos into a trustworthy local library

### Story 1.1: Local library foundation (config, layout, database)

As a **photographer**,  
I want **the app to use a clear library location and a reliable local database**,  
So that **my data has a stable home and upgrades apply safely**.

**Acceptance criteria:**

- **Given** no `PHOTO_TOOL_LIBRARY` env var, **when** the app resolves the library root, **then** it uses an absolute path under the OS user config area (`…/photo-tool/library` per architecture) **and** creating the library succeeds.
- **Given** `PHOTO_TOOL_LIBRARY` set to a path, **when** the app starts, **then** that path is used (absolute) and standard subdirs exist (`.phototool`, `.trash`, `.cache/thumbnails`).
- **Given** a fresh library, **when** the store opens, **then** SQLite is created under `.phototool/library.sqlite`, migrations apply, and `schema_meta.version` is **1**.
- **Given** the assets table exists, **when** two active rows would share the same `rel_path`, **then** the database rejects the second insert (partial unique index).
- **And** existing implementation in `internal/config`, `internal/paths`, `internal/filehash`, `internal/store` satisfies the above (regression tests stay green).

**Implements:** Architecture foundation; enables all FRs that need persistence.

---

### Story 1.2: Capture time and content hash for ingestion

As a **photographer**,  
I want **the system to read capture time and hash files consistently**,  
So that **placement and deduplication match the PRD and architecture**.

**Acceptance criteria:**

- **Given** a supported image file with readable EXIF/TIFF capture metadata, **when** the extractor runs, **then** it returns a UTC (or documented) capture instant used for folder placement (FR-01, FR-26 subset).
- **Given** a file without usable EXIF, **when** the extractor runs, **then** fallback order is **documented in code** (e.g. embedded time → filesystem mtime) and non-silent.
- **Given** any file path, **when** hashing completes, **then** the result is **SHA-256** lowercase hex matching architecture/NFR-03.
- **And** unit tests cover at least one EXIF sample (or golden file) and the “no EXIF” fallback path.

**Implements:** FR-01 (input to placement), FR-26 (partial), FR-02/FR-03 inputs, NFR-03.

---

### Story 1.3: Core ingest — copy into canonical storage and register asset

As a **photographer**,  
I want **new files copied into Year/Month/Day with unique names and DB registration**,  
So that **the library reflects what is on disk**.

**Acceptance criteria:**

- **Given** a source file not yet in the library, **when** ingest runs, **then** the file is copied under `{library}/{YYYY}/{MM}/{DD}/` using capture time from Story 1.2 and named per architecture (`SuggestedFilename` + hash prefix), and an **assets** row is inserted with `content_hash`, `rel_path`, `capture_time_unix`, `created_at_unix`.
- **Given** a file whose **size + hash** matches an existing asset, **when** ingest runs, **then** no duplicate copy is made and the outcome is counted as **skipped_duplicate** (FR-03).
- **Given** ingest processes multiple files, **when** it finishes, **then** it returns an **`OperationSummary`** (or equivalent) with stable fields: **added**, **skipped_duplicate**, **updated**, **failed** (NFR-04).
- **And** ingest uses streaming/chunked file read for hashing where appropriate (supports NFR-02 for large batches).

**Implements:** FR-01, FR-02, FR-03; NFR-02, NFR-03, NFR-04.

---

### Story 1.4: Collections schema and batch assignment API

As a **photographer**,  
I want **collections to exist in the database before upload confirmation flows**,  
So that **I can attach uploads to albums safely**.

**Acceptance criteria:**

- **Given** migrations run, **when** the store is opened, **then** `collections` and `asset_collections` tables exist with fields needed for FR-18 (name required, display_date optional with default rule).
- **Given** a set of asset IDs and a collection ID, **when** linking API runs, **then** relations are created idempotently or per documented rules.
- **Given** a collection delete request, **when** executed, **then** all asset–collection rows for that collection are removed then the collection row is deleted (FR-20).

**Implements:** FR-18 (persistence), FR-20; prerequisite for FR-04–FR-06.

---

### Story 1.5: Desktop upload flow with collection confirm and receipt

As a **photographer**,  
I want **to pick files, optionally assign an upload collection, and confirm before anything is created**,  
So that **I get predictable organization without surprise albums**.

**Acceptance criteria:**

- **Given** the user selects multiple images via file picker, **when** import completes successfully, **then** each new file is ingested per Story 1.3 and the UI shows an **operation receipt** with added / duplicate / failed counts (UX-DR6, FR-03, NFR-04).
- **Given** the upload flow offers collection assignment, **when** the user has not confirmed, **then** no new collection and no links are persisted (FR-06).
- **Given** the default collection name pattern `Upload YYYYMMDD`, **when** the user confirms, **then** the collection is created/updated and all batch assets are linked as specified (FR-04, FR-05).
- **Given** the user clears or renames the collection name before confirm, **when** they confirm, **then** the persisted name matches their input.
- **Given** a multi-file pick, **when** the confirm step is shown, **then** the UI surfaces **large previews** of the batch (image-first **ingest** stage per UX **Direction E**); the **operation receipt** remains readable and may be **collapsed** after the user has learned the pattern (UX spec **feedback patterns**).
- **Given** ingest work runs off the UI thread, **when** results are applied, **then** **Fyne** widgets update on the **main thread** (UX-DR17).

**Implements:** FR-04, FR-05, FR-06; NFR-04; UX-DR6, UX-DR17.

---

### Story 1.6: Scan CLI (`--dir`, `--recursive`, `--dry-run`)

As a **power user**,  
I want **to scan a folder tree into the canonical library with optional dry-run**,  
So that **I can reconcile large archives safely**.

**Acceptance criteria:**

- **Given** `phototool scan --dir PATH` (exact CLI shape per implementation), **when** run without dry-run, **then** supported images are discovered recursively (when flag set), hashed, deduped, copied if new, DB updated (FR-27).
- **Given** `--dry-run=true`, **when** scan runs, **then** **no** files are copied and **no** DB writes occur, but the summary counts are emitted (FR-27).
- **Given** scan completes, **when** output is printed, **then** **`OperationSummary`** categories match the GUI ingest receipt semantics (NFR-04).
- **And** a 10,000-file dry-run does not grow memory unbounded (NFR-02) — verified by streaming walk test or benchmark note in code comments.
- **And** the process exits with **non-zero status** when any per-file failures were recorded, **after** printing the summary.

**Implements:** FR-27; NFR-02, NFR-04.

---

### Story 1.7: Import CLI (register / backfill)

As a **power user**,  
I want **to register existing canonical files or backfill metadata**,  
So that **the DB matches disk after manual operations**.

**Acceptance criteria:**

- **Given** an import root **under** `libraryRoot` (after symlink resolution; no “path under library” tricks), **when** import runs without `--dry-run`, **then** supported images missing from the DB by **content hash** are **registered in place** (no copy): `rel_path` relative to library, capture time and hash consistent with scan/upload (FR-28, NFR-03).
- **Given** a file whose hash already exists on **another** `rel_path` in the DB, **when** import processes a second path with the same bytes, **then** the outcome is **skipped duplicate** (NFR-03).
- **Given** `--dry-run`, **when** import runs, **then** no `INSERT`/`UPDATE`/`DELETE` on `assets`, no file copies/moves under the library tree, and only summary output is produced (FR-28).
- **Given** backfill rules in implementation (same hash and `rel_path`, stale `capture_time_unix`), **when** import runs, **then** metadata is updated and **updated** count reflects it where applicable (NFR-04).
- **Given** import completes, **when** CLI prints results, **then** **`OperationSummary`** categories match **scan** / GUI: **added**, **skipped duplicate**, **updated**, **failed** (NFR-04).
- **And** large trees use the same **streaming walk** discipline as scan (NFR-02).
- **And** the process exits with **non-zero status** when any per-file failures were recorded, **after** printing the summary — **same rule as `scan`** so scripts and CI can detect partial failure.

**Implements:** FR-28; NFR-02, NFR-03, NFR-04.

---

### Story 1.8: Drag-and-drop upload entry

As a **photographer**,  
I want **to drop files onto a designated target**,  
So that **ingestion matches the picker path exactly**.

**Acceptance criteria:**

- **Given** the Upload view with a visible drop target, **when** the user drops supported image files, **then** the same ingest + receipt path runs as for the file picker (UX-DR14, FR-01).
- **Given** unsupported files, **when** dropped, **then** the user sees a clear, factual message without silent failure.

**Implements:** UX-DR14; FR-01 (parity with picker).

---

## Epic 2: Review, filter, organize, and curate

### Story 2.1: Application shell, navigation, and dual themes

As a **photographer**,  
I want **consistent navigation and dark/light themes**,  
So that **long sessions are comfortable and I always know where I am**.

**Acceptance criteria:**

- **Given** the app launches, **when** the main window loads, **then** primary areas exist: **Upload**, **Review**, **Collections**, **Rejected** (UX-DR13).
- **Given** theme toggle or preference, **when** switched, **then** both **dark** and **light** themes apply semantic roles (primary, destructive, reject/caution, focus) without feature gaps (UX-DR1).
- **And** focus visibility is visible on standard Fyne controls (baseline for UX-DR15).
- **And** primary **navigation** is **compact** (single obvious row / rail per **Direction A**); it does **not** compete with the **Review** image stage for vertical space (UX-DR16 baseline).

**Implements:** UX-DR1, UX-DR13, UX-DR16 (shell); enables FR-07+ UI work.

---

### Story 2.2: Filter strip (collection, min rating, tags)

As a **photographer**,  
I want **filters in a fixed order with sensible defaults**,  
So that **browsing matches my mental model**.

**Acceptance criteria:**

- **Given** the Review surface, **when** filters render, **then** order is **Collection → minimum rating → tags** (FR-15).
- **Given** a fresh session, **when** Review opens, **then** defaults are **no assigned collection** and **any rating** (FR-16).
- **Given** the user changes filters, **when** applied, **then** the result set updates without silent mismatch (UX-DR2).
- **And** keyboard users can traverse the strip with visible focus (UX-DR2).
- **And** the strip occupies **at most one** default **row** of controls; additional filters or sort/scope controls use **overflow** (sheet, drawer, or equivalent)—not a **second** horizontal **nav** (UX-DR2).

**Implements:** FR-15, FR-16; UX-DR2.

---

### Story 2.3: Paged thumbnail grid with rating and reject badges

As a **photographer**,  
I want **an image-forward grid that still shows rating and reject at a glance**,  
So that **photos stay primary while I triage quickly**.

**Acceptance criteria:**

- **Given** filtered assets, **when** the grid loads, **then** thumbnails load incrementally (paged or windowed) without loading all pixmaps at once (architecture).
- **Given** an asset with rating or reject state, **when** shown in grid, **then** badges reflect DB state within the PRD **1 second** guideline for local single-user use (FR-10, SC-3 / FR-07 baseline for display).
- **Given** decode failure, **when** a cell renders, **then** user sees failed-decode affordance (placeholder + explanation) per UX-DR3.
- **And** at default density, **thumbnail imagery** is the **largest** element in the cell; nonessential chrome **defers** to **hover/focus** where feasible (UX-DR3, UX-DR16).
- **And** **minimum** thumbnail **readability** at **1024×768** and **1920×1080** reference layouts is **recorded** in Story **2.11** / NFR-01 evidence (numeric thresholds—not ad hoc).

**Implements:** FR-07 (display tags/ratings context); supports FR-08/FR-29 display; UX-DR3, UX-DR16.

---

### Story 2.4: Review loupe with safe chrome and keyboard rating

As a **photographer**,  
I want **a large letterboxed view with keyboard 1–5 and prev/next**,  
So that **I can review on any aspect ratio monitor**.

**Acceptance criteria:**

- **Given** an asset opened from the grid, **when** loupe opens, **then** the image uses up to **90%** of the viewport and is fully visible without cropping (letterboxed) (FR-09, FR-12, FR-25 image rules).
- **Given** keyboard **1–5** or star control, **when** used, **then** rating persists without extra confirm dialog (FR-10).
- **Given** window aspects from **1:1** to **21:9**, **when** resized manually per NFR-01 matrix, **then** primary controls remain in viewport (FR-11, NFR-01).
- **And** Reject shortcut is **not** bound adjacent to **1–5** (UX-DR5).
- **And** **chrome** does **not** **overlap** the **letterboxed** image **by default**; **minimum loupe image region** is asserted in Story **2.11** evidence (UX-DR4, UX-DR16).

**Implements:** FR-09–FR-12, FR-25 (desktop portion); NFR-01; UX-DR4, UX-DR5, UX-DR16.

---

### Story 2.5: Tags editing in bulk review

As a **photographer**,  
I want **to add/edit tags on photos from bulk review**,  
So that **I can organize beyond stars**.

**Acceptance criteria:**

- **Given** tag schema exists (migration as needed), **when** the user edits tags on one or more assets, **then** values persist and participate in filter strip “tags” (FR-07, FR-15).
- **Given** empty tag filter, **when** applied, **then** behavior matches documented “any tag” semantics.

**Implements:** FR-07; extends FR-15.

---

### Story 2.6: Reject, session undo, Rejected/Hidden, and restore

As a **photographer**,  
I want **to hide bad shots without losing them**,  
So that **I can recover quickly and trust default views**.

**Acceptance criteria:**

- **Given** an asset in grid or loupe, **when** the user rejects it, **then** it disappears from default queries and does not appear in share selection later (FR-29).
- **Given** a reject action, **when** still in the same review context per architecture, **then** the user can undo from transient UI (FR-30 + architecture).
- **Given** **Rejected/Hidden** view, **when** the user restores an asset, **then** reject flag clears and it reappears in default surfaces (FR-30).
- **And** Reject uses **caution** styling; distinct from Delete (UX-DR5).

**Implements:** FR-29, FR-30; UX-DR5, UX-DR8.

---

### Story 2.7: Delete with confirmation and quarantine

As a **photographer**,  
I want **delete to be clearly separate from reject and require confirmation**,  
So that **I do not erase by accident**.

**Acceptance criteria:**

- **Given** delete is triggered, **when** the user has not confirmed, **then** no delete pipeline runs (FR-31).
- **Given** confirmed delete, **when** executed, **then** behavior matches architecture MVP: **soft-delete + file quarantine** under library `.trash` (FR-31 + architecture).
- **And** UI uses **destructive** styling distinct from reject (UX-DR5).

**Implements:** FR-31; UX-DR5.

---

### Story 2.8: Collections list and full-page collection detail

As a **photographer**,  
I want **to browse collections on their own page**,  
So that **albums feel like first-class places**.

**Acceptance criteria:**

- **Given** the Collections list, **when** the user selects a collection, **then** navigation goes to a **full-page** detail view (not a modal) (FR-21).
- **Given** collection detail, **when** shown, **then** assets sort by **capture time** by default (FR-22).
- **Given** default grouping, **when** detail loads, **then** grouping is by **star rating** descending with empty groups omitted; within group sort by capture time (FR-23).
- **Given** user action, **when** grouping switches, **then** **by day** and **by camera name** modes work per FR-24 (camera/lens field must be available from FR-26 persistence).

**Implements:** FR-21–FR-24.

---

### Story 2.9: Collection CRUD, multi-assign, and safe collection delete

As a **photographer**,  
I want **to create, rename, and delete collections without orphaning rules wrong**,  
So that **metadata stays trustworthy**.

**Acceptance criteria:**

- **Given** create/rename flows, **when** saved, **then** name is required and display date defaults per FR-18 when omitted.
- **Given** single-photo view, **when** the user assigns multiple collections or creates a new collection, **then** FR-19 is satisfied.
- **Given** collection delete, **when** confirmed, **then** relations detach then collection deletes (FR-20).

**Implements:** FR-18–FR-20.

---

### Story 2.10: Quick collection assign (hover) and filter workflow assign

As a **photographer**,  
I want **to assign collections from the grid and from filters**,  
So that **organization stays fast at scale**.

**Acceptance criteria:**

- **Given** a thumbnail with hover or equivalent quick action, **when** the user picks a collection, **then** assignment persists without opening loupe (FR-08).
- **Given** filter workflow selection, **when** assign-to-collection runs, **then** selected assets update per FR-17.

**Implements:** FR-08, FR-17.

---

### Story 2.11: Layout and display-scaling validation gate

As a **product owner**,  
I want **documented proof that layout holds across sizes and OS scaling**,  
So that **ultrawide and laptop both work**.

**Acceptance criteria:**

- **Given** the NFR-01 matrix (1024×768 through 5120×1440, square/16:9/21:9), **when** QA runs on tier-1 OS targets, **then** results are recorded (pass/fail + notes) for Review + loupe.
- **Given** 125% and 150% OS scaling on macOS and Windows, **when** checked each major milestone, **then** NFR-07 checklist is updated.
- **And** failures become tracked defects with UX/layout owner.
- **And** evidence captures **UX-DR16** thresholds where applicable: **min thumb edge** (grid), **min loupe image region**, **combined nav+filter height budget**, plus **UX-DR19**: at **minimum** window, **primary** actions **not clipped** and **Tab** order has **no** focus trap on **hidden** widgets.
- **And** each recorded threshold cites a **measurement anchor** (e.g. full window vs inner content pane) and **lifecycle moment** (e.g. initial layout vs after thumbnail decode vs after cache hit), consistent with [architecture.md](architecture.md) §3.8.1 and [initiative-fyne-image-first-phase1-party-2026-04-15.md](initiative-fyne-image-first-phase1-party-2026-04-15.md).

**Implements:** NFR-01, NFR-07; UX-DR16, UX-DR19.

---

### Story 2.12: Empty states and error tone

As a **photographer**,  
I want **helpful empty states and honest errors**,  
So that **I never feel lost after import or filtering**.

**Acceptance criteria:**

- **Given** empty library, no filter results, or empty Rejected, **when** the user opens that surface, **then** they see **one primary CTA** per UX-DR9.
- **Given** IO/DB errors, **when** shown, **then** copy is factual with next steps (UX spec “proportionate honesty”).
- **Given** the primary **Review** (or collection) **grid**, **when** in **loading** or **error** state, **then** UI matches the **AC-LIST-STATES** matrix (copy + which actions are disabled) (UX-DR18).

**Implements:** UX-DR9, UX-DR18; cross-cutting UX quality.

---

## Epic 3: Share a single photo for browser review

### Story 3.1: Share preview, confirm, and snapshot mint (desktop)

As a **photographer**,  
I want **to preview exactly what I share before a link exists**,  
So that **I never mint the wrong asset**.

**Acceptance criteria:**

- **Given** share from loupe, **when** the user has not confirmed, **then** no token/URL is persisted or copied (FR-32, UX-DR7).
- **Given** confirm, **when** mint succeeds, **then** a **snapshot** row exists tying token hash to asset identity at mint time (FR-32, architecture).
- **Given** a **rejected** asset, **when** user attempts default share flow, **then** share is blocked (FR-29, FR-32).
- **Given** mint **fails** (permissions, size limits, IO), **when** the preview sheet is open, **then** the user sees a **clear failure** state with **next steps**—**no** silent close and **no** URL copied (UX-DR7; UX spec **share preview** failure states).

**Implements:** FR-32; FR-29 (share side); UX-DR7.

---

### Story 3.2: Loopback HTTP server and token resolution

As a **photographer**,  
I want **the app to serve share links locally by default**,  
So that **tokens are not unnecessarily exposed**.

**Acceptance criteria:**

- **Given** default config, **when** share serving starts, **then** it binds **loopback** only unless explicit opt-in (architecture).
- **Given** a valid token, **when** requested, **then** handler resolves via **stored hash** (never plaintext token in DB) (architecture, NFR-06).
- **Given** invalid token, **when** requested, **then** safe 404 without leaking existence details.

**Implements:** FR-13 (technical enabler); NFR-06 baseline.

---

### Story 3.3: Read-only share HTML page (image + rating)

As a **recipient**,  
I want **to open the shared photo in a browser with correct layout**,  
So that **I can review without installing the app**.

**Acceptance criteria:**

- **Given** a valid share URL, **when** the page loads, **then** the same asset renders read-only with image fitted analogously to FR-12 (FR-14).
- **Given** the snapshot, **when** rendered, **then** **current rating at mint** is visible (FR-14).
- **Given** mobile width, **when** viewed, **then** primary content remains usable (PRD browser targets).

**Implements:** FR-13, FR-14.

---

### Story 3.4: Share page privacy and WCAG 2.1 Level A

As a **recipient**,  
I want **a page that respects privacy and basic accessibility**,  
So that **I feel safe viewing shared photos**.

**Acceptance criteria:**

- **Given** shared page render, **when** inspected, **then** **raw GPS** and restricted location panels are **not** exposed (PRD domain + UX-DR11).
- **Given** keyboard navigation, **when** user tabs the page, **then** controls are focusable with visible focus (WCAG A, UX-DR11).
- **Given** image alt, **when** no owner caption exists, **then** neutral policy applies (e.g. “Shared photo”) without auto-filling from filename/EXIF (UX-DR11).
- **Given** 200% zoom, **when** tested, **then** primary reading path remains usable (UX-DR12).
- **And** `prefers-reduced-motion` honored for non-essential motion (UX-DR12).

**Implements:** FR-14 (constraints); UX-DR11–UX-DR12; PRD domain requirements.

---

### Story 3.5: Share cold-load performance and abuse posture

As a **product owner**,  
I want **measurable share performance and documented rate limits**,  
So that **we meet NFR-05/NFR-06**.

**Acceptance criteria:**

- **Given** staging or CI measurement harness, **when** cold load runs, **then** median/goal aligns with **NFR-05** (document methodology and caveats).
- **Given** share endpoints, **when** documented, **then** rate-limit/abuse posture for public deployment is written (NFR-06) — in-repo doc or architecture appendix.

**Implements:** NFR-05, NFR-06.

---

## Epic 4 (Growth): Sharable packages

### Story 4.1: Multi-asset snapshot packages with manifest preview

As a **photographer**,  
I want **to share a curated set with manifest preview**,  
So that **friends see exactly the package I intended**.

**Acceptance criteria:**

- **Given** multi-select or filtered set, **when** package flow runs, **then** user sees **count + thumbnails or IDs** before mint (FR-33).
- **Given** optional audience preset, **when** selected, **then** it does **not** skip preview (FR-33).
- **Given** package mint, **when** complete, **then** recipients see fixed snapshot; **rejected** never included by default (FR-33, FR-29).

**Implements:** FR-33.

---

## Final validation record

| Check | Result |
|-------|--------|
| Every FR FR-01–FR-32 mapped to ≥1 story | **Pass** (FR-33 → Epic 4) |
| NFR-01–NFR-07 addressed in stories | **Pass** |
| UX-DR1–UX-DR19 covered | **Pass** |
| Epics ordered by user value, not layers | **Pass** |
| Stories avoid forward dependencies within epic | **Pass** (sequential enablement) |
| DB/migrations only when story needs them | **Pass** (1.1 assets; 1.4 collections/tags as stated) |
| Architecture starter | N/A (brownfield); Story 1.1 matches existing foundation |

**Workflow status:** Epics and stories are ready for **implementation** (`bmad-dev-story` / `bmad-quick-dev`) and sprint tracking (`bmad-sprint-planning`). For questions on what to run next, use **`bmad-help`**.

---

## Story authoring — files to reference (per story)

When you create **`_bmad-output/implementation-artifacts/<id>.md`**, set **`inputDocuments`** (or equivalent) to at least:

| Story | Always include | Also use for UI/layout |
|-------|----------------|-------------------------|
| **1.1** | [PRD.md](PRD.md) (persistence baseline), [architecture.md](architecture.md), **this file** §Story 1.1 | [ux-design-specification.md](ux-design-specification.md) §Design system (if touching paths/cache names) |
| **1.2** | PRD FR-01/FR-26, architecture EXIF/hash, **this file** §1.2 | — |
| **1.3** | PRD FR-01–FR-03, NFR-02–04, architecture ingest, **this file** §1.3 | — |
| **1.4** | PRD FR-18/FR-20, architecture DB, **this file** §1.4 | — |
| **1.5** | PRD FR-04–FR-06, NFR-04, **this file** §1.5 (**UX-DR6**, **UX-DR17**, Direction E) | UX spec **Journey A**, **feedback patterns**, [ux-design-directions.html](ux-design-directions.html) |
| **1.6** | PRD FR-27, NFR-02/04, **this file** §1.6 | — |
| **1.7** | PRD FR-28, NFR-02–04, **this file** §1.7 | — |
| **1.8** | PRD FR-01, **this file** §1.8 (**UX-DR14**) | UX spec upload / DnD patterns |
| **2.1** | **this file** §2.1 (**UX-DR1**, **UX-DR13**, **UX-DR15**, **UX-DR16** shell) | UX spec **navigation**, **color system**, **Direction A** in directions HTML |
| **2.2** | PRD FR-15/FR-16, **this file** §2.2 (**UX-DR2** one-row + overflow) | UX spec **Form patterns** (filter strip) |
| **2.3** | PRD FR-07/FR-10, **this file** §2.3 (**UX-DR3**, **UX-DR16**) | UX spec **Image stage and density**, Party follow-ups (thumbnail pipeline) |
| **2.4** | PRD FR-09–FR-12, FR-25, NFR-01, **this file** §2.4 | UX spec loupe / **reduced motion** where relevant |
| **2.5** | PRD FR-07/FR-15, **this file** §2.5 | UX spec tag/bulk patterns if expanded |
| **2.6** | PRD FR-29/FR-30, **this file** §2.6 (**UX-DR5**, **UX-DR8**) | UX spec button hierarchy (reject vs delete) |
| **2.7** | PRD FR-31, **this file** §2.7 | Same as 2.6 |
| **2.8** | PRD FR-21–FR-24, **this file** §2.8 (**UX-DR10**) | UX spec **Places not modes** |
| **2.9** | PRD FR-18–FR-20, **this file** §2.9 | — |
| **2.10** | PRD FR-08/FR-17, **this file** §2.10 | UX spec quick actions / grid |
| **2.11** | PRD NFR-01/NFR-07, **this file** §2.11; [nfr-01-layout-matrix-evidence.md](../implementation-artifacts/nfr-01-layout-matrix-evidence.md); [nfr-07-os-scaling-checklist.md](../implementation-artifacts/nfr-07-os-scaling-checklist.md) | UX spec testing strategy; `internal/domain/nfr_layout.go` for matrix constants |
| **2.12** | **this file** §2.12 (**UX-DR9**, **UX-DR18**) | UX spec **proportionate honesty**, list-state matrix |
| **3.1** | PRD FR-32, FR-29 (share), **this file** §3.1 (**UX-DR7**) | UX spec share preview failure states |
| **3.2** | PRD FR-13, architecture share HTTP, **this file** §3.2 | — |
| **3.3** | PRD FR-13/FR-14, **this file** §3.3 | — |
| **3.4** | PRD domain privacy, **this file** §3.4 (**UX-DR11**, **UX-DR12**) | Share HTML templates under `internal/share` |
| **3.5** | PRD NFR-05/NFR-06, **this file** §3.5 | — |
| **4.1** | PRD FR-33, **this file** §4.1 | UX spec packages / manifest preview |

**Global references for any Fyne story:** [architecture.md](architecture.md) (§ layering, §3.8 Fyne/thumbnails), [ux-design-specification.md](ux-design-specification.md) **Party mode follow-ups (2026-04-14)**, and **this file** §Requirements inventory (FR/NFR/**UX-DR1–19**).

---

_End of epic breakdown (v2)._
