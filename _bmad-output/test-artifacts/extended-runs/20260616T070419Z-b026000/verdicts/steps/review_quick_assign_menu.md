# Step UX verdict — review_quick_assign_menu

## Step

| Field | Value |
|-------|-------|
| **id** | `review_quick_assign_menu` |
| **flow** | `review` |
| **file** | `09_review_quick_assign_menu.png` |
| **intent** | Review: quick collection assign affordance visible |

## Verdict

**FAIL** — No valid real-binary capture is available for this step. The bundle has no `ui-real/steps.json` and no `ui-real/09_review_quick_assign_menu.png` (or `ui-real/review_quick_assign_menu.png` as referenced in the matrix); real capture failed during the run (`logs/ux-capture-real.txt` reports GLX `BadAccess` / `X_GLXMakeCurrent`). The only artifact is `ui-software/09_review_quick_assign_menu.png` backed by `ui-software/steps.json` with `"app_mode": "software_driver"`, which is explicitly out of scope for UX sign-off per the step judge spec (`software_driver_not_valid_for_ux_signoff`).

## Heuristics

- **Heuristic 4 (Consistency and standards):** Not evaluated — software-driver pixels are not valid sign-off input.
- **Heuristic 8 (Aesthetic / image dominance):** Not evaluated — software-driver pixels are not valid sign-off input.
- **Blocker/Major:** Major — missing `real_binary` capture blocks UX evaluation for this step.

STEP_UX_RESULT=fail
