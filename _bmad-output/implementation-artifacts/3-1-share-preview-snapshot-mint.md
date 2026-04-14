# Story 3.1: Share preview, confirm, and snapshot mint (desktop)

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->
<!-- create-story workflow (2026-04-14): Epic 3 Story 3.1 — desktop preview-before-mint, snapshot row (token hash), reject/delete eligibility; HTTP serving deferred to Story 3.2. -->
<!-- Party mode create1/2 (2026-04-14): risks/DoD/edge cases; implementation landed for migration + store mint + loupe flow. -->
<!-- Party mode create2/2 (2026-04-14): challenged TOCTOU — confirm-time loupe vs preview asset check; Cmd/Ctrl+Shift+S; `internal/share` path stub for 3.2; `loupeShareSelectionMatchesPreview` unit test. -->
<!-- Party mode dev 1/2 (2026-04-14): simulated Dev/TEA/Arch/UX — fail-closed nil loupe callback + invalid preview id; share mint error copy + log hint (AC6); tests updated. -->
<!-- Party mode dev 2/2 (2026-04-14): simulated Dev/TEA/Arch/UX — challenged “dialog text is always non-empty”: defensive mint fallback copy; soft-delete mint parity CountShareLinks; isSQLiteUniqueTokenHash table test; ShareHTTPPath token charset note; nil mint err test. -->

## Story

As a **photographer**,  
I want **to preview exactly what I share before a link exists**,  
So that **I never mint the wrong asset**.

**Implements:** FR-32; FR-29 (share side); UX-DR7.

## Acceptance Criteria

1. **No mint before confirm (FR-32, UX-DR7):** **Given** the user opens **Share** from **single-photo loupe** (the primary MVP entry point for this story), **when** they have **not** confirmed on the preview step, **then** **no** `share_links` row is written, **no** share token exists in persistent storage, and **no** share URL is placed on the system clipboard (including no auto-copy). **And** canceling or closing the preview **never** leaves a minted link behind.
2. **Snapshot row on successful mint (FR-32, architecture §3.3 / §3.5):** **Given** the user **confirms** the preview for an eligible asset, **when** mint succeeds, **then** a **`share_links`** row exists with **`token_hash`** = **SHA-256** of the raw token (hex or consistent project-wide encoding—match existing hash style in repo), **`asset_id`** set to the asset at mint time, **`created_at`** set, and optional **`payload`** JSON capturing **rating at mint** (and any other snapshot fields architecture requires for FR-14 later, e.g. content hash or rendition path placeholder if already available—keep minimal if not yet produced). **And** the **raw token** is **never** stored in SQLite (hash only).
3. **Rejected assets blocked (FR-29, FR-32):** **Given** the current loupe asset is **rejected** (`rejected = 1`), **when** the user attempts the **default** share flow from loupe, **then** share **does not** mint and the user sees a **clear, factual** message (proportionate tone per Story 2.12 patterns—use existing `userFacingDialogErrText` / helpers where appropriate). **And** no clipboard or DB side effects occur.
4. **Soft-deleted assets blocked (architecture §3.4, parity with Epic 3 eligibility):** **Given** the asset row is **soft-deleted** (`deleted_at_unix` set), **when** the user attempts the default share flow, **then** mint is blocked with user-appropriate messaging (same choke point as rejected). **And** `store.AssetEligibleForDefaultShare` (or successor) is invoked so eligibility stays **single-sourced**.
5. **Preview content (UX-DR7, PRD “explicit identity”):** **Given** the preview step is visible, **when** the user reviews it, **then** the UI makes the **asset identity** obvious for a single-photo MVP (e.g. **thumbnail or loupe-scale image** + **non-ambiguous label** such as capture date, library **relative path**, or **asset id**—choose the least misleading combination available from existing review/loupe data without leaking sensitive fields beyond what the operator already sees in-app). **And** primary actions are **Cancel** vs **Confirm / Create link** (wording aligned with PRD “confirm after preview”).
6. **Observability & errors:** **Given** mint fails (DB error, eligibility race, etc.), **when** the user has confirmed, **then** show **honest** error copy with a **next step** (retry, check library, contact logs) and **do not** partially persist a row unless the implementation uses a single transaction that rolls back on failure. **And** structured logging (`log/slog`) records mint failures **without** logging raw tokens.
7. **Regression tests:** **Given** CI runs `go test`, **when** store and app tests execute, **then** new tests cover: **(a)** successful mint inserts expected `share_links` columns; **(b)** rejected and soft-deleted assets fail eligibility; **(c)** cancel path leaves **zero** new `share_links` rows; **(d)** confirm path increases row count by one. Prefer **`internal/store`** table-driven tests for persistence; **`internal/app`** pure helpers tested headlessly (**(e)** preview vs loupe asset alignment predicate for AC8).
8. **Navigation / TOCTOU:** **Given** the loupe **Share** flow reads the row at **Share** tap time, **when** the user navigates prev/next **without** closing the preview confirm dialog, **then** **Create link** does **not** mint unless the loupe’s **current** asset id still matches the previewed asset (session 2/2 hardening). **When** they match, **then** mint targets that **asset id**. **When** they differ, **then** the user sees a factual message to close the preview and start Share again—**no** DB or clipboard side effects.

