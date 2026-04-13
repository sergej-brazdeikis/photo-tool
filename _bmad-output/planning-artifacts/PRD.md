---
workflowType: prd
workflow: edit
classification:
  domain: consumer_media
  projectType: desktop_and_web
  complexity: moderate
inputDocuments:
  - docs/input/initial-idea.md
  - _bmad-output/planning-artifacts/ux-design-specification.md
stepsCompleted:
  - step-e-01-discovery
  - step-e-01b-legacy-conversion
  - step-e-02-review
  - step-e-03-edit
lastEdited: '2026-04-12'
date: '2026-04-12'
validationReport: _bmad-output/planning-artifacts/validation-report-20260412-run2.md
editHistory:
  - date: '2026-04-12'
    changes: >-
      Converted legacy initial-idea.md into BMAD PRD structure; added measurable
      FRs/NFRs, user journeys, scope phases; preserved stack intent under
      project-type requirements.
  - date: '2026-04-12'
    changes: >-
      PRD validation (bmad-validate-prd): corrected User Journey B/D FR references;
      see validation-report-20260412.md.
  - date: '2026-04-12'
    changes: >-
      Edit workflow (post-validation): FR-14/FR-18 clarified; MVP out-of-scope;
      desktop/web platform matrix; SC-3 and domain share/GPS tightened.
  - date: '2026-04-12'
    changes: >-
      PRD validation run 2: overall Pass; MVP collections line aligned to display date;
      report validation-report-20260412-run2.md.
  - date: '2026-04-12'
    changes: >-
      Aligned with ux-design-specification.md: Reject vs Delete, share preview before mint
      and snapshot default, Rejected/Hidden recovery, CLI parity for reject counts, Growth
      sharable packages and audience presets; journeys and traceability updated.
---

## Executive Summary

**Photo Tool** is a photo-management product for importing, organizing, reviewing, and grouping personal images. Users bulk-upload or scan existing files, store each unique image once with deduplication, organize storage by capture date (not file creation time), and work with **collections**, **ratings (1–5)**, **tags**, **Reject** (soft-hide from normal browsing and from share selection), optional **Delete** (stronger removal, distinct from Reject), and rich **EXIF/TIFF metadata**. The product provides a **primary Fyne desktop UI** (dark and light themes) and **web-accessible views** where a **shareable URL** opens the same photo in review context—after **preview/confirm** in the app before the link is minted (**MVP**).

**Target users:** hobbyist and semi-pro photographers and organizers who manage large local libraries and want fast bulk operations, predictable on-disk layout, **noise reduction** via reject without default irreversible loss, and **audience-appropriate sharing** (single-photo links in MVP; **sharable packages** in Growth).

**Differentiator:** Single pipeline for upload and filesystem scan with identical deduplication and metadata rules; collection-centric browsing with flexible grouping (rating, day, camera); explicit keyboard and touch navigation in single-photo review; **Reject ≠ Delete**; **snapshot-first** sharing with **preview before publish**; CLI summaries aligned with GUI.

---

## Success Criteria

1. **Import integrity:** For a batch of mixed JPEG/RAW/common raster inputs, ≥ 99% of files with readable EXIF expose **capture datetime** used for folder placement; files without usable EXIF fall back to a documented default rule without silent data loss.
2. **Deduplication:** When two files match **size + content checksum**, only one stored instance exists in the library; the user receives a clear, countable summary of skipped duplicates per operation.
3. **Review speed:** Assigning a rating via keyboard **1–5** or star control persists visible state within **1 second** under **single-user, local-library** use (no concurrent remote editors; no multi-user load target in MVP).
4. **Layout correctness:** On viewport aspect ratios from **1:1 (square)** through **21:9 (ultrawide)**, primary navigation for review and collection single-photo views remains **fully visible** (no controls clipped outside the viewport); **full image** fits inside **90% of viewport** without cropping in single-photo modes.
5. **Share fidelity:** A shared **review link** loads the **same photo** and review context in the browser as in the app session that created the link (same asset identity and review state where applicable), and is only created after the user **confirms** a **preview** of what will be shared (**MVP**).
6. **Operational tooling:** Scan and import CLI paths (or equivalent) process a directory tree with **dry-run** mode reporting counts (candidates, duplicates, DB updates) without mutating storage when dry-run is enabled.
7. **Reject integrity:** **Rejected** photos do not appear in **default** browse, filter, collection, or **share/package** selection surfaces; users can **restore** from a dedicated **Rejected/Hidden** destination; **GUI and CLI** use the same meaning for reject state in summaries where applicable.

