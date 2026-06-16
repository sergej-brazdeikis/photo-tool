# Step UX verdict — review_quick_assign_menu

## Step

| Field | Value |
|-------|-------|
| **id** | `review_quick_assign_menu` |
| **flow** | `review` |
| **file** | `09_review_quick_assign_menu.png` |
| **intent** | Review: quick collection assign affordance visible |

## Verdict

**FAIL** — `ui-real/09_review_quick_assign_menu.png` is byte-identical to `01_upload_empty.png` and shows Upload idle. FR-08 quick collection assign requires the Review bulk row with **Assign selection** album dropdown and assign action visible (`ui-software/09_review_quick_assign_menu.png`). The real-binary capture never reached Review or exposed the assign affordance.

## Heuristics

- **Heuristic 6 (Recognition rather than recall):** Major — collection assign control not visible for evaluation.
- **Heuristic 8 (Aesthetic / image dominance):** Not assessable — review grid and bulk chrome absent.
- **Blocker/Major:** Major — step intent not represented in `ui-real/` capture.

STEP_UX_RESULT=fail
