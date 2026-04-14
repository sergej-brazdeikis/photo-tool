# Story 4.1: Multi-asset snapshot packages with manifest preview

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->
<!-- create-story workflow (2026-04-14): Epic 4 Story 4.1 — FR-33 manifest preview, audience presets without skipping preview, rejected excluded; extends Epic 3 share/token model. -->
<!-- 2026-04-14: Party mode create session 1/2 (4-1) — simulated PM/UX/Arch/TEA; snapshot-vs-live-resolve split, MVP max size, dedupe/order contract, domain.StableDedupeAssetIDs + tests. -->
<!-- 2026-04-14: Party mode create session 2/2 (4-1) — challenged: explicit resolve fork vs ResolveDefaultShareLink JOIN; preview UI cap decoupled from mint cap; normalized member table + CHECK; preset scope UI-only; store sentinel errors; 404 enumeration parity. -->
<!-- 2026-04-14: Party mode dev session 1/2 (4-1) — simulated dev round: AC1 confirm label restates eligible count; app test for wrapped package mint sentinels in userFacingShareMintErrText; HTTP GET/HEAD 404 header parity /s vs /i/{token}/0 for unknown token. -->
<!-- 2026-04-14: Party mode dev session 2/2 (4-1) — challenged: OOB package member index vs unknown token (NFR-06 404 shape); post-reject package HTML must still list snapshot slots + member id in captions; `http_test` coverage. -->

## Story

As a **photographer**,  
I want **to share a curated set with manifest preview**,  
So that **friends see exactly the package I intended**.

**Implements:** FR-33; FR-29 (package selection exclusion); aligns with PRD Journey F and UX-DR7-style preview discipline for multi-asset scope.

## Acceptance Criteria

1. **Manifest before mint (FR-33, UX Growth):** **Given** the user starts a **package** flow from **multi-select** (e.g. Cmd/Ctrl+click selection in bulk review grid) **or** from an explicit **“use current filtered set”** action (defined in UI copy—must not silently include assets the user did not intend), **when** the preview step is shown, **then** the UI displays **asset count** **and** either **thumbnails** and/or **stable identifiers** (e.g. ordered asset ids, capture date + rel_path fragment—choose the least misleading combination consistent with single-photo share labeling in Story 3.1). **And** **no** package share token is persisted, **no** `share_links` (or successor) row for the package exists, and **no** share URL is copied **until** the user confirms after this preview.
   - **Preview list throttle (decoupled from mint cap):** **Given** the ordered manifest exceeds **100** lines (MVP UI guardrail—tune with evidence), **when** the preview renders, **then** the UI shows the **first 100** rows in stable order plus explicit copy for **remaining count** (or an equivalent virtualized list that does not materialize unbounded widgets). **And** **Confirm** still applies to the **full** deduped candidate set subject to reject filter + **500 eligible** mint cap (AC3c)—preview truncation must **not** silently mint only the visible slice.
2. **Audience preset does not skip preview (FR-33, PRD):** **Given** an optional **audience preset** control exists (e.g. labels such as close friends / wider circle / family as **accelerators** only), **when** the user selects any preset, **then** the **full manifest preview** remains **mandatory** before confirm (preset may pre-fill **non-security** metadata such as display title or tags in `payload`, but **must not** auto-mint or hide the asset list). **MVP scope pin:** presets are **desktop-local UI state** only unless/until a later story defines persisted audience policy; they **must not** imply different token strength, TTL, or server-side ACL in this story.
3. **Rejected excluded by default (FR-33, FR-29):** **Given** the package candidate set is built from selection or filter, **when** the manifest is computed, **then** **rejected** assets are **omitted** from the package **by default** (no silent inclusion). **If** the only candidates were rejected (e.g. edge case from stale selection), **then** the flow **blocks** with clear copy (reuse tone patterns from `userFacingDialogErrText` / `DefaultShareBlockedUserMessage` family). **And** at **mint** time, eligibility is re-checked **transactionally** for **every** asset in the package (mirror `assetEligibleForDefaultShareTx` discipline from `MintDefaultShareLink`).
   - **Stable order + duplicate IDs:** **Given** the UI or filter pipeline may yield **duplicate** asset ids (e.g. merged selection paths), **when** the manifest is built, **then** duplicates are removed with **first-seen order preserved** (`domain.StableDedupeAssetIDs` or equivalent); the minted package must not contain two rows for the same `asset_id`.
   - **MVP package size cap:** **Given** a confirmed package would contain **more than 500** eligible assets, **when** the user confirms mint, **then** mint **fails** with clear, factual copy (and **no** partial rows). **And** the preview step may show **total selected** vs **eligible-after-reject-filter** so the user understands why confirm might fail. *(Cap is MVP guardrail; raise only with perf/UI evidence.)*
