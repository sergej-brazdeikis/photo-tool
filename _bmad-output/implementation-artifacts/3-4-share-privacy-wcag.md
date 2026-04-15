# Story 3.4: Share page privacy and WCAG 2.1 Level A

Status: in-progress

<!-- Sprint key: `3-4-share-page-privacy-wcag-2-1-level-a` → this spec (`3-4-share-privacy-wcag.md`). -->
<!-- Party mode dev session 1/2 (2026-04-14): simulated Dev/TEA/UX/Arch — 404 must not emit Referrer-Policy (AC1 successful-only); skip-link `:focus` + `:focus:not(:focus-visible)` reset; `TestShareHTTP_404_noReferrerPolicyHeader` + inlined CSS substring. -->
<!-- Party mode create session 2/2 (2026-04-14): challenged session-1 “HTML-only” referrer stance — /i/ 200 now mirrors Referrer-Policy; tests assert inlined reduced-motion CSS; json.Unmarshal ignore-unknown-keys called out in code; future “dump payload to template” risk added. -->
<!-- Party mode dev session 2/2 (2026-04-14): simulated Murat/Amelia/Sally/Winston — CSP + nosniff on successful HTML only (404 parity with Referrer); `tabindex="-1"` on `<main>` for skip-link focus target; tests lock CSP/X-CTO absent on 404. -->
<!-- create-story workflow (2026-04-14): Epic 3 Story 3.4 — privacy (no GPS/location UI), WCAG A focus/alt/contrast, 200% zoom, prefers-reduced-motion; builds on 3.3 HTML/CSS + handlers. -->
<!-- Ultimate context engine analysis completed - comprehensive developer guide created -->

## Story

As a **recipient**,  
I want **a page that respects privacy and basic accessibility**,  
So that **I feel safe viewing shared photos**.

**Implements:** FR-14 (constraints); UX-DR11–UX-DR12; PRD domain requirements.

## Acceptance Criteria

1. **No raw GPS or location panels (PRD domain + UX-DR11):** **Given** a successful share page render (`GET /s/{token}` HTML), **when** the response body, inline scripts, embedded JSON, `meta` tags, and server-injected debug/comment blocks are inspected (view source + network), **then** **raw GPS coordinates** (e.g. decimal lat/long), **map embeds**, **“location” / EXIF geo panels**, and **structured location dumps** are **absent**. **And** this holds even if the library stores GPS in SQLite for desktop use — the **share template and payload binding** must not surface it. **And** if `share_links.payload` JSON is extended later, **do not** emit location fields into HTML without an explicit product decision and sanitization pass (default: **omit**). **And** (session 2/2 — transport hygiene) **successful** **`GET`/`HEAD`** responses for **`/i/{token}`** (image bytes) include **`Referrer-Policy: no-referrer`**, **matching** the HTML route, so direct image hits do not adopt a looser referrer posture than the document.
2. **Keyboard + visible focus (WCAG 2.1 Level A, UX-DR11):** **Given** keyboard-only navigation, **when** the user moves focus through the page (`Tab` / `Shift+Tab`), **then** every **keyboard-operable** control (including any **skip link**, in-page links, or buttons added now or later) receives a **visible** `:focus-visible` (or equivalent) indicator with **≥ 2px** contrast against adjacent backgrounds (treat as **non-negotiable** for MVP; align with UX token “focus ring” intent in `ux-design-specification.md`). **And** focus order follows a **logical reading order** (skip link → primary landmark content → rating summary region). **Note:** The current MVP template may have **no** interactive controls besides a skip link; if the page is truly static, **add** a **skip link** (“Skip to main content”) that moves focus to `<main>` so the AC is testable and matches WCAG **2.4.1 Bypass Blocks** / **2.4.7 Focus Visible** intent.
3. **Neutral alt policy (UX-DR11):** **Given** no owner-authored caption exists in the snapshot payload (today’s `ShareSnapshotPayload` has **rating only**), **when** the page renders the primary `<img>`, **then** **`alt` text** is exactly the **neutral** phrase **`Shared photo`** (or the same single canonical string used in Story 3.3 — **do not** branch per asset). **And** **`alt` MUST NOT** be auto-filled from **filename**, **`rel_path`**, **EXIF** **Artist/Title/Description**, or other derived metadata unless a future story adds an **explicit** user caption field to the snapshot with review rules. **And** decorative star glyphs remain **`aria-hidden="true"`** with the **visible rating text** exposed as today (Story 3.3 `aria-labelledby` pattern preserved).
4. **200% zoom — primary path usable (UX-DR12):** **Given** browser zoom at **200%** (or OS scaling equivalent documented in test notes), **when** the user views the share page on a **typical desktop width** and a **narrow** width (e.g. ≤430 CSS px, per Story 3.3 mobile target), **then** the **primary reading path** — **full photo (letterboxed)** + **rating summary** — remains **usable** without **broken overlap** or **loss of essential text**. **And** **horizontal scrolling** on that primary path is **avoided** where **1.4.10 Reflow** applies (fixed chrome may scroll as a last resort, but image+rating must not require side-scroll to read).
5. **Reduced motion (UX-DR12):** **Given** `prefers-reduced-motion: reduce`, **when** the page renders, **then** **non-essential** motion is suppressed — specifically: **no** auto-animated rating chrome, **no** parallax, **no** looping decorative transitions. **And** essential layout (static letterbox) remains functional. **Implement** via `@media (prefers-reduced-motion: reduce)` in `web/share/share.css` (or equivalent) setting `animation: none` / `transition: none` for decorative rules, or guarding any future transitions.

