# Story 2.1: Application shell, navigation, and dual themes

Status: done

<!-- Ultimate context engine analysis completed — comprehensive developer guide created. -->
<!-- 2026-04-15: BMAD create-story merge — epics.md §Story 2.1 acceptance criteria inlined below; Status preserved. Sprint key aligned to **done** with story (party create 1/2 + 2/2); do not resurrect ready-for-dev for closed 2.1. -->

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a **photographer**,  
I want **consistent navigation and dark/light themes**,  
So that **long sessions are comfortable and I always know where I am**.

**Implements:** UX-DR1, UX-DR13, UX-DR15, UX-DR16 (shell baseline); enables FR-07+ UI work. *(Epics.md also lists UX-DR16 shell; UX-DR15 appears in Story 2.1 “And” clause.)*

## Acceptance Criteria

*Canonical wording from* `_bmad-output/planning-artifacts/epics.md` *§Story 2.1 is reflected in AC **1**, **5–8**, and **10** below; additional items elaborate architecture and Epic 1 continuity.*

1. **Given** the app launches, **when** the main window loads, **then** primary areas exist: **Upload**, **Review**, **Collections**, **Rejected** (UX-DR13).
2. **Given** the user launches the desktop app (no CLI subcommand), **when** the main window appears, **then** the UI exposes those **four** destinations in **consistent order and labeling**, each reachable without hidden gestures (persistent chrome such as sidebar or top tabs). **When** the user switches destinations, **then** the **content region** updates while **primary navigation remains visible** (shell pattern — not a one-shot dialog).
3. **Given** the **Upload** area, **when** opened, **then** it hosts the **existing** upload flow (`NewUploadView`; Story **1.5** + **1.8**) **without forking** ingest, receipts, or **`OperationSummary`** semantics — only **re-parented** into the shell content area (architecture §3.9, §4.5).
4. **Given** **Review**, **Collections**, or **Rejected** in this story, **when** opened, **then** each shows a **deliberate placeholder** (title + short “coming next” copy is acceptable) — **no** fake data or partial grids that imply FR-07+ is done; placeholders must **not** claim feature completeness. *(Original Story 2.1 scope; current `main` may replace these regions with real panels from later Epic 2 stories — trace feature completeness to those stories, not this AC in isolation.)*
5. **Given** theme toggle or preference, **when** switched, **then** both **dark** and **light** themes apply semantic roles (**primary**, **destructive**, **reject/caution**, **focus**) without feature gaps (UX-DR1).
6. **Given** the default first run, **when** the window loads, **then** the active theme is **dark** (UX spec: “Dark DAM default”) with **light** available as a **first-class peer** (UX-DR1, UX spec §Design Direction).
7. **Given** a **theme toggle** (menu, toolbar control, or settings action — pick one discoverable pattern and document it), **when** the user switches **dark ↔ light**, **then** **all** standard chrome in the shell (nav, placeholders, upload surface) redraws with the selected variant **without restart** (Fyne `fyne.App.Settings().SetTheme(...)` or equivalent).
8. **Given** either theme variant, **when** implementing custom colors, **then** semantic roles from UX are **all** defined for **both** variants: **background, surface (and elevated if used), border/divider, text primary/secondary, primary action, destructive, reject/caution, focus ring** — **no** role that exists only in light or only in dark (UX-DR1, UX spec §Color System → Core roles & Theme completeness).
9. **Given** **destructive** vs **reject/caution** styling, **when** sample buttons or labels demonstrate those roles (e.g. small “style preview” strip **or** documented `widget.Button` importance + theme color wiring), **then** **destructive** and **reject/caution** remain **visually distinct** in **both** themes (UX spec: reject distinct from destructive; star vs reject must not rely on hue alone — baseline for later Stories **2.6–2.7**).
10. **And** focus visibility is visible on standard Fyne controls (baseline for UX-DR15): **given** keyboard focus on **standard Fyne controls** (nav buttons, one `widget.Entry` if present on Upload), **when** the user tabs through, **then** **focus is visibly indicated** (focus color / focus ring from theme) in **both** dark and light. *(Full focus order filter → grid → loupe is Story **2.2+**.)*
11. **And** primary **navigation** is **compact** (single obvious row / rail per **Direction A**); it does **not** compete with the **Review** image stage for vertical space (UX-DR16 baseline). **Verify** during Story **2.11** / layout review if not already obvious; architecture §3.8.1 requires **measurement anchors** (which box, which lifecycle moment) to match UX evidence.

