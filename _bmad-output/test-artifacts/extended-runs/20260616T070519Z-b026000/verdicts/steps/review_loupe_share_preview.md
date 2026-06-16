# Step UX verdict — review_loupe_share_preview

## Step

| Field | Value |
|-------|-------|
| **id** | `review_loupe_share_preview` |
| **flow** | `review` |
| **file** | `04_review_loupe_share_preview.png` |
| **intent** | Share preview dialog |

## Verdict

**FAIL** — `ui-real/04_review_loupe_share_preview.png` is byte-identical to `01_upload_empty.png` and shows Upload idle, not the **Share preview** mint dialog. The software-driver reference (`ui-software/04_review_loupe_share_preview.png`) shows a modal with a large preview image, file metadata, and **Cancel** / **Create link** actions overlaid on the loupe. None of that appears in the real-binary PNG.

## Heuristics

- **Heuristic 4 (Consistency and standards):** Major — share mint dialog missing; cannot verify destructive/share affordance separation.
- **Heuristic 8 (Aesthetic / image dominance):** Major — share preview spec requires the shared asset as the largest element in the dialog scope; dialog is absent.
- **Blocker/Major:** Major — step intent not represented in `ui-real/` capture.

STEP_UX_RESULT=fail
