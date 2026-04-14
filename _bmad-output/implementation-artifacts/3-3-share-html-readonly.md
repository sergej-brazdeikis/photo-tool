# Story 3.3: Read-only share HTML page (image + rating)

Status: review

<!-- Sprint key: `3-3-read-only-share-html-page` → this spec (`3-3-share-html-readonly.md`). -->
<!-- create-story workflow (2026-04-14): Epic 3 Story 3.3 — HTML/CSS share page, image bytes route, snapshot rating; builds on 3.1 mint payload + 3.2 loopback handler. -->
<!-- Party mode create 1/2 (2026-04-14): simulated John/Sally/Winston/Murat — AC/tasks/risks tightened (/i/ parity, rating 0 vs unrated, mux + libraryRoot, cross-route 404 bytes). -->
<!-- Party mode create 2/2 (2026-04-14): simulated John/Sally/Winston/Murat — challenged: FR-14 “current” vs snapshot, non-color-only rating, query + path-decoding parity, nosniff/inline image, Range/ETag fingerprint risks, HTML source must not leak ids/paths. -->
<!-- Party mode dev 1/2 (2026-04-14): simulated Amelia/Murat/Sally/Winston — challenged prior impl: rating strip aria-label masked visible rating for AT; Range+404 table covered /i/ only — extend to /s/ parity. -->
<!-- Party mode dev 2/2 (2026-04-14): simulated Amelia/Murat/Sally/Winston — challenged: HEAD HTML Content-Length vs GET body; HEAD+Range on 404; landmark vs “test-only” scope. -->
<!-- Ultimate context engine analysis completed - comprehensive developer guide created -->

## Story

As a **recipient**,  
I want **to open the shared photo in a browser with correct layout**,  
So that **I can review without installing the app**.

**Implements:** FR-13, FR-14.

## Acceptance Criteria

