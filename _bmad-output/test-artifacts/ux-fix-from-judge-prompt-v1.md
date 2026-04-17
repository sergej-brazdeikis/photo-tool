# UX fix from judge — implementer (local / Cursor `agent` only)

**Prerequisites:** A judge bundle path `BUNDLE` (e.g. `_bmad-output/test-artifacts/judge-bundles/<stamp>/`) with `verdict/judge-output.md` from [judge-prompt-v2-screenshots.md](judge-prompt-v2-screenshots.md) showing `UX_JUDGE_RESULT=fail`.

## Role

You are the **implementer**. Read the judge verdict and **minimal** UX/layout fixes in the **photo-tool** Go + Fyne codebase.

## Inputs

- `BUNDLE/verdict/judge-output.md` — gaps and per-step failures
- `BUNDLE/ui/steps.json` — which screens were captured (`flow`, `id`, `intent`, `file` per step). Steps with `id` containing `nfr01_min_window` are **1024×768** — fixes for clipping there must not regress wider layouts (run `go test ./internal/app/...` including layout gates).
- Planning context (as needed): `_bmad-output/planning-artifacts/ux-design-specification.md`, `epics-v2-ux-aligned-2026-04-14.md`

## Requirements

1. Implement **smallest** changes that address **Blocker** and **Major** items first; then **Minor** if time allows. When the verdict cites layout or “image-first,” align with the UX spec subsection **Normative criteria: image dominance (all primary flows)** — prefer progressive disclosure, horizontal scroll on strips, or collapsing chrome **before** shrinking photo real estate (thumbnails, loupe band, upload previews). If the judge cited **story / FR gaps** tied to **`context/requirements-trace.md`**, prefer the **minimal** automated tests or `ui/steps.json` / capture updates that file lists for those rows.
2. Prefer [`internal/app/upload.go`](../../internal/app/upload.go), [`internal/app/shell.go`](../../internal/app/shell.go), [`internal/app/review.go`](../../internal/app/review.go), [`internal/app/review_grid.go`](../../internal/app/review_grid.go), [`internal/app/review_loupe.go`](../../internal/app/review_loupe.go), [`internal/app/share_loupe.go`](../../internal/app/share_loupe.go), [`internal/app/collections.go`](../../internal/app/collections.go), [`internal/app/rejected.go`](../../internal/app/rejected.go) — do not refactor unrelated packages.
3. Run from repo root:

   ```bash
   go test ./...
   go build .
   ```

4. **Do not** re-run the judge inside this task; the human or [`scripts/ux-judge-loop.sh`](../../scripts/ux-judge-loop.sh) will assemble a **new** bundle and call the judge again.

## Output

- Code + tests updated.
- Short note in the PR or story file if your team tracks Dev Agent Record.