4. **Snapshot package mint:** **Given** the user **confirms** after preview, **when** mint succeeds, **then** persistence captures a **fixed** ordered set of **eligible** asset ids **as of mint time** (snapshot semantics consistent with FR-32), with **token_hash** storage matching the existing share model (SHA-256 of raw token, **never** store raw token in SQLite). **And** mint is **all-or-nothing**: either the full package row(s) commit or none (no partial package on failure).
5. **Recipients see fixed snapshot:** **Given** a package link is opened via the existing loopback/share HTTP path (extend as needed), **when** the token resolves, **then** the **package index** lists the **minted ordered members** from persistence even if an asset is **later rejected or soft-deleted** (snapshot membership is **not** re-derived from live `assets` eligibility flags). **When** a recipient requests **bytes** for a member, **then** behavior matches **single-asset** `/i/{token}` posture: **404** when the file is unavailable or the asset row is ineligible (deleted/rejected), without leaking whether the slot existed at mint time. **And** HTML/read-only presentation reuses **`web/share`** / `internal/share` patterns from Epic 3 where possible (package gallery or index page; **WCAG** focus order documented for multi-thumb grids).
   - **Resolve fork (challenged in session 2/2):** **Given** `ResolveDefaultShareLink` joins `assets` with live eligibility, **when** implementing packages, **then** recipient **index** resolution **must** use a **separate** store/API path (e.g. `ResolvePackageShareLink` + member query) that selects the parent row by `token_hash` **without** requiring the parent `asset_id` to pass live eligibility, then loads **ordered** member ids from persisted package rows. **Do not** “fix” package listing by weakening single-asset rules—keep two explicit code paths.
   - **404 / enumeration parity (NFR-06):** **Given** an invalid token or failed lookup, **when** the server responds, **then** status and **generic** body/headers match **single-asset** unknown-token behavior (no extra headers or JSON fields that reveal whether the token was a package, member counts, or which leg failed).
6. **No mint before confirm (parity with FR-32):** **Given** the user cancels or closes the preview dialog, **when** the flow ends, **then** **zero** new package share rows exist and **no** clipboard side effects occur.
7. **Regression tests:** **Given** `go test ./...`, **when** store and app tests run, **then** tests cover: **(a)** manifest filtering drops rejected ids; **(b)** mint refuses ineligible ids inside the transaction; **(c)** happy path inserts expected schema for a multi-asset package; **(d)** cancel path leaves row counts unchanged; **(e)** preset selection does not bypass preview (assertable via API/UI test hook or documented pure helper); **(f)** duplicate input ids mint a single position per asset; **(g)** package resolve returns member list after a member is rejected post-mint (index still lists; `/i/` or per-member image route 404s as for single share); **(h)** `errors.Is` mapping for `ErrPackageTooManyAssets` / `ErrPackageNoEligibleAssets` in mint failure paths (distinct from `ErrShareAssetIneligible`); **(i)** HTTP tests: invalid package token response shape matches invalid single-share token (no package-specific leakage); **(j)** HTTP tests: **valid** package token with **out-of-range** member index returns the **same** generic 404 body/headers as an unknown token (no “valid token, bad slot” oracle). Prefer **`internal/store`** tests for persistence; **`internal/domain`** tests for dedupe/order; **`internal/app`** headless tests for selection → manifest derivation where feasible; **`internal/share`** `httptest` for (i)–(j).

## Tasks / Subtasks

