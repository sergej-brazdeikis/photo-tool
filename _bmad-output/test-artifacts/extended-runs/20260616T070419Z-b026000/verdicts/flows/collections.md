# Flow UX verdict — collections

**Bundle:** `20260616T070419Z-b026000`  
**Flow:** `collections`  
**Evaluator:** LLM flow judge (extended matrix, local only)

## Flow summary

The collections flow cannot receive UX sign-off in this bundle. Real-binary capture failed entirely (`logs/ux-capture-real.txt`: Fyne call-thread warnings followed by GLX `BadAccess` / `X_GLXMakeCurrent`), so there is no `ui-real/` directory or `ui-real/steps.json`. The matrix step `collections_album_list` exists only under `ui-software/` with `"app_mode": "software_driver"`, which the judge spec rejects (`software_driver_not_valid_for_ux_signoff`). The step verdict is **FAIL** with **Major** severity for missing valid captures; album-list clarity, list-to-detail navigation, grouping controls, and NFR-01 min-window layout cannot be assessed on real pixels.

## Step rollup

| Step id | Result | Note |
|---------|--------|------|
| `collections_album_list` | **FAIL** | No `ui-real/10_collections_album_list.png`; only software-driver fallback |

Additional collections journey frames in `ui-software/steps.json` (`collections_new_album_form`, `collections_album_detail_stars`, `collections_album_group_by_day`, `collections_album_group_by_camera`, `collections_back_to_album_list`) likewise lack real-binary counterparts and have no step verdicts in this bundle; they reinforce the capture failure but are not separately rolled up here.

## Cross-step consistency

**Navigation:** Not evaluable — no valid real-binary collections screenshots; album list → new album → detail → group-by variants → back-to-list progression cannot be vision-checked.

**Density / NFR-01:** Not evaluable — rubric-contractual `collections_album_list_nfr01_min_window` and `collections_album_detail_nfr01_min_window` frames were not captured in `ui-real/`; min-window clip checks require real pixels per rubric.

**Destructive affordances:** Not evaluable — album CRUD and empty-album CTAs (`Back to albums`, `Go to Review`) were not captured for vision review on real pixels.

**Capture integrity:** Invalid for UX sign-off per judge spec: all collections step PNGs are software-driver only; the matrix step verdict cites `software_driver_not_valid_for_ux_signoff`. Re-run real-binary capture (`TestUXJourneyCapture` without GLX failure) before re-judging this flow.

## Open issues

| Severity | Issue |
|----------|-------|
| **Major** | Missing `real_binary` capture for `collections_album_list` (blocks flow sign-off) |
| **Major** | Real capture environment failure (GLX `BadAccess`) documented in `logs/ux-capture-real.txt` |

FLOW_UX_RESULT=fail