## Tasks / Subtasks

- [x] **Shell layout + navigation** (AC: 1–2, 4, 11)
  - [x] Introduce a **single main window content** pattern: persistent nav + **swappable** central content (`container.Border`, `container.AppTabs`, or `fyne.Container` with manual visibility — justify choice in a one-line code comment).
  - [x] Wire **four** nav entries with exact labels **Upload**, **Review**, **Collections**, **Rejected** (order matches UX-DR13). *(Implementation: four `widget.Button` entries; **active** section uses `HighImportance`, inactive use `MediumImportance`. RadioGroup was rejected: re-tapping **Collections** must fire `OnTapped` for Story **2.8 AC12** list reset — radio controls typically no-op when the same item is selected.)*
  - [x] Add **placeholder** `fyne.CanvasObject` factories for Review / Collections / Rejected (minimal `widget.Label` + padding acceptable). *(Superseded in tree: later stories swap in `NewReviewView`, collections UI, `NewRejectedView`; `NewSectionPlaceholder` remains in `placeholder.go` for honest empty states.)*
- [x] **Integrate existing Upload** (AC: 3)
  - [x] Refactor `main.go` / app bootstrap so **library open** + **`NewUploadView(win, db, root)`** runs **inside** the Upload tab/pane — **same** window reference as today for dialogs and `SetOnDropped` (Story 1.8).
  - [x] Confirm **no** duplicate `store.Open` / double-close of DB when switching tabs.
- [x] **Custom Fyne theme** (AC: 5–6, 8–9)
  - [x] Implement **`fyne.Theme`** (e.g. `internal/app/theme.go`) with **two** constructors or a variant enum: **Dark (default)** and **Light**, delegating to `theme.DefaultTheme()` where sensible and **overriding** `Color` for semantic roles (architecture §3.8).
  - [x] Map UX roles to **`theme.ColorName`** and/or **custom** `theme.ColorName` values per Fyne v2.7 patterns; **document** the mapping table in code comments at top of theme file.
  - [x] Persist user theme choice via **`fyne.Preferences`** (app ID already fixed: `internal/app/fyne_app_id.go` — **do not change** `FyneAppID`).
  - [x] On startup, **load preference** → `SetTheme`; default **dark** when unset.
- [x] **Theme toggle UI** (AC: 7)
  - [x] Add user-visible toggle bound to preference + `SetTheme`.
- [x] **Focus visibility check** (AC: 10)
  - [x] Verify **Focus** color contrasts on **background** in both variants; adjust theme until Tab focus is obvious on nav and one input.
- [x] **Tests / verification** (AC: 5–10)
  - [x] **Unit tests** for theme type: e.g. `Color` returns non-nil for each role, dark ≠ light for at least **background** and **primary**; optional **golden** hex string asserts if stable.
  - [x] **Regression tests** for primary nav: `PrimaryNavLabels` + **`PrimaryNavKeys`** order/labels/keys and unique keys (`shell_test.go`; party dev **1/2** adds key/label parity + **focus ≠ primary** in `theme_test.go` for AC **10**; party dev **2/2** adds **focus ≠ warning/error** + **separator ≠ background**).
  - [x] **Manual QA** note in Dev Agent Record: launch, switch theme, tab through nav + upload entry, confirm Upload ingest still works.
- [x] **Spec refresh (2026-04-15)** — BMAD create-story: merged epics.md §Story 2.1 BDD criteria into AC **1**, **5**, **10**, **11**; sprint key `2-1-app-shell-navigation-themes` → `ready-for-dev` for tracking (story **Status** remains **done**).

## Dev Notes

### Technical requirements