1. **Read-only HTML page for valid token (FR-13, FR-14):** **Given** a **valid** share URL pointing at `GET /s/{token}` (same path contract as Story 3.2), **when** the response succeeds, **then** the **`Content-Type`** is **`text/html; charset=utf-8`**, the body is a **complete HTML document** (not the plain-text stub), and the page is **read-only** (no controls that change rating, tags, or library state). **And** behavior for **invalid / ineligible** tokens remains **indistinguishable** from Story 3.2 (**same generic 404** body bytes and **non-revealing** headers — no regression to 405 or existence leaks). **And** the **`/i/{token}`** image route follows the **same method + path-shape rules** as Story 3.2 for **`/s/{token}`**: only **GET/HEAD** succeed when eligible; **POST/OPTIONS/TRACE/…** and **`/i/{token}/…` extra segments** yield the **same** generic **404** (not 405, not path-specific errors). **And** a **query string** on **`/s/{token}`** or **`/i/{token}`** must follow Story 3.2 AC3 (**strip / ignore for resolution**; no redirect that echoes the token; success and failure behavior unchanged vs the same path without a query). **And** `{token}` path matching uses the same **decoded** request path rules as Story 3.2 (`net/http` **`Request.URL.Path`** is unescaped; tokens from mint are URL-safe — no double-unescape or alternate encoding that could desync hash lookup).
2. **Image fitted analogously to FR-12 (FR-14):** **Given** the HTML page, **when** it renders in a desktop browser, **then** the **shared photo** is visible **without cropping the image content** (letterboxed / **object-fit: contain** or equivalent) inside a **primary viewport region** that uses **most of the visible area** (product analog of the loupe **~90% image region** — see `loupeImageLayout` / FR-09–FR-12 in app). **And** both **portrait** and **landscape** assets display the **full image** within that region (no clipped edges at default zoom).
3. **Rating at mint visible (FR-14):** **Scope:** PRD inventory phrasing says “**current**” star rating; for share links, **architecture snapshot semantics** apply — the page shows **`payload.rating` at mint time only**, not a live poll of `assets.rating` after mint (no AJAX/refresh to “catch up” with the library). **Given** a resolved `share_links` row, **when** the page renders, **then** that **snapshot rating** is **shown clearly** with **more than color alone** (e.g. **1–5 stars** **and** a visible **numeric label** and/or text like “Unrated” / “Rating: 3” so the state is not **color-only** — minimal accessibility toward Story **3.4**). **When** the payload has **`rating: null`** or the field is **absent** (SQLite `NULL` → omitted/`null` per `shareSnapshotPayload`), **then** the page shows an explicit **neutral unrated** state (not a misleading default). **Note:** Today **`assets.rating`** is **1–5** or **NULL** (`UpdateAssetRating`); mint therefore emits **`rating` in 1–5** or omits/`null` for unrated — include **`0`** only in **unit/golden** parsing tests for forward-compatibility; **if `0` ever appears in payload**, treat as **invalid/out-of-band** and show the same **unrated** state (or document a single interpretation — prefer **unrated** to avoid implying a real zero-star rating). **And** the page **does not** offer browser-side rating edits (FR-14 MVP). **And** avoid **decorative motion** on the rating chrome (no auto-animating stars); full **`prefers-reduced-motion`** is Story **3.4**.
4. **Image bytes (same photo):** **Given** the HTML page, **when** it loads the image, **then** the browser requests a **second route** served by the same loopback server that returns **the same library file** as the **snapshot `asset_id`** (stream bytes from disk under the configured library root with **the same path-escape rules** as destructive operations — reuse or share logic with `assetPrimaryPath` / store helpers; **no** raw `rel_path` concatenation). **And** successful image responses use a correct **`Content-Type`** (from DB **`mime`** when trustworthy, else sniff only within documented safe limits), **`Cache-Control: no-store`** unless a later story explicitly relaxes caching, and **`X-Content-Type-Options: nosniff`** when serving browser-displayed image types (reduces MIME confusion if metadata is wrong). **And** serve the bytes **inline** for MVP (**no** `Content-Disposition: attachment` that forces download by default). **And** **`http.ServeContent`** (or equivalent) may emit **206 Partial Content** / **Range** behavior on **success**; that is acceptable. **On misses** (unknown token, ineligible asset, malformed path), **GET/HEAD** must remain **indistinguishable** from **`/s/{token}`** — no alternate404 bodies because the request had a **`Range`** header. **And** **HEAD** support matches Story 3.2 parity expectations for **both** HTML and image routes (no “short” error representations; success HEAD includes **`Content-Length`** consistent with GET body length where applicable). **And** for **unknown / ineligible tokens**, **GET** and **HEAD** on **`/i/{token}`** are **indistinguishable** from the same case on **`/s/{token}`** (**same** `NotFoundBody` bytes, **`Content-Length`**, and header policy — no “image-only” error strings).
5. **Mobile width usability (PRD browser targets):** **Given** a **narrow** viewport (typical phone width, e.g. ≤430 CSS px), **when** the page is viewed, **then** the **image remains the primary focus**, remains **fully visible without horizontal scroll** at default zoom, and the **rating** remains **readable** without overlapping the image in a broken way (prefer a **dedicated strip** or block **outside** the letterboxed image region rather than text drawn on top of the photo). **And** include a proper **`viewport`** meta for mobile scaling.
6. **Observability & security (carry-forward from 3.1 / 3.2):** **Given** any request, **when** logged at info or error, **then** **raw tokens** and **full share URLs** are **not** logged. **And** template / HTML output for this story **must not** embed **raw GPS**, filenames, or **EXIF-derived location** strings (defer full WCAG + alt policy to Story **3.4**, but use a **single neutral** image description such as **“Shared photo”** in the `alt` attribute to avoid filename leakage). **And** the rendered HTML **must not** leak implementation identifiers in the markup (**no** `rel_path`, **no** numeric **`asset_id`**, **no** `content_hash`, **no** raw token in `src` query params — the image URL is **`/i/{token}`** only). **Optional follow-up (Story 3.4):** `Referrer-Policy` to reduce accidental cross-origin referrer leakage; not required for loopback MVP if the page is same-origin only.

## Tasks / Subtasks

