# Flow UX verdict — share_desktop

**Bundle:** `20260616T070419Z-b026000`  
**Flow:** `share_desktop`  
**Evaluator:** LLM flow judge (extended matrix, local only)

## Flow summary

The share_desktop flow cannot receive UX sign-off in this bundle. Real-binary capture failed entirely (`logs/ux-capture-real.txt`: Fyne call-thread warnings followed by GLX `BadAccess` / `X_GLXMakeCurrent`), so there is no `ui-real/` directory or `ui-real/steps.json`. Both matrix steps — single-asset loupe share preview (FR-32) and package share preview (FR-33) — exist only under `ui-software/` with `"app_mode": "software_driver"`, which the judge spec rejects (`software_driver_not_valid_for_ux_signoff`). The package step verdict is **FAIL** with **Major** severity; the loupe share step has no step verdict but shares the same invalid capture state. Image-first preview layout, copyable URL affordances, and single-vs-package dialog consistency cannot be assessed on real pixels.

## Step rollup

| Step id | Result | Note |
|---------|--------|------|
| `review_loupe_share_preview` | **FAIL** | No `ui-real/04_review_loupe_share_preview.png`; only software-driver fallback (`ui-software/04_review_loupe_share_preview.png`) |
| `review_package_share_preview` | **FAIL** | No `ui-real/review_package_share_preview.png`; step verdict cites `software_driver_not_valid_for_ux_signoff` |

Additional share-related journey frames in `ui-software/steps.json` (`review_loupe_share_preview` at step 04, `review_package_share_preview` at step 24) likewise lack real-binary counterparts. The rubric also references `review_loupe_share_preview_nfr01_min_window` for NFR-01 min-window clip checks; that frame was not captured in `ui-real/` either.

## Cross-step consistency

**Navigation:** Not evaluable — no valid real-binary screenshots; loupe → Share → mint-preview progression cannot be vision-checked.

**Density / NFR-01:** Not evaluable — `review_loupe_share_preview_nfr01_min_window` (contractual per rubric) was not captured under `ui-real/`; min-window primary-chrome clip checks require real pixels.

**Destructive affordances:** Not evaluable — mint dialogs typically pair primary confirm with Cancel; prominence and copy patterns for single-asset vs package mint cannot be compared without valid captures.

**Single vs package share:** Not evaluable — FR-32 loupe share preview and FR-33 package share preview should share consistent dialog chrome, preview band sizing, and metadata treatment; both steps are software-driver only, blocking cross-step consistency review.

**Capture integrity:** Invalid for UX sign-off per judge spec: all share_desktop matrix step PNGs are software-driver only. Re-run real-binary capture (`TestUXJourneyCapture` without GLX failure) before re-judging this flow.

## Open issues

| Severity | Issue |
|----------|-------|
| **Major** | Missing `real_binary` captures for both share_desktop matrix steps (blocks flow sign-off) |
| **Major** | Real capture environment failure (GLX `BadAccess`) documented in `logs/ux-capture-real.txt` |
| **Major** | `review_package_share_preview` step verdict FAIL — `software_driver_not_valid_for_ux_signoff` |

FLOW_UX_RESULT=fail