- **Prerequisites:** Epic 1 upload path (`main.go`, `internal/app/upload.go`, `internal/app/drop_paths.go`, `ingest`, `store`) must remain functional; this story **restructures presentation only** for Upload and adds placeholders elsewhere.
- **Single pipeline / summaries:** Do **not** change `domain.OperationSummary` or ingest from the shell story (architecture §3.9, §4.5).
- **Boundaries:** All new Fyne wiring stays in **`internal/app`** (or `cmd/` if main moves later); **no** SQL or ingest imports added for placeholders (architecture §5.2).

### Architecture compliance

- **§3.8:** Custom `fyne.Theme` with UX semantic roles; dark default + light peer; distinct primary / destructive / reject-caution.
- **§3.8.1:** Layout ↔ async coherence — any numeric layout budget (e.g. UX-DR16 nav height vs Review stage) must name the **same measurement box** and **lifecycle moment** as UX/Story **2.11** evidence; avoid dual interpretations.
- **§5.1:** `internal/app` owns Fyne application, navigation, theme wiring.
- **§5.2:** `app (Fyne) → domain, store, ingest` — shell only **routes** to existing use cases.
- **§3.12 implementation order:** Step 4 (“Fyne shell + themes + navigation placeholders”) is **this story**.

### Library / framework requirements

- **Fyne** `fyne.io/fyne/v2` **v2.7.3** (`go.mod`): implement `fyne.Theme`; use `fyne.App.Settings().SetTheme` and **Preferences** for persistence.
- Prefer **composition** over forking Fyne internals (UX spec — Design System → Customization Strategy).

### File structure requirements

- **Primary:** `main.go` — construct app, window, open store, attach **shell** instead of raw `SetContent(NewUploadView(...))` only.
- **New or extended:** `internal/app/` — e.g. `shell.go` (navigation + content stack), `phototool_theme.go` or `theme.go` (types implementing `fyne.Theme`), small `placeholder.go` if it avoids clutter.
- **Avoid:** scattering theme color literals across widgets — **centralize** in the `Theme` implementation.

### Testing requirements

- **Table-driven** unit tests for theme color presence / variant distinction (architecture §4.4).
- Full navigation E2E optional; **must** keep `go test ./...` green.

### Continuity from Epic 1 (not a prior Epic 2 story)

- **`main.go`** currently sets `w.SetContent(ptapp.NewUploadView(...))` only — replace with shell that **embeds** that view for Upload.
- **Story 1.8:** `SetOnDropped` and window-scoped dialogs assume a **stable** `fyne.Window` — preserve the same `win` reference passed into `NewUploadView`.

### Latest technical information

- Fyne **v2.7** theme API: implement `Theme` interface (`Color`, `Font`, `Icon`, `Size`); use `theme.DefaultTheme()` as delegate for uncustomized slots to reduce drift.
- For **OS appearance sync**: optional follow-up — **out of scope** unless trivial; preference + manual toggle satisfies AC.

### Project structure notes

- Architecture allows future `cmd/phototool`; **do not** block that — keep bootstrap logic readable for a later move.
- **Rejected** nav label matches UX-DR13 (“Rejected”); if product copy later prefers “Rejected/Hidden”, track as UX change — **use spec labels for MVP**.

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Epic 2, Story 2.1 (canonical AC), requirements inventory UX-DR1, UX-DR13, UX-DR15–UX-DR16]
- [Source: _bmad-output/planning-artifacts/epics-v2-ux-aligned-2026-04-14.md — Story 2.1 rollup, UX-DR1–UX-DR19 cross-reference]
- [Source: _bmad-output/planning-artifacts/architecture.md — §3.8 Desktop UI (Fyne), §3.8.1 layout measurement anchor, §5.1 layout, §5.2 boundaries, §3.12 order step 4]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — Design System Foundation §1.1; Visual Design Foundation §Color System (Core roles, Theme completeness); §Design Direction (Dark DAM default + light peer)]
- [Source: _bmad-output/planning-artifacts/PRD.md — Primary Fyne UI, dark/light themes]
- [Source: main.go — GUI bootstrap, `NewUploadView`]
- [Source: internal/app/upload.go — `NewUploadView`]
- [Source: internal/app/fyne_app_id.go — `FyneAppID`]

