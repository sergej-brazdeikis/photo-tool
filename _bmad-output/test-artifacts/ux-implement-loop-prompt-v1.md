# UX judge loop — single-agent driver prompt (local only)

Use when you want **one** Cursor `agent` session to orchestrate the **capture → judge → fix** loop instead of running the shell script yourself.

**Do not use in CI.** GitHub Actions for this repo runs `go test` and static checks only.

## Action

1. From the **photo-tool repository root**, run exactly:

   ```bash
   ./scripts/ux-judge-loop.sh
   ```

   Or: `make ux-judge-loop`.

2. If the command is not available or fails, align with the same behavior manually each round:
   - `UX_BUNDLE_EMIT_PATH_ONLY=1 ./scripts/assemble-judge-bundle.sh` → note the printed bundle path.
   - Judge: follow [`judge-prompt-v2-screenshots.md`](judge-prompt-v2-screenshots.md) against that bundle; ensure `verdict/judge-output.md` ends with `UX_JUDGE_RESULT=pass` or `UX_JUDGE_RESULT=fail`.
   - On fail: follow [`ux-fix-from-judge-prompt-v1.md`](ux-fix-from-judge-prompt-v1.md), then start a new round with a **fresh** bundle.

## Environment

| Variable | Meaning |
|----------|---------|
| `UX_LOOP_MAX` | Maximum judge+fix rounds (default `5`). |
| `UX_AGENT` | Cursor CLI binary name if not `agent`. |
| `CURSOR_AGENT` | Alias for `UX_AGENT` in `ux-judge-loop.sh`. |

## Stop condition

Success when the judge writes **`UX_JUDGE_RESULT=pass`** as the **sole content of the last line** of `verdict/judge-output.md` inside the latest bundle. If rounds exceed `UX_LOOP_MAX`, stop and report.