## Tasks / Subtasks

- [x] **Schema: `share_links`** (AC: 2, 7)
  - [x] Add forward-only migration under `internal/store/migrations/` (next numeric after `005_camera_meta.sql`), defining `share_links` per architecture: `id`, `token_hash` (unique), `asset_id` FK to `assets`, `created_at_unix` (or consistent timestamp column naming with existing migrations), `payload` nullable TEXT/JSON as used elsewhere.
  - [x] Wire migration in `internal/store/migrate.go` (or equivalent) and bump `schema_meta` expectations if tests assert version.
- [x] **Store: mint + eligibility** (AC: 2, 3, 4, 6, 7)
  - [x] Implement `MintDefaultShareLink` (`internal/store/share.go`): generate **32 random bytes** (`crypto/rand`), URL-safe token encoding for the path segment, compute **SHA-256** hash for storage (lowercase hex), insert row in a **single transaction** with snapshot payload (rating at mint minimum).
  - [x] Enforce eligibility **inside** the store API (transaction-scoped check mirrors `AssetEligibleForDefaultShare`) so UI cannot bypass.
  - [x] Unit tests in `internal/store/share_test.go`: happy path, rejected blocked, deleted blocked, hash never equals raw token; `DefaultShareBlockedUserMessage` smoke.
- [x] **Desktop UI: loupe share flow** (AC: 1, 3, 4, 5, 6)
  - [x] Add **Share** affordance to **loupe** chrome (`Share…` in `review_loupe.go`; flow in `share_loupe.go`).
  - [x] Keyboard shortcut: **Cmd/Ctrl+Shift+S** (same flow as **Share…**; documented in `review_loupe.go`; avoids 1–5 and **R** per UX-DR5).
  - [x] **Modal** pattern: **Preview** (`ShowCustomConfirm`) → **Create link** → `MintDefaultShareLink`. **Cancel** closes without mint or clipboard.
  - [x] After successful mint, **success** dialog with **Copy token** (explicit control; no auto-copy); **no** loopback URL until Story **3.2** (token + copy only in 3.1).
  - [x] Map errors through `userFacingDialogErrText` / informational copy for ineligible assets.
- [x] **Cross-story alignment** (AC: 2, 7)
  - [x] Resolve `TODO(Epic 3)` on `AssetEligibleForDefaultShare` by documenting mint’s transactional re-check (`reject.go`).
  - [x] **Port / base URL:** none in 3.1; Story **3.2** owns `http://127.0.0.1:{port}/s/{token}` (or agreed canonical form).

## Risks & mitigations (party mode create 1/2 + 2/2)

