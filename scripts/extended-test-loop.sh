#!/usr/bin/env bash
# Hands-free extended test loop until EXTENDED_RESULT=pass or EXTENDED_LOOP_MAX.
set -euo pipefail

log() { printf '%s\n' "$*" >&2; }

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export REPO
# shellcheck source=extended-test-env.sh
source "${REPO}/scripts/extended-test-env.sh"

MAX="${EXTENDED_LOOP_MAX:-20}"
PARALLEL="${EXTENDED_PARALLEL_FIX:-4}"
AGENT="${EXTENDED_AGENT_BIN}"
FIX_SPEC="${REPO}/_bmad-output/test-artifacts/extended-fix-from-issue-prompt-v1.md"
STEP_SPEC="${REPO}/_bmad-output/test-artifacts/judge-prompt-step-ux.md"
FLOW_SPEC="${REPO}/_bmad-output/test-artifacts/judge-prompt-flow-ux.md"

for f in "${FIX_SPEC}" "${STEP_SPEC}" "${FLOW_SPEC}"; do
	if [[ ! -f "${f}" ]]; then
		log "error: missing ${f}"
		exit 1
	fi
done

"${SCRIPT_DIR}/extended-test-setup.sh"

run_agent() {
	"${AGENT}" --print --trust --workspace "${REPO}" --yolo "$1"
}

parallel_step_judges() {
	local run_dir="$1"
	local flows=(upload review collections rejected delete packages)
	local pids=()
	for flow in "${flows[@]}"; do
		(
			run_agent "photo-tool extended step UX judge (flow=${flow}). Read ${STEP_SPEC}. Run dir: ${run_dir}. Judge each step in flow ${flow} using ui-real/ PNGs only. Write verdicts/steps/<step_id>.md; last line STEP_UX_RESULT=pass or fail."
		) &
		pids+=($!)
	done
	for pid in "${pids[@]}"; do
		wait "${pid}" || true
	done
}

parallel_flow_judges() {
	local run_dir="$1"
	local flows=(upload review collections rejected share_desktop)
	local pids=()
	for flow in "${flows[@]}"; do
		(
			run_agent "photo-tool extended flow UX judge (flow=${flow}). Read ${FLOW_SPEC}. Run dir: ${run_dir}. Write ${run_dir}/verdicts/flows/${flow}.md ending FLOW_UX_RESULT=pass or fail."
		) &
		pids+=($!)
	done
	for pid in "${pids[@]}"; do
		wait "${pid}" || true
	done
}

parallel_fix_agents() {
	local run_dir="$1"
	local issues_dir="${run_dir}/issues"
	[[ -d "${issues_dir}" ]] || return 0
	mapfile -t issue_files < <(find "${issues_dir}" -name 'ISSUE-*.json' | sort)
	if [[ ${#issue_files[@]} -eq 0 ]]; then
		return 0
	fi
	local pids=()
	local n=0
	for jf in "${issue_files[@]}"; do
		while [[ $(jobs -rp | wc -l) -ge ${PARALLEL} ]]; do
			wait -n 2>/dev/null || wait || true
		done
		(
			run_agent "photo-tool extended fix from issue. Read ${FIX_SPEC}. Repo: ${REPO}. Issue: ${jf}. Run dir: ${run_dir}. Minimal fix + affected go test."
		) &
		pids+=($!)
		n=$((n + 1))
	done
	for pid in "${pids[@]}"; do
		wait "${pid}" || true
	done
	log "fix agents dispatched: ${n}"
}

all_green() {
	local run_dir="$1"
	local summary="${run_dir}/summary.md"
	[[ -f "${summary}" ]] || return 1
	grep -q 'EXTENDED_RESULT=pass' "${summary}" 2>/dev/null
}

round=0
while true; do
	round=$((round + 1))
	if [[ "${round}" -gt "${MAX}" ]]; then
		log "extended-test-loop: exceeded EXTENDED_LOOP_MAX=${MAX}"
		exit 1
	fi
	log "extended-test-loop: round ${round}/${MAX}"

	STAMP="$(date -u +%Y%m%dT%H%M%SZ)-$(git -C "${REPO}" rev-parse --short HEAD 2>/dev/null || echo local)"
	RUN_DIR="${REPO}/_bmad-output/test-artifacts/extended-runs/${STAMP}"
	mkdir -p "${RUN_DIR}"

	"${SCRIPT_DIR}/extended-test-matrix-gen.sh" -out "${RUN_DIR}"
	set +e
	"${SCRIPT_DIR}/extended-test-run.sh" --all --out "${RUN_DIR}" --judges
	run_rc=$?
	set -e

	parallel_step_judges "${RUN_DIR}"
	parallel_flow_judges "${RUN_DIR}"

	queue_out="$(cd "${REPO}" && go run ./tests/extended/cmd/build-issues -out "${RUN_DIR}" 2>&1)" || true
	log "${queue_out}"

	if echo "${queue_out}" | grep -q 'EXTENDED_ISSUE_QUEUE=empty' && [[ "${run_rc}" -eq 0 ]]; then
		if all_green "${RUN_DIR}"; then
			log "extended-test-loop: EXTENDED_RESULT=pass"
			echo "EXTENDED_RESULT=pass"
			exit 0
		fi
	fi

	if echo "${queue_out}" | grep -q 'EXTENDED_ISSUE_QUEUE=open'; then
		parallel_fix_agents "${RUN_DIR}"
		set +e
		"${SCRIPT_DIR}/extended-test-run.sh" --failed-only --out "${RUN_DIR}"
		set -e
		continue
	fi

	if [[ "${run_rc}" -eq 0 ]] && all_green "${RUN_DIR}"; then
		log "extended-test-loop: EXTENDED_RESULT=pass"
		echo "EXTENDED_RESULT=pass"
		exit 0
	fi

	log "extended-test-loop: round ${round} incomplete; continuing"
done
