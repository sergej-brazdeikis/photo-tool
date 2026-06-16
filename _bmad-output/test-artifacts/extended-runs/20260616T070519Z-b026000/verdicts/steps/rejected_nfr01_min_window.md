# Step UX verdict — rejected_nfr01_min_window

## Step

| Field | Value |
|-------|-------|
| **id** | `rejected_nfr01_min_window` |
| **flow** | `rejected` |
| **file** | `18_rejected_nfr01_min_window.png` |
| **intent** | Rejected NFR min window |

## Verdict

**FAIL** — `ui-real/18_rejected_nfr01_min_window.png` is byte-identical to `19_upload_empty_nfr01_min_window.png` and shows the **Upload** empty state at the NFR-01 minimum window size (1024×768), not Rejected. The capture lacks Rejected primary-nav selection, filter strip, **Rejected: N** header, **Back to Review**, and rejected thumbnail grid that must be evaluated for clipped primary chrome at minimum size per NFR-01. The real-binary journey did not resize and navigate to Rejected for this contractual min-window frame.

## Heuristics

- **Heuristic 4 (Consistency and standards):** Major — NFR-01 min-window step id targets Rejected but pixels show Upload at the same dimensions.
- **Heuristic 8 (Aesthetic / image dominance):** Not assessable — rejected grid and aggregate photo region are absent; cannot verify image dominance or chrome clip at NFR-01 minimum for the Rejected flow.
- **Blocker/Major:** Major — NFR-01 Rejected min-window sign-off not represented in `ui-real/` capture.

STEP_UX_RESULT=fail
