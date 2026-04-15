# Story 3.5: Share cold-load performance and abuse posture

Status: in-progress

<!-- Sprint key: `3-5-share-cold-load-performance-abuse-posture` → this spec (`3-5-share-performance-abuse.md`). -->
<!-- create-story workflow (2026-04-14): Epic 3 Story 3.5 — NFR-05 measurable cold load + NFR-06 documented (and architecture-aligned) abuse posture. -->
<!-- Party mode create session 1/2 (2026-04-14): PM/UX/Arch/TEA simulated round; rate limiter + NFR-05 gate + canonical docs; sprint → in-progress. -->
<!-- Party mode create session 2/2 (2026-04-14): challenged session-1 “ship it” — bounded rate-limit map + keep-alive caveat in NFR-05 doc; burst-80 called out as loopback-only compromise; sprint → review. -->
<!-- Party mode dev session 1/2 (2026-04-14): Dev/TEA/Arch/UX — NFR-05 harness uses separate httptest client for /i/ (no keep-alive reuse); methodology doc aligned; sprint unchanged review. -->
<!-- Party mode dev session 2/2 (2026-04-14): challenged dev1 — two srv.Client() still share httptest’s cached Transport; gate now uses http.Client+DisableKeepAlives for true per-leg cold TCP; docs updated. -->

## Story

As a **product owner**,  
I want **measurable share performance and documented rate limits**,  
so that **we meet NFR-05/NFR-06**.

**Implements:** NFR-05, NFR-06.

## Acceptance Criteria

1. **NFR-05 (epic):** **Given** staging or CI measurement harness, **when** cold load runs, **then** median/goal aligns with **NFR-05** (document methodology and caveats). **Implementation detail (for dev):** define **cold load** in that write-up (minimum: **first** `GET /s/{token}` after fresh server+handler, through full HTML body; **recommended:** also **first** `GET /i/{token}` for same token through full image bytes). Match PRD exclusions (**user network** out of scope for the number). Use a **small web-sized** disk fixture so the gate does not measure **RAW decode** on the share path. Prefer a **`go test`** gate in `internal/share` (median over **N** trials) plus a short methodology section in **`architecture.md`** and/or **`docs/`** linked from §3.5. **Dev session 2/2:** the gate uses an `http.Client` with **`Transport.DisableKeepAlives: true`** so each leg opens a **new** connection — two `httptest.Server.Client()` values still **share** the server’s cached `Transport`, so they did **not** guarantee cold `/i/` (see `docs/share-cold-load-nfr05.md`).

2. **NFR-06 (epic):** **Given** share endpoints, **when** documented, **then** rate-limit/abuse posture for public deployment is written (NFR-06) — in-repo doc or architecture appendix. **Must include:** loopback-by-default (Story 3.2), token entropy + hash-at-rest (Stories 3.1–3.2), guidance for **edge/reverse-proxy** limits if ever exposed beyond loopback, and limits of **in-process** mitigation (single process, restart clears state, `X-Forwarded-For` trust hazards). **Session 2 note:** in-process limiter must not allow **unbounded memory** from distinct client keys; document **cap + eviction** (or equivalent) if the service could see many source IPs.

## Tasks / Subtasks

- [x] **NFR-05 methodology + harness** (AC: 1)
  - [x] Write methodology + caveats (CI variance, `-short` policy if any); link from `architecture.md` §3.5 or §8.
  - [x] Implement `internal/share` gate test: `httptest.Server` + temp DB + library file + minted link; assert median **≤ 3s** (or document threshold override for `-short`).
  - [x] Confirm `.github/workflows/go.yml` runs the test (no silent skip).
  - [x] Record sample local/CI timings in **Dev Agent Record**.
- [x] **NFR-06 documentation** (AC: 2)
  - [x] Add or extend **`architecture.md`** and/or **`docs/share-abuse-posture.md`** (single canonical home + cross-link).
  - [x] Describe operational posture for **public deployment** even if MVP remains loopback-only.
