# Story 3.2: Loopback HTTP server and token resolution

Status: review

<!-- Sprint key: `3-2-loopback-http-server-token-resolution` → this spec (`3-2-loopback-http-token.md`). -->
<!-- create-story workflow (2026-04-14): Epic 3 Story 3.2 — loopback bind, GET /s/{token} hash resolution, safe 404; builds on 3.1 mint + share_links. -->

## Story

As a **photographer**,  
I want **the app to serve share links locally by default**,  
So that **tokens are not unnecessarily exposed**.

**Implements:** FR-13 (technical enabler); NFR-06 baseline.

## Acceptance Criteria

1. **Loopback by default (architecture §3.5, NFR-06):** **Given** default configuration, **when** share HTTP serving is started, **then** the listener binds to **loopback only** (default host **`127.0.0.1`** for predictable clipboard URLs and to avoid `localhost` DNS/rebinding ambiguity—document if a platform **requires** a different loopback literal; **`::1`** is supported by setting `PHOTO_TOOL_SHARE_HTTP_HOST`—`internal/config` uses `net.JoinHostPort` so IPv6 literals bracket correctly). **And** binding on `0.0.0.0` / LAN-wide exposure exists **only** behind an **explicit** opt-in (`PHOTO_TOOL_SHARE_HTTP_BIND_ALL`, alongside `PHOTO_TOOL_SHARE_HTTP_HOST` / `PHOTO_TOOL_SHARE_HTTP_PORT`) **and** is accompanied by a clear security warning in code comment + user-facing copy if the UI exposes it.
2. **Resolve valid token via stored hash (architecture §3.5, NFR-06):** **Given** a request to `GET /s/{token}` where `{token}` is the **raw URL-safe token** string produced by `MintDefaultShareLink` (same encoding as mint), **when** the handler runs, **then** it computes **SHA-256** of the raw token and looks up **`share_links.token_hash`** using the same **lowercase hex** encoding as `internal/store/share.go` (`MintDefaultShareLink`). **And** the database continues to store **hash only**—no new columns for plaintext tokens. **And** the **primary** secret equality check is the **parameterized** `token_hash = ?` lookup (unique index); use `crypto/subtle.ConstantTimeCompare` **only** when comparing **multiple candidate digests in Go** or comparing **raw secrets in memory**—do not require a redundant byte compare on the hash string after SQLite already matched the row unless an implementation path materializes several candidates. **And** avoid **early-return** branches that skip the DB round-trip for “obviously bad” tokens unless timing is shown equivalent (otherwise prefer one code path: hash → query → 404 on any miss). **And** request handling **never** logs the raw token or full share URL at info level (align with Story 3.1 logging rules).
3. **Invalid token — safe 404:** **Given** a malformed path, wrong-length token, unknown hash, **or** a share row whose asset is **no longer eligible** (missing row, soft-deleted/trashed, or rejected—mirror mint eligibility), **when** the handler runs, **then** the response is **404 Not Found** with a **generic** body (e.g. plain “Not Found”) **without** revealing whether the failure was malformed token, unknown hash, or ineligible asset. **And** do not emit different error messages or headers that fingerprint validity. **And** disallowed methods (e.g. `POST`), extra path segments after the token (`/s/{tok}/extra`), or a missing `/s/` prefix **must** yield the **same** 404 body and equivalent non-revealing headers (no `405 Method Not Allowed`—method-based responses fingerprint the route). **And** a query string on `GET /s/{token}` must not change resolution logic except as explicitly documented (default: strip query for lookup; never echo token in redirects). **And** **`HEAD`** failures use the **same** status and **non-revealing** headers as **`GET`**; **`HEAD` on a miss** sends an **empty body** with **`Content-Length` equal to the GET 404 body length** so caches/clients do not see a distinct “short” error representation.
4. **Configurable port:** **Given** the server starts, **when** default port is busy or unset, **then** behavior is deterministic: default **`8765`** when `PHOTO_TOOL_SHARE_HTTP_PORT` is unset, try up to **10** successive ports on **address already in use**, then fail with actionable `slog` error—**without** falling back to non-loopback bind.
5. **Lifecycle:** **Given** the desktop app is running, **when** the user successfully mints a share link (Story 3.1 path), **then** the app ensures the loopback server is **available** for that session (start on first need or at shell init—choose one approach, document in Dev Notes). **And** shutting down the app **stops** the HTTP server (defer / `Close` on context) so ports do not leak.
6. **Regression tests:** **Given** `go test ./...`, **when** tests run, **then** new tests cover: **(a)** handler returns **404** for unknown token; **(b)** handler returns **200** (or agreed success code) for a row inserted with known hash **and** eligible asset (use `httptest` + in-memory or temp DB per existing store tests); **(c)** server `Listen` address resolves to loopback under default config (table-test or string assert on configured address); **(d)** **404 parity:** unknown token vs ineligible asset (rejected or `deleted_at_unix` set) vs malformed path (`/s/a/b`, wrong method) produce the **same** body bytes (and no distinguishing `Allow` / method-specific headers); **(e)** mint → hash → resolve round-trip in **`internal/store/share_test.go`**. Prefer **`internal/share`** tests for HTTP surface and **`internal/store`** tests for any new query API.

