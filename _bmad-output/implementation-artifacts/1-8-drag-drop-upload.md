# Story 1.8: Drag-and-drop upload entry

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a **photographer**,  
I want **to drop files onto a designated target**,  
So that **ingestion matches the picker path exactly**.

**Implements:** UX-DR14; FR-01 (parity with picker).

## Acceptance Criteria

1. **Given** the Upload view with a **visible, labeled drop target** (not window-chrome-only), **when** the user drops **one or more supported image files** onto that target, **then** the application runs the **same ingest + receipt + post-import (collection confirm) path** as when the user accumulates files via the file picker and taps **Import selected files** — specifically **`ingest.IngestWithAssetIDs`**, **`batchStart`**, receipt labels, **`postImport`** visibility, and the existing confirm/cancel collection behavior (UX-DR14, FR-01, Story 1.5 parity).
2. **Given** a drop that includes **only** paths that are **not** supported image files (wrong extension, non-file URI, or empty list after filtering), **when** the drop is processed, **then** the user sees a **clear, factual** explanation (e.g. dialog or inline error) — **no** silent no-op and **no** ingest run that hides the rejection (UX-DR14; aligns with UX **Errors: proportionate honesty**).
3. **Given** a **mixed** drop (some supported, some unsupported), **when** processed, **then** supported files follow **AC1** and the user is informed about the unsupported items **without** silent failure (same honesty bar as AC2).
4. **Extension and path rules** match the file picker: use the **same allow-list** as **`storage.NewExtensionFileFilter(ingest.PickerFilterExtensions())`** / **`internal/ingest/extensions.go`** — **no** duplicate extension tables in the UI package.
5. **Given** an external drop whose **release position** is **outside** the designated drop target (per canvas hit-test), **when** the drop is delivered to the window, **then** the app **performs no ingest** and shows **no** error (drops elsewhere on the window are intentionally ignored to avoid accidental imports).

## Tasks / Subtasks

- [x] **Refactor upload batch runner** (AC: 1)
  - [x] Extract the logic currently in the **Import selected files** handler in `internal/app/upload.go` (set `batchStart`, call `ingest.IngestWithAssetIDs`, update `lastSummary` / `lastAssetIDs`, `showReceipt`, reset collection widgets, show `postImport`) into a **single function or closure** invoked by both the Import button and the drop path.
- [x] **Drop target UI** (AC: 1)
  - [x] Add a **designated** control or bordered region (e.g. label + subtle border / background using theme colors) that reads as “drop images here,” placed near the existing **Add images…** / list affordances so the flow is discoverable (UX-DR14, UX spec — upload entry).
- [x] **OS file drop wiring** (AC: 1–3, 5)
  - [x] Use **`fyne.Window.SetOnDropped`** (Fyne **v2.4+**, project uses **v2.7.3**) to receive `[]fyne.URI`. For each URI, resolve a **local filesystem path** (`uri.Scheme()` / `uri.Path()` per `fyne.URI` rules); ignore non-`file:` schemes with a **factual** user message or include in the “unsupported” summary.
  - [x] **Hit-test** (recommended): only treat a drop as targeting the upload drop zone when the drop **position** intersects the drop target’s canvas bounds — so arbitrary drops on the window do not start an ingest. Document the hit-test approach in a short code comment.
  - [x] **Directories:** If a dropped URI is a **directory**, treat as **unsupported** for this story (clear message) unless you explicitly add recursive expansion — **out of scope** unless expanded in a follow-up story.
- [x] **Validation + user feedback** (AC: 2–4)
  - [x] Centralize “is this path a supported ingest extension?” using **`ingest.PickerFilterExtensions()`** (or a small helper in `internal/ingest` if needed — **not** a second extension list in `app`).
  - [x] For unsupported-only drops: **`dialog.ShowError`** or **`dialog.ShowInformation`** with a concrete reason (e.g. file type / path), not an empty receipt.
  - [x] For mixed drops: run ingest for supported set per AC1; surface unsupported names/types in the same honesty pattern (dialog or inline — pick one consistent with Story 1.5 error style).
- [x] **State / regression guards**
  - [x] Reuse **`addAbsolute`** / path de-dup rules where drops **append** to the visible list **if** product choice is “drop adds to batch like picker”; **or** if drops **only** trigger the shared batch runner without mutating the list, document that choice — **default recommendation:** drops feed the **same** `paths` slice + `pathList` refresh so picker and DnD are visually consistent, then call the shared import runner (matches “same pipeline” mental model).
  - [x] Respect existing Story 1.5 behaviors: **`importBtn` enablement**, `postImport` gating, collection confirm — align with any **in-progress review fixes** on `upload.go` (orphan collection, double-import) so DnD does not bypass those guards.