---

## Product Scope

### MVP

- Bulk upload with **Year/Month/Day** folders, filenames including **time + content hash**.
- EXIF-based **capture datetime**; duplicate detection **size + checksum**; user-visible duplicate summary.
- Optional post-upload **collection** assignment: default name `Upload YYYYMMDD`, editable before confirm; **no** collection created without confirmation.
- Bulk review: tags, rating 1–5, hover/quick assign to collection, modal or full-stage view up to **90%** screen; keyboard rating.
- Filters: order **Collection → Min rating → tags**; defaults **No assigned collection** + **Any rating**; assign-to-collection from filter context.
- Collections CRUD (**name**, **display date** per FR-18); multi-assign from single-photo view; safe delete (detach relations then delete).
- Collections list page; collection detail as **full page** (not modal), default grouping by **star sections** (hide empty ratings), intra-section sort by capture time; optional grouping by **day** or **camera name**.
- Single-photo view: fit image without crop, **90%** max footprint, stars + keyboard **1–5**, prev/next via UI, keyboard, swipe; arrows vertically centered at image left/right.
- **Reject:** soft-hide from default surfaces and from **share** selection; **undo** and/or **Rejected/Hidden** recovery (exact undo rule: architecture/product).
- **Delete:** distinct from **Reject**, **guarded** confirmation; **persistence** (library-only vs file removal) defined in architecture.
- **Share (MVP):** **Preview/confirm** before **token/URL** is minted; default **snapshot** link to the asset as of mint time; URL shown with explicit **Copy** control (auto-copy only if user opts in).
- Metadata extraction per **Photo data** section in source (camera, lens, exposure, GPS, resolution, orientation, flash, metering, white balance).
- **Shareable browser URL** for review photo view (read-only per FR-14).
- **Scan** and **import** tools aligned with source doc behavior (EXIF extract, dedup, DB registration, dry-run); summaries include **reject-related** counts when the operation touches reject semantics (**CLI parity** with GUI).
- **Upload:** file picker plus **drag-and-drop** onto a designated target when feasible (**same pipeline** as multi-select).

#### Out of scope (MVP)

- **Accounts and multi-tenant hosting** — single operator / local deployment assumption unless Growth adds auth.
- **Browser-based rating edits via share link** — recipients view only; see FR-14.
- **RAW conversion pipeline, face recognition, map UI** — listed under Growth or Vision.
- **Mobile native apps** — touch supported in browser for shared links; no iOS/Android store apps in MVP.
- **Cloud backup, sync across devices, collaborative real-time editing.**

### Growth

- **Sharable packages:** multi-asset **snapshot** links with **preview manifest** (count + thumbnails/IDs) before mint; **audience presets** (e.g. close friends, wider circle, family) as **accelerators**, not a substitute for preview; **rejected** excluded by default.
- **Live** collection-linked shares (if ever offered) require explicit recipient copy that content **may change**—not default in MVP.
- Stronger format coverage and RAW/TIFF edge cases; map previews where needed.
- Richer tag management (bulk tag, tag filters persisted).
- Optional multi-user/auth model for shared libraries (not assumed in MVP).

### Vision

- Face/object smart albums, map view from GPS, offline-first sync (only if product direction confirms).

---

## User Journeys

### Journey A — Bulk import and optional collection