## Tasks / Subtasks

- [x] **Config: bind + port + LAN opt-in** (AC: 1, 4)
  - [x] Add small share-server settings (port, bind loopback vs LAN) in `internal/config` (`PHOTO_TOOL_SHARE_HTTP_HOST`, `PHOTO_TOOL_SHARE_HTTP_PORT`, `PHOTO_TOOL_SHARE_HTTP_BIND_ALL`); **default** = loopback only.
  - [x] Document env var names in Dev Notes (below).
- [x] **Store: resolve share link by raw token** (AC: 2, 3, 6)
  - [x] Add exported API `ResolveDefaultShareLink(ctx, db, rawToken)` — hashes like mint, `share_links` **JOIN `assets`**, eligible asset only; miss → `(nil, nil)`.
  - [x] Unit tests in `internal/store/share_test.go`: round-trip, unknown, rejected-after-mint, soft-delete-after-mint.
- [x] **HTTP: `net/http` server in `internal/share`** (AC: 1–6)
  - [x] Handler for `/s/{token}` single segment; wrong method/extra segments → same 404 as unknown (not `405`).
  - [x] `Loopback` + `http.Server`; LAN bind only when `PHOTO_TOOL_SHARE_HTTP_BIND_ALL` set (code comment warning).
  - [x] Valid token: minimal **200** plain stub (`Shared photo`); FR-14 HTML is Story **3.3**.
  - [x] `HEAD` mirrors `GET` on success; `HEAD` on miss matches GET 404 contract (AC3 addendum).
- [x] **App integration: start/stop + mint success UX** (AC: 5)
  - [x] `main.go` constructs `share.Loopback`, defers `Close`; `NewMainShell` / Review grid receive loopback; **ensure on first successful mint**.
  - [x] `showLoupeShareMintSuccess`: **Copy link** (loopback base + `ShareHTTPPath`) + **Copy token**; loopback failure falls back to token-only UI + `slog`.
- [x] **Observability** (AC: 2)
  - [x] `slog` on resolve/store errors and serve errors; **never** log raw tokens.

## Risks & mitigations

| Risk | Mitigation |
|------|------------|
| Timing side channels on token validation | Prefer single path: hash + query; avoid faster “reject before DB” unless benchmarked/neutralized; use `subtle.ConstantTimeCompare` only when Go compares multiple secrets; keep404 bodies identical (AC3). |
| Accidental LAN exposure | Default `127.0.0.1`; LAN requires explicit opt-in + warning. |
| `localhost` / DNS ambiguity in copied URLs | Default clipboard base `http://127.0.0.1:{port}`; document in Dev Notes. |
| Port collisions | Document default; retry next port or surface error—do not widen bind address as a “fix”. |
| Double server / race on mint | Singleflight or mutex around `Listen`; idempotent “ensure running”. |
| Valid token, ineligible asset (rejected after mint) | Resolve joins/filters asset eligibility; same 404 as unknown (AC3). |

## Definition of done