- [x] **Tests / verification** (AC: 1–5)
  - [x] **Unit-test** pure helpers if extracted (extension classification, URI → path, mixed unsupported lists, **axis-aligned hit rect**, **duplicate URIs in one drop**) without starting Fyne.
  - [x] **Manual QA** checklist in Dev Agent Record: single file drop, multi-file drop, unsupported-only, mixed supported/unsupported, duplicate paths in one drop, **drop while `postImport` visible → informational dialog; Add/Clear/Import disabled until Confirm/Cancel** (session 2/2).

## Dev Notes

### Technical requirements

- **Prerequisites:** Story **1.5** upload flow (`internal/app/upload.go`, `ingest.IngestWithAssetIDs`, receipt + collection gate). Story **1.3** ingest semantics unchanged.
- **Single pipeline:** Drops must **not** fork ingest, dedup, or summary types — architecture **§3.9**, **§4.5** (one `OperationSummary`, one dedup path).
- **Boundaries:** Fyne-only in **`internal/app`**; **no** Fyne imports under **`internal/ingest`** (architecture **§5.2**).

### Architecture compliance

- **§3.8 / §5.1–5.2:** Fyne desktop UI in `internal/app`; compose primitives; keep **no SQL in widgets**.
- **§3.9:** Receipt fields and meanings stay aligned with CLI (`domain.OperationSummary`).
- **§4.2–4.3:** Wrap errors with **`%w`** at boundaries; user-facing copy separate from logged errors.

### Library / framework requirements

- **Fyne** `fyne.io/fyne/v2` **v2.7.3** (per `go.mod` / architecture): **`Window.SetOnDropped(func(pos fyne.Position, uris []fyne.URI))`** for OS file drops; combine with canvas **position** / **size** of the drop target for hit-testing.
- Reuse **`fyne.URI`** / **`storage`** patterns already used by the file picker in `upload.go`.

### File structure requirements

- **Primary:** `internal/app/upload.go` — drop target UI, shared import runner, `SetOnDropped` registration (ensure callback is set **once** per window / view lifetime to avoid duplicate handlers if `NewUploadView` is called multiple times — document or guard).
- **Optional:** tiny helper file under `internal/app/` only if `upload.go` becomes unwieldy; prefer keeping upload surface cohesive.
- **Optional:** `internal/ingest` helper for “allowed extension for path string” if it improves reuse for CLI/GUI parity — **do not** diverge from `PickerFilterExtensions`.

### Testing requirements

- Prefer **table-driven** tests for classification logic (architecture **§4.4**).
- Full Fyne DnD E2E is often OS-dependent; **minimum** is strong unit coverage + documented manual matrix (platforms: macOS baseline per project).

### Previous story intelligence (1.5 / 1.7)

- **1.5:** Picker uses repeated **Add images…** + **Import**; `IngestWithAssetIDs` returns summary + asset IDs for collection linking; **Journey A** order: ingest before collection confirm. **Review findings** on `upload.go` (double import before confirm, orphan collection, non-atomic create+link) — DnD must not worsen those; consider fixing in Story 1.5 or as part of 1.8 if trivially the same closure.
- **1.7:** Reinforces **one extension source** (`extensions.go`) and **honest** operation accounting — same spirit for “unsupported drop” messaging (NFR-04 observability tone).

### Latest technical information

- **Fyne file drop:** `SetOnDropped` is the supported cross-platform hook for external files (since v2.4); verify edge cases (sandboxed macOS paths, `file://` encoding) against [Fyne window / URI documentation](https://docs.fyne.io/) and `pkg.go.dev/fyne.io/fyne/v2`.

### Project structure notes

- Aligns with architecture **§5.1** `internal/app` for Fyne.
- Executable naming (`photo-tool` vs `phototool`) unchanged by this story.

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Story 1.8, Epic 1, UX-DR14, FR-01]
- [Source: _bmad-output/planning-artifacts/PRD.md — Upload: picker + DnD same pipeline]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — Component strategy: upload entry; UX-DR14; error tone]
- [Source: _bmad-output/planning-artifacts/architecture.md — §3.8 Fyne, §3.9 OperationSummary parity, §5.1–5.2 boundaries]
- [Source: _bmad-output/implementation-artifacts/1-5-upload-confirm-receipt.md — upload behavior, Journey A, review notes]
- [Source: internal/app/upload.go — `NewUploadView`, picker, `IngestWithAssetIDs`]
- [Source: internal/ingest/extensions.go — `PickerFilterExtensions`]
- [Source: internal/domain/summary.go — `OperationSummary`]