- [x] **Routing & handler wiring** (AC: 1, 4, 6)
  - [x] Register **`/s/{token}`** and **`/i/{token}`** on the **same** loopback server via **`http.ServeMux`** (or a thin wrapper) so both paths share listen config; avoid a second HTTP server.
  - [x] Extend `internal/share` HTTP surface so `GET /s/{token}` serves HTML (replacing the `Shared photo\n` stub) and add a **token-only** image route (recommended: **`GET /i/{token}`**) that reuses **`store.ResolveDefaultShareLink`** — **do not** use `/s/{token}/...` extra segments (Story 3.2 AC3 explicitly forbids them).
  - [x] Factor **token + path validation** into a shared helper (same rules for `/s/` and `/i/`: **exactly** one path segment, URL cleaning consistent with `sharePathToken`).
  - [x] Change **`NewHTTPHandler`** (and **`NewLoopback`** if needed) to accept **`libraryRoot` absolute** (or an options struct: `db`, `libraryRoot`) so image bytes can open files; update **`main.go`**, **`internal/share/http_test.go`**, **`internal/share/loopback_test.go`**, and any other call sites.
  - [x] Preserve **404 parity** for unknown tokens, ineligible assets, wrong methods, and **`HEAD`** vs **`GET`** for **both** routes (extend `internal/share/http_test.go` tables — no fingerprint regressions; include **cross-route** rows: e.g. unknown token **GET `/i/…`** body equals unknown token **GET `/s/…`**).
  - [x] Add **`/i/{token}`** **query-string** parity tests mirroring `TestShareHTTP_queryStringDoesNotAffectResolution` (200 unchanged vs no query; 404 body same with and without `?…`).
  - [x] Add a **regression** that successful HTML response body **does not contain** substrings for **`asset_id`**, **`rel_path`**, or **`content_hash`** from the fixture row (lightweight `strings.Contains` on rendered output).
- [x] **Templates & static assets** (AC: 1, 2, 3, 5, 6)
  - [x] Add `html/template` + small CSS per architecture §3.6 — prefer `web/share/` for `.html` / `.css` (or `internal/share/templates/` if embedding with `embed.FS`; stay consistent with §5.1 directory table).
  - [x] Implement layout: flex/grid **viewport-height** shell, **contain**-fitted `<img>`, typography for rating; **responsive** rules for narrow width.
  - [x] Bind template data: parsed **`payload`** rating, image **URL** path for `/i/{token}` (relative URL sufficient for same-origin).
- [x] **File streaming** (AC: 4, 6)
  - [x] Resolve `asset_id` → `rel_path` (+ `mime`) via store (new small read-only query if needed); join path with library root using **hardened** resolution.
  - [x] Stream with `http.ServeContent` or equivalent; handle **missing file** on disk as **404 identical** to unknown token (no “file missing” fingerprint if feasible — align with store-layer error mapping).
- [x] **Payload parsing** (AC: 3)
  - [x] Parse `ResolvedDefaultShareLink.Payload` JSON in **`internal/share`** (consider exporting a shared DTO from `internal/store` **or** duplicating the minimal struct with **matching JSON tags** — avoid drift; unit-test golden cases: `null`, `0`, `5`, absent field).
