# Initiative Fyne image-first — Phase 1 party synthesis (2026-04-15)

**Context:** Headless YOLO Phase 1 pass per `initiative-fyne-image-first-bmad.md` — spec fidelity before Stories **1.5**, **2.1–2.3**, **2.11**. Voices: **Sally** (UX), **Winston** (Architect). Two rounds; round 2 challenges round 1.

**Related:** [epics-v2-ux-aligned-2026-04-14.md](epics-v2-ux-aligned-2026-04-14.md) (UX-DR16–19, Story ACs), [architecture.md](architecture.md) §3.8.1.

---

## Round 1/2 — agent transcripts

### Sally (UX Designer) — Round 1

Picture someone dragging a folder in, breathing out when thumbnails appear, then freezing because they cannot tell *what happened*—receipt missing, confirm buried, or the same chrome they use for *living in the library* shouting over the moment of *committing imports*. Phase 1 is not “pretty screens”; it is **one mental model** from Upload → Review so nobody has to guess which surface is “temporary work” versus “home.”

**Lock these spec decisions now**

1. **Two modes, one shell.** Shell/nav/themes (2.1) must define a single frame: where upload flow *enters*, where it *lands* after confirm, and that the filter strip is **never** a second navigation rail (UX-DR2). If the strip can look like “another sidebar,” coherence breaks before a single image loads.

2. **Height budget on paper.** UX-DR16 is not aspirational here—it is the spine. Lock: approximate vertical slices for chrome vs grid vs loupe (or loupe-adjacent region), minimum thumbnail edge, and “grid/loupe dominates” as a **ratio or pixel band**, not a vibe. Without that, 2.3 and 2.11 cannot prove they match the bar.

3. **Upload slice = linear story, not a dashboard.** Story 1.5 needs a fixed sequence: previews visible → **receipt** (what will be ingested, counts, failures surfaced clearly) → **confirm** (irreversible-feeling primary action). Lock what lives *on* confirm vs *before* confirm so implementers do not invent a third step.

4. **Threading contract visible in UX copy/behavior.** UX-DR17: the user should *feel* responsiveness (spinners, disabled-but-clear states), but Fyne stays on the main thread—lock which UI elements reflect async work so `-race` and ingest/grid stories do not reinterpret “loading” as “broken.”

5. **Grid states as a finite set.** UX-DR18: empty, loading, partial error, populated—each gets one primary affordance and one honest message. Lock those states so “image-forward” does not become “mystery blank.”

6. **Minimum window = primary actions survive.** UX-DR19: at min size, upload confirm and core review actions (e.g. proceed, dismiss flow, critical nav) must not clip; focus order documented. That is how you avoid “works on my laptop.”

**Measurable checks to document for Story 2.11 (layout matrix evidence)**

- **Chrome vs content:** At reference breakpoints (min window, “comfortable,” max useful), record **heights/widths** of persistent chrome vs **grid/loupe** region; assert grid+loupe meets the DR16 budget (e.g. ≥ X% of client area, or ≥ Y px for loupe).

- **Thumbnails:** **Minimum edge length** of a cell at min window; **max cells visible** without horizontal scroll where the spec says so; **tap/click target** meets minimum (document the number).

- **Filter strip:** One row; **overflow opens sheet**—capture screenshots or matrix cells: collapsed vs overflow menu/sheet; assert no second persistent nav column.

- **States:** Matrix row per UX-DR18 state with **expected primary CTA** and **focus order** (tab stops listed).

- **Upload flow:** Time-to-first-preview threshold or “progressive disclosure” checklist: receipt fields present, confirm enabled only when rules say so.

**If tradeoffs appear: upload vs review**

**Upload wins on clarity and completion; review wins on sustained browsing.** In a conflict, bias **upload** for *blocking errors, receipt truth, and confirm affordance*—because a wrong import is harder to undo than a slightly cramped grid. Bias **review** for *density, filter behavior, and grid/loupe dominance* once the user is “in the library.” Stated bluntly: do not steal upload’s certainty to polish review chrome; do not sacrifice DR16’s “images dominate” on the review side to stack upload controls on every screen.