| Risk | Mitigation |
|------|------------|
| User navigates loupe with preview open; confirms thinking the on-screen photo is shared (2/2) | On confirm, require `currentLoupeAssetID == previewAssetID` before `MintDefaultShareLink`; otherwise informational dialog only. |
| Share shortcut clashes with rating/reject keys (2/2) | **Cmd/Ctrl+Shift+S** (same modifier pattern as delete), documented beside loupe shortcuts. |
| Raw token logged on failure | Mint errors use `slog` with `asset_id` / `err` only; never log token string. |
| Clipboard before confirm (UX-DR7) | No `Clipboard` calls until success dialog; preview copy explicitly deferred. |
| TOCTOU: reject/delete during preview | Transactional eligibility at mint; user sees `ErrShareAssetIneligible` path with gate copy. |
| `token_hash` collision | Retry insert with new token (bounded attempts) inside one transaction. |
| Modal-on-modal focus | Fyne default; manual QA if focus order regresses. |

## Definition of done (3.1)

- [x] Migration `006_share_links.sql` applied; `targetSchemaVersion` = 6; tests updated.
- [x] `MintDefaultShareLink` + `DefaultShareBlockedUserMessage` + store tests green.
- [x] Loupe **Share…** → preview → confirm → row persisted; ineligible paths show factual copy; success shows copyable token without auto-copy.
- [x] Headless unit test for AC8 alignment predicate; keyboard shortcut documented in loupe header.
- [x] Session 2/2: TOCTOU confirm gate + `internal/share` `ShareHTTPPath` stub for Story 3.2 path segment.

## Dev Notes

### Product / architecture rules

