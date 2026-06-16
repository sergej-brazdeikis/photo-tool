# Flow UX verdict — upload

**Bundle:** `20260616T070419Z-b026000`  
**Flow:** `upload`  
**Evaluator:** LLM flow judge (extended matrix, local only)

## Flow summary

The upload flow cannot receive UX sign-off in this bundle. Real-binary capture failed entirely (`logs/ux-capture-real.txt`: GLX `BadAccess` / `X_GLXMakeCurrent` after Fyne call-thread warnings), so there is no `ui-real/` directory or `ui-real/steps.json`. Every upload step PNG exists only under `ui-software/` with `"app_mode": "software_driver"`, which the judge spec rejects (`software_driver_not_valid_for_ux_signoff`). All three matrix step verdicts are **FAIL** with **Major** severity for missing valid captures; coherence, task clarity, and image-dominance cannot be assessed on real pixels.

## Step rollup

| Step id | Result | Note |
|---------|--------|------|
| `upload_empty` | **FAIL** | No `ui-real/01_upload_empty.png`; only software-driver fallback |
| `upload_paths_staged` | **FAIL** | No `ui-real/20_upload_paths_staged.png`; only software-driver fallback |
| `upload_fr06_collection_assign` | **FAIL** | No `ui-real/21_upload_fr06_collection_assign.png`; only software-driver fallback |

Additional upload journey frames in `ui-software/steps.json` (`upload_empty_nfr01_min_window`, `upload_after_confirm_idle`) likewise lack real-binary counterparts and have no step verdicts in this bundle; they reinforce the capture failure but are not separately rolled up here.

## Cross-step consistency

**Navigation:** Not evaluable — no valid real-binary upload screenshots.

**Density / NFR-01:** Not evaluable — `19_upload_empty_nfr01_min_window.png` exists only under `ui-software/`; min-window clip checks require real pixels per rubric.

**Destructive affordances:** Not evaluable — FR-06 confirm/cancel and Import/Clear affordances were not captured for vision review.

**Capture integrity:** Invalid for UX sign-off per judge spec: all upload step PNGs are software-driver only; step verdicts cite `software_driver_not_valid_for_ux_signoff`. Re-run real-binary capture (`TestUXJourneyCapture` without GLX failure) before re-judging this flow.

## Open issues

| Severity | Issue |
|----------|-------|
| **Major** | Missing `real_binary` captures for all upload matrix steps (blocks flow sign-off) |
| **Major** | Real capture environment failure (GLX `BadAccess`) documented in `logs/ux-capture-real.txt` |

FLOW_UX_RESULT=fail