- [x] **Per-IP rate limiter (architecture §3.5 — expected implementation)** (supports AC: 2)
  - [x] Wrap `NewHTTPHandler` chain with small in-memory per-IP limiter (`internal/share`, e.g. `ratelimit.go`); `429` with **safe** body; **no** token existence oracle via rate-limit responses.
  - [x] Document limiter defaults, burst policy (allow HTML **+** image fetch without false429), and IP extraction rules in the same NFR-06 doc.
  - [x] Table-driven `httptest` in `internal/share/http_test.go` (or sibling): under-limit vs over-limit; preserve **404 discipline** from Stories 3.2–3.4.
  - [x] **Session 2:** Bound distinct IP buckets (cap + single-entry eviction) + unit test `TestIPRateLimiter_visitorsCapEviction`; document that default **burst 80** is a loopback-oriented compromise, not a public-facing posture.
- [x] **Regression** (AC: 1–2)
  - [x] `go test ./...` green.

## Risks & mitigations

| Risk | Mitigation |
|------|------------|
| CI timing flakes | Use median over **N** runs; document variance; optional build tag for “strict” timing only on main. |
| Rate limit breaks legitimate page load (HTML + image) | Set burst ≥ **2** per navigation **or** count HTML+image as one “page view” if using paired logic — document chosen policy. |
| `X-Forwarded-For` spoofing if ever proxied | Default **ignore** forwarded headers for IP key when not behind a trusted proxy; document. |
| Duplicating NFR-05 doc in too many places | Single **canonical** methodology doc + one link from architecture. |
| Per-IP limiter map grows with spoofed or scanned sources | **Cap** distinct keys (evict one arbitrary entry at overflow); still rely on edge limits off-loopback. |
| Default burst feels “too loose” for NFR-06 | Call out in posture doc: **loopback/desktop** tuning; tighten at reverse proxy for remote exposure. |

## Definition of done

- [x] AC1–AC2 satisfied; PRD **NFR-05** / **NFR-06** traceable from epics; architecture **§3.5** limiter implemented or **explicit deferral** recorded in the posture doc with rationale.
- [x] No regression in Stories **3.2–3.4** behavior (404 parity, headers, privacy).
- [x] `go test ./...` passes on CI matrix.
- [x] Story status advanced per sprint workflow after dev-story (`ready-for-dev` → `in-progress` → `review`).

## Dev Notes

### NFR text (authoritative)

- **NFR-05 (PRD):** Shared review URL resolves in under **3 seconds** on broadband for **cold load** of a single photo page (**excluding user network** variability; measured in **CI or staging**).
- **NFR-06 (PRD):** Share links use **non-guessable** tokens; **rate-limit or abuse posture documented** before public deployment.

### Architecture alignment

| Topic | Rule |
|--------|------|
| **Performance** | Minimal HTML/CSS; **web-sized** renditions; **no** on-the-fly full RAW decode on share path ([Source: `_bmad-output/planning-artifacts/architecture.md` — §3.5 NFR-05]). |
| **Abuse** | Loopback bind + token entropy + **per-IP rate limit** (in-memory) on share routes; document **reverse-proxy** limits if deployed remotely ([Source: `architecture.md` — §3.5 NFR-06]). |
| **Stack** | `net/http` handler in `internal/share`; static assets `web/share/` ([Source: `architecture.md` — §3.6, §5.1, §5.3]). |

### Previous story intelligence (3.4)

- **Tests** for share HTTP live in `internal/share/http_test.go` — extend with rate-limit cases; preserve **404 header parity** lessons (Referrer/CSP/nosniff rules are **success-path only** for HTML).
- **Handler** entry: `NewHTTPHandler` in `handler.go` — wrap or branch **before** `ServeHTTP` for limiter.
- **Privacy / security** headers must not regress when adding `429` responses (decide whether **429** responses carry same minimal safe headers as **404**; document).

### Project Structure Notes

- Keep limiter and perf gate tests **inside** `internal/share` or `internal/store` test helpers — **no** Fyne imports in share HTTP path.
- Documentation: prefer **`docs/`** for operational posture if `architecture.md` should stay high-level; otherwise appendix in `architecture.md` is acceptable.

### References

- [Source: `_bmad-output/planning-artifacts/epics.md` — Epic 3; Story 3.5 acceptance criteria]
- [Source: `_bmad-output/planning-artifacts/PRD.md` — NFR-05, NFR-06]
- [Source: `_bmad-output/planning-artifacts/architecture.md` — §3.5 Share service; §3.6 web stack; §5.3 NFR-05/06 mapping; §8 traceability]
- [Source: `_bmad-output/implementation-artifacts/3-4-share-privacy-wcag.md` — prior share hardening, `http_test` patterns]
- [Source: `internal/share/handler.go` — `/s/` and `/i/` routing]
- [Source: `.github/workflows/go.yml` — CI matrix for `go test ./...`]

