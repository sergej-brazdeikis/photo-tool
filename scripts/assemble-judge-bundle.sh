#!/usr/bin/env bash
# Assemble a text-first judge bundle under _bmad-output/test-artifacts/judge-bundles/<stamp>/.
# Run from any directory; requires git checkout and Go toolchain. Advisory / local use only.
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || {
  echo "error: not inside a git repository" >&2
  exit 1
}
cd "$ROOT"

SHORT="$(git rev-parse --short HEAD)"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)-${SHORT}"
OUT="${ROOT}/_bmad-output/test-artifacts/judge-bundles/${STAMP}"
RUBRIC_SRC="${ROOT}/_bmad-output/test-artifacts/e2e-usability-rubric.md"

mkdir -p "${OUT}/logs" "${OUT}/context" "${OUT}/verdict"

if [[ ! -f "${RUBRIC_SRC}" ]]; then
  echo "error: missing ${RUBRIC_SRC}" >&2
  exit 1
fi

cp "${RUBRIC_SRC}" "${OUT}/context/rubric.md"

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

cat >"${OUT}/README.md" <<EOF
# Judge bundle (${STAMP})

Assembled for qualitative LLM review (advisory only). Do not commit secrets.

- Read \`context/rubric.md\` and \`logs/go-test.txt\`.
- Apply instructions in \`_bmad-output/test-artifacts/judge-prompt-v1.md\` (repo root).
- Write findings to \`verdict/judge-output.md\`.
EOF

# Minimal JSON (no jq): escape branch for JSON string
BRANCH_ESC="${BRANCH//\\/\\\\}"
BRANCH_ESC="${BRANCH_ESC//\"/\\\"}"

cat >"${OUT}/manifest.json" <<EOF
{
  "bundle_version": "1",
  "created_at": "${CREATED}",
  "git_commit": "${COMMIT}",
  "git_branch": "${BRANCH_ESC}",
  "go_version": "${GOVER}",
  "os": "${GOOS}",
  "arch": "${GOARCH}",
  "commands_run": [
    "go test ./... -count=1",
    "go test -tags ci ./... -count=1"
  ],
  "exit_codes": {
    "go_test": ${GO_TEST_EC},
    "go_test_ci": ${GO_TEST_CI_EC}
  },
  "judge_prompt_id": "judge-v1",
  "omissions": ["logs/e2e-cli.txt omitted; tests/e2e is included in logs/go-test.txt"]
}
EOF

echo "Wrote bundle: ${OUT}"
echo "go test exit codes: main=${GO_TEST_EC} ci=${GO_TEST_CI_EC}"