**“Control soup” risks**

- **Upload confirm** tempts teams to add every option (naming, tags, albums, duplicates, AI toggles) on one sheet. That reads like *settings*, not *commit*—users hesitate or mis-tap. Lock **one primary** on confirm; park advanced actions behind explicit “more” or post-import review.

- **Review** tempts parallel controls (filters + sort + multi-select + batch + loupe tools) without hierarchy. Without overflow/sheet discipline (DR2), you get two competing “command layers.” Spec should name **primary action per context** (e.g. browsing vs selecting) so Phase 2 does not decode intent from Figma guesses.

Headless YOLO only works if these are **tables and numbers** in the spec, not paragraphs of taste. Lock the frame, the budget, the upload sequence, the state machine, and the min-window survival list—then implementers are painting inside lines, not improvising a second product.

### Winston (Architect) — Round 1

**Phase 1 documentation (architecture view)**

**ADRs (short, binding, testable)**

- **Thumbnail pipeline contract**: what runs on the UI thread (layout, invalidation, binding updates) versus worker pools (decode, disk/IO, hashing, EXIF). Explicitly state *who* posts results back and *how* (channel, `fyne.Do`, equivalent)—this is the mechanical spine of **UX-DR17**.

- **Cancellation & lifecycle**: scroll, folder change, and “upload batch preview” must define cancel tokens or epoch IDs so stale decode jobs never repaint. Without this, image-first layout and async work *will* fight in ways that look like flaky UI bugs.

- **Cache policy**: max bytes, max entries, eviction order, and what happens under memory pressure. Bounds are not optimization; they are correctness for scroll and batch ingest.

- **Shell / vertical budget**: how the filter strip and viewport compete for height; where flex vs fixed lives; what degrades when space is tight. Tie numeric thresholds to **UX-DR16** via the **NFR-01** matrix so “magic numbers” live in evidence, not chat.

**Dev Notes pointers**

- One page that maps **user journeys** → **goroutine boundaries** → **UI callbacks** (ingest complete, thumb ready, error). Developers should not infer this from code in Phase 1.

- A **threading diagram** (even ASCII): producer → queue → decode → GPU/upload-ready handoff → main-thread paint. Mark every crossing.

**Acceptance hooks**

- **CI**: static checks where possible—e.g. lint or review gates that forbid blocking calls in UI handlers for listed packages; build tags or interface seams for “headless” thumbnail pipeline tests.

- **Manual / matrix**: **NFR-01** viewport × density × scroll behavior; document which cells are automated vs manual so the spec is not “everything is QA’s problem.”

- **Definition of done** per story: e.g. “no frame > X ms blocked on decode under scripted scroll” or “cancelled job never updates UI”—pick measurable proxies, not vibes.

**Tie to UX-DR17 and the thumbnail pipeline**

**UX-DR17** is satisfied when the spec says: decode and cache never hold the main thread; ingest callbacks only enqueue work and schedule a single coalesced refresh. Phase 1 should document **coalescing** (debounce invalidations during fast scroll) and **priority** (visible range first). Without that, image-first layout will thrash: layout runs, thumbs arrive out of order, and you get jank that looks like a Fyne bug but is a missing contract.

**Where I refuse Phase 1 scope creep**

- **Full YOLO integration**, model tuning, or “smart” detection UX—only **thread boundaries and validation split** (CI vs matrix) belong here; not the vision stack.

- **General-purpose async framework** or abstractions “for all future features.” Ship the minimal pipeline for thumbs + ingest.

- **Pixel-perfect cross-platform parity** beyond what **NFR-01** explicitly lists—document the matrix, don’t chase every edge in code.

- **Deep perf engineering** without baselines—instrumentation hooks yes; rewriting hot paths no, until stories 2.x are green against the written contracts.

Phase 1 wins when the documents make it *hard* to block the UI thread by accident and *obvious* when a change crosses a thread boundary. Everything else waits for evidence from the matrix and the first integrated vertical slice.