## Definition of Done (create / closure)

- **Epics parity:** Story AC **1**, **5**, **10**, **11** cover `epics.md` §Story 2.1 bullets (nav areas, dual themes + semantic roles, focus baseline, compact nav baseline); elaborations AC **2–9** trace to UX-DR1 / shell architecture without contradicting Epic 2 scope.
- **Automation:** `go test ./...` and `go build .` green on the paths in **File List**; theme + nav regression tests present (`theme_test.go` including primary≠destructive + focus contrast, `shell_test.go`).
- **Operational truth:** Sprint key `2-1-app-shell-navigation-themes` matches story **Status** (`done`) so downstream agents are not sent to “ready-for-dev” on closed work.
- **Manual QA:** One pass per **Manual QA (Story 2.1)** — especially theme toggle, Tab focus on nav + upload entry, and **1024×768** sanity (nav + optional preview strip + Review filter strip per NFR-01 structural tests).

## Risks & follow-ups

| Risk | Mitigation / note |
|------|-------------------|
| **Sprint vs story status drift** | **Mitigated (2026-04-15 create):** `sprint-status.yaml` set to **done** when story file is **done**; DoD above requires both to agree. |
| **Semantic preview buttons** read as real actions | Labels include `(preview)`; handlers are no-ops; strip documents roles only (AC9). |
| **Disabled-button preview** hid true Danger/Warning chrome | Use **enabled** no-op buttons so Fyne renders real importance colors (party session 2). |
| **Corrupt / migrated prefs** (`appearance.themeVariant`) | Invalid values fall back to **dark**; covered by `TestLoadThemeVariantFromPrefs_invalidFallsBackToDark` (party session 1). |
| **OS / system theme sync** | Out of scope per story; preference + View menu is the contract. |
| **Full focus order (filter → grid → loupe)** | Explicitly Story 2.2+ (UX-DR15); this story only baselines nav + upload entry. |
| **Star vs reject not hue-only** | AC9 here is destructive vs caution; star/reject grid semantics wait for Stories 2.3–2.7. |
| **Flat nav buttons hide “you are here”** | Resolved: **importance** styling marks the active section; RadioGroup avoided to preserve **Collections** re-tap (2.8 AC12). |
| **Nav importance change without redraw** | Fyne `Button.Importance` is a plain field — dev session 2/2 calls `Refresh()` on each nav control after `setNavSelection` so programmatic hops match tap behavior. |
| **Programmatic `gotoReview` vs tap ordering** | **Mitigated (party create 2/2):** `gotoReview` now runs **`clearReviewUndoIfLeftReview` → `prevNavKey` → `setNavSelection` → `selectPanel`** (same as buttons for the shared steps). Collections reset is N/A here (`nextKey` is always `review`, so AC12 re-tap predicate is false). |
| **Epics §2.1 omits preview strip** | Left-rail semantic preview is an **AC9 evidence** aid, not a separate epic deliverable; height risk remains deferred to Story **2.11** (existing Review finding). |

## Dev Agent Record

### Agent Model Used

Party mode **dev session 2/2** (simulated roundtable — `bmad-party-mode` solo path; roster from `_bmad/_config/agent-manifest.csv`) + Cursor agent implementation.

### Debug Log References

### Completion Notes List

