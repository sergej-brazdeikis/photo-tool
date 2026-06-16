# Step UX judge — single step (extended matrix, local only)

Use with an **extended run** directory (`_bmad-output/test-artifacts/extended-runs/<stamp>/`). **Do not** use in CI.

## Scope

Judge **one** journey step PNG against rubric + UX spec **image dominance** for that step's `intent` in `ui-real/steps.json`.

## Inputs (required)

- `context/rubric.md` — usability heuristics
- `ui-real/steps.json` — must declare `"app_mode": "real_binary"`
- `ui-real/<steps[].file>` — PNG for the assigned step only
- Optional: `context/requirements-trace.md`

## Reject software-driver captures

If the PNG path is under `ui-software/` **or** `steps.json` has `"app_mode": "software_driver"` (or missing), write **FAIL** with rationale `software_driver_not_valid_for_ux_signoff` and end with `STEP_UX_RESULT=fail`.

## Output

Write `verdicts/steps/<step_id>.md` under the run directory:

1. **Step** — `id`, `flow`, `file`, `intent` from `steps.json`
2. **Verdict** — PASS or FAIL with one paragraph rationale citing the PNG
3. **Heuristics** — note any Blocker/Major against rubric 4/8 and UX spec image dominance
4. **Machine line** — last line exactly:

```text
STEP_UX_RESULT=pass
```

or

```text
STEP_UX_RESULT=fail
```

Use **fail** for any Major/Blocker or software-driver rejection. Do **not** edit repository source from this role.