- [x] Default bind is loopback only; LAN requires explicit opt-in with warning (env + code comment).
- [x] `GET /s/{token}` resolves via `token_hash` (SHA-256 hex, mint-compatible); failures → indistinguishable generic 404 (`HEAD` miss uses empty body + matching `Content-Length`).
- [x] Mint success UI offers copyable loopback URL; server starts on mint; `Close` on app exit.
- [x] Tests: `internal/share` httptest (200, 404 parity, HEAD); `internal/store/share_test` resolve; `internal/config/share_http_test` defaults + IPv6 join.
- [x] No plaintext token in DB or logs.

## Dev Notes

### Product / architecture rules

| Topic | Rule |
|--------|------|
| **Transport** | In-process `net/http` from desktop; loopback default; LAN only with explicit opt-in ([Source: `_bmad-output/planning-artifacts/architecture.md` — §3.5]). |
| **Token model** | 32-byte random, RawURLEncoding token in URL; **SHA-256 lowercase hex** in `share_links.token_hash` ([Source: `internal/store/share.go` — `MintDefaultShareLink`; `architecture.md` — §3.5]). |
| **Constant-time** | Compare digests with `crypto/subtle` per architecture §3.5. |
| **404 hygiene** | No existence leak (AC3); aligns with NFR-06 “unguessable tokens” posture. |
| **HTML / FR-14** | Full read-only share page is Story **3.3**; this story may return a minimal 200 body. |
| **Rate limits** | Architecture §3.5 mentions per-IP limits; **detailed** abuse documentation is Story **3.5**—optional minimal stub here if trivial, otherwise defer. |
| **HEAD** | Optional: `HEAD /s/{token}` may mirror `GET` status/headers without body for future cache clients; if unimplemented, **must** still fall into the same 404 contract as `GET` (no special error bodies). |

### Epic boundary: Story 3.2 vs 3.1 / 3.3

- **3.1 (done/review):** Mint, `share_links` row, loupe preview—**token-only** success copy today.
- **This story:** HTTP server, **hash** resolution route, loopback URL in UX, shutdown lifecycle.
- **3.3:** Rich HTML, image bytes, rating visible per FR-14.

### Existing implementation to extend

- **Path helper:** `internal/share/path.go` — `ShareHTTPPath(rawURLSafeToken)`.
- **Mint + hash:** `internal/store/share.go` — `MintDefaultShareLink` + `ResolveDefaultShareLink`.
- **HTTP:** `internal/share/handler.go`, `internal/share/loopback.go`.
- **Config:** `internal/config/share_http.go` — env-driven listen + clipboard host.
- **Success dialog:** `internal/app/share_loupe.go` — `showLoupeShareMintSuccess`.
- **App lifecycle:** `main.go` (`Loopback` + defer `Close`); `internal/app/shell.go` / `review_grid.go` plumb loopback into Review.

### Testing standards

- `httptest.Server` for handlers; temp DB pattern from `internal/store/share_test.go`.
- No real browser or network flake in unit tests.
- Table-driven store tests.

### Project Structure Notes

- **`internal/share`:** HTTP mux, server wrapper, handlers (per architecture §5 tree: `internal/share` for server + templates).
- **`internal/store`:** SQL for resolution only; no Fyne imports.
- **`internal/app`:** Start/stop orchestration and clipboard URL copy.

### References

- [Source: `_bmad-output/planning-artifacts/epics.md` — Epic 3; Story 3.2 acceptance criteria]
- [Source: `_bmad-output/planning-artifacts/architecture.md` — §3.3 `share_links`; §3.5 share service; §3.10 security; §5 directory layout]
- [Source: `_bmad-output/planning-artifacts/PRD.md` — FR-13 context]
- [Source: `_bmad-output/implementation-artifacts/3-1-share-preview-snapshot-mint.md` — mint contract, `ShareHTTPPath`, logging rules, Epic boundaries]
- [Source: `internal/store/share.go` — `MintDefaultShareLink`, token encoding]
- [Source: `internal/share/path.go` — `ShareHTTPPath`]
- [Source: `internal/app/share_loupe.go` — mint success UX to update]

## Dev Agent Record

### Agent Model Used

Composer (automated party mode orchestrator + implementation)

### Debug Log References

### Completion Notes List

