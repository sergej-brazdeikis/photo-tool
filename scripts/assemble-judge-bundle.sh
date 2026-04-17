#!/usr/bin/env bash
# Assemble a judge bundle under _bmad-output/test-artifacts/judge-bundles/<stamp>/.
# Includes go test logs (full module, -tags ci, CLI e2e under tests/e2e), rubric, FR/story requirements trace,
# UI journey PNGs (Fyne capture), and steps.json.
# Run from any directory; requires git + Go. LLM/judge runs locally only (not CI).
#
# Env:
#   UX_BUNDLE_ALLOW_E2E_FAIL — if 1, `go test ./tests/e2e/...` failure is recorded in manifest omissions only;
#                              default (unset) fails the script so the UX loop does not run the judge on a red tree.
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
REQ_TRACE_SRC="${ROOT}/_bmad-output/test-artifacts/bundle-requirements-trace.md"

mkdir -p "${OUT}/logs" "${OUT}/context" "${OUT}/verdict" "${OUT}/ui"

if [[ ! -f "${RUBRIC_SRC}" ]]; then
	log "error: missing ${RUBRIC_SRC}"
	exit 1
fi
if [[ ! -f "${REQ_TRACE_SRC}" ]]; then
	log "error: missing ${REQ_TRACE_SRC}"
	exit 1
fi

cp "${RUBRIC_SRC}" "${OUT}/context/rubric.md"
cp "${REQ_TRACE_SRC}" "${OUT}/context/requirements-trace.md"

cat >"${OUT}/ui/README.md" <<'UIREAD'
# `ui/` screenshots

PNG files and `steps.json` are produced by `TestUXJourneyCapture` when `PHOTO_TOOL_UX_CAPTURE_DIR` points at this directory **and** `PHOTO_TOOL_UX_JOURNEY_TEST=1` (see `scripts/assemble-judge-bundle.sh`). Most steps use **1280×800** and default **light** theme (`PHOTO_TOOL_TEST_THEME=dark` for dark). The harness also captures **Rejected** and **Upload** at **1024×768** (NFR-01 minimum) so judges see horizontal clip / density issues. After a successful capture, `capture-files.txt` lists each expected PNG and its size (bytes) for judges and tooling that skip binary files in directory listings.

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
go test ./tests/e2e/... -count=1 >"${OUT}/logs/go-test-e2e.txt" 2>&1
GO_TEST_E2E_EC=$?
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
export _JBM_GO_TEST_E2E_EC="${GO_TEST_E2E_EC}"
export _JBM_UX_CAPTURE_EC="${UX_CAPTURE_EC}"
export _JBM_ALLOW_E2E_FAIL="${UX_BUNDLE_ALLOW_E2E_FAIL:-}"

cat >"${OUT}/README.md" <<EOF
# Judge bundle (${STAMP})

## Text-only (v1)

