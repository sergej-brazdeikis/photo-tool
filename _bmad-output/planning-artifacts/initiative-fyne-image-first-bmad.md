# Initiative: Fyne image-first UX (BMAD workflow)

**Goal:** Close the gap between **revised UX** (image stage dominates; minimal control soup) and **current Fyne**, using BMAD skills in a repeatable order.

**Canonical inputs (put these in every new story’s `inputDocuments`):**

| Artifact | Role |
|----------|------|
| `_bmad-output/planning-artifacts/ux-design-specification.md` | Party follow-ups, UX consistency patterns, image stage / density |
| `_bmad-output/planning-artifacts/ux-design-directions.html` | Direction A (shell), Direction E (ingest previews) |
| `_bmad-output/planning-artifacts/epics-v2-ux-aligned-2026-04-14.md` | UX-DR16–19, Story 1.5 / 2.1–2.3 / 2.11 AC language |
| `_bmad-output/planning-artifacts/initiative-fyne-image-first-phase1-party-2026-04-15.md` | Phase 1 party synthesis: layout budget template, risk rows, UX and architecture coherence |
| `_bmad-output/planning-artifacts/initiative-fyne-image-first-tea-notes.md` | **Murat / TEA:** risk-based test strategy, AC-to-automation map, NFR-01 headless vs manual, **UX-DR17** `-race` guidance |
| `_bmad-output/planning-artifacts/PRD.md` | FRs unchanged — layout is NFR-01 + UX-DR layer |
| `_bmad-output/planning-artifacts/architecture.md` | Fyne boundaries, thumbnail pipeline, `fyne.Do` / threading |

---

## Phase 0 — Orient (optional, 5–10 min)

| Skill | Invoke when | Outcome |
|-------|-------------|---------|
| `bmad-help` | You are unsure which skill to run next | Next-step recommendation from `_bmad/_config/bmad-help.csv` |
| `bmad-sprint-status` | You want a snapshot of story states | Risk/readiness summary from epics + `sprint-status.yaml` |

---

## Phase 1 — Spec fidelity (before new story files)

Lock **layout intent** so implementation stories do not drift.

**Headless YOLO (e.g. `scripts/initiative-fyne-image-first-yolo.sh`):** Phase 1 runs **two** party rounds (UX + architect); round 2 challenges round 1. Outcomes land in [initiative-fyne-image-first-phase1-party-2026-04-15.md](initiative-fyne-image-first-phase1-party-2026-04-15.md) and small deltas here / epics / `architecture.md` §3.8.1 — **no production Go** in this phase.

**Deliverables to carry into stories**

- **Layout budget table** (region × min/max height × scroll owner × **measurement box** × **lifecycle moment** × fixed vs flexible loading reservation).
- **Risk register** rows tying UX contracts to pipeline ADRs (see party doc example **R-UX-01**).
- **Journey × emotional state × recovery** one-pager (record requirement in party doc; implement in UX spec or a short `ux-journey-latency-states.md` on next UX edit).
- **CI vs manual matrix:** classify by **user-visible invariant** first, then map to CI / headless Fyne / manual NFR-01 — not “whatever is easy in CI.”

| Skill | Invoke when | Outcome |
|-------|-------------|---------|
| `bmad-agent-ux-designer` (Sally) | Ambiguity on thumb min size, receipt collapse, filter overflow | Concrete Fyne-feasible notes; optional deltas to UX spec sections |
| `bmad-agent-architect` (Winston) | Thumbnail decode/cache, main-thread rules, scroll cancel | ADR-sized notes in architecture or story Dev Notes |

Optional: **`bmad-party-mode`** — one round on “Upload vs Review vs shell” priority if Sally/Winston disagree.

---

## Phase 2 — Story breakdown (one spec file per shippable slice)

Use **`bmad-create-story`** so each slice gets AC, tasks, file pointers, and tests — **do not** jump straight to code without a story.

**Suggested slices (create in this order for dependency sanity):**

| Order | Focus | Epic anchor in v2 epics | Primary code touchpoints |
|-------|--------|-------------------------|---------------------------|
| 1 | **Upload:** large batch previews + collapsible / secondary receipt; fewer competing buttons where possible | Story **1.5** (+ UX-DR6, UX-DR17) | `internal/app/upload.go`, tests alongside |
| 2 | **Shell:** compact nav / vertical budget for Review (Direction A) | Story **2.1** (+ UX-DR16) | `internal/app/shell.go`, `theme.go` if needed |
| 3 | **Filter strip:** single row + overflow (sheet / “More filters”) | Story **2.2** (+ UX-DR2) | `internal/app/review.go` |
| 4 | **Grid:** image-forward cells; responsive columns / min cell size; defer nonessential chrome | Story **2.3** (+ UX-DR3, UX-DR16) | `internal/app/review_grid.go`, `review.go` |
| 5 | **Evidence:** record min thumb / loupe region / chrome height in NFR matrices | Story **2.11** (+ UX-DR16, UX-DR19) | `nfr-01-layout-matrix-evidence.md`, `internal/domain/nfr_layout.go` as needed |