- Party mode **create** session **2/2** implemented loopback server, resolver, tests, mint UX (**Copy link**), and sprint status → **review**.
- Party mode **dev** session **1/2** (simulated **Amelia** / **Murat** / **Winston** / **Sally**): **Murat** pushed AC3 regression for **query strings** (must not change resolution or 404 shape); **Winston** + **Sally** insisted **OPTIONS** must not become **405** with `Allow` (fingerprint); **Winston** noted **GET 404** should expose the same **Content-Length** as **HEAD404** for proxy/cache parity — implemented in `writeNotFound`; parity tests assert **Content-Length** on all 404 cases.
- Party mode **dev** session **2/2** (simulated **Amelia** / **Murat** / **Winston** / **Sally**): **Murat** challenged session-1 coverage — **soft-delete after mint** must be indistinguishable from unknown at the HTTP layer (AC3), not only in `store` tests; **Winston** argued **TRACE** (and any future method) must stay on the uniform 404 path, not 405; **Amelia** pushed **lifecycle** tests — `EnsureRunning` idempotency, concurrent first calls, **Close** then serve again — because mutex behavior is easy to regress; **Sally** noted real users hit **port churn** rarely but **restart** after Close matters for tests and future “toggle server” UX. → `TRACE` added to 404 parity table; `TestShareHTTP_softDeleteAfterMint_404MatchesUnknown`; new `internal/share/loopback_test.go`.
- Dev-story verification (2026-04-14): `go test ./...` and `go build .` green; **AC2** alignment — `ResolveDefaultShareLink` always uses hash + DB lookup (removed empty-token short-circuit); `TestResolveDefaultShareLink_emptyToken` added.

### File List

- `internal/config/share_http.go`, `internal/config/share_http_test.go`
- `internal/store/share.go`, `internal/store/share_test.go`
- `internal/share/handler.go`, `internal/share/loopback.go`, `internal/share/http_test.go`, `internal/share/loopback_test.go`
- `main.go`
- `internal/app/shell.go`, `internal/app/review.go`, `internal/app/review_grid.go`, `internal/app/rejected.go`, `internal/app/share_loupe.go`
- `internal/app/review_test.go`, `internal/app/nfr01_layout_gate_test.go`

## Change Log

- 2026-04-14 — party mode **dev** session **2/2** (headless): challenged session-1 — **TRACE → same 404** as other disallowed methods; **HTTP-level** soft-delete-after-mint vs unknown **byte-identical** 404; **`internal/share/loopback_test.go`** (idempotent `EnsureRunning`, concurrent startup, **Close** + re-**EnsureRunning** serves **200** again); `go test ./...` green.
- 2026-04-14 — party mode **dev** session **1/2** (headless, no `agent-manifest.csv` in repo): simulated dev roundtable — **GET/HEAD 404** both send explicit **Content-Length** (`writeNotFound`); **404 parity** tests extended (**OPTIONS**, **Content-Length** assert); **`TestShareHTTP_queryStringDoesNotAffectResolution`** for AC3 query-strip contract; `go test ./...` green.
- 2026-04-14 — dev-story run: verified AC/tasks against codebase; `ResolveDefaultShareLink` empty-token path now uses same hash→query flow as other misses; store test added.
- 2026-04-14 — party mode **create** session **2/2** (simulated **Winston** / **Sally** / **Murat** / **Amelia**): challenged session-1 assumptions — **IPv6 `::1`** as explicit host (JoinHostPort + tests) vs “IPv4-only” wording; **HEAD404** must not become a shorter/different representation (Content-Length parity); **port collision** cap documented (10 tries from default **8765**); **LAN bind** = `PHOTO_TOOL_SHARE_HTTP_BIND_ALL` with clipboard still defaulting to loopback literal for same-machine copy (remote recipients out of scope for MVP); **EnsureRunning** errors stay user-safe (token-only fallback, `slog` only). → spec deltas applied; **implementation landed** (review).
- 2026-04-14 — party mode **create** session **1/2** (simulated Winston/Mary/Sally/Murat): challenged AC2 constant-time scope (SQLite primary compare; subtle only for multi-candidate/raw compare); AC3 expanded (ineligible asset, non-GET → same 404, no 405); AC1 default host `127.0.0.1` + config env naming; AC6/store tasks for 404 parity + eligibility join; risks (`localhost`, post-mint reject).
- 2026-04-14 — create-story: initial spec for Epic 3 Story 3.2 (loopback server + token resolution).