### Sally — Round 2 (challenge / deepen)

**Where Winston risks “correct but mean.”** When we optimize for cancellation epochs, cache bounds, and ADR completeness, we can ship something that is *technically* never wrong but *emotionally* punishing: thumbnails that pop in with no affordance for “we threw your work away,” scroll that feels “stuck” because coalescing wins over responsiveness, or errors that are accurate but read like blame (“stale decode”) instead of recovery. Engineering-centricity here shows up as treating **latency variance** and **aborted work** as invisible implementation details. Users experience them as trust events. The mean version is: silent drops, jittery grids, and status text that mirrors goroutine state instead of human intent.

**Where UX-DR16 (ratio) can break on Fyne.** Ratios assume we control layout the way design tools do. Fyne’s min sizes, scroll containers, and platform chrome can force **minimum heights and widths** that fight a pure ratio—especially with filters, badges, and focus rings. We may end up with “ratio in name only” (letterboxing that wastes vertical budget) or **content clipping** if we refuse to break ratio. The failure mode is not ugliness; it’s **unreadable labels** and **unreachable controls** at small windows—exactly where UX-DR19 (min-window survival) matters.

**One artifact so headless YOLO doesn’t flatten nuance.** A **“journey × emotional state × recovery”** one-pager (not a wireframe): for each critical path (open folder, scroll burst, upload, confirm), list what the user *feels* at p50 vs p95 latency, what we show, and what they can do next. Headless CI can assert timings; this artifact forces **language, tone, and affordances** to stay in the spec so automation doesn’t optimize away empathy.

**Explicit disagreement with Winston.** Sally pushes back on framing **CI vs manual matrix split** primarily as a coverage problem. For Phase 1, it’s a **risk to perceived quality**: what we automate becomes “truth,” and everything else quietly erodes. She wants the split justified by **user-visible outcomes** (e.g., “we never ship a build where the grid can look empty without explanation”), not only by what’s easy to hook in CI.

### Winston — Round 2 (challenge / deepen)

**Where Sally underspecifies implementation / re-layout thrash**

- **Upload vs review** is named as two modes but not as **two stable layout contracts**: who owns scroll (thread vs page), whether the composer is **sticky vs in-flow**, and whether the image strip is **above or beside** the thread at each breakpoint. Without pinned rules, “linear 1.5 flow” will thrash when preview height, keyboard, or safe-area changes mid-session.

- **Numeric height budget (DR16)** needs **which box is measured** (viewport, content pane, thread column, composer block) and **when** (initial paint, after images decode, after font load). Without that, engineering will pick one interpretation and UX will pick another—classic relayout when intrinsic size arrives.

- **Loading states as contract (DR17)** should state **layout reservation**: skeleton fixed height vs flexible growth. Flexible loading → cumulative layout shift when data lands; fixed reservation → empty gutters if cancelled—both need an explicit choice per region.

- **Finite grid states (DR18)** needs **transition graph**: allowed edges (e.g. empty → loading → populated → error) and **forbidden** oscillations (e.g. error ↔ loading loop). Without it, “finite” is a label, not a guardrail for state machines and tests.

- **Min-window primary actions (DR19)** must fix **action bar vs overflow**: at min width, do actions collapse to a single “More” with **stable hit targets**, or reflow to two rows? Unspecified → reflow thrash when the window is resized across the threshold.

**Headless YOLO failure modes (specs without human taste)**

- **Normative drift**: agents “optimize” copy and AC until they **pass checks** but **lose product intent**.

- **Silent scope expansion**: specs absorb **implementation-friendly** shortcuts without stakeholder tradeoff—then UX ratios and reality diverge.

- **Matrix theater**: evidence grids filled with **passing rows** that don’t encode **real user paths** (cancel mid-upload, slow decode, tiny window).

- **Aesthetic vacuum**: density and “control soup” delegated to Fyne defaults; confirm vs review still feels crowded.

