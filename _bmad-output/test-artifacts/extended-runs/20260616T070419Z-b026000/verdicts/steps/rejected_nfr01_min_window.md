# Step UX verdict — rejected_nfr01_min_window

## Step

| Field | Value |
|-------|-------|
| **id** | `rejected_nfr01_min_window` |
| **flow** | `rejected` |
| **file** | `18_rejected_nfr01_min_window.png` |
| **intent** | Rejected NFR min window |

## Verdict

**FAIL** — No valid real-binary capture is available for this step. The bundle has no `ui-real/steps.json` and no `ui-real/18_rejected_nfr01_min_window.png`; real capture failed during the run (`logs/ux-capture-real.txt` reports GLX `BadAccess` / `X_GLXMakeCurrent`). The only PNG is under `ui-software/18_rejected_nfr01_min_window.png` with `"app_mode": "software_driver"` in `ui-software/steps.json`, which must be rejected for UX sign-off (`software_driver_not_valid_for_ux_signoff`). NFR-01 min-window layout cannot be signed off without a `real_binary` frame at 1024×768.

## Heuristics

- **Heuristic 4 (Consistency and standards):** Not evaluated — software-driver pixels are not valid sign-off input.
- **Heuristic 8 (Aesthetic / image dominance):** Not evaluated — software-driver pixels are not valid sign-off input.
- **Blocker/Major:** Major — missing `real_binary` capture blocks UX evaluation for this step; NFR-01 min-window steps are contractual per rubric.

STEP_UX_RESULT=fail
