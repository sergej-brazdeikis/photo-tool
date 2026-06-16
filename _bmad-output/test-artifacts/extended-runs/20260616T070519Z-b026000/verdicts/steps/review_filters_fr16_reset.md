# Step UX verdict — review_filters_fr16_reset

## Step

| Field | Value |
|-------|-------|
| **id** | `review_filters_fr16_reset` |
| **flow** | `review` |
| **file** | `08_review_filters_fr16_reset.png` |
| **intent** | Review: filters reset defaults |

## Verdict

**FAIL** — `ui-real/08_review_filters_fr16_reset.png` is byte-identical to `01_upload_empty.png` and shows Upload idle. FR-16 reset expects Review filter controls returned to defaults (Collection **Unassigned**, **Any rating**, **Any tag**) with a populated grid visible afterward (`ui-software/08_review_filters_fr16_reset.png`). The real-binary PNG does not show Review or any filter strip.

## Heuristics

- **Heuristic 3 (User control and freedom):** Major — reset-filters recovery path not demonstrable in capture.
- **Heuristic 8 (Aesthetic / image dominance):** Not assessable — review grid not shown.
- **Blocker/Major:** Major — step intent not represented in `ui-real/` capture.

STEP_UX_RESULT=fail
