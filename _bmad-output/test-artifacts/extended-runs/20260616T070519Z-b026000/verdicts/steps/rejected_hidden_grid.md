# Step UX verdict — rejected_hidden_grid

## Step

| Field | Value |
|-------|-------|
| **id** | `rejected_hidden_grid` |
| **flow** | `rejected` |
| **file** | `16_rejected_hidden_grid.png` |
| **intent** | Rejected: grid |

## Verdict

**FAIL** — `ui-real/16_rejected_hidden_grid.png` is byte-identical to `01_upload_empty.png` and shows the **Upload** idle screen (empty staging list, drop zone, disabled **Import selected files**), not Rejected. The capture lacks the filter strip (Collection / Minimum rating / Tags), **Rejected: N** count, **Back to Review**, bulk-delete hint, and rejected thumbnail grid with **Restore** that the step intent requires. Primary nav highlights **Upload** instead of **Rejected**; the real-binary journey never reached the Rejected view for vision sign-off.

## Heuristics

- **Heuristic 4 (Consistency and standards):** Major — step id and filename imply Rejected but pixels show Upload; nav active state contradicts flow.
- **Heuristic 8 (Aesthetic / image dominance):** Not assessable on this frame — no rejected grid or photographic pixels are present.
- **Blocker/Major:** Major — step intent not represented in `ui-real/` capture.

STEP_UX_RESULT=fail
