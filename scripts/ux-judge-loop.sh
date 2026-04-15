#!/usr/bin/env bash
# Local-only: assemble judge bundle → vision judge (Cursor `agent`) → optional fix agent → repeat.
# CI must never run this script (no LLM / no agent in GitHub Actions for this repo).
#
# Prerequisites: `agent` on PATH (`agent login` or CURSOR_API_KEY), python3 (for manifest in assemble script).
#
# Env:
#   UX_LOOP_MAX       — max rounds (default 5)
#   UX_AGENT          — agent binary name if not `agent` (default: agent)
#   CURSOR_AGENT      — alias for UX_AGENT (plan name); UX_AGENT wins if both set
#
# Human kill switch: Ctrl+C between iterations.
#
# Usage (from repo root):
#   ./scripts/ux-judge-loop.sh
set -euo pipefail

log() { printf '%s\n' "$*" >&2; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "$REPO"

AGENT_BIN="${UX_AGENT:-${CURSOR_AGENT:-agent}}"
if ! command -v "${AGENT_BIN}" >/dev/null 2>&1; then
	log "error: '${AGENT_BIN}' not on PATH (set UX_AGENT if your binary has another name)"
	exit 1
fi

JUDGE_SPEC="${REPO}/_bmad-output/test-artifacts/judge-prompt-v2-screenshots.md"
FIX_SPEC="${REPO}/_bmad-output/test-artifacts/ux-fix-from-judge-prompt-v1.md"
for f in "${JUDGE_SPEC}" "${FIX_SPEC}"; do
	if [[ ! -f "${f}" ]]; then
		log "error: missing ${f}"
		exit 1
	fi
done

max="${UX_LOOP_MAX:-5}"
run_agent() {
	local prompt="$1"
	"${AGENT_BIN}" --print --trust --workspace "${REPO}" --yolo "${prompt}"
}

for i in $(seq 1 "${max}"); do
	log "ux-judge-loop: iteration ${i}/${max}"
	BUNDLE="$(UX_BUNDLE_EMIT_PATH_ONLY=1 "${SCRIPT_DIR}/assemble-judge-bundle.sh")"
	log "ux-judge-loop: bundle ${BUNDLE}"

	run_agent "photo-tool — **UX judge (v2, screenshots)**. Run locally only.

Read and follow the instructions in this file (open it and use as the contract):
${JUDGE_SPEC}

**Bundle directory (absolute):** ${BUNDLE}

- Use \`context/rubric.md\`, \`ui/steps.json\`, ordered \`ui/*.png\`, and \`logs/go-test.txt\` from that bundle.
- Write **only** under that bundle: \`verdict/judge-output.md\` with structure required by the spec.
- The **last line** of \`verdict/judge-output.md\` must be exactly \`UX_JUDGE_RESULT=pass\` or \`UX_JUDGE_RESULT=fail\` (no other text on that line)."

	verdict="${BUNDLE}/verdict/judge-output.md"
	if [[ ! -f "${verdict}" ]]; then
		log "ux-judge-loop: missing ${verdict}; treating as fail"
	elif [[ "$(tail -n 1 "${verdict}" | tr -d '\r')" == "UX_JUDGE_RESULT=pass" ]]; then
		log "ux-judge-loop: UX_JUDGE_RESULT=pass — done"
		exit 0
	else
		log "ux-judge-loop: judge did not end with UX_JUDGE_RESULT=pass (see ${verdict})"
	fi

	run_agent "photo-tool — **UX implementer (fix from judge)**.

Read and follow:
${FIX_SPEC}

**Repository root:** ${REPO}
**Bundle (read verdict here):** ${BUNDLE}

Implement minimal Fyne/layout changes to address gaps in \`verdict/judge-output.md\`. Run \`go test ./...\` and \`go build .\` from the repo root. Do **not** re-run the judge in this invocation; the outer loop will re-bundle and re-judge."

	log "ux-judge-loop: fix iteration complete; next loop will re-capture"
done

log "ux-judge-loop: exceeded UX_LOOP_MAX=${max} without pass"
exit 1