- 2026-04-15 (party mode **dev session 2/2**, hook **dev** — automated headless, simulated round): **Sally** challenged session **1/2** as still too **primary-centric**: keyboard users land on **Warning** (reject/caution) and **Danger** (destructive) controls — if a future palette retune collapses **focus** into **warning** or **error**, AC **10** fails silently. **Amelia** wanted the smallest fix: **three** color inequality tests, no new exports. **Winston** pushed back on refactoring `gotoUpload` / `OnTapped` into a shared closure — accepted a **one-line prelude comment** documenting identical ordering instead. **John** asked for **AC8** traceability: **separator** must not equal **background** or dividers disappear in QA photography. **Applied:** `TestPhotoToolTheme_focusDistinctFromWarning`, `TestPhotoToolTheme_focusDistinctFromError`, `TestPhotoToolTheme_separatorDistinctFromBackground`, `gotoUpload` prelude godoc; `go test ./...`.
- 2026-04-15 (party mode **dev session 1/2**, hook **dev** — automated headless, simulated round): **Amelia** pushed **key/label export parity** (`PrimaryNavKeys`) so panel wiring tests cannot drift from UX-DR13 labels silently. **Sally** argued the real AC **10** foot-gun is **focus color colliding with primary** (both blue-leaning in light themes); demanded a regression beyond focus≠background. **Winston** resisted widening the public API — accepted **one** narrow `PrimaryNavKeys` mirror of existing `PrimaryNavLabels` over refactoring `primaryNavItems` to a new package. **John** asked for an explicit **epic wording bridge**: epics §2.1 says “row / rail”; code uses a **vertical** rail — document in-shell to stop false “wrong layout” bugs. **Applied:** `PrimaryNavKeys`, `TestPrimaryNavKeys_matchItemsAndLabels`, `TestPhotoToolTheme_focusDistinctFromPrimary`, Border-layout godoc; `go test ./...`.
- 2026-04-15 (**dev-story workflow**, earlier same day): Re-verified all tasks [x] and AC **1–11** against `main.go`, `internal/app/shell.go`, `internal/app/theme.go`, and tests; `go test ./...` and `go build .` green at that time (superseded by party dev **1/2** test additions below). Story **Status** remains **done** per Definition of Done (sprint `2-1-app-shell-navigation-themes: done`).
- 2026-04-15 (party mode **create session 2/2**, hook **create** — automated headless, simulated round): **John** challenged **residual doc debt**: the story banner comment and Change Log still mentioned `ready-for-dev` after sprint was corrected to **`done`**, which re-opens the same routing bug in prose. **Mary** argued the real spec risk is **nav state coherence**: `gotoReview` used to call `selectPanel` **before** committing `prevNavKey`, unlike nav buttons — low probability today, easy latent foot-gun. **Winston** noted a full **collections** prelude cannot run in `gotoReview` (constructor order); **Amelia** accepted **`clearReviewUndoIfLeftReview` + `prevNavKey` before `selectPanel`** plus **`TestPhotoToolTheme_primaryDistinctFromDestructive`**. **Paige** fixed **AC** references in tests/comments (focus = **AC10**, preview strip = **AC9**). **Applied:** `gotoReview` ordering + tests + story/changelog hygiene; `go test ./...`.
- 2026-04-15 (party mode **create session 1/2**, hook **create** — automated headless, simulated round): **John** flagged **process debt**: sprint still said `ready-for-dev` while the story and code were **done**, which mis-routes sprint automation. **Mary** asked for an explicit **Definition of Done** tying `epics.md` §2.1 to the story’s expanded AC set so “gold plating” is traceable, not scope creep. **Sally** defended keeping the **semantic preview strip** as long as **Manual QA** names **1024×768** and the strip is optional in structural tests (`omitSemanticStylePreview`); do not pretend the rail is zero-height. **Winston** disagreed with adding noisy runtime logging for bad `selectPanel` keys; preferred a **fail-fast construction check** that every nav key has a non-nil panel. **Applied:** `Definition of Done` + risk row + sprint **`done`** + shell startup invariant after `panels` map; `go test ./...`.
- 2026-04-14 (party mode **session 1/2**, dev hook — automated, simulated round): **Amelia** flagged **doc drift**: story tasks still described **RadioGroup** while `shell.go` ships **buttons** + importance (AC12). **Sally** insisted Manual QA must not train testers on the wrong control. **Winston** kept scope tight: fix markdown + `NewMainShell` godoc only — no nav rewrite. **Paige** asked for a single traceability sentence on **AC4** so “placeholder” language does not contradict the live tree. Applied: story task/risk/Manual QA alignment, AC4 footnote, shell comment truth. Re-ran `go test ./...`.
- 2026-04-14 (party mode **session 2/2**, dev hook — automated, simulated round): **Amelia** challenged session-1 “documentation-only” closure — `setNavSelection` mutates `Importance` but never **`Refresh()`**; **gotoReview** / **gotoUpload** can leave the wrong chrome until the next full repaint. **Sally** added UX-DR1 nuance: **caution** must not be confusable with **primary** (not only vs destructive). **Winston** accepted the smallest fix: refresh the four nav buttons inside `setNavSelection`; no new abstractions. **John** asked for a one-line **risk** + a **theme** regression so primary ≠ warning in both variants. Applied: `shell.go` refresh loop, `TestPhotoToolTheme_cautionDistinctFromPrimary`, risks table; re-ran `go test ./...`.
- 2026-04-13 (historical): RadioGroup spike for AC1 “selected” affordance — **superseded** by button nav + importance for Story **2.8 AC12** (re-tap Collections). Do not resurrect without re-validating AC12.
- 2026-04-13 (party mode **session 1/2**, dev hook — automated): Roundtable (Amelia / Sally / Winston / Murat) challenged prefs corruption and “visible focus” (now AC **10**) testability; applied extra unit tests in `theme_test.go` (invalid pref → dark; focus ≠ background per variant). Sprint story remains **done**.
- 2026-04-14 (dev-story verification): Re-ran `go test ./...` and `go build .`; all green. AC1–11 (post–2026-04-15 spec numbering) remain satisfied at time of verification: `main.go` wires `NewMainShell`, theme + View menu toggle + prefs, semantic preview strip, `PhotoToolTheme` roles/tests. Review/Collections/Rejected are implemented panels from follow-on Epic 2 stories (replacing the original placeholders). Primary nav uses **four labeled buttons** with High/Medium importance for the active section so re-tapping **Collections** still runs `OnTapped` (Story 2.8 AC12); this supersedes the earlier RadioGroup note in task prose.
- 2026-04-13 (dev-story verification): Re-ran `go test ./...` and `go build .`; all green. Codebase matched AC1–9 (prior numbering) and all tasks; no code changes required.
- Shell: `container.NewBorder` + left nav + `container.Stack` center; pre-built panels swapped with `RemoveAll`/`Add` so Upload state survives tab changes and `SetOnDropped` stays on the same `fyne.Window`.
- Theme: `PhotoToolTheme` delegates Font/Icon/Size to `theme.DefaultTheme()` and forces light/dark via an internal variant so preferences work without a public `SetThemeVariant` API.
- Theme toggle: **View → Use dark theme / Use light theme**; persisted under `appearance.themeVariant` in app preferences.
- Style preview: **enabled** **Danger** vs **Warning** sample buttons with no-op taps and `(preview)` labels so Fyne renders real importance colors (AC9; session 2 fixes disabled-state washout).

