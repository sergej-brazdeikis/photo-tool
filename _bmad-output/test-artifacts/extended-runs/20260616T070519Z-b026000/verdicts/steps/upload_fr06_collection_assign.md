# Step UX verdict — upload_fr06_collection_assign

## Step

| Field | Value |
|-------|-------|
| **id** | `upload_fr06_collection_assign` |
| **flow** | `upload` |
| **file** | `21_upload_fr06_collection_assign.png` |
| **intent** | FR-06 collection assign |

## Verdict

**FAIL** — `ui-real/21_upload_fr06_collection_assign.png` is byte-identical to `20_upload_paths_staged.png` and still shows the empty pre-import Upload view. The PNG lacks the FR-06 **Batch preview** band (file count, thumbnails, import summary), the **Collection (after import)** choice (Skip / Assign to collection), and **Confirm** / **Cancel** actions that appear in the software-driver reference for this step. The capture never reached the post-import collection-assign gate.

## Heuristics

- **Heuristic 4 (Consistency and standards):** Major — expected FR-06 confirm/cancel affordances and collection radios are absent; user cannot recognize the collection-decision task from pixels alone.
- **Heuristic 8 (Aesthetic / image dominance):** Major — batch preview thumbnails should be the dominant content block; this frame shows only empty chrome, failing the cross-cutting Upload staging/receipt bar.
- **Blocker/Major:** Major — FR-06 step intent not represented in `ui-real/` capture.

STEP_UX_RESULT=fail