- [x] **Schema: package representation** (AC: 4, 5, 7)
  - [x] Add forward-only migration `007_*.sql` (or next free number): **prefer normalized** `share_link_members(share_link_id, position, asset_id, …)` (or equivalent name) with FK to `share_links` + `assets`, plus `share_links.link_kind` (`single`|`package`) and **`asset_id` nullable on package rows** with a SQLite `CHECK` enforcing `(link_kind='single' AND asset_id IS NOT NULL) OR (link_kind='package')`. Avoid storing the full ordered id list only in JSON unless a spike proves otherwise. Bump `targetSchemaVersion` in `internal/store/migrate.go`.
  - [x] Document chosen shape in package mint API godoc (single upgrade path from Story 3.1 single-asset rows). **Constraint:** package **member listing** for recipients must not depend on `JOIN assets ... rejected=0` for snapshot rows (see AC5); eligibility checks apply to **image bytes**, not to “is this id still in the package”.
- [x] **Store: mint package + resolve** (AC: 3, 4, 5, 7)
  - [x] Implement `MintPackageShareLink` (name as you prefer) in `internal/store/share.go` (or colocated file): token generation **consistent** with `MintDefaultShareLink` (32-byte URL-safe, SHA-256 hash storage, bounded retry on hash collision).
  - [x] Apply `domain.StableDedupeAssetIDs` (or equivalent) at the manifest boundary **before** reject filtering and mint.
  - [x] Enforce **AC3c** max (500) inside the mint transaction after eligibility filtering; return **`store.ErrPackageTooManyAssets`** (distinct from per-asset `ErrShareAssetIneligible`).
  - [x] When the candidate set is non-empty before filtering but **all** ids are rejected/deleted/missing after eligibility, return **`store.ErrPackageNoEligibleAssets`** (maps to the same tone family as `DefaultShareBlockedUserMessage`).
  - [x] Transaction: verify **each** asset id eligible; build snapshot payload (per-asset rating or shared package payload—align with existing `ShareSnapshotPayload` evolution; add explicit **`kind`** or version field in JSON if the payload becomes polymorphic).
  - [x] Add **`ResolvePackageShareLink`** (or equivalent) for package tokens **plus** keep `ResolveDefaultShareLink` for single-asset semantics; HTTP `GET /s/{token}` (or documented route) branches on kind. **Do not** satisfy package index by removing the assets JOIN from the single-asset resolver.
  - [x] Unit tests: rejected filtered before mint; transactional reject mid-mint; collision retry; count of child rows matches manifest; post-mint reject does not remove member from package resolve.
- [x] **HTTP + HTML: package recipient view** (AC: 5, 7)
  - [x] Extend `internal/share` mux/handlers to serve a **read-only** package page (gallery or list) using minted ordering; preserve NFR-06 / privacy headers posture from Epic 3.
  - [x] Reuse or extend templates under `web/share/`; no raw GPS; WCAG patterns from Story 3.4 where applicable.
- [x] **Desktop UI: entry + preview + preset** (AC: 1, 2, 6, 7)
  - [x] Entry: from **bulk review** when `len(SelectedAssetIDs()) > 0` and/or **“Share filtered set…”** (explicit)—`internal/app/review.go` / `review_grid.go` patterns; do **not** offer package mint from **Rejected** grid for default flow (FR-29).
  - [x] Preview dialog: scrollable manifest (thumbnails via existing thumbnail/cache helpers if practical); **Confirm create package link** vs **Cancel**; map errors via Story 2.12 helpers.
  - [x] Audience preset: `Select` or `RadioGroup` that **only** affects accelerators / metadata—**no** skip of preview step (testable).
  - [x] After success: same **explicit copy** affordance as single share (loopback URL + token per Story 3.2 conventions); **no** auto-copy unless product already allows opt-in for single share.
- [x] **Cross-story compatibility** (AC: 4, 5)
  - [x] Existing **single-asset** `share_links` rows and `GET /s/{token}` behavior **remain** valid; package tokens resolve to new branch without breaking single-photo links.

## Risks & mitigations