## Risk register (party mode / create review, session 1/2)

- **Fyne drop position:** On desktop, `SetOnDropped` is invoked with the window’s tracked pointer position (not a separate OS-reported release point). Hit-testing remains valid but may diverge on exotic drivers — document and re-check on Windows/Linux when available.
- **Scroll / layout:** The drop target lives inside a `Scroll`; `AbsolutePositionForObject` should include scroll offset, but high-DPI + nested scroll regressions warrant a manual pass.
- **Handler lifetime:** `SetOnDropped` replaces prior callbacks; `main` attaches one `NewUploadView` per window — do not mount multiple upload roots on the same `fyne.Window` without revisiting registration.
- **Mixed-drop UX:** A second dialog after import lists skipped items (honest but slightly chatty). Session 2 may propose consolidating into receipt-adjacent copy if product wants fewer modals.

## Party mode round (automated headless, **dev** hook, session **1/2**)

_Roster: `agent-manifest.csv` not present in-repo; round simulated. Communication: English (default)._

**Amelia (Dev):** The shared `runImportBatch` + `awaitingPostImportStep` gate matches AC1 and the post-import checklist. I’d still **tighten URI coverage**: table-drive `uriLocalPath` for `file://` edge cases so we don’t regress on a Fyne bump. I’m **against** adding `slog` in the drop path for MVP — noise without a log sink policy.

**Murat (Test Architect):** Disagree with Amelia on “tests are enough.” We still **don’t** unit-test `dropHitTest` because it binds `fyne.CurrentApp().Driver()` — call that out as **manual / GUI** risk, not pretend it’s covered. I want **one** more boundary case on `uriLocalPath` (`file://` with no path) in the table — that’s cheap insurance.

**Winston (Architect):** I’ll push back on Sally’s instinct to soften failures: **unsupported-only** must stay *factual* and *visible*; whether that’s `ShowError` vs `ShowInformation` is UX policy, not architecture. What matters is **no silent no-op** — we already satisfy that. For **scroll + hit-test**, document reliance on Fyne’s absolute positioning so we don’t “fix” it twice on every platform.

**Sally (UX Designer):** I disagree with Murat framing unsupported-only as an “error.” Users who drop a `.pdf` didn’t break the app — **error dialogs feel punitive**. Prefer **`ShowInformation`** with title **“No supported images”** and the same detail lines; keep **`ShowError`** for real I/O or DB failures elsewhere. Mixed-drop can stay information + receipt flow.

**Orchestrator synthesis (applied):** Use **`dialog.ShowInformation("No supported images", …)`** for unsupported-only drops (Sally + Winston: honest, proportionate tone). Extend **`TestURILocalPath`** table with **`file://`** empty-path case (Murat + Amelia). Document **scroll / absolute position** assumption beside `SetOnDropped` (Winston). Explicitly note **`dropHitTest` is not headless-unit-tested** in the risk register below. No `slog` in drop handler for this story.

## Risk register (party mode / dev review, session **1/2**)

- **`dropHitTest` / driver coupling:** Remains **integration-heavy**; validate hit target on scrolled content in manual QA (macOS baseline; Windows/Linux when available).
- **Unsupported-only tone (closed in code):** Information dialog with factual detail lines; still **no** ingest or hidden rejection.

## Party mode round (automated headless, **create** hook, session **2/2**)

_Roster: `agent-manifest.csv` not present in-repo; round simulated. Communication: English (default)._

**Sally (UX Designer):** I am pushing back on session 1's "mixed drop = second dialog after import." That pattern is honest but **modal-heavy**; we should eventually fold skip reasons into receipt-adjacent copy. For **this** story, keep it — changing receipt layout is scope creep — but track it as UX debt in the risk register.

**Winston (Architect):** Session 1 underplayed **state coupling**. While `postImport` is visible, the `paths` slice still holds the ingested batch; a second drop or another **Import** would **re-run ingest on the full list** — that violates "same pipeline" *semantics* (duplicate work, confusing receipts). **Hard gate** the batch until Confirm/Cancel: disable picker actions too, not only the drop path.

**Mary (Analyst):** AC5 says drops off-target are silent; good. We still lack proof that **`Visible()`** on nested stacks matches the user mental model for "blocked." Prefer an explicit **`awaitingPostImportStep` flag** tied to the same transitions as the receipt UI — auditable in code review.

