# Step UX verdict — review_loupe

## Step

| Field | Value |
|-------|-------|
| **id** | `review_loupe` |
| **flow** | `review` |
| **file** | `03_review_loupe.png` |
| **intent** | Review loupe: image band + controls |

## Verdict

**FAIL** — `ui-real/03_review_loupe.png` is byte-identical to `01_upload_empty.png` and still shows the Upload idle panel. There is no loupe image band, star-rating row, **Prev/Next**, **Reject photo**, **Share…**, or album/tag chrome that appear in the software-driver reference (`ui-software/03_review_loupe.png`). The capture did not open Review loupe.

## Heuristics

- **Heuristic 4 (Consistency and standards):** Major — wrong flow entirely; loupe controls absent.
- **Heuristic 8 (Aesthetic / image dominance):** Major — UX spec loupe criterion requires the active photo to dominate the central band; this frame has no photo at all.
- **Blocker/Major:** Major — step intent not represented in `ui-real/` capture.

STEP_UX_RESULT=fail