**Privacy / hardening headers (not a substitute for AC1):** Set **`Referrer-Policy: no-referrer`** on **successful** **`GET`/`HEAD`** for **`/s/{token}`** and **`/i/{token}`**. On **successful HTML only**, also send **`Content-Security-Policy`** (no scripts; same-origin **`img-src 'self'`** for `/i/{token}`; **`style-src 'unsafe-inline'`** for the inlined sheet), **`X-Content-Type-Options: nosniff`**, and keep **no** CSP/nosniff/Referrer on **404** (parity with Story 3.2–3.3 metadata discipline). Story 3.3 treated Referrer on HTML as optional; **3.4 locks** Referrer on both routes **and** adds HTML CSP + nosniff (dev session 2/2).

## Tasks / Subtasks

- [x] **Privacy audit — HTML + handler** (AC: 1)
  - [x] Trace all template fields into `web/share/share.html` / Go template structs — confirm **no** GPS, filename, `asset_id`, `rel_path`, `content_hash`, or raw token in markup (extend Story 3.3 substring tests if new fields appear).
  - [x] Confirm `ShareSnapshotPayload` / JSON binding cannot inject location strings; document **denylist** for future payload keys (`latitude`, `longitude`, `gps`, `location`, …) in Dev Notes.
  - [x] Add **`Referrer-Policy: no-referrer`** on successful **`/s/`** and **`/i/`**; **`Content-Security-Policy`** + **`X-Content-Type-Options: nosniff`** on successful **`/s/`** only; extend `internal/share/http_test.go` (headers, malicious payload `TestShareHTTP_HTML_extraPayloadJSON_doesNotLeakGeo`, 404 must not emit CSP/nosniff/Referrer).
- [x] **Keyboard + focus visibility** (AC: 2)
  - [x] Add **skip link** as first focusable element in `share.html`; target **`id`** on `<main>` (e.g. `id="share-main"`) with **`tabindex="-1"`** so activating the skip link moves focus into the landmark; visually hidden until focused (accessible CSS pattern).
  - [x] Add **`:focus-visible`** styles in `share.css` for in-page controls (`.shell :focus-visible`) + skip-link focus ring (`box-shadow` ≥ 2px).
  - [x] **httptest** checks: skip link + main `id` in `internal/share/http_test.go`.
- [x] **Alt + rating semantics** (AC: 3)
  - [x] Lock `alt="Shared photo"` in template — regression via `TestShareHTTP_resolve200_HTML`.
  - [x] Preserve **`role="group"`** + **`aria-labelledby="share-rating-summary"`** from Story 3.3; ensure **`id`** remains unique and stable.
- [x] **200% zoom + reflow** (AC: 4)
  - [x] Review `share.css` for **fixed** heights/widths that break at 200% zoom; prefer **flex** + **`min()`** patterns already used; adjust `rating-strip` typography/`max-width` if text clips.
  - [x] Document **manual QA matrix** (browser + zoom + narrow width) in Dev Agent Record — CI cannot replace visual zoom checks.
- [x] **prefers-reduced-motion** (AC: 5)
  - [x] Add `@media (prefers-reduced-motion: reduce)` block; `animation`/`transition` neutralized (incl. explicit `none` +0.01ms fallback for stubborn rules).
- [x] **Regression** (AC: 1–5)
  - [x] `go test ./...` green; `http_test` asserts Referrer-Policy on HTML + image, inlined `prefers-reduced-motion` substring.

## Risks & mitigations