- \`context/rubric.md\`, \`logs/go-test.txt\`
- [_bmad-output/test-artifacts/judge-prompt-v1.md](../judge-prompt-v1.md)
- Write \`verdict/judge-output.md\`

## Screenshots + rubric (v2, local \`agent\` only)

- \`context/rubric.md\`, \`context/requirements-trace.md\` (FR/story ↔ tests/capture matrix for the judge + QA loop)
- \`ui/*.png\`, \`ui/steps.json\`, \`ui/capture-files.txt\` (PNG manifest)
- [_bmad-output/test-artifacts/judge-prompt-v2-screenshots.md](../judge-prompt-v2-screenshots.md)
- Write \`verdict/judge-output.md\` with final line \`UX_JUDGE_RESULT=pass\` or \`UX_JUDGE_RESULT=fail\`

## Closed loop

- [\`scripts/ux-judge-loop.sh\`](../../scripts/ux-judge-loop.sh) (requires \`agent\` on PATH; not for CI): judge → UX fix → optional post-fix review → optional **QA agent** → optional **QA fix** (when \`QA_LOOP_RESULT=fail\`; \`UX_LOOP_QA_AGENT\`, \`UX_LOOP_QA_FIX\`) → optional **self-improve** (\`UX_LOOP_SELF_IMPROVE\`).
- **QA logs:** \`logs/go-test-e2e.txt\` — \`go test ./tests/e2e/...\` (CLI black-box). Strict by default; set \`UX_BUNDLE_ALLOW_E2E_FAIL=1\` when assembling to allow a failing E2E without aborting (recorded in \`manifest.json\` \`omissions\`).
EOF

python3 <<'PY'
import json
import os
import sys

out = os.environ["_JBM_OUT"]
omissions = [
    "logs/go-test-e2e.txt holds go test ./tests/e2e/... (CLI black-box); full module tests in logs/go-test.txt",
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

validation_failed = False
min_png = 32
ui_dir = os.path.join(out, "ui")
capture_list_path = os.path.join(ui_dir, "capture-files.txt")
lines_out = []
missing = []
if ux_ec == 0 and steps_meta:
    for s in steps_meta:
        fn = (s or {}).get("file") or ""
        if not fn:
            missing.append("(step without file)")
            continue
        fp = os.path.join(ui_dir, fn)
        if not os.path.isfile(fp):
            missing.append(fn)
            continue
        sz = os.path.getsize(fp)
        if sz < min_png:
            missing.append(f"{fn} (size {sz})")
            continue
        lines_out.append(f"{fn}\t{sz}\n")
    if missing:
        validation_failed = True
        omissions.append(
            "ui PNG validation failed (steps.json vs files on disk): " + ", ".join(missing[:12])
            + (" …" if len(missing) > 12 else "")
        )
    else:
        with open(capture_list_path, "w", encoding="utf-8") as cf:
            cf.write("# One row per steps.json screenshot: filename<TAB>size_bytes\n")
            cf.writelines(lines_out)
elif ux_ec == 0 and not steps_meta and os.path.isfile(steps_path):
    validation_failed = True
    omissions.append("ux capture exited 0 but steps.json has no steps[]")
elif ux_ec == 0 and not os.path.isfile(steps_path):
    validation_failed = True

e2e_ec = int(os.environ["_JBM_GO_TEST_E2E_EC"])
allow_e2e_fail = os.environ.get("_JBM_ALLOW_E2E_FAIL", "").strip() == "1"
e2e_hard_fail = e2e_ec != 0 and not allow_e2e_fail
if e2e_ec != 0 and allow_e2e_fail:
    omissions.append(
        f"go test ./tests/e2e/... failed (exit {e2e_ec}); see logs/go-test-e2e.txt (UX_BUNDLE_ALLOW_E2E_FAIL=1)"
    )

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
        "go test ./tests/e2e/... -count=1",
        "PHOTO_TOOL_UX_JOURNEY_TEST=1 PHOTO_TOOL_UX_CAPTURE_DIR=<bundle>/ui go test ./internal/app -run TestUXJourneyCapture -count=1",
    ],
    "exit_codes": {
        "go_test": int(os.environ["_JBM_GO_TEST_EC"]),
        "go_test_ci": int(os.environ["_JBM_GO_TEST_CI_EC"]),
        "go_test_e2e": e2e_ec,
        "ux_capture": ux_ec,
    },
    "judge_prompt_v1": "_bmad-output/test-artifacts/judge-prompt-v1.md",
    "judge_prompt_v2_screenshots": "_bmad-output/test-artifacts/judge-prompt-v2-screenshots.md",
    "fix_prompt_v1": "_bmad-output/test-artifacts/ux-fix-from-judge-prompt-v1.md",
    "post_fix_code_review_loop": "_bmad-output/test-artifacts/ux-post-fix-code-review-loop.md",
    "ux_judge_loop_self_improve": "_bmad-output/test-artifacts/ux-judge-loop-self-improve.md",
    "ux_qa_post_fix_loop": "_bmad-output/test-artifacts/ux-qa-post-fix-loop.md",
    "ux_fix_from_qa_prompt_v1": "_bmad-output/test-artifacts/ux-fix-from-qa-prompt-v1.md",
    "requirements_trace": "context/requirements-trace.md",
    "ui_steps_file": "ui/steps.json",
    "ui_capture_files_list": "ui/capture-files.txt",
    "steps": steps_meta,
    "capture_manifest": capture_meta,
    "omissions": omissions,
}
with open(os.path.join(out, "manifest.json"), "w", encoding="utf-8") as f:
    json.dump(manifest, f, indent=2)
    f.write("\n")
if validation_failed:
    print(
        "error: judge bundle UI capture incomplete — see omissions in manifest.json and logs/ux-capture.txt",
        file=sys.stderr,
    )
if ux_ec != 0:
    print(
        f"error: TestUXJourneyCapture failed (exit {ux_ec}); see {out}/logs/ux-capture.txt — not running judge on this bundle",
        file=sys.stderr,
    )
if validation_failed or ux_ec != 0:
    sys.exit(1)
if e2e_hard_fail:
    print(
        f"error: go test ./tests/e2e/... failed (exit {e2e_ec}); see {out}/logs/go-test-e2e.txt "
        "(set UX_BUNDLE_ALLOW_E2E_FAIL=1 to assemble anyway — recorded in manifest omissions)",
        file=sys.stderr,
    )
    sys.exit(1)
PY

log "Wrote bundle: ${OUT}"
log "exit codes: go_test=${GO_TEST_EC} go_test_ci=${GO_TEST_CI_EC} go_test_e2e=${GO_TEST_E2E_EC} ux_capture=${UX_CAPTURE_EC}"

if [[ "${UX_BUNDLE_EMIT_PATH_ONLY:-}" == "1" ]]; then
	printf '%s\n' "${OUT}"
fi