| Risk | Mitigation |
|------|------------|
| Large selection → UI/memory blowup in preview | Cap manifest preview page size with “showing first N” + total count, or virtualized list; still mint **full** confirmed set only if product confirms—otherwise enforce max package size with clear error. |
| Filter vs selection ambiguity | Two explicit actions: “Share selection” vs “Share all matching current filters”; document which query/store API defines the filtered id set (`ListAssetsForReview` with current `domain.ReviewFilters`). |
| Schema migration breaks006 assumptions | Prefer additive migration; keep single-asset rows unchanged; use `kind` or NULLable column with CHECK constraint. |
| TOCTOU: selection changes during preview | On confirm, re-read selection or freeze ids at preview open—document chosen rule; transactional eligibility at mint catches drift. |
| Overloading `ResolveDefaultShareLink` for packages | Keep **two** resolver entry points; add `ResolvePackageShareLink` + ordered member load; batch or lazy-fetch metadata for thumbnails to avoid N+1 storms. |
| Recipient index uses live eligibility JOIN | **Violates AC5** — members disappear after reject; use persisted manifest/child table for listing; keep strict eligibility only for bytes. |
| 500-cap UX feels arbitrary | Surface count in preview + link to “reduce selection”; log cap hits at info level for tuning. |
| Preview shows 100 rows but user thinks that is the full package | Always show **total eligible** + **visible window** copy; Confirm must restate full count before mint (AC1 throttle). |
| SQLite `CHECK` + nullable `asset_id` migration surprises | Prototype migration on copy of prod-shaped DB; verify legacy single-row inserts unchanged. |
| Package vs single 404 drift leaks link type | Centralize unknown-token response builder; `httptest` parity per AC5 enumeration bullet. |

## Definition of done

- [x] Migration applied; `targetSchemaVersion` updated; `go test ./...` green.
- [x] Package mint + resolve + minimal HTML package page implemented; single-asset shares unaffected; package resolve is **not** implemented by reusing `ResolveDefaultShareLink` JOIN semantics.
- [x] Preview → confirm → mint; preset never skips preview; rejected excluded by default with tests; duplicate-id + post-mint reject snapshot tests (`internal/domain` + `internal/store`).
- [x] Dev Agent Record filled after implementation (this story spec is pre-dev).

## Dev Notes

### Product / architecture rules

