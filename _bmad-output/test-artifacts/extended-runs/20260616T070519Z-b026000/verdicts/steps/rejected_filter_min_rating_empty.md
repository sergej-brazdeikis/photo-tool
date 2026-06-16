# Step UX verdict — rejected_filter_min_rating_empty

## Step

| Field | Value |
|-------|-------|
| **id** | `rejected_filter_min_rating_empty` |
| **flow** | `rejected` |
| **file** | `17_rejected_filter_min_rating_empty.png` |
| **intent** | Rejected: narrow filter empty |

## Verdict

**FAIL** — `ui-real/17_rejected_filter_min_rating_empty.png` is byte-identical to `01_upload_empty.png` and shows the **Upload** idle screen, not Rejected with a narrowed filter. The capture is missing the Rejected filter strip with **Minimum rating: 5**, **Rejected: 0** count, and the empty-state copy (“No rejected photos match these filters…”) that should appear when no rejected assets match the filter. Primary nav highlights **Upload**; the real-binary journey did not navigate to Rejected or apply the min-rating filter for this step.

## Heuristics

- **Heuristic 1 (Visibility of system status):** Major — no filter-empty feedback or rejected count is visible because the wrong view was captured.
- **Heuristic 4 (Consistency and standards):** Major — step id and filename imply Rejected filter-empty state but pixels show Upload.
- **Heuristic 8 (Aesthetic / image dominance):** Not assessable — no rejected view chrome or empty-state camera placeholder is present.
- **Blocker/Major:** Major — step intent not represented in `ui-real/` capture.

STEP_UX_RESULT=fail