| Risk | Mitigation |
|------|------------|
| Payload JSON grows with EXIF/geo fields | **Omit from template**; unit test that HTML never contains coordinate-like patterns if fixtures add bad data (lightweight regex or substring denylist in test). |
| HTML CSP too strict for a future same-origin asset | Relax **`img-src`** / add **`font-src`** only with architecture review; default stays minimal. |
| Developer binds **raw payload JSON** or `template.JS` dump into HTML “for debugging” | **Forbidden** for share templates without product review; `encoding/json` already **drops unknown keys** on `ShareSnapshotPayload` — do not bypass with generic `map`/`RawMessage` in the render path. |
| Skip link breaks layout | Use **visually-hidden-until-focus** pattern; no layout shift for mouse users. |
| Focus ring fails contrast on dark bg | Use **light** outline (`#fff` or `Highlight`) + **offset**; verify against **WCAG 2.1 1.4.11** non-text contrast for UI components where applicable. |
| False sense of “WCAG done” | Story targets **Level A** for this page only; document **manual** zoom + keyboard checks; optional **axe** in CI is follow-up unless already wired. |

## Definition of done

- [x] AC1–AC5 satisfied; PRD **no raw GPS on web** and UX-DR11/12 traceable.
- [x] No identifier or path leaks regressions vs Story 3.3 (`TestShareHTTP_HTML_doesNotLeakIdentifiers`, geo payload test).
- [x] `go test ./...` passes; **manual** note for **200% zoom** documented in Dev Agent Record (visual QA matrix).
- [x] Story status → **review** after dev-story / QA sign-off.

## Dev Notes

### UX-DR11 / UX-DR12 (authoritative phrasing)

From `epics.md`: **UX-DR11** — Share page **WCAG 2.1 Level A**: focus, labels, contrast; **no raw GPS** on web; neutral **alt** without leaking EXIF/filename. **UX-DR12** — **200% zoom** primary path usable; **`prefers-reduced-motion`** for non-essential motion.

### Product / architecture rules

| Topic | Rule |
|--------|------|
| **PRD domain** | Shared web page **must not** display **raw GPS**, map embeds, or EXIF location panels; desktop may show full metadata ([Source: `_bmad-output/planning-artifacts/PRD.md` — Personal media / SC-3; FR-14]). |
| **Architecture** | Strip **raw GPS** in **share template**; desktop may show full EXIF ([Source: `_bmad-output/planning-artifacts/architecture.md` — §3.5 Privacy; §4.5 Agent MUST — Share HTML must not emit raw GPS]). |
| **Web stack** | `html/template` + CSS under `web/share/`; `net/http` handlers in `internal/share` ([Source: `architecture.md` — §3.6, §5.1]). |
| **Snapshot** | Rating from mint payload only (Story 3.3); do not add live DB reads for metadata on this story. |

### Epic boundary: 3.3 vs 3.4

- **3.3:** Read-only layout, image route, snapshot rating, baseline neutral alt, **main** landmark.
- **3.4 (this story):** **Privacy hardening** (GPS/location UI absent, Referrer locked + HTML CSP/nosniff on success), **WCAG A** focus/contrast/skip link + main focus target, **alt policy** locked, **200% zoom** + **reduced-motion** CSS.

### Existing implementation to extend

- **Templates / CSS:** `web/share/share.html`, `web/share/share.css`, `web/share/embed.go` (if template constants change).
- **HTTP:** `internal/share/handler.go` — **`Referrer-Policy: no-referrer`** on **successful** `/s/` **and** `/i/`; **`ShareHTMLContentSecurityPolicy`** + **`nosniff`** on **successful** `/s/` only; do not weaken 404 parity from Stories 3.2–3.3.
- **Tests:** `internal/share/http_test.go` — HTML golden / substring tests from Story 3.3.
- **Payload:** `internal/store/share.go` — `ShareSnapshotPayload` (rating only today); `ParseShareSnapshotPayloadJSON` **ignores unknown JSON keys** — they must not be surfaced via HTML unless a future story adds a vetted field and sanitization.

### Testing standards

- Prefer **table-driven** `httptest` consistent with `internal/share/http_test.go`.
- **Manual:** Keyboard-only pass (skip link + focus ring visibility); **200% zoom** on Chrome/Safari/Firefox at least one engine.
- Optional: local **axe** CLI against saved HTML fixture — not required unless project already automates it.

### Project Structure Notes

- Keep **all** share HTML/CSS under `web/share/` per architecture §5.1; **no** Fyne imports in `internal/share`.
- **Do not** add client-side JS bundles for MVP unless required for an AC (default: **no** JS).

### References

- [Source: `_bmad-output/planning-artifacts/epics.md` — Epic 3; Story 3.4 acceptance criteria; UX-DR11–UX-DR12 definitions]
- [Source: `_bmad-output/planning-artifacts/PRD.md` — Personal media / GPS on share; FR-14; Accessibility share page WCAG 2.1 Level A]
- [Source: `_bmad-output/planning-artifacts/architecture.md` — §3.5 Share service; §3.6 web stack; §4.5 agent rules; §5.1 `web/share/`]
- [Source: `_bmad-output/planning-artifacts/ux-design-specification.md` — Accessibility Considerations (share page WCAG A, 200% zoom, reduced motion); Focus ring / tokens]
- [Source: `_bmad-output/implementation-artifacts/3-3-share-html-readonly.md` — prior share page, landmark, rating `aria-labelledby`, 404 parity constraints]
- [Source: `internal/share/handler.go` — HTML + image routes]
- [Source: `web/share/share.html`, `web/share/share.css` — current template/layout]

