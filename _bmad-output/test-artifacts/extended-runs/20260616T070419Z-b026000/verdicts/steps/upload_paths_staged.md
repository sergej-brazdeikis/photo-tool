# Step UX verdict — upload_paths_staged

## Step

| Field | Value |
|-------|-------|
| **id** | `upload_paths_staged` |
| **flow** | `upload` |
| **file** | `20_upload_paths_staged.png` |
| **intent** | Upload: staged paths |

## Verdict

**FAIL** — No valid real-binary capture is available for this step. The bundle has no `ui-real/steps.json` and no `ui-real/20_upload_paths_staged.png` (or `ui-real/upload_paths_staged.png` as referenced in the matrix); real capture failed during the run (`logs/ux-capture-real.txt` reports GLX `BadAccess`). The only PNG is under `ui-software/20_upload_paths_staged.png` with `"app_mode": "software_driver"` in `ui-software/steps.json`, which must be rejected for UX sign-off (`software_driver_not_valid_for_ux_signoff`).

## Heuristics

- **Heuristic 4 (Consistency and standards):** Not evaluated — software-driver pixels are not valid sign-off input.
- **Heuristic 8 (Aesthetic / image dominance):** Not evaluated — software-driver pixels are not valid sign-off input.
- **Blocker/Major:** Major — missing `real_binary` capture blocks UX evaluation for this step.

STEP_UX_RESULT=fail
