# Step UX verdict — upload_paths_staged

## Step

| Field | Value |
|-------|-------|
| **id** | `upload_paths_staged` |
| **flow** | `upload` |
| **file** | `20_upload_paths_staged.png` |
| **intent** | Upload: staged paths |

## Verdict

**FAIL** — `ui-real/20_upload_paths_staged.png` does not depict staged paths. The capture shows the pre-import idle Upload screen: “Files staged for import” is empty, **Import selected files** is disabled, and there are no listed paths or preview thumbnails. That contradicts the step intent and the bundle’s valid software-driver reference (`ui-software/20_upload_paths_staged.png`), which shows two staged JPG paths with a “Selected photos” preview strip. The real-binary journey did not freeze the post-seed state for vision sign-off.

## Heuristics

- **Heuristic 4 (Consistency and standards):** Major — control states (disabled Import, empty list) imply the wrong journey phase for a “staged paths” checkpoint.
- **Heuristic 8 (Aesthetic / image dominance):** Major — normative image-dominance for Upload staging requires an aggregate photographic preview region; this PNG has none, so the vision gate for Story 1.5 photographic receipt remains unmet.
- **Blocker/Major:** Major — wrong capture state; not evaluated from `ui-software/` (rejected per spec).

STEP_UX_RESULT=fail