| Topic | Rule |
|--------|------|
| **FR-32 / UX-DR7** | Preview → confirm → mint; **no** token persistence or clipboard **before** confirm ([Source: `_bmad-output/planning-artifacts/epics.md` — Story 3.1; `_bmad-output/planning-artifacts/ux-design-specification.md` — Preview-before-share, Journey B]). |
| **Snapshot semantics** | Row captures **asset id at mint** + **rating at mint** in `payload`; FR-14 rendering is Story **3.3** ([Source: `_bmad-output/planning-artifacts/architecture.md` — §3.3 `share_links`, §3.5 semantics]). |
| **Token security** | **32-byte** random token, **SHA-256** hash in DB; constant-time compare on **lookup** belongs to Story **3.2** ([Source: `architecture.md` — §3.5, §3.10]). |
| **Reject / delete** | Default share **blocked** for rejected and soft-deleted assets ([Source: `architecture.md` — §3.4; `internal/store/reject.go` — `AssetEligibleForDefaultShare`]). |
| **Layering** | Fyne in **`internal/app`**; SQL and mint in **`internal/store`**; **`internal/share`** holds path helpers for 3.2+; HTTP server in 3.2 per architecture ([Source: `architecture.md` — §5, directory table). |

### Epic boundary: Story 3.1 vs 3.2

- **This story:** Desktop **preview, confirm, mint** + **`share_links` persistence** + eligibility enforcement + tests.
- **Story 3.2:** **Loopback HTTP server**, token **resolution** by hash, safe **404** behavior—**not** required to close 3.1 ACs unless an AC explicitly references serving (it does not).

### Existing implementation to extend

- **Loupe layout & actions:** `internal/app/review_loupe.go` (rating, reject, delete patterns—mirror **Importance** / caution semantics for share as **primary** or **default** action, not destructive).
- **Eligibility helper:** `internal/store/reject.go` — `AssetEligibleForDefaultShare`.
- **Error tone:** `internal/app/collection_store_err_text.go` — `userFacingDialogErrText` and related helpers (Story 2.12).
- **Migrations:** `internal/store/migrations/*.sql`, `internal/store/migrate.go`.

### Testing standards

- **Table-driven** store tests per architecture guidance; **no** real network in unit tests.
- **Headless Fyne** tests in `internal/app` where feasible; inject `fyne.App` test instance as in existing review tests.
- **Never** assert on full token strings in logs; snapshot **hash** in DB only.

### Project Structure Notes

- Align new packages with **`internal/store`** for persistence and **`internal/app`** for Fyne; avoid introducing SQL from widgets.
- **`internal/share`** — `ShareHTTPPath(rawToken)` returns `/s/{token}` for Story **3.2** full URL assembly; no HTTP in 3.1.

### References

- [Source: `_bmad-output/planning-artifacts/epics.md` — Epic 3 goal; Story 3.1 acceptance criteria]
- [Source: `_bmad-output/planning-artifacts/PRD.md` — FR-13–FR-14 context; FR-29; FR-32; Share MVP preview/confirm; snapshot default]
- [Source: `_bmad-output/planning-artifacts/architecture.md` — §3.3 `share_links`; §3.4 reject/delete; §3.5 token + snapshot; §3.6 web stack note]
- [Source: `_bmad-output/planning-artifacts/ux-design-specification.md` — UX-DR7 share assurance; preview-before-publish; Journey B]
- [Source: `_bmad-output/implementation-artifacts/2-12-empty-states-error-tone.md` — dialog / error mapping patterns]
- [Source: `_bmad-output/implementation-artifacts/2-6-reject-undo-hidden-restore.md` — loupe actions, reject semantics]
- [Source: `_bmad-output/implementation-artifacts/2-7-delete-quarantine.md` — `AssetEligibleForDefaultShare`, soft-delete exclusion]
- [Source: `internal/app/review_loupe.go` — loupe chrome and keyboard conventions]
- [Source: `internal/store/reject.go` — `AssetEligibleForDefaultShare`]

## Dev Agent Record

### Agent Model Used

Party mode create1/2 (simulated): PM / UX / Architect / TEA; orchestrator applied implementation (2026-04-14).  
Party mode create2/2 (simulated): PM / UX / Architect / Dev; TOCTOU confirm gate, shortcut, `internal/share`, alignment test (2026-04-14).  
Dev-story (2026-04-14): AC4 eligibility gate calls `AssetEligibleForDefaultShare`; preview confirm gate helper + `TestLoupeSharePreviewProceedToMint` for cancel/drift paths; `go test ./...` and `go build .` green.  
Party mode dev 1/2 (2026-04-14): `loupeSharePreviewProceedToMint` fail-closed without selection callback; `userFacingShareMintErrText`; `TestUserFacingShareMintErrText_logsHint` + invalid preview id table case.  
Party mode dev 2/2 (2026-04-14): `userFacingShareMintErrText` never empty for non-nil `err`; `TestMintDefaultShareLink_softDeletedBlocked` asserts zero `share_links`; `TestIsSQLiteUniqueTokenHash`; `ShareHTTPPath` doc (RawURLEncoding path safety); `TestUserFacingShareMintErrText_nilIsEmpty`.

### Debug Log References

### Completion Notes List

- `internal/store/migrations/006_share_links.sql`, `migrate.go` v6, `share.go`, `share_test.go`
- `internal/app/share_loupe.go`, `review_loupe.go` (Share… button)
- `reject.go`: Epic 3 TODO cleared in favor of mint transactional gate
- `store_test.go`: schema version expectations for v6; `share_links` table presence in migrate test
- `DefaultShareBlockedUserMessage` invokes `AssetEligibleForDefaultShare` before tailored copy (AC4 single-sourced gate)
- `loupeSharePreviewProceedToMint` + `TestLoupeSharePreviewProceedToMint`: cancel does not enter mint path; drift vs proceed (AC7c/AC8); nil callback / invalid preview id fail closed
- `userFacingShareMintErrText` for mint failures (AC6 log hint without token leakage)

### File List

- `internal/store/migrations/006_share_links.sql`
- `internal/store/migrate.go`
- `internal/store/share.go`
- `internal/store/share_test.go`
- `internal/store/reject.go`
- `internal/store/store_test.go`
- `internal/app/share_loupe.go`
- `internal/app/share_loupe_test.go`
- `internal/app/review_loupe.go`
- `internal/share/path.go`
- `internal/share/path_test.go`

## Change Log

- 2026-04-14 — Dev-story close: AC4 `AssetEligibleForDefaultShare` in `DefaultShareBlockedUserMessage`; preview confirm gate tests for cancel/selection drift; status → review.
- 2026-04-14 — Party mode dev 1/2: AC8 fail-closed if loupe selection callback missing; AC6 mint dialog adds log-output next step; tests extended.
