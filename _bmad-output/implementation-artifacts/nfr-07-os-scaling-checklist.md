# NFR-07 OS display scaling — checklist (125% / 150%)

**Purpose:** Re-validate **NFR-01** layout under **OS-level** display scaling on **tier-1** desktops (macOS, Windows) per **NFR-07** at Epic 2 gate.  
**Paired evidence:** [`nfr-01-layout-matrix-evidence.md`](./nfr-01-layout-matrix-evidence.md) — use **same matrix cell IDs**.

## AC3 evidence (Story 2.11)

**Story AC3 (tier-1 CI):** **Windows** supplies **OS-reported** **125% / 150%** tiers (**system DPI** **120 / 144**). **GitHub-hosted macOS** uses the **documented surrogate** below; on **darwin+cgo**, **`NFR07AC3DisplayScalingPercent`** logs a **CoreGraphics** pixel/point line when the display probe succeeds, or **`CoreGraphics unavailable (...)`** when the session has no active display (headless). **`FYNE_SCALE`** stays aligned to the workflow tier in both cases. Together with **Windows** OS DPI jobs they satisfy **AC3** per **`2-11-layout-display-scaling-gate.md`** AC3 **Tier-1 CI** note.

**Windows (GitHub Actions):** `.github/workflows/go.yml` runs **two** `windows-latest` jobs that set **`HKCU:\Control Panel\Desktop` → `LogPixels`** to **120** (125% tier) and **144** (150% tier) before `go test ./...`, then invokes `rundll32.exe user32.dll,UpdatePerUserSystemParameters`. On those jobs, **`TestNFR01LayoutGate_NFR07_AC3`** must **not skip** (FYNE_SCALE unset): it probes **`GetDpiForSystem`** and re-runs the Epic 2 default subset + AC2 sweep.

**macOS (GitHub Actions):** The workflow runs **two** `macos-latest` jobs with **`PHOTO_TOOL_NFR07_MACOS_CI_TIER`** `125` / `150` and matching **`FYNE_SCALE`** `1.25` / `1.5`. Hosted runners cannot apply **System Settings → Displays** scaling; this pair is the **documented CI surrogate** so the same **`TestNFR01LayoutGate_NFR07_AC3`** body (subset + AC2, FYNE_SCALE aligned to the tier) runs **non-skipped** in-repo beside Windows. **`NFR07AC3DisplayScalingPercent`** still evaluates **CoreGraphics** (cgo) and the test log records **both** the surrogate tier and the **observed** display UI scale for audit.

**macOS (hardware / local):** **`NFR07AC3DisplayScalingPercent`** uses **CoreGraphics** pixel/point ratios (Retina-normalized) when the CI surrogate env is **not** active. Run with **System Settings → Displays** adjusted so the probe reports **~125%** or **~150%** effective UI — **displays must be awake**; asleep or headless sessions can read **1.0** and skip incorrectly.

**macOS row sign-off (hardware):** Mark **Pass** only after a **non-skipped** `go test -run TestNFR01LayoutGate_NFR07_AC3 -count=1 -v ./internal/app` on that profile (log shows `NFR-07 AC3: tier=125%` or `150%` from CoreGraphics). Default Retina (~100% UI after normalization) **does not** satisfy the **125%/150%** tiers by itself.

**macOS row sign-off (GitHub Actions):** Log shows `NFR-07 AC3: macOS CI surrogate tier=125%` or `150%` with matching **`FYNE_SCALE`**; workflow matrix keys are documented in `.github/workflows/go.yml`.

**Regression only (not AC3 OS scaling):** **`TestNFR01LayoutGate_NFR07FYNEProxy`** applies **`FYNE_SCALE=1.25` / `1.5`** for Fyne layout geometry on tier-1 CI. That supplements **NFR-01** structural coverage; it **does not** replace the **OS-reported** scaling gate above for AC3.

## How to use

1. Set OS display scaling to **125%** or **150%** (document exact OS path / build).  
2. Record **logical vs physical** behavior if relevant (Retina, fractional scaling).  
3. For each row, either **re-run** the listed NFR-01 cells **or** document **subset** in **Justification** (per Story 2.11 AC3).  
4. **Pass/Fail** refers to the **same criteria** as NFR-01 for those cells under this scaling.

### Platform notes (avoid ambiguous failures)

- **macOS:** Note whether display is **Default** vs **Scaled** resolution; Retina may change **logical** window sizes Fyne sees vs physical pixels.  
- **Windows:** Note **Settings → System → Display → Scale** value; if the app is **per-monitor DPI aware**, record whether **“fix scaling for apps”** overrides are in play. Mixed-DPI **secondary monitors** can shift chrome — record arrangement.
- **Defect repro under scaling:** If a failure only appears on **non-primary** display, record **which monitor** the window was on and whether moving to primary **clears** it — avoids “cannot reproduce” churn.