**Amelia (Dev):** Disagree with Winston that we need a new queue system — a **flag + disable Add/Clear/Import** plus a drop dialog is smaller and testable. Extract **pure geometry** for hit-test (`rectContainsPoint`) so we are not "testing Fyne" but we still lock the math.


**Orchestrator synthesis (applied):** Add **`awaitingPostImportStep`**; after successful batch ingest, **disable Add / Clear / Import** until Confirm/Cancel resets UI; **block drop** on the target with a factual dialog. Add **unit tests** for **deduped drops** and **hit-rectangle containment**. Keep mixed-drop second dialog for now; note consolidation as follow-up UX. Story → **review**; sprint item → **review**.

## Risk register (party mode / create review, session 2/2)

- **Post-import re-entrancy (closed in code):** Drops or extra imports while the collection step is active could re-ingest prior paths — mitigated by **`awaitingPostImportStep`** + disabled batch controls + drop message.
- **Modal load (open):** Mixed-drop still uses an extra information dialog after import; acceptable for MVP per Sally/Winston — revisit if users complain.
- **Windows `file:` URI edge cases (open):** `uriLocalPath` uses `fyne.URI.Path()`; validate on Windows CI when available (drive letters, encoded paths).

## Party mode round (automated headless, **dev** hook, session **2/2**)

_Roster: `agent-manifest.csv` not present in-repo; round simulated. Communication: English (default). Session **2/2** challenges prior conclusions — not a recap of dev 1/2._

**Murat (Test Architect):** Dev 1/2 nailed **classification** and **post-import** gates, but we still had a **logical hole**: drop always called **`runImportBatch`** after merging supported paths. If the batch list **already** contained every dropped file (picker-first workflow), that was a **silent second ingest** of the same list — not covered by `classifyDroppedURIs` tests. I’m not buying “manual QA will catch it”; we need a **deterministic guard** plus a **unit** on path dedupe.

**Amelia (Dev):** Pushback: that’s not a Fyne bug — it’s our **mutation contract**. **`tryAddUniquePath`** (or `addAbsolute` returning **added?**) fixes it cleanly and also avoids pointless **`Refresh`** when the picker selects the same file twice. I’ll wire **`anyNew`** so we **skip** `runImportBatch` when the drop adds nothing.

**Winston (Architect):** Murat is right on the symptom; Amelia’s fix matches the architecture rule: **a batch ingest is a function of the current path set**, and you don’t advance the operation without a **set delta**. This is smaller than a queue and doesn’t fork **`IngestWithAssetIDs`**.

**Sally (UX Designer):** If nothing new lands, don’t punish users with an **error** — **`ShowInformation("No new files to import", …)`** with a short explanation. When it’s **all duplicates plus** skipped junk, **one** dialog that stacks “already in list” **then** skip lines beats two back-to-back modals.

**Orchestrator synthesis (applied):** Introduce **`tryAddUniquePath`**; drop handler sets **`anyNew`** and **only then** calls **`runImportBatch`**; if **`!anyNew`**, show **`ShowInformation("No new files to import", …)`** combining duplicate-list copy with unsupported lines when present. Add **`TestTryAddUniquePath`**. Extend manual QA with **“drop files already in list (after picker add)”**. **`dropHitTest`** remains integration-heavy (unchanged from dev 1/2).

## Risk register (party mode / dev review, session **2/2**)

- **Pre-import duplicate re-ingest (closed in code):** Drops that resolve only to paths **already** in the upload list no longer trigger **`runImportBatch`**; user sees **information**, not a second silent ingest.
- **`dropHitTest` / driver coupling (open):** Same as dev 1/2 — manual / GUI validation on scrolled layouts.
- **Windows `file:` URI edge cases (open):** Unchanged; validate on Windows when available.

## Dev Agent Record

### Agent Model Used

BMAD party mode (automated headless, **create** hook), session **1/2**, 2026-04-13 — `agent-manifest.csv` missing in repo; round simulated + synthesis applied in-tree.

BMAD party mode (automated headless, **create** hook), session **2/2**, 2026-04-13 — simulated round (Sally / Winston / Mary / Amelia); **post-import re-entrancy** guard + **`rectContainsPoint`** tests + **dedupe drop** test applied.

BMAD party mode (automated headless, **dev** hook), session **1/2**, 2026-04-13 — simulated round (Amelia / Murat / Winston / Sally); **unsupported-only → information dialog**, **URI table tests**, **scroll/hit-test comment**, **dropHitTest** risk documented.

