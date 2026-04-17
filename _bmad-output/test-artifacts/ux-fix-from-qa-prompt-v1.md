# UX loop — fix from QA (implementer, local / Cursor `agent` only)

**Prerequisites:** A judge bundle `BUNDLE` with `verdict/qa-loop-output.md` whose **last line** is `QA_LOOP_RESULT=fail` (from [ux-qa-post-fix-loop.md](ux-qa-post-fix-loop.md)).

## Role

You are the **implementer** for **QA findings**: failing automated tests, brittle coverage called out in the QA report, or small product/code changes needed so **log evidence** and **traceability** gaps are addressed. This is **orthogonal** to the UX vision judge; you may touch **tests** and **production** code as needed, minimally.

## Inputs

- `BUNDLE/verdict/qa-loop-output.md` — summary, log status, traceability/gaps, **Feature behavior matrix** (required by [ux-qa-post-fix-loop.md](ux-qa-post-fix-loop.md)), recommendations
- `BUNDLE/context/requirements-trace.md` — FR/story matrix; use it to prioritize **GAP** / **PARTIAL** rows called out by QA
- `BUNDLE/verdict/judge-output.md` — optional context if QA tied issues to UX verdict items
- `BUNDLE/verdict/code-review-post-fix.md` — optional
- `BUNDLE/logs/go-test.txt`, `BUNDLE/logs/go-test-e2e.txt`, `BUNDLE/logs/go-test-ci.txt` — re-read failures cited in the QA output
- Primary test locations: [`tests/e2e/`](../../tests/e2e/), [`internal/app/`](../../internal/app/), [`internal/share/`](../../internal/share/)

## Requirements

1. Address **blocking** items first: make **`go test ./...`** and **`go test ./tests/e2e/...`** pass if the QA report or logs show failures. Then close **Feature behavior matrix** and **`context/requirements-trace.md`** **GAP**s in priority order (especially **Review — rating** exclusivity, persistence, and keyboard path, and any trace **Loop risk** **GAP** / **PARTIAL** rows QA tied to release risk)—prefer the **smallest** tests, capture-step updates, or UX fixes that prove the behavior.
2. Prefer **targeted** edits in the packages above; do not refactor unrelated modules.
3. From repo root:

   ```bash
   go test ./...
   go test ./tests/e2e/... -count=1
   go build .
   ```

4. **Do not** re-run the vision judge or the QA agent in this task; the outer [`scripts/ux-judge-loop.sh`](../../scripts/ux-judge-loop.sh) will **re-assemble** on the next iteration.

## Output

- Code and/or tests updated; CI-relevant tags unchanged unless the QA report required it (`go test -tags ci` is exercised on the next bundle assemble).