1. User selects many files in **Upload Photos**.
2. System computes hashes, skips duplicates, writes new files under `Year/Month/Day` with **capture** time from EXIF in filename.
3. System offers assignment to one or more collections; default label `Upload YYYYMMDD`; user edits label or cancels.
4. User confirms → collection created/updated and links applied; cancel → no new collection.

**Requirements touched:** FR-01–FR-06.

### Journey B — Bulk review and rating

1. User opens bulk review; sees grid with hover actions.
2. User sets rating via **1–5** keys or stars; assigns collection from dropdown on hover.
3. User opens a photo → large view (~90% viewport); UI remains reachable on ultrawide and square layouts.
4. User starts share → **preview/confirm** asset identity → system **mints** token/URL → user copies link (e.g. via **Copy** control); recipient opens browser → same photo in **read-only** review layout (MVP); rating edits stay in the desktop app.

**Requirements touched:** FR-07–FR-14, FR-29–FR-32.

### Journey C — Browse by collection

1. User opens **Collections** list → selects one.
2. Detail page shows images sorted by capture time, grouped by stars (only non-empty groups); user switches grouping to day or camera.
3. User opens single photo → rates, assigns multiple collections, creates new collection inline, navigates prev/next.

**Requirements touched:** FR-15–FR-21.

### Journey D — Scan / import existing disk

1. User runs scan (e.g. `external-photos` or custom `--dir`) with optional `--dry-run`.
2. Tool finds supported images, extracts EXIF, dedupes, copies into canonical layout (scan) or registers existing files (import), updates DB per rules.

**Requirements touched:** FR-27–FR-28.

### Journey E — Reject, delete, and recovery

1. User in bulk review or single-photo view **rejects** a photo → it **disappears** from default grid/collections/filters and cannot be added to a **share** selection unless explicitly viewing **Rejected/Hidden**.
2. User **undoes** reject per product rule and/or opens **Rejected/Hidden** and **restores** to the library.
3. If user chooses **Delete** (distinct control), system requires **confirmation**; post-confirm behavior per architecture (library row vs file removal).

**Requirements touched:** FR-29–FR-31.

### Journey F — Sharable package (Growth)

1. User selects photos or a filtered set → starts **package** flow.
2. Optional **audience preset** → **preview manifest** → **confirm** → **mint** snapshot package link.
3. Recipients see fixed set; **rejected** never included by default.

**Requirements touched:** FR-33 (Growth); FR-29 for exclusion rules.

---

## Domain Requirements

- **Personal media:** GPS and camera metadata are **personally sensitive**. **MVP:** the **shared web review page does not display raw GPS coordinates, map embeds, or EXIF location panels** to link holders; the desktop app may show full metadata (including GPS) to the **local operator** only. Growth may add policy-controlled sharing of location metadata.
- **Provenance:** Displayed **capture time** must prefer EXIF/TIFF over filesystem mtime when available; document fallback ordering.

---

## Innovation Analysis

- Unified **upload + scan + import** pipeline with identical dedup and path rules reduces “two truths” between disk and database.
- **Shareable review URLs** bridge desktop workflow and lightweight browser review without exporting files manually.
- **Reject vs Delete** plus **preview-before-publish** sharing reduces embarrassment risk and keeps curation **authored**; **Growth packages** extend the same trust model to multi-asset **snapshots**.

---

## Project-Type Requirements

- **Stack (product charter):** Implementation uses **Go** with **Fyne** for the primary UI (**dark and light** themes, same semantic roles), plus **web-served** surfaces for shared review links (implementation may be minimal HTML/CSS and/or **Fyne WASM**—architecture decides). PRD behavior is stated in product terms; APIs and module boundaries are left to architecture.
- **Image formats:** Support “best effort” display for common consumer formats in UI and browser; exact codec list is an architecture/test artifact, MVP must include **JPEG** and documented behavior for missing decoders.
- **Platforms:** Desktop windowing with responsive layout breakpoints; browser view for shared links must work on **desktop and mobile** width (touch swipe for navigation).

### Desktop and web targets (MVP)