BMAD party mode (automated headless, **dev** hook), session **2/2**, 2026-04-13 — simulated round (Murat / Amelia / Winston / Sally); **pre-import duplicate-drop re-ingest** guard via **`tryAddUniquePath`** + **`anyNew`**, combined **“No new files to import”** dialog, **`TestTryAddUniquePath`**.

### Debug Log References

### Completion Notes List

- Implemented shared `runImportBatch`, themed drop target, `SetOnDropped` with hit-test, URI→path + extension/directory classification via `ingest.IsSupportedIngestExt`, user messaging for unsupported-only and mixed drops; unit tests for URI/path classification (`internal/app/drop_paths_test.go`).
- Session **2/2:** **`awaitingPostImportStep`** blocks accidental **re-ingest** (drop dialog; **Add / Clear / Import** disabled until collection Confirm/Cancel). **`rectContainsPoint`** + tests; **duplicate URI** classification test.
- Session **dev 1/2:** Unsupported-only drops use **`ShowInformation("No supported images", …)`**; table-driven **`TestURILocalPath`** includes `file://` empty path; **`SetOnDropped`** comment documents scroll/absolute-position assumption; story risk register notes **`dropHitTest`** not covered headless.
- Session **dev 2/2:** **`tryAddUniquePath`** centralizes list dedupe; drop path skips **`runImportBatch`** when the drop adds **no** new paths (avoids duplicate ingest when files were already accumulated via picker); **`TestTryAddUniquePath`**.
- **Tests / verification:** `go test ./...` and `go build .` pass. Manual QA matrix captured below (execute on macOS GUI baseline before marking story **done**).

### Manual QA checklist (macOS baseline; GUI)

Run the desktop app with a writable library root. Check each row; note failures in sprint retro or a new defect.

- [ ] **Single file drop** — Drop one `.jpg` (or other allowed type) onto the bordered “Drop images here” target. Expect: path appears in list, ingest runs, receipt + post-import (collection) UI matches picker + Import behavior.
- [ ] **Multi-file drop** — Drop several allowed images at once. Expect: all appear (deduped if duplicates), single import batch, receipt reflects combined outcome.
- [ ] **Unsupported-only** — Drop only `.txt`, a folder, or non-`file` expectation where applicable. Expect: **information** dialog titled **“No supported images”** with factual detail lines; **no** silent no-op; **no** ingest/receipt that hides rejection.
- [ ] **Mixed supported/unsupported** — Drop one good image + one bad item. Expect: supported ingested per normal pipeline; user sees follow-up about skipped items (second dialog after import is acceptable for this story).
- [ ] **Duplicate paths in one drop** — Drop the same file URI twice in one gesture. Expect: one logical add (no duplicate rows); ingest behaves consistently.
- [ ] **Drop files already in list (picker first)** — Use **Add images…** to add a file, **do not** tap Import; drop the **same** file onto the drop target. Expect: **information** dialog **“No new files to import”** explaining files are already listed; **no** ingest/receipt; user can still tap **Import selected files** once to run the batch.
- [ ] **Drop while `postImport` visible** — After a successful drop/import, while receipt/collection UI is showing: attempt another drop on the target. Expect: **informational** dialog (“Finish collection step…”); **Add images…**, **Clear list**, and **Import** remain **disabled** until **Confirm** or **Cancel**; then controls reset per existing Story 1.5 flow.
- [ ] **Off-target drop (AC5)** — Release files on the window **outside** the drop zone (e.g. on the path list). Expect: **no** ingest and **no** error dialog.

### File List

- `internal/app/upload.go`
- `internal/app/drop_paths.go`
- `internal/app/drop_paths_test.go`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/implementation-artifacts/1-8-drag-drop-upload.md`

## Change Log

- **2026-04-13:** Marked Tests / verification complete; added Manual QA checklist to Dev Agent Record; confirmed `go test ./...` and `go build .` green.
- **2026-04-13 (party dev1/2):** Unsupported-only drop → information dialog; expanded `TestURILocalPath`; scroll/hit-test comment in `upload.go`; story risk register + party round recorded.
- **2026-04-13 (party dev2/2):** `tryAddUniquePath` + drop **`anyNew`** gate prevents duplicate ingest when drops add only paths already in the list; `TestTryAddUniquePath`; story party round + manual QA row updated.

---

**Story context:** Ultimate context engine analysis completed — comprehensive developer guide created (BMAD create-story workflow, 2026-04-13).