### Manual QA (Story 2.1)

- Launch GUI (`go run .` with no args): confirm default dark, primary nav shows **four buttons** with the **active** destination at **High** importance (inactive at **Medium**), Upload ingest + DnD still work, View menu switches theme without restart, Tab shows focus on nav and upload/collections inputs as applicable; Review/Collections/Rejected behavior is owned by later Epic 2 stories.
- At **1024×768**, confirm left rail (nav + optional semantic preview strip) plus Review **filter strip** still read as usable (NFR-01 structural baseline; preview strip may be omitted in headless layout tests via `omitSemanticStylePreview`).

### File List

- `main.go`
- `main_fyne_ci_test.go`
- `internal/app/fyne_app_id.go`
- `internal/app/theme.go`
- `internal/app/theme_test.go`
- `internal/app/shell.go`
- `internal/app/shell_test.go`
- `internal/app/placeholder.go`

### Review Findings

_BMAD code review (2026-04-15), scoped to Story 2.1 paths + working-tree diff; layers: Blind Hunter, Edge Case Hunter, Acceptance Auditor; headless run (no patch batch)._

- [x] [Review][Mitigated] `selectPanel` + `panels[...]` — **2026-04-15:** after building `panels`, shell **panics in dev** if any `primaryNavItems` key is missing or maps to a nil object, so the nav→panel map cannot silently drift; callers should still only pass internal keys (unexpected runtime keys remain an implementation bug, not a user path).