## Dev Agent Record

### Agent Model Used

Party mode create 1/2 + 2/2 (simulated headless); implementation: Cursor agent.

### Debug Log References

### Completion Notes List

- `internal/share/ratelimit.go` — per-IP token-bucket wrapper; `NewHTTPHandler` uses `newShareMuxHandler` + default limiter (12 req/s refill, burst 80). Tests use `wrapRateLimitedHandler` + `newShareMuxHandler` with tight limits. **Session 2:** `maxRateLimitVisitorEntries` (4096) with single-entry eviction before insert at cap.
- `internal/share/nfr05_cold_load_test.go` — `TestNFR05_ShareColdLoadMedian` (9 trials, median ≤ 3s HTML + `/i/`); skips under `-short`.
- `internal/share/ratelimit_test.go` — 429 oracle-safety + HEAD 429 + visitor-cap eviction.
- `docs/share-cold-load-nfr05.md`, `docs/share-abuse-posture.md` — canonical methodology + NFR-06 posture; `architecture.md` §3.5 cross-links. **Session 2:** keep-alive caveat + visitor-map cap + burst-as-loopback note.
- Local `go test ./...`: `internal/share` ~1.1s (2026-04-14); full tree green.
- **Dev-story verification (2026-04-14):** `go test ./...` and `go build .` green; `go test -count=1 ./internal/share/...` green (includes NFR-05 median gate when not `-short`). Sprint `3-5-share-cold-load-performance-abuse-posture` already `review`.
- **Party dev1/2 (2026-04-14):** attempted distinct `httptest` clients for HTML vs `/i/`; **superseded** by dev2/2 (see below).
- **Party dev2/2 (2026-04-14):** NFR-05 harness — `http.Client` + `DisableKeepAlives` (httptest `Client()` shares one cached `Transport`); `docs/share-cold-load-nfr05.md` corrected.

### File List

- `internal/share/ratelimit.go`
- `internal/share/handler.go`
- `internal/share/nfr05_cold_load_test.go`
- `internal/share/ratelimit_test.go`
- `docs/share-cold-load-nfr05.md`
- `docs/share-abuse-posture.md`
- `_bmad-output/planning-artifacts/architecture.md`
- `go.mod`, `go.sum`

### Review Findings

<!-- BMAD code-review workflow (2026-04-14): headless run; patch items left open. -->

- [ ] [Review][Patch] Add an explicit **under-limit** rate-limit regression (default `NewHTTPHandler` limits): e.g. rapid sequential `GET /s/{token}` + `GET /i/{token}` both `200`, matching the story task’s under-limit vs over-limit pairing — today over-limit is covered in `ratelimit_test.go`, under-limit is only indirect via other suites [`internal/share/ratelimit_test.go`].
- [ ] [Review][Patch] Extend **NFR-06 posture** with package / multi-image behavior: parallel browser `/i/` fetches on large galleries vs default **burst 80** (loopback-oriented); point operators to edge tuning if the listener is ever off-machine [`docs/share-abuse-posture.md`].
- [x] [Review][Defer] **Visitor-map eviction** removes an arbitrary bucket at cap — a high-cardinality source-IP fanout can churn buckets; acceptable documented tradeoff for bounded memory vs LRU complexity [`internal/share/ratelimit.go:56-60`] — deferred, pre-existing design choice.
- [x] [Review][Defer] **NFR-05 gate** remains a localhost, small-fixture median test; extreme CI contention or very large libraries may need more trials or staging replay — caveats already noted in `docs/share-cold-load-nfr05.md` [`internal/share/nfr05_cold_load_test.go`] — deferred, pre-existing scope limit.

## Change Log

- 2026-04-14: BMAD code-review — two patch follow-ups (under-limit test, package/burst posture note); two deferred design/scope items; story **in-progress**; sprint synced.
- 2026-04-14: Dev-story workflow — verified AC1–AC2 against codebase and CI; regression `go test ./...` / `go build .`; story remains **review** (implementation unchanged).
- 2026-04-14: Party mode dev 1/2 — NFR-05 harness uses separate `httptest` client for `/i/`; `docs/share-cold-load-nfr05.md` caveat updated.