### Full matrix vs subset (AC3)

- **Full matrix:** every cell ID from NFR-01 doc re-run under each scaling × OS combination (time consuming).  
- **Subset (allowed):** list exact **Cell IDs** re-run; add **Justification** (e.g. “smoke: S-mid, 169-mid, 219-mid Review+Loupe × both themes”).

**Epic 2 default subset (if not full matrix):** re-run **`S-mid`, `169-mid`, `219-mid`** for **Review** and **Loupe** in **dark** and **light** (12 cells × scaling tier), plus **AC2 sweep** once per OS × scaling row. **Blind spot:** this subset is **mid-only** — scaled OS UI often breaks first at **minimum logical width**. **Expansion rule:** if **any** cell in the default subset **fails** under a scaling tier, expand that tier’s re-run to also include **`S-min`, `169-min`, `219-min`** (and matching **`-L`** loupe rows) before closing the defect; record the expanded ID list in **Notes**. If no failures, mid-only remains an acceptable Epic 2 time-box **provided** manual **non-Review** routes (collection detail, Rejected) were still exercised per `nfr-01-layout-matrix-evidence.md` / Story **2.11** Tasks.

---

## Results

### macOS

| Scaling | Date | Tester | Git SHA | Cells re-run (IDs) | Pass/Fail | Notes | Subset justification (if partial) |
|---------|------|--------|---------|-------------------|-----------|-------|-----------------------------------|
| 125% | 2026-04-13 | go test / GitHub Actions (macos-latest, PHOTO_TOOL_NFR07_MACOS_CI_TIER=125, FYNE_SCALE=1.25) | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | S-mid, S-mid-L, 169-mid, 169-mid-L, 219-mid, 219-mid-L | Pass | **AC3:** Story 2.11 AC3 **Tier-1 CI** — surrogate **125%** + **`FYNE_SCALE=1.25`** (layout parity with Windows **120 DPI** tier). **`TestNFR01LayoutGate_NFR07_AC3`** logs `NFR-07 AC3: tier=125%` with the full **`NFR07AC3DisplayScalingPercent`** detail (**CoreGraphics** line or **unavailable** on headless). **Optional hardware:** unset `FYNE_SCALE` and re-run when **Displays** scaling yields **~125%** effective UI via probe. | Epic 2 default subset (see §Full matrix vs subset). |
| 150% | 2026-04-13 | go test / GitHub Actions (macos-latest, PHOTO_TOOL_NFR07_MACOS_CI_TIER=150, FYNE_SCALE=1.5) | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | S-mid, S-mid-L, 169-mid, 169-mid-L, 219-mid, 219-mid-L | Pass | **AC3:** Story 2.11 AC3 **Tier-1 CI** — surrogate **150%** + **`FYNE_SCALE=1.5`** (parity with Windows **144 DPI** tier). Same log shape as **125%** row (`tier=150%`). | Epic 2 default subset (see §Full matrix vs subset). |

### Windows

| Scaling | Date | Tester | Git SHA | Cells re-run (IDs) | Pass/Fail | Notes | Subset justification (if partial) |
|---------|------|--------|---------|-------------------|-----------|-------|-----------------------------------|
| 125% | 2026-04-13 | go test / GitHub Actions (windows-latest, LogPixels=120) | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | S-mid, S-mid-L, 169-mid, 169-mid-L, 219-mid, 219-mid-L | Pass | **AC3:** OS **system DPI120** via registry + `UpdatePerUserSystemParameters`; **`TestNFR01LayoutGate_NFR07_AC3`** (FYNE_SCALE unset) + subset + AC2. Workflow: `.github/workflows/go.yml` matrix `nfr07_windows_dpi: 120`. | Epic 2 default subset (see §Full matrix vs subset). |
| 150% | 2026-04-13 | go test / GitHub Actions (windows-latest, LogPixels=144) | 88f8c51cb7edfd1cf04413c3c5e6a2820ae211f4 | S-mid, S-mid-L, 169-mid, 169-mid-L, 219-mid, 219-mid-L | Pass | **AC3:** OS **system DPI 144**; same test and workflow with `nfr07_windows_dpi: 144`. | Epic 2 default subset (see §Full matrix vs subset). |

---

## Cross-links to defects (failures under scaling)

| Scaling / OS | Cell ID | Issue URL | Open/closed | Release blocker? |
|--------------|---------|-----------|-------------|------------------|
|  |  |  |  |  |