- **Desktop OS:** **macOS** and **Windows** are tier-1; **Linux** is tier-2 (same Fyne build targets architecture supports).
- **Desktop displays:** Layout rules in NFR-01; support for laptop through ultrawide as specified there.
- **Share-link browsers (latest two major versions):** **Chrome**, **Safari**, **Firefox** on desktop; **Mobile Safari** (iOS) and **Chrome** (Android) for shared URLs.
- **Accessibility (shared web page only):** Meet **WCAG 2.1 Level A** for the share-link review view (e.g. keyboard focusable controls, text alternatives for meaningful icons, visible labels). Desktop Fyne accessibility posture is architecture-defined.

---

## Functional Requirements

### Import and storage

- **FR-01:** Users can upload multiple images in one action; system places each new file under `{Year}/{Month}/{Day}/` using **EXIF capture datetime** when present.
- **FR-02:** System names each stored file using capture time plus a **content hash** (algorithm fixed in architecture) so names remain unique and traceable.
- **FR-03:** System detects duplicates by **file size and content checksum**; retains one copy; reports count of skipped duplicates for the operation.
- **FR-04:** Users can assign all images from an upload batch to one or more collections before or immediately after upload completes.
- **FR-05:** Default collection name for that flow is `Upload YYYYMMDD` (calendar date of upload batch initiation or documented rule); user can clear or rename before confirming.
- **FR-06:** System creates no collection and assigns no links until the user **explicitly confirms**.

### Review (bulk and shared)

- **FR-07:** Users can apply **tags** and **ratings 1–5** in bulk review.
- **FR-08:** Users can assign a collection from a **hover** or equivalent quick action on a thumbnail without opening full view.
- **FR-09:** Users can open an image in a large view using up to **90%** of the available viewport.
- **FR-10:** Users can set rating by clicking **1–5** on keyboard or by clicking stars; change saves **without** extra confirmation.
- **FR-11:** In large review view, **layout adapts** so controls remain visible from **1:1** through **21:9** aspect ratios (addresses ultrawide clipping called out in source).
- **FR-12:** Large review view shows the **entire image** (letterboxed as needed) within the 90% region for both portrait and landscape assets.
- **FR-13:** Users can obtain a **shareable URL** that opens the **same photo** in review context in a browser.
- **FR-14:** **MVP:** Anyone with a valid share URL can **open the same photo** in a **read-only** review layout (image fitted per FR-12 analog in browser; **current star rating visible**). **Changing rating from the browser is out of scope for MVP**—ratings are edited only in the desktop app. **Growth** may add token-scoped rating (or light auth) without full multi-user product scope.
- **FR-29:** Users can **reject** a photo (**soft-hide**): it **does not appear** in default bulk review, collection browsing, or filter results except within a dedicated **Rejected/Hidden** view; **rejected** photos are **excluded** from **share** and **package** selection by default.
- **FR-30:** Users can open **Rejected/Hidden** and **restore** photos to the active library (clear reject state). **Undo** for reject may also apply per product rule (time- and/or navigation-bound—architecture/product).
- **FR-31:** Users can **delete** a photo with semantics **distinct** from **Reject**; **Delete** requires **explicit confirmation** and uses **destructive** styling/flow separate from reject. Whether delete removes **DB row only**, moves files to **trash**, or **erases** bytes is **architecture-defined** and documented before release.
- **FR-32:** **MVP:** Before a share URL/token is **finalized**, the user **confirms** after a **preview** of **which asset** will be shared (minimal for single-photo). Default link semantics are **snapshot** (fixed asset identity at mint time) unless **Growth** defines **live** links with recipient disclosure. **Rejected** assets **cannot** be shared via default flow.
- **FR-33 (Growth):** Users can create **sharable packages** (multi-asset **snapshot**): **preview manifest** (count + thumbnails or IDs) before mint; optional **audience presets**; **rejected** excluded by default.

### Filtering and collections

