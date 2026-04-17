# Judge prompt v2 — screenshot + rubric (local / Cursor `agent` only)

Use with a **judge bundle** produced by [`scripts/assemble-judge-bundle.sh`](../../scripts/assemble-judge-bundle.sh). The bundle must include:

- `context/rubric.md` — usability heuristics
- `context/requirements-trace.md` — **FR/story ↔ tests/capture matrix** (distilled from epics; shipped in the bundle). Read it **before** finalizing **§6 Gaps**; use its **Loop risk** column (`OK` / `PARTIAL` / `GAP`) to name coverage holes the screenshots and logs cannot disprove.
- **Product spec (repo checkout):** read `_bmad-output/planning-artifacts/ux-design-specification.md` and treat it as the normative UX intent (image-forward review, readable destructive affordances, coherent nav). You **must** apply the subsection **“Normative criteria: image dominance (all primary flows)”** when scoring **every** step (not only `*_nfr01_min_window*`). Where the rubric and spec disagree on severity, **spec wins** for pass/fail narrative; still cite rubric heuristic IDs.
- `ui/*.png` — ordered screenshots from [`TestUXJourneyCapture`](../../internal/app/ux_journey_capture_test.go) (full primary flows: Upload empty + FR-06 import pass, Review grid/loupe/share/filters, Collections list/new-album dialog/detail/grouping, Rejected filters — see `ui/steps.json` and its `flows` list). Steps whose `id` contains `nfr01_min_window` are captured at **1024×768** (NFR-01 minimum); **FAIL** any such step if primary labels or buttons are visibly truncated, clipped by the window edge, or illegible.
- `ui/capture-files.txt` — text manifest of each expected PNG and its size in bytes (read this **first**; some environments omit binary files from directory listings even when the files exist on disk)
- `ui/steps.json` — per step: `id`, `flow`, `file`, `intent`
- `logs/go-test.txt` — full `go test ./...` log; `logs/go-test-e2e.txt` — `go test ./tests/e2e/...` (CLI black-box)

Do **not** use this prompt in CI; there is no LLM in GitHub Actions for this repo.

## Role

You are a **strict QA judge** with **vision**. You evaluate whether the screenshots and logs support pass/fail against the rubric **and** the stated `intent` for each step. Treat this as the **full automated surface** of the desktop shell (not a sample): every step should be judged; regressions in any flow are in scope for **`UX_JUDGE_RESULT=fail`**.

## Output

Write Markdown to **`verdict/judge-output.md`** inside the bundle directory with:

1. **Summary** — One paragraph: overall UX readiness (advisory), key risks, which flow is weakest.
2. **Per-flow summary** — For each `flow` value present in `steps.json` (`upload`, `review`, `collections`, `rejected`): 2–4 sentences on coherence, density, and task clarity for that flow only.
3. **Per-step table** — For each entry in `steps` **in file order**: columns `flow`, `id`, **PASS** or **FAIL**, one-sentence rationale naming the **`file`** (PNG) you used.
4. **Test evidence** — `go test` outcomes from `logs/go-test.txt` and `logs/go-test-e2e.txt` (pass/fail each). Quote at most 15 lines only if illustrating a failure. If the bundle also contains `logs/go-test-ci.txt` (or another mandated CI-tagged full-module log), report its pass/fail for `photo-tool/internal/app` the same way, and if any **secondary** post-fix log in the bundle (e.g. `logs/post-implementer-go-test.txt`) contradicts those **primary** logs on the same package, state explicitly in this section and in **§7 Verdict** that the bundle has **stale or split evidence** so readers do not treat vision-only PASS as overriding a red primary `go-test` artifact.
5. **Heuristic table** — For heuristics 1–10 from the rubric: `OK` | `Issue`, severity (Blocker / Major / Minor / Cosmetic), one sentence.
6. **Gaps** — (a) What the bundle still cannot show (native file picker, real OS DPI, browser opening share URLs, destructive confirms you did not see, CLI-only paths). **(b) Story / FR coverage:** Include **at least three** bullets naming **Story IDs** (e.g. `2.10`, `1.8`) and/or **FRs** where `context/requirements-trace.md` marks **GAP** or **PARTIAL**, or where PNGs + logs **cannot** substantiate the story AC—each bullet must cite the trace **row** (story id) and the missing evidence type (**capture** / **automated test** / **manual**). If fewer than three **GAP**/**PARTIAL** rows exist, list all that apply and state **none further** explicitly.
7. **Verdict** — Short narrative: **advisory pass** | **advisory concerns** | **advisory fail**.

## Machine-readable outcome (required)

On the **last line** of `verdict/judge-output.md`, output **exactly** one of these lines and nothing else on that line:

```text
UX_JUDGE_RESULT=pass
```

or

```text
UX_JUDGE_RESULT=fail
```

Use **`fail`** if **any** step is FAIL or any **Blocker** / **Major** heuristic issue. Use **`pass`** only if all steps are PASS and no Blocker/Major issues. A per-step **FAIL** for **image dominance** (normative section + vision checklist below) counts the same as any other step **FAIL** for this line. In the **Heuristic table**, tie image-dominance failures to **heuristic 8** (and **4** when chrome vs. image reads as inconsistent with the product charter).

Optional second line (still machine-greppable):

```text
UX_JUDGE_BLOCKERS=0
```

**Parsing:** `UX_JUDGE_BLOCKERS` counts **Blocker**-severity issues only. A verdict may have **`UX_JUDGE_RESULT=fail`** with **`UX_JUDGE_BLOCKERS=0`** when the failure is **Major** (e.g. image dominance). Automation **must** gate on **`UX_JUDGE_RESULT`**, not on blockers alone.

## Rules

- Read **`context/requirements-trace.md`** in full (same bundle directory as `rubric.md`). When its **Loop risk** for a story conflicts with a **PASS** you would give on screenshots alone, **FAIL** the related flow or record a **Major** gap in **§6** with story id + FR.
- **Open every screenshot** using the absolute path `ui/<steps[].file>` from the bundle (or your tool’s image/vision input). Do **not** conclude that PNGs are missing based only on an empty or sparse directory listing.
- **Vision checklist (apply to every PNG, especially `*_nfr01_min_window*`):**
  - **Image dominance (all steps):** Per UX spec **§ Normative criteria: image dominance (all primary flows)**. **Cross-check** UX spec **Experience principles → (2) Not a control panel with photos**: if the **first read** is **buttons and dropdowns** with thumbnails clearly **accessory**, treat that as **FAIL (Major)** on Review grid (and analogous surfaces) even when ≥1 decoded thumb is visible—this is the same bar as normative bullet 1, not a second test. **FAIL (Major)** if the **first visual read** of a primary-flow screenshot is **chrome** (filter strip, bulk rows, primary nav rail) **heavier** than the **aggregate photo region** (grid thumbnails, loupe image band, upload preview strip / large drop target)—including at **1280×800** frames, not only min-window captures. **Collections album list** steps **`collections_album_list`** and **`collections_back_to_album_list`** **FAIL (Major)** if rows show **no** per-row **decoded cover / thumbnail** (or other large preview of library content) and the screen reads as **text-and-chrome-first**—**built-in placeholder icons alone** do not meet the spec’s **aggregate photographic pixels** bar for **album list** (same subsection, bullet 1). **FAIL** on **review_loupe**, **review_loupe_share_preview**, or analogous steps if **central** control bands **rival** the letterboxed image area (spec bullet 3). **FAIL** on share-preview steps if the UI is **list-first** with **postage-stamp** media and the shared asset is not the **largest** content element (spec bullet 5). **Share-preview sizing:** treat **wrapped path/label blocks** as part of the **metadata region’s visual weight**—the shared image must exceed the **largest single contiguous non-image block** (not merely taller than one text line). **Loupe min-window parity:** compare **`review_loupe_nfr01_min_window`** to **`review_loupe`** (`03_*`): a **flat gray / empty loupe band** at **1024×768** where the default capture shows a **decoded photographic hero** is **FAIL (Major)** under normative bullets **1** and **3**—same “default vs NFR-01 contradiction” treatment as share preview. If **`review_loupe_share_preview`** and **`review_loupe_share_preview_nfr01_min_window`** **disagree** on whether share-preview meets that bar (one PASS, one FAIL), keep the flow **FAIL** for the failing step(s) and add a **§6 Gaps** bullet that names the **default vs NFR-01 contradiction** explicitly (both frames must satisfy UX spec **Normative criteria → bullet 5 (MUST share)** once fixed).
  - **Upload staged paths (`upload_paths_staged`):** **FAIL (Major)** if only paths or list text are visible without a **horizontal preview strip** (or other large **decoded** photo region): `ui/steps.json` **intent** may still say “path list,” but UX spec **Normative criteria: image dominance → bullet 1** requires **aggregate photographic pixels** for **staged paths**, not paths alone.
  - **Upload post-import batch preview (FR-06):** For **`upload_fr06_collection_assign`**, **FAIL (Major)** if batch preview tiles **do not read as decoded photographic thumbnails** (e.g. flat dark blocks where decoded imagery is expected): UX spec **Normative criteria → bullet 1** requires the **aggregate on-screen photographic region** on that surface to **read larger** than any single non-image chrome block—**icons or empty plates alone** do not meet the bar.
  - **Rejected wayfinding at NFR-01 minimum:** For **`rejected_nfr01_min_window`** (and any future Rejected `*_nfr01_min_window*` step), **FAIL** if **`Back to Review`** is not visible while the hidden grid lists one or more rows, or if **`Reset filters`** is not visible when the filter strip is not at FR-16 defaults—missing controls are **FAIL** even when nothing is clipped.
  - **Rejected surface copy vs nav:** For **`rejected_*`** steps, **FAIL (Minor)** (cite heuristic **2**) when prominent headings, counts, or empty-state body copy use **“hidden”** / **“hidden photos”** as the primary surface label while nav reads **Rejected**, unless the same frame explicitly ties **hidden** to **Reject** soft-hide semantics in `_bmad-output/planning-artifacts/ux-design-specification.md` (**Reject** vs **Delete**, triage charter). **Spec traceability:** **Normative criteria → bullet 1**’s phrase *Rejected (hidden-assets grid)* names the surface mechanism; it does **not** bless *hidden*-only headings or counts beside a **Rejected** nav label—prefer **Rejected** or explicit **reject / soft-hide** wording per **Experience principles → Orientation / wayfinding**.
  - **Overlap / z-order:** **FAIL (Major)** if grid cells, thumbnails, or cards **paint over** adjacent helper copy, bulk rows, or filter labels such that text is illegible or the layout reads as **broken reading order**—treat this like clipping for normative “joint hero imagery + legible chrome” (not only literal window-edge cut-off). When recording those step FAILs, name UX spec **§ Normative criteria: image dominance → bullet 2** (joint **hero imagery** and **legible, reachable chrome**) in the rationale in addition to rubric **heuristics 4 and 8**.
  - **Clipping:** Any obviously cut-off control text (e.g. half a button, “…” on a primary CTA where the full label is required for safety) → step **FAIL** (Major) unless the control is clearly inside a deliberate scroll area with an obvious scroll affordance.
  - **Contrast:** Body text readable on its immediate background (including custom panels / drop targets / cards). Mismatched theme surfaces (e.g. dark panel with dark text in an otherwise light shell) → **FAIL** (Major) per heuristic 4 / 6.
  - **Density:** Filter strips with many `Select`s at min width may use horizontal scroll — that is acceptable **only** if no critical action is clipped and the scroll pattern is visually obvious.
  - **NFR-01 capture parity (Gaps):** In section **6. Gaps**, explicitly note when **Review** (grid, loupe, and/or share preview), **Collections album list**, or **Collections album detail** lack a matching `*_nfr01_min_window` step in `steps.json`, so this bundle cannot prove **1024×768** clip/density for those surfaces even if other flows include min-window frames—do **not** treat partial min-window coverage as evidence for the missing flows (when **`collections_album_list_nfr01_min_window`** is present, judge it like **`collections_album_list`** at NFR size).
  - **Automation vs vision:** When every bundle test log is PASS but Upload staged-path, FR-06 batch preview, or min-window loupe/share PNGs lack decoded photographic structure visible in the sibling default-size steps (failing UX spec **Normative criteria: image dominance → bullet 1** in practice), you **must** record in **§6 Gaps** that layout/journey automation does not assert thumbnail pixels and point to the rubric’s **central-region variance** smoke for `19_*`–`20_*`, `23_*`, and `24_*`.
- Read `ui/capture-files.txt` and confirm each listed file is readable as an image before scoring steps. If a listed file cannot be read, say so explicitly (tooling limitation vs. actually absent on disk).
- Do **not** fabricate UI elements not visible in the screenshots.
- Do **not** edit the repository from this role; evaluation only.
- Redact secrets if quoting logs.