| Topic | Rule |
|--------|------|
| **FR-33** | Multi-asset **snapshot**; **manifest preview** (count + thumbnails or ids); **audience presets** are accelerators only ([Source: `_bmad-output/planning-artifacts/epics.md` — Story 4.1; `PRD.md` — Journey F, FR-33]). |
| **FR-29 / FR-32 parity** | Rejected **excluded** from default package construction and from mint eligibility; preview-before-mint ([Source: `architecture.md` — §3.4, §3.5; `epics.md` — FR-29 note on packages]). |
| **Growth schema** | “Extend `share_links` (or sibling table) with manifest JSON and multi-asset resolution” ([Source: `architecture.md` — §3.1 item 9, §3.3 `share_links` bullet). |
| **Security** | Same token model as Epic 3: random raw token, **SHA-256** hash only in DB; constant-time compare on lookup; never log raw token ([Source: `architecture.md` — §3.5; `internal/store/share.go`). |
| **Layering** | Fyne in **`internal/app`**; SQL in **`internal/store`**; HTTP/templates in **`internal/share`** + `web/share` ([Source: `architecture.md` — §5). |

### Existing code to extend

- Single-asset mint + eligibility: `internal/store/share.go` — `MintDefaultShareLink`, `assetEligibleForDefaultShareTx`, `ResolveDefaultShareLink`, `ShareSnapshotPayload`, package mint sentinels `ErrPackageTooManyAssets` / `ErrPackageNoEligibleAssets` (wired by `MintPackageShareLink` in dev).
- Manifest dedupe helper: `internal/domain/share_package_manifest.go` — `StableDedupeAssetIDs` (Story 4.1; wire in app/store at implementation time).
- Review multi-select: `internal/app/review_grid.go` — `SelectedAssetIDs`, `toggleSelected`.
- Share UI patterns: `internal/app/share_loupe.go`, `review_loupe.go` (mirror preview/confirm **modal** discipline).
- HTTP loopback: `internal/share` (Epic 3 stories3.2–3.5).
- Migrations: `internal/store/migrations/006_share_links.sql`, `migrate.go`.

### Testing standards

- Table-driven store tests; **no** real network in unit tests (httptest for handlers).
- Headless Fyne tests where patterns exist in `internal/app/review*_test.go`.
- Never assert raw tokens in logs.

### Project Structure Notes

- Keep package types and JSON payloads in **`store`** or **`domain`** as appropriate; avoid SQL from Fyne widgets.
- If `ShareSnapshotPayload` grows, version or discriminate `kind` in JSON explicitly for future template compatibility.

### References

- [Source: `_bmad-output/planning-artifacts/epics.md` — Epic 4 goal; Story 4.1 acceptance criteria]
- [Source: `_bmad-output/planning-artifacts/PRD.md` — Journey F; FR-29; FR-33; sharable packages wording]
- [Source: `_bmad-output/planning-artifacts/architecture.md` — §3.1 Growth packages; §3.3 share_links; §3.4–§3.6 share stack]
- [Source: `_bmad-output/planning-artifacts/ux-design-specification.md` — Journey F; preview-before-publish for packages; audience presets]
- [Source: `_bmad-output/planning-artifacts/implementation-readiness-report.md` — FR-33 coverage]
- [Source: `_bmad-output/implementation-artifacts/3-1-share-preview-snapshot-mint.md` — preview/confirm/mint patterns, payload, eligibility]
- [Source: `_bmad-output/implementation-artifacts/3-2-loopback-http-token.md` — token resolution HTTP]
- [Source: `_bmad-output/implementation-artifacts/3-3-share-html-readonly.md` — HTML serving patterns]
- [Source: `internal/store/share.go` — current mint/resolve implementation]
- [Source: `internal/app/review_grid.go` — bulk selection]

### Git intelligence summary

- Recent **committed** history in this clone is sparse; Epic 3/4 behavior should be validated against the **current working tree** (`internal/store/share.go`, `internal/share`, `internal/app/review*.go`) rather than only `git log`.

## Dev Agent Record

### Agent Model Used

Cursor dev agent — BMAD dev-story workflow, 2026-04-14.

### Debug Log References

_(none)_

### Completion Notes List

- Migration `007_share_packages.sql`: `share_links` recreated with `link_kind` + nullable `asset_id` + CHECK; `share_link_members` for ordered snapshot ids; `targetSchemaVersion` = 7.
- Store: `PackagePrepareEligibleForMint`, `MintPackageShareLink` (dedupe + transactional eligibility + 500 cap), `ResolvePackageShareLink` (no live-asset JOIN for index); `ShareSnapshotPayload` extended with `kind` / display metadata; `ResolveDefaultShareLink` requires `link_kind='single'`.
- HTTP: package `GET /s/{token}` after package resolve; member bytes `GET /i/{token}/{position}`; same 404 body/headers for unknown tokens (parity test).
- Review UI: “Share selection as package…” / “Share filtered set as package…”; preview caps at 100 rows with copy; audience `Select` only affects JSON metadata; success uses existing loopback copy pattern. Rejected grid unchanged (no package entry).
- Regression tests in `internal/store/share_test.go`, `internal/share/http_test.go`, `internal/app/share_package_flow_test.go`; `store_test` schema expectation bumped to 7 + `share_link_members` table.
- Dev session 1/2: `share_package_flow.go` confirm button shows `Create package link (N eligible)`; `share_loupe_test.go` — `TestUserFacingShareMintErrText_wrappedPackageSentinels`; `http_test.go` — invalid-token parity extended to `Content-Type` / `Cache-Control` / `Content-Length` and HEAD 404 legs.
- Dev session 2/2: `http_test.go` — `TestShareHTTP_packageMemberIndexOutOfRange404MatchesUnknown` (GET/HEAD vs bogus token); `TestShareHTTP_packageHTMLAndSnapshotImage404AfterReject` asserts package HTML after reject still references member image path and snapshot caption id (AC5 index vs bytes).

### File List

- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `internal/store/migrations/007_share_packages.sql`
- `internal/store/migrate.go`
- `internal/store/share.go`
- `internal/store/share_test.go`
- `internal/store/review_query.go`
- `internal/store/store_test.go`
- `internal/share/handler.go`
- `internal/share/http_test.go`
- `internal/share/path.go`
- `internal/share/path_test.go`
- `internal/app/review.go`
- `internal/app/share_loupe.go`
- `internal/app/share_package_flow.go`
- `internal/app/share_package_flow_test.go`
- `web/share/embed.go`
- `web/share/share.css`
- `web/share/share_package.html`
- `_bmad-output/implementation-artifacts/4-1-multi-asset-snapshot-packages.md`

## Change Log

- **2026-04-14:** Implemented Story 4.1 — package share schema, mint/resolve, package HTML + `/i/{token}/{n}` bytes, Review package preview flow, tests; story and sprint status set to **review**.