- **FR-15:** Filter panel order is **Collection**, then **minimum rating**, then **tags**.
- **FR-16:** Default filter selections are **No assigned collection** and **Any rating**.
- **FR-17:** Users can assign selected photos to a collection from the filter workflow.
- **FR-18:** Users can create, rename/edit, and delete collections. Each collection has **name** (required) and **display date** (optional user metadata, e.g. event or album date). If the user omits **display date**, the system stores **collection creation calendar date** as the default. **Display date** does not alter member photos’ capture times or on-disk layout.
- **FR-19:** Users can assign one photo to **multiple collections** from single-photo view and create a new collection there.
- **FR-20:** Deleting a collection **removes all photo–collection relations** for that collection, then deletes the collection record.
- **FR-21:** Collections list view navigates to a **dedicated full page** (not a popup) for one collection’s photos.

### Collection detail and single-photo

- **FR-22:** Collection detail sorts photos by **capture time** (EXIF-first).
- **FR-23:** Default grouping is by **star rating** descending; empty rating groups are **omitted**; within a group, sort by capture time.
- **FR-24:** Users can switch grouping to **by day** or **by camera name**.
- **FR-25:** Single-photo view uses up to **90%** viewport, shows full image without cropping, supports rating via keyboard and stars, prev/next via on-screen arrows (mid-height, left/right of image), keyboard left/right, and swipe on touch devices.

### Metadata

- **FR-26:** System extracts and stores/display fields: camera make/model (TIFF/EXIF as needed), capture datetime, lens, exposure (shutter, aperture, ISO, compensation), focal length, GPS, resolution/DPI, orientation, flash, metering mode, white balance—when present in source file.

### Scan and import tools

- **FR-27:** Scan tool accepts `--dir`, `--recursive`, `--dry-run`; discovers supported images; extracts EXIF (minimum: capture time, camera, lens); applies dedup; copies non-duplicates into canonical storage layout; updates database—**no copy or DB write** when `dry-run=true`.
- **FR-28:** Import tool walks an uploads directory (configurable path), registers files not in DB or backfills missing EXIF on existing rows per source rules; supports `--dry-run` with summary only.

---

## Non-Functional Requirements

- **NFR-01 (Layout):** On window resize between **1024×768** and **5120×1440**, review and single-photo views keep primary navigation controls within the viewport **100%** of the time in manual UI test matrix (square, 16:9, 21:9).
- **NFR-02 (Performance):** Import/scan reports completion progress or batch logs so a **10,000-file** dry-run completes without unbounded memory growth (streaming or chunked processing—architecture defines mechanism).
- **NFR-03 (Integrity):** Duplicate decisions are deterministic: same file bytes → same checksum → same dedup outcome across upload, scan, and import.
- **NFR-04 (Observability):** Each import-like operation emits a summary: **added**, **skipped duplicate**, **updated metadata**, **failed** (with codes). When reject semantics apply to an operation class, summaries remain **consistent** between **GUI and CLI** (same categories and meanings).
- **NFR-05 (Browser share):** Shared review URL resolves in under **3 seconds** on broadband for cold load of a single photo page (excluding user network variability; measured in CI or staging).
- **NFR-06 (Security):** Share links use non-guessable tokens or equivalent (architecture); rate-limit or abuse posture documented before public deployment.
- **NFR-07 (Display scaling):** Layout acceptance (**NFR-01**) is validated at **non-100% OS UI scaling** (e.g. **125% / 150%**) on **macOS** and **Windows** at least once per major milestone, in addition to raw pixel window sizes.

---

## Traceability note

| Area              | Primary FRs           |
|-------------------|------------------------|
| Upload/dedup      | FR-01 – FR-06         |
| Bulk review/share | FR-07 – FR-14, FR-29–FR-32 |
| Reject/delete     | FR-29 – FR-31         |
| Packages (Growth) | FR-33                 |
| Collections       | FR-15 – FR-25         |
| Metadata          | FR-26                 |
| Scan/import       | FR-27 – FR-28         |

Success criteria **SC-1–SC-7** map to the above clusters for acceptance testing.