- [x] **Regression tests** (AC: 1–6)
  - [x] `httptest`: valid mint → **GET /s/** returns HTML containing expected substrings / `Content-Type`; **GET /i/** returns **200** with image bytes or expected type for test fixture JPEG.
  - [x] **404 parity** extended to image route and HTML route for unknown token, ineligible asset, disallowed methods, extra path segments, and **HEAD** — including **byte equality** where AC requires indistinguishable bodies.
  - [x] **HEAD success (HTML):** `Content-Length` on **HEAD** equals **GET** body length (guards a silent drift if HEAD skips rendering).
  - [x] **404 + HEAD + Range:** unknown token **HEAD** with **`Range`** on **`/s/`** and **`/i/`** stays **404**, empty body, same **`Content-Length`** as generic miss (no alternate metadata).
  - [x] **Payload golden tests** (unit): `rating` JSON variants — absent, `null`, `1`–`5`, and **`0`** if applicable — parsed into template data without drift from `shareSnapshotPayload`.
  - [x] Narrow-viewport / template tests optional (lightweight string checks); heavy visual QA manual.

## Risks & mitigations

| Risk | Mitigation |
|------|------------|
| Path traversal via `rel_path` | Reuse `assetPrimaryPath` semantics (`internal/store/delete.go`); add store-level read helper if needed. |
| **Split-brain** resolve: HTML uses share row but image skips eligibility | Both routes call **`ResolveDefaultShareLink`** (or the same helper) before streaming; never trust `asset_id` without the same join/filter as 3.2. |
| **Differential** errors: missing file on disk vs unknown token | Map disk-miss to **same** generic 404 representation as resolve-miss where feasible; no distinct body for “file gone” in MVP. |
| Huge RAW/TIFF decode in browser | MVP: serve bytes as stored; if `mime` is non-browser-safe, document graceful failure (placeholder or 404 parity — decide in impl, note NFR-05 follow-up). |
| HEAD / Content-Length drift for dynamic HTML | Render template to buffer for HEAD or compute length once; keep parity with 3.2 tests. |
| `http.ServeContent` **Range** / **Last-Modified** behavior | Accept default `ServeContent` in MVP; **do not** add headers that leak existence beyond 3.2’s bar; if CI flakes, document deterministic test pattern (full GET only). |
| **Strong ETag** correlating one on-disk file across **different** share tokens | Loopback-only MVP risk is low; document residual: if ETags are byte-identical for the same file, a recipient with two tokens could correlate — defer hardening (opaque per-token ETag or disable ETag) to **3.5** / abuse posture unless trivial to avoid. |
| **Symlink** or unexpected file type at resolved path | Reuse `assetPrimaryPath` normalization; prefer opening via resolved path; if platform allows, consider **no symlink follow** (`O_NOFOLLOW`) in a follow-up if `os.Open` behavior is too permissive — call out in implementation notes. |
| Wrong **`mime`** in DB → browser sniffing | Pair trustworthy `Content-Type` with **`nosniff`**; if type is non-image, document graceful failure (broken `<img>` vs 404 parity — pick one consistently). |
| Duplicating `shareSnapshotPayload` JSON shape | Single source of truth: export type or shared `internal/domain` snapshot DTO + tests. |
| Template loading: disk **`web/share/`** vs **`embed.FS`** | Pick one for release builds; document how tests locate templates (e.g. `testdata/` copy or embed) so CI stays hermetic. |

## Definition of done

- [x] Valid share link: browser shows **HTML** page with **letterboxed** image and **mint-time rating** (incl. explicit **unrated** when payload has no rating).
- [x] Image route serves **library file** for same token; **loopback-only** default unchanged; **`/s/`** and **`/i/`** registered on **one** server.
- [x] **404 / method / path-shape parity** holds for **both** routes, including **cross-route** indistinguishable bodies for unknown tokens.
- [x] No token/URL logging; no GPS / filename leaks in HTML for this story.
- [x] Rendered HTML contains **no** `rel_path`, **asset id**, **content_hash**, or token-in-query patterns; rating is **not color-only** (stars + label/text).
- [x] `go test ./...` green; share HTTP tests cover HTML + image + 404 parity + payload parse goldens + **`/i/`** query parity + identifier-leak checks as listed in Tasks.

## Dev Notes

### Product / architecture rules

| Topic | Rule |
|--------|------|
| **FR-12 analog** | Browser: **full image visible**, letterboxed in a **large central region** (~90% viewport analogy); PRD FR-12 + FR-14 ([Source: `_bmad-output/planning-artifacts/PRD.md` — FR-12, FR-14; Desktop/web targets]). |
| **Snapshot rating** | Display **`payload.rating` at mint**, not live DB rating — reconciles PRD “current” wording with **frozen snapshot** at open ([Source: `_bmad-output/planning-artifacts/architecture.md` — §3.5 snapshot semantics; `internal/store/share.go` — `shareSnapshotPayload`). |
| **Web stack** | `html/template` + static CSS, `net/http` ([Source: `architecture.md` — §3.6, §3.5 routes]). |
| **404 / HEAD parity** | Non-negotiable carry-forward from Story 3.2 ([Source: `internal/share/handler.go` — `writeNotFound`; `3-2-loopback-http-token.md`]). |
| **Privacy / a11y** | No raw GPS in template; neutral `alt` only — full **WCAG A** is Story **3.4** ([Source: `epics.md` — Story 3.4; `architecture.md` — §4.5]). |
| **NFR-05** | Keep HTML/CSS minimal; avoid embedding megabyte images inline ([Source: `architecture.md` — §3.5 NFR-05]). |

### Epic boundary: 3.2 vs 3.3 vs 3.4

- **3.2:** Loopback + **`ResolveDefaultShareLink`** + plain stub body.
- **3.3 (this story):** **HTML + CSS + image bytes** + rating display + mobile-friendly layout.
- **3.4:** Privacy hardening, keyboard focus, alt policy detail, zoom / `prefers-reduced-motion`.

### Existing implementation to extend

- **HTTP entry:** `internal/share/handler.go`, `internal/share/loopback.go` — `NewHTTPHandler` currently **`(*sql.DB)` only**; extend signature ( **`libraryRoot`** ) and replace **`Handler: NewHTTPHandler(...)`** in `EnsureRunning` with a mux or composite handler.
- **App wiring:** `main.go` — `share.NewLoopback(db, shareCfg)` must receive **resolved absolute library root** (same value used to open the store) so bytes can be read.
- **Resolution:** `internal/store/share.go` — `ResolveDefaultShareLink`, `ResolvedDefaultShareLink`.
- **Path safety:** `internal/store/delete.go` — `assetPrimaryPath` (pattern for library-root join).
- **Loupe reference (fit semantics):** `internal/app/review_loupe.go` — `loupeImageLayout` (~90% region comment).

### Testing standards

- **Table-driven** `httptest` in `internal/share`; temp DB + library dir patterns from `internal/store/share_test.go`.
- **No** headless browser requirement in CI for this story; manual spot-check on Mobile Safari / Chrome per PRD targets optional.

### Project Structure Notes

- Prefer **`web/share/`** for templates/CSS per `architecture.md` §5.1; **`internal/share`** owns handlers and template execution.
- **Do not** import Fyne into `internal/share`.

### References

- [Source: `_bmad-output/planning-artifacts/epics.md` — Epic 3 goal; Story 3.3 acceptance criteria]
- [Source: `_bmad-output/planning-artifacts/PRD.md` — FR-12, FR-13, FR-14; §Desktop and web targets (mobile browsers)]
- [Source: `_bmad-output/planning-artifacts/architecture.md` — §3.5 share service; §3.6 web stack; §5.1 `web/share/`; §4.5 agent rules (no GPS in HTML)]
- [Source: `_bmad-output/implementation-artifacts/3-1-share-preview-snapshot-mint.md` — `payload` rating at mint, mint transaction]
- [Source: `_bmad-output/implementation-artifacts/3-2-loopback-http-token.md` — 404 parity, HEAD, `/s/{token}`-only path shape]
- [Source: `internal/share/handler.go` — current stub 200 body]
- [Source: `internal/store/share.go` — `MintDefaultShareLink`, `ResolveDefaultShareLink`, payload JSON]
- [Source: `internal/app/review_loupe.go` — loupe 90% / letterbox semantics]

## Dev Agent Record

### Party mode create (1/2) — 2026-04-14

Roundtable (solo simulation): PM pushed **recipient clarity** and **post-mint rating drift** (out of scope — snapshot only); UX pushed **rating strip outside image** and **no overlay clutter** on mobile; Architect pushed **single mux + constructor threading** for `libraryRoot` and **shared token rules**; TEA pushed **cross-route 404 byte identity**, **forbidden methods on `/i/`**, and **payload goldens** (`0` vs `null`). Synthesis applied to AC1/3/4/5, Tasks, Risks, DoD, and Dev Notes above.

### Party mode dev (1/2) — 2026-04-14

**Amelia (Dev):** Implementation is solid on mux and parity, but the template’s **`aria-label="Rating at time of share"`** on the rating strip is a foot-gun: many ATs treat that as the **accessible name** and **de-prioritize** the visible **"Rating: N" / "Unrated"** copy — undermining AC3’s “more than color alone” intent for real users of assistive tech, not just sighted users.

**Murat (TEA):** **`TestShareHTTP_404_rangeDoesNotChangeBody`** only hit **`/i/`**. AC4 says **GET/HEAD** misses on **both** routes stay indistinguishable; **`Range`** must not mint a **206-ish** or alternate body on **`/s/`** 404s either. Extend the test to **`/s/{token}`** or you’re green while half the contract is unenforced.

**Sally (UX):** I’m fine dropping the vague strip-level label if the **numeric/text label** remains the **primary** spoken name. Prefer **`role="group"`** + **`aria-labelledby`** pointing at the **same** element that shows **"Rating: 4"** so the group’s name **is** the rating string — no duplicate or competing labels.

**Winston (Architect):** No new routes or resolution forks — this stays a **template + test** tightening. Keep **`/i/{token}`** as the only image URL shape; the **`id`** we add is a **stable hook** for Story **3.4** focus/labels without leaking library identifiers.

**Orchestrator synthesis:** (1) Replace strip **`aria-label`** with **`role="group"`** and **`aria-labelledby`** referencing the visible rating label **`id`**. (2) Subtest **`Range`** on **unknown token** for **both** **`ShareHTTPPath`** and **`ShareImageHTTPPath`**; assert **404** body **`NotFoundBody`**. (3) Assert successful HTML still contains the summary **`id`** in **`TestShareHTTP_resolve200_HTML`**.

### Party mode dev (2/2) — 2026-04-14

**Amelia (Dev):** Session 1 nailed **GET + Range** on misses, but **HEAD** success for HTML only proved “non-zero **Content-Length**.” Nothing stopped a future “fast HEAD” that skips **`Execute`** and returns the wrong length — that breaks proxies and **AC4**’s **HEAD** expectations without failing the old test.

**Murat (TEA):** Agree on HEAD length, and I’m adding **HEAD + `Range`** on **404**: weird stacks still send **Range** with **HEAD**. We already forbid **206** on misses for **GET**; **HEAD** must not sprout a different **`Content-Length`** or body behavior than a plain **HEAD** miss.

**Sally (UX):** Semantics matter before Story **3.4**: wrap the page shell in **`<main class="shell">`** so the primary content is a real landmark — same CSS hook, no layout churn, better screen-reader page structure than an anonymous **`div`**.

**Winston (Architect):** I’d take **tests-only** if **`<main>`** weren’t a one-line rename — landmarks are UX/a11y debt either way. Constraint: **do not** fork CSS; keep **`.shell`** on **`main`**.

**Orchestrator synthesis:** (1) **`web/share/share.html`**: **`div.shell` → `main.shell`**. (2) **`TestShareHTTP_HEAD_success_and_404`**: after successful **HEAD** **`/s/{token}`**, **GET** the same URL and assert **`Content-Length == len(body)`**. (3) **`TestShareHTTP_404_HEAD_withRange`**: **HEAD** unknown token with **`Range: bytes=0-1`** for **`/s/`** and **`/i/`** — **404**, empty body, **`Content-Length`** matches **`NotFoundBody`**. (4) **`TestShareHTTP_resolve200_HTML`**: expect **`<main class="shell">`**.

### Party mode create (2/2) — 2026-04-14

**John (PM):** The PRD inventory still says "**current** star rating." Without an explicit gate, someone could wire the page to **live** `assets.rating`. AC3 now **pins** snapshot-only semantics and names the PRD vs architecture tension.

**Sally (UX):** Session 1 fixed layout; stars with **no text** risk "color alone" failure. Require a **numeric or textual label** with the stars and **no decorative animation** on the rating chrome; defer full **`prefers-reduced-motion`** to Story 3.4.

**Winston (Architect):** "Same path rules" needs **query-string parity** and explicit **decoded-path** alignment with 3.2 so `ServeMux` and `Request.URL.Path` cannot drift between `/s/` and `/i/`. One shared token validator should own segment count, cleaning, and encoding assumptions.

**Murat (TEA):** **206 Range** on success is fine; **404** must stay **byte-identical** even when clients send **`Range`**. Add **HTML source** regressions: no `asset_id`, `rel_path`, or `content_hash` in the document. **Strong ETag** equality across two tokens for the same bytes is a **documented residual** unless disabled later.

**Orchestrator synthesis:** Folded **query + path decoding** into AC1; expanded AC3 (**PRD vs snapshot**, **non-color-only** rating, **`0` → unrated**, **no decorative motion**); AC4 (**nosniff**, **inline**, **Range on success only**); AC6 (**no identifier leaks** in HTML); Tasks (**`/i/`** query tests, substring regression); Risks (**ETag correlation**, **symlink**, **bad mime**); DoD updated. Spec-only **create** round; implementation follows in dev-story.

### Agent Model Used

Cursor agent (GPT-based composer)

### Debug Log References

_(none)_

### Completion Notes List

- Party dev **1/2:** rating strip accessibility — `role="group"` + `aria-labelledby` + `id="share-rating-summary"` (avoids `aria-label` masking the visible rating); `TestShareHTTP_404_rangeDoesNotChangeBody` covers **`/s/`** and **`/i/`** unknown-token **`Range`** requests.
- Party dev **2/2:** **`main.shell`** landmark; **`TestShareHTTP_HEAD_success_and_404`** asserts **HEAD** **`Content-Length`** matches **GET** HTML body length; **`TestShareHTTP_404_HEAD_withRange`** locks **HEAD+Range** miss behavior on **`/s/`** and **`/i/`**.
- Implemented a single `handler` on the loopback server: `path.Clean` + `pathTokenAfterPrefix` for `/s/` and `/i/` (one segment only, same rules as Story 3.2).
- `GET/HEAD /s/{token}` renders `web/share` embedded HTML/CSS: letterboxed `object-fit: contain`, viewport meta, rating strip with Unicode stars plus **Unrated** / **Rating: N** from mint JSON (`store.ShareSnapshotPayload` + `ParseShareSnapshotPayloadJSON`); rating `0` or out-of-range treated as unrated.
- `GET/HEAD /i/{token}` resolves share link, loads `rel_path`/`mime` via `AssetLibraryFileForShare`, opens file with `store.AssetPrimaryPath`, sets `Content-Type` / `Cache-Control: no-store` / `nosniff` for image types, streams with `http.ServeContent` (no `Content-Disposition: attachment`).
- 404/method/path/query/Range parity covered in `http_test.go`; missing-on-disk image maps to generic `NotFoundBody`.
- `NewHTTPHandler(db, libraryRoot)` and `NewLoopback(db, libraryRoot, cfg)`; `main.go` passes resolved library root.

### Change Log

- 2026-04-14: Party dev 2/2 — `<main class="shell">`; HEAD HTML length vs GET; HEAD+Range 404 parity tests.
- 2026-04-14: Party dev 1/2 — share page rating group `aria-labelledby`; Range+404 regression for `/s/` and `/i/`.
- 2026-04-14: Story 3.3 implementation — read-only HTML share page, `/i/{token}` bytes, store helpers and tests (`go test ./...`, `go build .` green).

### File List

- `main.go`
- `web/share/embed.go`
- `web/share/share.css`
- `web/share/share.html`
- `internal/share/handler.go`
- `internal/share/http_test.go`
- `internal/share/loopback.go`
- `internal/share/loopback_test.go`
- `internal/share/path.go`
- `internal/share/path_test.go`
- `internal/share/rating_view_test.go`
- `internal/store/delete.go`
- `internal/store/share.go`
- `internal/store/share_payload_test.go`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/implementation-artifacts/3-3-share-html-readonly.md`
