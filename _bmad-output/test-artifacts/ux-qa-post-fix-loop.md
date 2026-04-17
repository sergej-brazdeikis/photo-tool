# QA loop (UX judge loop — non-interactive, local `agent` only)

You run from `scripts/ux-judge-loop.sh` **after** the post-fix code review step when **`UX_LOOP_QA_AGENT=1`**. Intent aligns with BMAD test review / QA test generation skills ([`bmad-testarch-test-review`](../../.claude/skills/bmad-testarch-test-review/SKILL.md), [`bmad-qa-generate-e2e-tests`](../../.claude/skills/bmad-qa-generate-e2e-tests/SKILL.md)) in **one pass**: **end-user behavior** vs tests and bundle evidence, **traceability**, regression risk, and **test gap** notes—not a second code style review.

Treat [`_bmad-output/planning-artifacts/ux-design-specification.md`](../../_bmad-output/planning-artifacts/ux-design-specification.md) as the **behavioral source of truth** when judging whether a feature is adequately guarded (rating semantics, reject vs delete, share confirm, ingest receipts, etc.).

**Loop constraints (must follow):**

- **Do not** modify application source code or run `go test` / `go build` (read **bundle logs** only).
- **Do not** re-run the UX vision judge.
- **Do** write only under the bundle: `verdict/qa-loop-output.md`.

## Inputs (paths come from the shell prompt)

- **Repository root** — for optional `git diff` / `git status` if you need file paths (same as post-fix review).
- **Bundle directory** — read:
  - `verdict/judge-output.md` — what the UX judge required
  - `verdict/code-review-post-fix.md` — if present, post-fix code review
  - `context/requirements-trace.md` — **FR/story ↔ tests/capture matrix** (distilled from epics; cross-check every matrix row against it)
  - `logs/go-test.txt`, `logs/go-test-ci.txt`, **`logs/go-test-e2e.txt`**
  - `ui/steps.json` — which flows were captured (coverage hint vs. code changes)

## Scope

1. **Deterministic evidence** — From logs: did **full** `go test ./...`, **`-tags ci`**, and **CLI e2e** (`./tests/e2e/...`) succeed? Quote at most **10 lines** total across logs only if illustrating a problem.
2. **Feature behavior matrix (required)** — Build an explicit **coverage table** in `verdict/qa-loop-output.md` (see structure below). Include a **`Story / FR`** column: for each row, cite the **matching story id and/or FR** from `context/requirements-trace.md` (or state **none** if the area is NFR-only). **Cross-check:** if your **Verdict** for a row disagrees with that file’s **Loop risk** (`OK` / `PARTIAL` / `GAP`) for the same story/FR, explain the mismatch in **Traceability and gaps** (cite the trace **Story** cell). For **each** row, state whether **automated tests** and/or **bundle capture steps** (`ui/steps.json` + `intent`) give **credible** evidence that the behavior matches the UX spec, or mark **`GAP`**. You are doing a **product QA retest pass in the abstract**: you cannot click the live app here, but you **must** infer from repo tests + journey capture + logs whether a human could regress core flows without detection. Rows **must** include at minimum:
   - **Review — rating:** One **numeric rating (1–5) per asset** at rest (DB + UI); changing rating updates, does not imply multiple concurrent ratings; keyboard **1–5** in loupe where spec applies; **Reject** affordances not conflated with rating (**UX spec** triage / safety).
   - **Review — grid & loupe:** Open loupe, prev/next, close; filter strip (collection → minimum rating → tags) and reset-to-defaults path when the bundle shows those steps.
   - **Review — bulk / albums:** Selection + assign-to-album (or document **GAP** if no test/capture path).
   - **Upload / ingest:** Staged paths, import, FR-06 batch / collection confirm (per bundle steps + `tests/e2e` / `internal/app` tests).
   - **Collections:** List, detail, grouping/back navigation as far as steps + tests show.
   - **Rejected:** Hidden grid, filters, restore (or **GAP**).
   - **Share:** Mint / loopback / HTTP contract — cite [`internal/share/http_test.go`](../../internal/share/http_test.go) or mark **GAP** if judge raised share UX but HTTP tests do not cover it.
   - **CLI:** `phototool` scan/import semantics — `tests/e2e/cli_test.go` (or **GAP**).

   If the **judge** or **post-fix review** called out a **Major/Blocker** on a flow, that flow’s matrix rows **cannot** be **`OK`** on testimony alone; tie them to **tests** or **capture** or mark **`GAP`**.

3. **Traceability** — Map **Major/Blocker** items in `verdict/judge-output.md` to **test layers** that should catch regressions (`internal/app/*_test.go`, `tests/e2e`, `internal/share/http_test.go`). Flag **gaps** (no plausible automated guard).
4. **Diff-aware gaps** — If you use `git diff HEAD` / `git status`, tie only **testable** risks to the diff; otherwise skip diff and stay log+verdict-based.

## Output (required)

Create:

`{BUNDLE}/verdict/qa-loop-output.md`

Structure:

```markdown
# QA loop output

**Summary:** One paragraph: overall test posture after this iteration, top gap if any.

## Log status
- go_test: pass|fail (evidence: logs/go-test.txt)
- go_test_ci: pass|fail
- go_test_e2e: pass|fail (logs/go-test-e2e.txt)

## Traceability and gaps
- ...

## Feature behavior matrix
| Area | Story / FR (from `context/requirements-trace.md`) | Expected behavior (short) | Evidence (test paths, `steps.json` ids, or log) | Verdict |
|------|--------------------------------------------------|---------------------------|---------------------------------------------------|---------|
| …    | …                                                | …                         | …                                                 | OK or GAP |

Add one **synthesis** bullet under the table: which **GAP** rows are most dangerous for the next release.

## Recommendations (non-binding)
- ...
```

Replace `{BUNDLE}` with the absolute bundle path from the prompt.

## Machine-readable outcome (required)

**Last line** of `verdict/qa-loop-output.md` must be exactly one of:

```text
QA_LOOP_RESULT=pass
```

or

```text
QA_LOOP_RESULT=fail
```

Use **`fail`** if:

- any referenced **go test** log shows failure, or
- any **Major/Blocker** judge item lacks **traceable** automated or capture-backed evidence, or
- the **Feature behavior matrix** marks **`GAP`** for **Review — rating** (single rating per asset / keyboard / exclusivity vs spec), **Upload / ingest**, or **Share** contract smoke, **unless** you document in the body why the gap is **acceptably deferred** (one sentence per deferred row), or
- logs are green but **`context/requirements-trace.md`** marks a story as **PARTIAL** with **Loop risk** commentary that treats it as **release-critical** (or the trace row’s FR is in the same class as the bullet above) and the matrix cannot show **OK** with credible test + capture evidence—in that case **`QA_LOOP_RESULT=fail`** even without a red `go test` line (same bar as “Major/Blocker without traceable evidence,” keyed off the trace file).

Otherwise **`pass`**.

**Note:** When **`UX_LOOP_QA_FIX`** is not `0` (default), [`scripts/ux-judge-loop.sh`](../../scripts/ux-judge-loop.sh) runs a **QA implementer** pass after this step if the last line is **`QA_LOOP_RESULT=fail`** (see [ux-fix-from-qa-prompt-v1.md](ux-fix-from-qa-prompt-v1.md)). Loop exit is still driven by **`UX_JUDGE_RESULT`** and **`UX_LOOP_MAX`** until the vision judge passes.
