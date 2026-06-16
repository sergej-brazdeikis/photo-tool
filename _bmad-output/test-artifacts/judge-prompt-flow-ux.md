# Flow UX judge — aggregate flow (extended matrix, local only)

Use with an **extended run** directory. **Do not** use in CI.

## Scope

Judge **one flow** (`upload`, `review`, `collections`, `rejected`, `share_desktop`) for coherence, task clarity, and cross-step consistency using **real-binary** PNGs only.

## Inputs

- `context/rubric.md`
- `ui-real/steps.json` with `"app_mode": "real_binary"`
- All `ui-real/*.png` whose `flow` matches the assigned flow
- `verdicts/steps/<step_id>.md` for steps in that flow (if present)

## Reject invalid captures

Fail the flow if any step PNG is only in `ui-software/` or step verdicts cite `software_driver_not_valid_for_ux_signoff`.

## Output

Write `verdicts/flows/<flow>.md`:

1. **Flow summary** — 2–4 sentences on weakest/strongest aspects
2. **Step rollup** — table: step id, PASS/FAIL, one-line note
3. **Cross-step consistency** — nav, density, destructive affordances
4. **Machine line** — last line exactly:

```text
FLOW_UX_RESULT=pass
```

or

```text
FLOW_UX_RESULT=fail
```

Fail if any step in the flow FAILs or any Major/Blocker remains. Evaluation only — no repo edits.
