# Step UX verdict — rejected_hidden_grid

## Step

| Field | Value |
|-------|-------|
| **id** | `rejected_hidden_grid` |
| **flow** | `rejected` |
| **file** | `16_rejected_hidden_grid.png` |
| **intent** | Rejected: grid |

## Verdict

**FAIL** — No valid real-binary capture is available for this step. The bundle has no `ui-real/steps.json` and no `ui-real/16_rejected_hidden_grid.png` (or `ui-real/rejected_hidden_grid.png` as referenced in the matrix); real capture failed during the run (`logs/ux-capture-real.txt` reports GLX `BadAccess` / `X_GLXMakeCurrent`). The only PNG is under `ui-software/16_rejected_hidden_grid.png` with `"app_mode": "software_driver"` in `ui-software/steps.json`, which must be rejected for UX sign-off (`software_driver_not_valid_for_ux_signoff`).

## Heuristics

- **Heuristic 4 (Consistency and standards):** Not evaluated — software-driver pixels are not valid sign-off input.
- **Heuristic 8 (Aesthetic / image dominance):** Not evaluated — software-driver pixels are not valid sign-off input.
- **Blocker/Major:** Major — missing `real_binary` capture blocks UX evaluation for this step.

STEP_UX_RESULT=fail
