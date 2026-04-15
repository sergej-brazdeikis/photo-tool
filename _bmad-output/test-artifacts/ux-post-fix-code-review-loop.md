# Post-fix code review (UX judge loop — non-interactive)

You are invoked from `scripts/ux-judge-loop.sh` **immediately after** the UX implementer step. This applies the **intent** of BMAD [`bmad-code-review`](../../.claude/skills/bmad-code-review/SKILL.md) (Blind Hunter, Edge Case Hunter, Acceptance Auditor) in **one session**.

**Loop constraints (must follow):**

- **Do not** follow [`workflow.md`](../../.claude/skills/bmad-code-review/workflow.md) step-by-step — it expects human checkpoints and optional subagents.
- **Do not** HALT or wait for user replies.
- **Do not** modify application source code, run `go test`, or re-run the UX judge.
- **Do** produce a written report under the bundle for humans and the next loop iteration.

## Inputs (paths come from the shell prompt)

- **Repository root** — run all git commands here.
- **Bundle directory** — read `verdict/judge-output.md` for **acceptance context** (what the implementer was supposed to fix).

## Scope: what to review

1. Run `git diff HEAD` and capture the full unified diff.
2. Run `git status --short`. For **untracked** files that are part of the change (e.g. under `internal/`, `tests/`, `*.go` at repo root), include them via `git diff --no-index /dev/null <path>` or by quoting relevant new file contents — so new files are not invisible to the review.

If there is **no** diff and **no** relevant untracked files, write the output file with a single section stating there was nothing to review and stop.

## Three layers (you perform all three)

1. **Blind Hunter** — Using **only** the diff text (no other files): suspicious logic, unclear APIs, obvious bugs, style hazards.
2. **Edge Case Hunter** — With **read access** to the repo: boundaries, errors/nil, concurrency, Fyne UI lifecycle, test gaps tied to the changed code.
3. **Acceptance Auditor** — Compare the diff to **`verdict/judge-output.md`**: whether Blocker/Major/Minor items appear addressed; flag contradictions or partial fixes.

## Triage

Normalize and dedupe findings. Classify each surviving finding as exactly one of:

- **patch** — Unambiguous code fix (still only *document* here; do not apply).
- **defer** — Pre-existing or clearly out of scope for this UX pass.
- **dismiss** — Noise or false positive — **omit** from the written report (count them in the summary).

If `review_mode` were `no-spec`, `decision_needed` would map to patch or defer; here treat ambiguous product intent as **defer** with explanation.

## Output (required)

Create:

`{BUNDLE}/verdict/code-review-post-fix.md`

Use this structure:

```markdown
# Post-fix code review

**Summary:** N patch, M defer, R dismissed (noise).

## Patch
- ...

## Defer
- ...

## Notes
- ...
```

Replace `{BUNDLE}` with the absolute bundle path from the prompt.