If a slice is large, split into **two** `bmad-create-story` runs (e.g. “upload previews only” then “receipt collapse only”) rather than one mega story.

After each story file exists or changes:

| Skill | Outcome |
|-------|---------|
| `bmad-sprint-planning` | Refresh `_bmad-output/implementation-artifacts/sprint-status.yaml` keys and `development_status` |

---

## Phase 3 — Implement

| Skill / automation | When |
|--------------------|------|
| `bmad-dev-story` | Full workflow: implement story file AC + tasks, update Dev Agent Record |
| `bmad-quick-dev` | Smaller tweak when story already exists and scope is narrow |
| `./scripts/bmad-story-workflow.sh --phase=dev --skip-readiness --skip-retro … <story-id>` | Headless Cursor `agent` driving dev-story equivalent |

Keep **`go test ./...`** and Fyne headless tests green; use **`GO_TEST_RACE=1`** selectively for ingest/grid paths if you touch goroutines (**UX-DR17**).

---

## Phase 4 — Quality gates

| Skill | When |
|-------|------|
| `bmad-tea` (Murat) | Test design for new states (loading/error/list), resize, thread safety — see [initiative-fyne-image-first-tea-notes.md](initiative-fyne-image-first-tea-notes.md) |
| `bmad-checkpoint-preview` | Human walkthrough before you mark story **done** |
| `bmad-code-review` | Adversarial pass before merge / “done” |

Optional: **`bmad-qa-generate-e2e-tests`** — extend CLI or headless Fyne coverage if AC allow.

### CI vs local UX automation

| Where | What runs | What does **not** run |
|-------|-----------|------------------------|
| **CI** ([`.github/workflows/go.yml`](../../.github/workflows/go.yml)) | `go test ./...`, `bash -n` on helper shell scripts, deterministic Fyne/layout tests (for example release-shell invariants in `internal/app/ux_layout_invariants_test.go`) | Cursor **`agent`**, LLM judge, screenshot review loop |
| **Local (developer machine)** | [`scripts/assemble-judge-bundle.sh`](../../scripts/assemble-judge-bundle.sh) → fresh `logs/`, `ui/*.png`, `ui/steps.json`, `manifest.json`; optional [`make ux-judge-loop`](../../Makefile) / [`scripts/ux-judge-loop.sh`](../../scripts/ux-judge-loop.sh) for **capture → vision judge → fix → repeat** until `UX_JUDGE_RESULT=pass` or `UX_LOOP_MAX` | N/A |

Prompts (local only): [`judge-prompt-v2-screenshots.md`](../test-artifacts/judge-prompt-v2-screenshots.md), [`ux-fix-from-judge-prompt-v1.md`](../test-artifacts/ux-fix-from-judge-prompt-v1.md). For a single agent invocation that should drive the loop from chat, see [`ux-implement-loop-prompt-v1.md`](../test-artifacts/ux-implement-loop-prompt-v1.md). Cursor CLI binary: prefer `UX_AGENT`, or `CURSOR_AGENT` as an alias in `ux-judge-loop.sh`.

---

## Phase 5 — Close

| Skill | When |
|-------|------|
| `bmad-sprint-planning` | Final sync of sprint YAML after statuses → **done** |
| `bmad-retrospective` | After a milestone epic or initiative chunk — process improvements for the next UI pass |

---

## One-line “default path” for Cursor chat

1. `bmad-create-story` for the next slice (cite **epics-v2** section + UX spec).  
2. `bmad-dev-story` on that file (or workflow script `--phase=dev`).  
3. `bmad-checkpoint-preview` + `bmad-code-review`.  
4. `bmad-sprint-planning` to update **`sprint-status.yaml`**.

---

## Initiative exit criteria (initiative-level)

- Upload confirm step shows **large** batch previews; receipt is **readable** and **collapsible** after first-use path (per v2 Story 1.5).
- Review: **filter strip** does not become a second nav; grid **defaults** image-dominant with documented **min** sizes in **2.11** evidence.
- No regression on **NFR-01** / **UX-DR17** (layout + main-thread updates).

_Last updated: 2026-04-15 — Phase 1 party synthesis added; still aligns with `epics-v2-ux-aligned-2026-04-14.md`._
