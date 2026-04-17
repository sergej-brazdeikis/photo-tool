# UX judge loop — self-improve pass (local / Cursor `agent` only)

You run **once per loop iteration** (after judge + implementer + optional post-fix review) when `scripts/ux-judge-loop.sh` invokes you. Goal: **tighten the machine** so the next iteration catches regressions earlier and stays aligned with the UX spec.

## Inputs

- **Repository root** — read-only for product intent; **edits allowed only** to artifacts listed under “Allowed edits”.
- **Version control:** This contract file is listed as a prerequisite in `scripts/ux-judge-loop.sh`; keep it **committed** so clean checkouts do not fail before the first iteration.
- **Bundle directory** — read `verdict/judge-output.md`, `verdict/code-review-post-fix.md` (if present), `ui/steps.json`, `manifest.json`, and skim `logs/ux-capture.txt` if the manifest flags capture issues.

## Tasks

1. **Delta review** — In 5–10 bullets, list what **slipped past** this bundle’s capture + judge contract (e.g. missing window size, missing theme path, rubric too vague, prompt missing a checklist item). Tie each bullet to a **root cause** (capture / prompt / rubric / implementer / structural). If **`context/requirements-trace.md`** in the bundle disagrees with shipped reality (epics, tests, or `ui/steps.json`), call out **distillate drift** and whether `_bmad-output/test-artifacts/bundle-requirements-trace.md` needs an update.
2. **Spec alignment** — Note any mismatch between `_bmad-output/planning-artifacts/ux-design-specification.md` and what the judge was asked to enforce; propose **one** concrete sentence to add to `judge-prompt-v2-screenshots.md` if needed. Cross-check **`bundle-requirements-trace.md`** against `_bmad-output/planning-artifacts/epics.md` (or `epics-v2-ux-aligned-2026-04-14.md`) when story closure or capture coverage changed this iteration—flag rows that should flip **Loop risk** or gain new **Journey capture** ids.
3. **Automation proposal** — Propose the **smallest** next automation (prefer `TestUXJourneyCapture`, `nfr01_layout_gate_test`, or headless shell tests) that would have caught the top slip; if the slip was a **trace GAP** only, say whether updating **`bundle-requirements-trace.md`** (without new code) is enough or tests/capture must change. If none is worth the cost, say so explicitly.

## Allowed edits (optional; keep diffs minimal)

You **may** apply small patches **only** to:

- `_bmad-output/test-artifacts/judge-prompt-v2-screenshots.md`
- `_bmad-output/test-artifacts/e2e-usability-rubric.md`
- `_bmad-output/test-artifacts/bundle-requirements-trace.md` (FR/story matrix copied into each bundle as `context/requirements-trace.md`; keep aligned with epics when stories ship or capture steps change)
- `_bmad-output/test-artifacts/ux-fix-from-judge-prompt-v1.md`
- `_bmad-output/test-artifacts/ux-qa-post-fix-loop.md`
- `_bmad-output/test-artifacts/ux-fix-from-qa-prompt-v1.md`
- `_bmad-output/test-artifacts/ux-fix-from-gui-e2e-macos-prompt-v1.md`
- `_bmad-output/test-artifacts/ux-judge-loop-self-improve.md` (this file — meta only, e.g. clarify instructions)
- `scripts/ux-judge-loop.sh` (comments / env docs / ordering only unless a clear bug)
- `scripts/assemble-judge-bundle.sh` (comments / manifest docs only unless a clear bug)
- `scripts/run-gui-e2e-macos.sh` (comments / env docs / robustness only unless a clear bug)
- `internal/app/ux_journey_capture_test.go` (extra capture steps / env toggles)
- `tests/gui_macos/README.md` (permissions / flakiness / env docs only unless a clear bug)

Do **not** change application UI code in this role (that is the implementer’s job). Do **not** commit secrets.

## Output (required)

Write **`verdict/loop-self-improve.md`** inside the **bundle** directory with:

1. **Summary** — What to change next iteration (1 short paragraph).
2. **Slipped past** — Bullets from task 1.
3. **Proposed changes** — Bullets: file → one-line intent (implement in-repo or leave as human backlog).
4. **Repo edits applied** — List each file touched with one line each, or state **none**.

If you applied repo edits, end with: `SELF_IMPROVE_REPO_EDITS=1`  
Otherwise: `SELF_IMPROVE_REPO_EDITS=0`