- [x] [Review][Defer] Left-rail semantic preview strip adds non-trivial vertical chrome; UX-DR16 “compact shell” is tracked for Story 2.11 verification (`internal/app/shell.go:145-149`) — deferred, NFR follow-up.

- [x] [Review][Defer] `gotoReview` resolves the destination via `keyByLabel[labels[1]]` (second nav slot). Reordering `primaryNavItems` without updating this helper desyncs programmatic navigation from button `OnTapped` paths that use `item.key`. Prefer resolving the Review panel by stable internal key (e.g. `"review"`) or a single shared lookup helper — [`internal/app/shell.go` ~69–81].

- [x] [Review][Defer] Theme regression tests use exact `color.Color` inequality only; they do not assert contrast ratios or perceptible focus for AC10. Keep Manual QA / Story 2.11 evidence as the contract unless contrast or ΔE tests are added — [`internal/app/theme_test.go`].

## Change Log

- 2026-04-15: Party mode **dev session 2/2** (hook **dev**) — `TestPhotoToolTheme_focusDistinctFromWarning` / `focusDistinctFromError` / `separatorDistinctFromBackground`, `gotoUpload` prelude comment; sprint comment; **Status** unchanged **done**.
- 2026-04-15: Party mode **dev session 1/2** (hook **dev**) — `PrimaryNavKeys` + nav/key parity test, `TestPhotoToolTheme_focusDistinctFromPrimary`, shell Border godoc (vertical rail); sprint comment; **Status** unchanged **done**.
- 2026-04-15: **BMAD dev-story** — verification pass on explicit story path; `go test ./...` and `go build .` green; Dev Agent Record updated (**Status** unchanged **done**).
- 2026-04-15: Party mode **create session 1/2** (hook **create**) — simulated John / Mary / Sally / Winston; **Definition of Done**, sprint/story alignment (**`done`**), risk row, Review finding mitigated for nav→panel map drift; `internal/app/shell.go` fail-fast panel registration.
- 2026-04-15: BMAD **create-story** — merged `epics.md` §Story 2.1 BDD into AC **1**, **5**, **10**, **11**; renumbered AC **2–11**; tasks + risks AC references updated; architecture §3.8.1 + epics-v2 references added; **Status** remains **done**; sprint key later aligned to **done** (party create 1/2) — supersede any interim **ready-for-dev** note.
- 2026-04-15: Party mode **create session 2/2** (2-1) — `gotoReview` prelude parity + `TestPhotoToolTheme_primaryDistinctFromDestructive`; AC comment fixes; story banner + risks + Dev Agent Record.
- 2026-04-14: Party mode dev **session 2/2** — simulated Amelia/Sally/Winston/John; nav `Refresh` after importance changes, `TestPhotoToolTheme_cautionDistinctFromPrimary`, risk row; sprint-status comment; historical RadioGroup note compressed.
- 2026-04-14: Party mode dev **session 1/2** — simulated Amelia/Sally/Winston/Paige; aligned story tasks, AC4 traceability, risks, Manual QA, and `shell.go` godoc with **button** nav + **2.8 AC12** rationale; sprint-status comment.
- 2026-04-14: Dev-story workflow verification — full `go test ./...` and `go build .` green; story Completion Notes updated (nav/buttons vs RadioGroup, placeholder AC superseded by later Epic 2 UI).
- 2026-04-13: Party mode session **2/2** (dev): simulated round challenged flat nav; implemented `RadioGroup` + `PrimaryNavLabels` regression tests + preview copy clarification.
- 2026-04-13: Party mode session **1/2** (dev): documented round + added theme preference / focus visibility regression tests.
- 2026-04-13: Dev-story verification pass — full test suite and release build succeeded; File List completed with Fyne app ID + CI guard files.
