#!/usr/bin/env bash
# Assemble a judge bundle under _bmad-output/test-artifacts/judge-bundles/<stamp>/.
# Includes go test logs, rubric, UI journey PNGs (Fyne capture), and steps.json.
# Run from any directory; requires git + Go. LLM/judge runs locally only (not CI).
set -euo pipefail

log() { printf '%s\n' "$*" >&2; }

ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || {
	log "error: not inside a git repository"
	exit 1
}
cd "$ROOT"

SHORT="$(git rev-parse --short HEAD)"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)-${SHORT}"
OUT="${ROOT}/_bmad-output/test-artifacts/judge-bundles/${STAMP}"
RUBRIC_SRC="${ROOT}/_bmad-output/test-artifacts/e2e-usability-rubric.md"

mkdir -p "${OUT}/logs" "${OUT}/context" "${OUT}/verdict" "${OUT}/ui"

if [[ ! -f "${RUBRIC_SRC}" ]]; then
	log "error: missing ${RUBRIC_SRC}"
	exit 1
fi

cp "${RUBRIC_SRC}" "${OUT}/context/rubric.md"

cat >"${OUT}/ui/README.md" <<'UIREAD'
# `ui/` screenshots

PNG files and `steps.json` are produced by `TestUXJourneyCapture` when `PHOTO_TOOL_UX_CAPTURE_DIR` points at this directory **and** `PHOTO_TOOL_UX_JOURNEY_TEST=1` (see `scripts/assemble-judge-bundle.sh`).

**Manual overrides:** If headless capture fails for a step on your platform, add or replace `NN_*.png` and edit `steps.json` so `steps[].file`, `flow`, and `intent` stay accurate for [judge-prompt-v2-screenshots.md](../judge-prompt-v2-screenshots.md). Keep numeric prefixes in order. Re-run the bundle script so `manifest.json` picks up the updated `steps.json`.
UIREAD

CREATED="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
COMMIT="$(git rev-parse HEAD)"
BRANCH="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)"
GOVER="$(go env GOVERSION 2>/dev/null || echo unknown)"
GOOS="$(go env GOOS 2>/dev/null || echo unknown)"
GOARCH="$(go env GOARCH 2>/dev/null || echo unknown)"

set +e
go test ./... -count=1 >"${OUT}/logs/go-test.txt" 2>&1
GO_TEST_EC=$?
set -e

set +e
go test -tags ci ./... -count=1 >"${OUT}/logs/go-test-ci.txt" 2>&1
GO_TEST_CI_EC=$?
set -e

set +e
PHOTO_TOOL_UX_JOURNEY_TEST=1 PHOTO_TOOL_UX_CAPTURE_DIR="${OUT}/ui" go test ./internal/app -run TestUXJourneyCapture -count=1 -v >"${OUT}/logs/ux-capture.txt" 2>&1
UX_CAPTURE_EC=$?
set -e

export _JBM_OUT="${OUT}"
export _JBM_CREATED="${CREATED}"
export _JBM_COMMIT="${COMMIT}"
export _JBM_BRANCH="${BRANCH}"
export _JBM_GOVER="${GOVER}"
export _JBM_GOOS="${GOOS}"
export _JBM_GOARCH="${GOARCH}"
export _JBM_GO_TEST_EC="${GO_TEST_EC}"
export _JBM_GO_TEST_CI_EC="${GO_TEST_CI_EC}"
export _JBM_UX_CAPTURE_EC="${UX_CAPTURE_EC}"

cat >"${OUT}/README.md" <<EOF
# Judge bundle (${STAMP})

## Text-only (v1)

- \`context/rubric.md\`, \`logs/go-test.txt\`
- [_bmad-output/test-artifacts/judge-prompt-v1.md](../judge-prompt-v1.md)
- Write \`verdict/judge-output.md\`

## Screenshots + rubric (v2, local \`agent\` only)

- \`ui/*.png\`, \`ui/steps.json\`
- [_bmad-output/test-artifacts/judge-prompt-v2-screenshots.md](../judge-prompt-v2-screenshots.md)
- Write \`verdict/judge-output.md\` with final line \`UX_JUDGE_RESULT=pass\` or \`UX_JUDGE_RESULT=fail\`

## Closed loop

- [\`scripts/ux-judge-loop.sh\`](../../scripts/ux-judge-loop.sh) (requires \`agent\` on PATH; not for CI)
EOF

python3 <<'PY'
import json
import os

out = os.environ["_JBM_OUT"]
omissions = [
    "logs/e2e-cli.txt omitted; tests/e2e is included in logs/go-test.txt",
]
ux_ec = int(os.environ["_JBM_UX_CAPTURE_EC"])
if ux_ec != 0:
    omissions.append(
        f"ux journey capture failed (exit {ux_ec}); see logs/ux-capture.txt"
    )
steps_path = os.path.join(out, "ui", "steps.json")
steps_meta = []
capture_meta = {}
if not os.path.isfile(steps_path):
    omissions.append("ui/steps.json missing after capture")
else:
    try:
        with open(steps_path, encoding="utf-8") as sf:
            capture_meta = json.load(sf)
        steps_meta = capture_meta.get("steps", [])
    except OSError as exc:
        omissions.append(f"ui/steps.json unreadable: {exc}")
    except json.JSONDecodeError as exc:
        omissions.append(f"ui/steps.json invalid JSON: {exc}")
manifest = {
    "bundle_version": "2",
    "created_at": os.environ["_JBM_CREATED"],
    "git_commit": os.environ["_JBM_COMMIT"],
    "git_branch": os.environ["_JBM_BRANCH"],
    "go_version": os.environ["_JBM_GOVER"],
    "os": os.environ["_JBM_GOOS"],
    "arch": os.environ["_JBM_GOARCH"],
    "commands_run": [
        "go test ./... -count=1",
        "go test -tags ci ./... -count=1",
        "PHOTO_TOOL_UX_JOURNEY_TEST=1 PHOTO_TOOL_UX_CAPTURE_DIR=<bundle>/ui go test ./internal/app -run TestUXJourneyCapture -count=1",
    ],
    "exit_codes": {
        "go_test": int(os.environ["_JBM_GO_TEST_EC"]),
        "go_test_ci": int(os.environ["_JBM_GO_TEST_CI_EC"]),
        "ux_capture": ux_ec,
    },
    "judge_prompt_v1": "_bmad-output/test-artifacts/judge-prompt-v1.md",
    "judge_prompt_v2_screenshots": "_bmad-output/test-artifacts/judge-prompt-v2-screenshots.md",
    "fix_prompt_v1": "_bmad-output/test-artifacts/ux-fix-from-judge-prompt-v1.md",
    "post_fix_code_review_loop": "_bmad-output/test-artifacts/ux-post-fix-code-review-loop.md",
    "ui_steps_file": "ui/steps.json",
    "steps": steps_meta,
    "capture_manifest": capture_meta,
    "omissions": omissions,
}
with open(os.path.join(out, "manifest.json"), "w", encoding="utf-8") as f:
    json.dump(manifest, f, indent=2)
    f.write("\n")
PY

log "Wrote bundle: ${OUT}"
log "exit codes: go_test=${GO_TEST_EC} go_test_ci=${GO_TEST_CI_EC} ux_capture=${UX_CAPTURE_EC}"

if [[ "${UX_BUNDLE_EMIT_PATH_ONLY:-}" == "1" ]]; then
	printf '%s\n' "${OUT}"
fi