**Require in architecture / initiative (numeric UX vs thread/cache)**

- **Single source of truth diagram**: thread model, upload pipeline, cache layers, and **which UI regions read from which layer**.

- **Layout budget table**: for each DR16–DR19 region, columns: **min/max height**, **resize trigger**, **scroll owner**, **loading reservation** (fixed vs flexible).

- **Coherence clause**: any numeric ratio in UX docs must cite **the same measurement anchor** as architecture (viewport vs pane) and **the lifecycle moment** (before/after decode, after cache hit). If UX and architecture disagree, **one** must win in writing (update ADR or UX spec explicitly).

- **Cancellation & coalescing**: explicit rules for in-flight uploads/previews and how the UI **must not** flash intermediate grid states (ties DR17/18 to pipeline ADRs).

---

## Orchestrator synthesis (applied)

### Locked for Phase 1 (both voices)

| Theme | Commitment |
|-------|------------|
| Shell & modes | One frame for Upload → Review; filter strip never reads as second nav (UX-DR2). |
| Numbers | Height budget, min thumb edge, loupe region: **ratios or px** + **same measurement box** everywhere (see §Layout budget template). |
| Upload 1.5 | Linear: previews → receipt → confirm; one primary on confirm; advanced options deferred. |
| Async | UX-DR17: workers decode/IO; main thread mutates Fyne; coalescing + cancellation/epoch documented. |
| Grid | UX-DR18: finite states **plus** transition rules (allowed edges, no oscillation loops). |
| Min window | UX-DR19: primary actions + focus; explicit overflow vs second-row behavior. |
| Evidence | Story 2.11 matrix records anchors, states, filter overflow, upload checklist. |

### Resolved tension: CI vs manual matrix

**Synthesis:** Classify checks by **user-visible invariant** first (e.g. “no unexplained empty grid,” “cancelled job never repaints”), then map each to **CI** (automated/hook), **headless** (where Fyne tests allow), or **manual matrix** (NFR-01 / OS scaling). The split is documented in the layout/risk artifacts so “automated” is not mistaken for “sufficient.”

### New / updated artifacts (this initiative phase)

1. **Journey × emotional state × recovery** (Sally): add as section in [ux-design-specification.md](ux-design-specification.md) when next edited, or a one-page `planning-artifacts/ux-journey-latency-states.md` — *deferred until UX spec touch*; this file records the requirement now.

2. **Layout budget table** — template below; first fill belongs with Story 2.11 / NFR evidence.

3. **Risk register** — column format and starter row below.

---

## Layout budget table (template)

Use one row per region (nav, filter strip, grid, loupe chrome, upload preview band, receipt, confirm bar). Copy into Story 2.11 notes or `nfr-01-layout-matrix-evidence.md` when created.

| Region | Min H | Max H | Resize trigger | Scroll owner | Measurement box (viewport / pane / cell) | Lifecycle moment (initial / post-decode / post-cache) | Loading reservation (fixed / flexible) | Notes (UX-DR) |

---

## Phase 1 risk register (format + example)

| ID | Risk | Cause | Impact | Likelihood | Detection | Mitigation | Owner | Status |
|----|------|-------|--------|------------|-----------|------------|-------|--------|
| R-UX-01 | Composer / upload-review relayout thrash under decode | DR16 budget not anchored to content pane + decode timing | DR19 failures, perceived jank | Med | Resize + slow decode matrix | Fixed skeleton heights; single scroll owner; ADR for measurement anchor | Arch + UX | Open |

_Add rows as Phase 1 spec work continues._

---

## Headless YOLO guardrails (orchestrator)

- After automated edits to planning artifacts, **spot-check** that numeric thresholds still cite one measurement anchor.

- Prefer **user-visible invariants** in AC over implementation-only wording so normative drift is visible in review.

- Fill matrix rows for **stress paths** (cancel mid-upload, min window, slow decode), not only happy path.

---

_Last updated: 2026-04-15 (Phase 1 party, initiative YOLO script driver)._
