# Flow UX verdict — rejected

**Bundle:** `20260616T070419Z-b026000`  
**Flow:** `rejected`  
**Evaluator:** LLM flow judge (extended matrix, local only)

## Flow summary

The rejected flow cannot receive UX sign-off in this bundle. Real-binary capture failed entirely (`logs/ux-capture-real.txt`: Fyne call-thread warnings followed by GLX `BadAccess` / `X_GLXMakeCurrent`), so there is no `ui-real/` directory or `ui-real/steps.json`. All three matrix rejected steps exist only under `ui-software/` with `"app_mode": "software_driver"`, which the judge spec rejects (`software_driver_not_valid_for_ux_signoff`). Step verdicts are uniformly **FAIL** with **Major** severity for missing valid captures; hidden-grid visibility, filter-empty empty state, and NFR-01 min-window layout cannot be assessed on real pixels.

## Step rollup

| Step id | Result | Note |
|---------|--------|------|
| `rejected_hidden_grid` | **FAIL** | No `ui-real/16_rejected_hidden_grid.png`; only software-driver fallback |
| `rejected_filter_min_rating_empty` | **FAIL** | No `ui-real/17_rejected_filter_min_rating_empty.png`; only software-driver fallback |
| `rejected_nfr01_min_window` | **FAIL** | No `ui-real/18_rejected_nfr01_min_window.png`; only software-driver fallback; NFR-01 contractual step |

## Cross-step consistency

**Navigation:** Not evaluable — no valid real-binary rejected screenshots; **Back to Review** and tab coherence cannot be vision-checked.

**Density / NFR-01:** Not evaluable — `rejected_nfr01_min_window` exists only under `ui-software/`; min-window clip checks require real pixels per rubric.

**Destructive affordances:** Not evaluable — reject/restore affordances and filter-strip behavior on the Rejected tab were not captured in `ui-real/`; undo-reject prominence and empty-state guidance cannot be assessed.

**Capture integrity:** Invalid for UX sign-off per judge spec: all rejected matrix step PNGs are software-driver only; step verdicts cite `software_driver_not_valid_for_ux_signoff`. Re-run real-binary capture (`TestUXJourneyCapture` without GLX failure) before re-judging this flow.

## Open issues

| Severity | Issue |
|----------|-------|
| **Major** | Missing `real_binary` captures for all rejected matrix steps (blocks flow sign-off) |
| **Major** | Real capture environment failure (GLX `BadAccess`) documented in `logs/ux-capture-real.txt` |

FLOW_UX_RESULT=fail