## Dev Agent Record

### Agent Model Used

Cursor AI agent — BMAD dev-story workflow (2026-04-14).

### Debug Log References

### Completion Notes List

- Party mode **dev session 1/2**: `TestShareHTTP_404_noReferrerPolicyHeader` (GET/HEAD × `/s/` `/i/`); skip-link CSS pairs `:focus` with `:focus:not(:focus-visible)` reset (keyboard-visible ring, legacy-safe); `TestShareHTTP_resolve200_HTML` asserts the pairing in inlined CSS.
- Party mode **create session 2/2**: `/i/` Referrer-Policy parity; stricter reduced-motion CSS; `http_test` asserts inlined `prefers-reduced-motion`; `ParseShareSnapshotPayloadJSON` godoc warns on unknown keys + anti–raw-payload-dump.
- Party mode **dev session 2/2**: **`ShareHTMLContentSecurityPolicy`** + HTML **`nosniff`**; **`<main tabindex="-1">`** for skip target; 404 asserts no CSP/nosniff/Referrer; HEAD HTML matches GET CSP/nosniff.
- **Manual QA matrix (AC4 — 200% zoom / reflow; CI cannot replace):** Run against a **live** `GET /s/{token}` page (loopback or local build). Verify **primary path** = letterboxed full photo + rating summary: no broken overlap, rating text readable, **no horizontal scroll** required on that path. Use **200% browser zoom** (or OS display scaling documented in `nfr-07-os-scaling-checklist.md` as equivalent). Repeat at **typical desktop** width (e.g. **1280×720** CSS px) and **narrow** width (e.g. **≤430** CSS px, Story 3.3 mobile target). Spot-check **≥2 engines**: e.g. **Chrome + Firefox**, or **Chrome + Safari**. **Keyboard:** `Tab` / `Shift+Tab` — skip link first, visible `:focus-visible` on focusable controls, logical order into `<main>`.

### File List

- `internal/share/handler.go` — Referrer-Policy; HTML CSP constant + `nosniff` on successful `/s/`.
- `web/share/share.html` — `tabindex="-1"` on `<main>` for skip-link focus.
- `web/share/share.css` — reduced-motion block hardened; skip-link focus pairing.
- `internal/share/http_test.go` — image Referrer-Policy; HTML CSP + nosniff; 404 CSP/nosniff/Referrer absent; inlined CSS probe; skip-link `:focus-visible` substring.
- `internal/store/share.go` — payload parse godoc.
- `_bmad-output/implementation-artifacts/3-4-share-privacy-wcag.md` — tasks, Dev Agent Record, status.
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — story `3-4-share-page-privacy-wcag-2-1-level-a` → review.

### Review Findings

_BMAD code-review (2026-04-14), scoped to Epic 3 Story 3.4; headless run — patch/decision handling left to implementer._

- [ ] [Review][Decision] **Package share HTML exposes filename and internal asset id in captions** — Story 3.4 Tasks/Subtasks require tracing template fields and confirming no `filename`, `asset_id`, or `rel_path` in markup for successful `GET /s/{token}` HTML. `servePackageHTML` builds captions with `pathpkg.Base(relPath)`, numeric `asset id`, and capture date (`internal/share/handler.go` ~239–251). `TestShareHTTP_packageHTMLAndSnapshotImage404AfterReject` asserts captions still contain `id %d` (`internal/share/http_test.go` ~1057–1058`). This matches Epic 4.1 snapshot UX but conflicts with the 3.4 privacy-audit checklist and DoD “no identifier … leaks”. **Needs product call:** neutralize package captions (e.g. “Photo 1 of N” only) vs keep identifiers for package shares and revise Story 3.4 scope/tasks.

- [ ] [Review][Patch] **Extend privacy tests to package HTML once decision above is made** — If captions stay identifier-rich, update Story 3.4 tasks/DoD to carve out package shares and add a test that documents allowed patterns; if captions are stripped, add `TestShareHTTP_packageHTML_doesNotLeakIdentifiers` mirroring `TestShareHTTP_HTML_doesNotLeakIdentifiers` and adjust package regression tests.

## Change Log

- **2026-04-14:** Story 3.4 dev-story closed — manual zoom/reflow QA matrix recorded; all tasks complete; status **review**; sprint status updated.
- **2026-04-14:** Party mode dev session **2/2** — HTML CSP + nosniff (success-only); main `tabindex="-1"`; tests + story spec aligned; status remains **review**.
