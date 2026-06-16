#!/usr/bin/env bash
# Run extended matrix layers: functional, software capture, real-app capture, optional judges.
# Phased with preflight — stops on first hard failure when EXTENDED_STOP_ON_FAIL=1 (default).
set -euo pipefail

log() { printf '%s\n' "$*" >&2; }

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export REPO
# shellcheck source=extended-test-env.sh
source "${REPO}/scripts/extended-test-env.sh"

OUT=""
LAYER="all"
FLOW=""
FAILED_ONLY=0
RUN_JUDGES=0
STOP_ON_FAIL="${EXTENDED_STOP_ON_FAIL:-1}"

while [[ $# -gt 0 ]]; do
	case "$1" in
	--out)
		OUT="$2"
		shift 2
		;;
	--layer=*)
		LAYER="${1#*=}"
		shift
		;;
	--flow=*)
		FLOW="${1#*=}"
		shift
		;;
	--all)
		LAYER="all"
		shift
		;;
	--failed-only)
		FAILED_ONLY=1
		shift
		;;
	--judges)
		RUN_JUDGES=1
		shift
		;;
	--no-stop-on-fail)
		STOP_ON_FAIL=0
		shift
		;;
	*)
		log "unknown arg: $1"
		exit 2
		;;
	esac
done

fail_or_continue() {
	local phase="$1"
	local rc="$2"
	if [[ "${rc}" -ne 0 ]]; then
		log "error: phase failed: ${phase} (exit ${rc})"
		if [[ "${STOP_ON_FAIL}" == "1" ]]; then
			write_summary "${rc}"
			exit "${rc}"
		fi
	fi
	return "${rc}"
}

preflight_go() {
	if [[ -z "${GO_BIN}" ]] || ! command -v go >/dev/null 2>&1; then
		log "error: go not on PATH — run ./scripts/extended-test-setup.sh"
		return 1
	fi
	return 0
}

shell_env() {
	printf "export PATH=%q" "${PATH}"
	[[ -n "${PKG_CONFIG_PATH:-}" ]] && printf " PKG_CONFIG_PATH=%q" "${PKG_CONFIG_PATH}"
	[[ -n "${CGO_CFLAGS:-}" ]] && printf " CGO_CFLAGS=%q" "${CGO_CFLAGS}"
	[[ -n "${CGO_LDFLAGS:-}" ]] && printf " CGO_LDFLAGS=%q" "${CGO_LDFLAGS}"
}

if [[ -z "${OUT}" ]]; then
	STAMP="$(date -u +%Y%m%dT%H%M%SZ)-$(git -C "${REPO}" rev-parse --short HEAD 2>/dev/null || echo local)"
	OUT="${REPO}/_bmad-output/test-artifacts/extended-runs/${STAMP}"
fi
mkdir -p "${OUT}/logs" "${OUT}/ui-software" "${OUT}/ui-real" "${OUT}/context" "${OUT}/verdicts/steps" "${OUT}/verdicts/flows" "${OUT}/issues"
cd "${REPO}"

preflight_go || exit 1

if [[ ! -f "${OUT}/matrix.json" ]]; then
	"${REPO}/scripts/extended-test-matrix-gen.sh" -out "${OUT}"
fi

RUBRIC="${REPO}/_bmad-output/test-artifacts/e2e-usability-rubric.md"
TRACE="${REPO}/_bmad-output/test-artifacts/bundle-requirements-trace.md"
[[ -f "${RUBRIC}" ]] && cp "${RUBRIC}" "${OUT}/context/rubric.md"
[[ -f "${TRACE}" ]] && cp "${TRACE}" "${OUT}/context/requirements-trace.md"

SHELL_EXPORTS="$(shell_env)"

run_go_test() {
	local logfile="$1"
	shift
	set +e
	"$@" >"${logfile}" 2>&1
	local rc=$?
	set -e
	return "${rc}"
}

run_functional() {
	log ">>> functional layer"
	# PHOTO_TOOL_GUI_E2E_LINUX=1 activates tests/gui_linux — run those in capture phase only.
	local saved_gui="${PHOTO_TOOL_GUI_E2E_LINUX:-}"
	unset PHOTO_TOOL_GUI_E2E_LINUX
	local pkgs_ci pkgs_all
	pkgs_ci="$(go list -tags ci ./... | grep -v '/tests/gui_linux$' | tr '\n' ' ')"
	pkgs_all="$(go list ./... | grep -v '/tests/gui_linux$' | tr '\n' ' ')"
	local groups=(
		"epic1_foundation:go test ./internal/config ./internal/store ./internal/exifmeta ./internal/filehash ./internal/paths ./internal/ingest -count=1"
		"epic1_cli:go test ./internal/cli ./tests/e2e -count=1"
		"epic1_upload:go test ./internal/app -run 'Upload|Drop|FR06|UX_upload|Classify' -count=1"
		"epic2_review:go test ./internal/app -run 'Review|Rejected|Collection|Story210|Story212|Reject' -count=1"
		"epic2_layout:go test ./internal/app -run 'NFR01|E2E_shell|ux_layout|UXDR' -count=1"
		"epic3_share:go test ./internal/share -count=1"
		"epic4_packages:go test ./internal/app -run 'Package' -count=1"
		"root_ci:go test -tags ci ${pkgs_ci} -count=1"
	)
	local rc=0
	for entry in "${groups[@]}"; do
		local name="${entry%%:*}"
		local cmd="${entry#*:}"
		log "  ${name}"
		if ! run_go_test "${OUT}/logs/${name}.txt" bash -lc "${SHELL_EXPORTS}; cd '${REPO}' && ${cmd}"; then
			rc=1
			if [[ "${STOP_ON_FAIL}" == "1" ]]; then
				log "  stopping at first functional failure: ${name} (see ${OUT}/logs/${name}.txt)"
				return "${rc}"
			fi
		fi
	done
	run_go_test "${OUT}/logs/go-test.txt" bash -lc "${SHELL_EXPORTS}; cd '${REPO}' && go test ${pkgs_all} -count=1" || rc=1
	run_go_test "${OUT}/logs/go-test-ci.txt" bash -lc "${SHELL_EXPORTS}; cd '${REPO}' && go test -tags ci ${pkgs_ci} -count=1" || rc=1
	run_go_test "${OUT}/logs/go-test-e2e.txt" bash -lc "${SHELL_EXPORTS}; cd '${REPO}' && go test ./tests/e2e/... -count=1" || rc=1
	[[ -n "${saved_gui}" ]] && export PHOTO_TOOL_GUI_E2E_LINUX="${saved_gui}"
	return "${rc}"
}

run_software_capture() {
	log ">>> software-driver capture (ui-software/) — Tier A, not used for step_ux sign-off"
	PHOTO_TOOL_UX_JOURNEY_TEST=1 \
		PHOTO_TOOL_UX_CAPTURE_DIR="${OUT}/ui-software" \
		PHOTO_TOOL_UX_CAPTURE_SOFTWARE_SUBDIR="" \
		go test ./internal/app -run TestUXJourneyCapture -count=1 -timeout=10m \
		>"${OUT}/logs/ux-capture-software.txt" 2>&1
}

run_real_capture() {
	log ">>> real-binary capture (ui-real/) — Tier B, authoritative for step_ux/flow_ux"
	local lib="${OUT}/library"
	mkdir -p "${OUT}/bin" "${lib}"
	log "  go build -o ${OUT}/bin/photo-tool ."
	if ! go build -o "${OUT}/bin/photo-tool" .; then
		log "error: go build failed"
		return 1
	fi
	if [[ ! -x "${OUT}/bin/photo-tool" ]]; then
		log "error: missing executable ${OUT}/bin/photo-tool"
		return 1
	fi
	local runner=(env
		PHOTO_TOOL_LIBRARY="${lib}"
		PHOTO_TOOL_UX_JOURNEY_TEST=1
		PHOTO_TOOL_UX_CAPTURE_DIR="${OUT}/ui-real"
		PHOTO_TOOL_GUI_E2E_LINUX=1
		PHOTO_TOOL_UX_CAPTURE_APP_MODE=real_binary
	)
	if [[ -n "${FLOW}" ]]; then
		runner+=(PHOTO_TOOL_UX_CAPTURE_FLOWS="${FLOW}")
	fi
	local bin="${OUT}/bin/photo-tool"
	set +e
	if [[ "${EXTENDED_USE_XVFB:-}" == "1" ]] || [[ -z "${DISPLAY:-}" && -z "${WAYLAND_DISPLAY:-}" ]]; then
		if ! command -v xvfb-run >/dev/null 2>&1; then
			log "error: need DISPLAY or xvfb-run for real GUI capture"
			set -e
			return 1
		fi
		xvfb-run -a "${runner[@]}" "${bin}" >"${OUT}/logs/ux-capture-real.txt" 2>&1
	else
		"${runner[@]}" "${bin}" >"${OUT}/logs/ux-capture-real.txt" 2>&1
	fi
	local rc=$?
	set -e
	if [[ "${rc}" -ne 0 ]]; then
		log "error: real-binary journey exited ${rc} — see ${OUT}/logs/ux-capture-real.txt"
		tail -n 20 "${OUT}/logs/ux-capture-real.txt" >&2 || true
		return "${rc}"
	fi
	if ! go run ./tests/extended/cmd/validate-ux-bundle -out "${OUT}"; then
		log "error: ui-real bundle validation failed"
		return 1
	fi
	return 0
}

run_gui_linux_smoke() {
	log ">>> gui_linux smoke (real binary journey)"
	if [[ -z "${DISPLAY:-}" && -z "${WAYLAND_DISPLAY:-}" ]]; then
		log "  skip: no DISPLAY"
		return 0
	fi
	PHOTO_TOOL_GUI_E2E_LINUX=1 \
		go test ./tests/gui_linux/... -run 'TestLinuxGUIE2E_journeyRealAppCapture' -count=1 -timeout=5m \
		>"${OUT}/logs/gui-linux-journey.txt" 2>&1
}

validate_before_judges() {
	log ">>> validate UX judge inputs (reject software-driver tier)"
	if ! go run ./tests/extended/cmd/validate-ux-bundle -out "${OUT}"; then
		log "error: cannot run step_ux/flow_ux judges without valid ui-real/ real_binary captures"
		return 1
	fi
	return 0
}

run_step_judges() {
	local spec="${REPO}/_bmad-output/test-artifacts/judge-prompt-step-ux.md"
	[[ -f "${spec}" ]] || { log "missing ${spec}"; return 1; }
	for flow in upload review collections rejected delete packages; do
		if [[ -n "${FLOW}" && "${FLOW}" != "${flow}" ]]; then
			continue
		fi
		log ">>> step judge flow=${flow} (ui-real/ only)"
		"${EXTENDED_AGENT_BIN}" --print --trust --workspace "${REPO}" --yolo \
			"photo-tool extended step UX judge. Read ${spec}. Run dir: ${OUT}. Flow: ${flow}. Use ONLY ui-real/ PNGs where steps.json app_mode is real_binary. Reject ui-software/ with STEP_UX_RESULT=fail and rationale software_driver_not_valid_for_ux_signoff. Write verdicts/steps/<step_id>.md; last line STEP_UX_RESULT=pass or fail." \
			|| true
	done
}

run_flow_judges() {
	local spec="${REPO}/_bmad-output/test-artifacts/judge-prompt-flow-ux.md"
	for flow in upload review collections rejected share_desktop; do
		log ">>> flow judge flow=${flow}"
		"${EXTENDED_AGENT_BIN}" --print --trust --workspace "${REPO}" --yolo \
			"photo-tool extended flow UX judge. Read ${spec}. Run dir: ${OUT}. Flow: ${flow}. Use ONLY ui-real/ captures. Write ${OUT}/verdicts/flows/${flow}.md ending FLOW_UX_RESULT=pass or fail." \
			|| true
	done
}

write_summary() {
	local rc="$1"
	python3 - "${OUT}" "${rc}" <<'PY'
import json, os, sys, datetime
out, rc = sys.argv[1], int(sys.argv[2])
summary = {
  "finished_at": datetime.datetime.utcnow().isoformat() + "Z",
  "exit_code": rc,
  "machine_line": "EXTENDED_RESULT=pass" if rc == 0 else "EXTENDED_RESULT=fail",
}
with open(os.path.join(out, "summary.json"), "w") as f:
    json.dump(summary, f, indent=2)
with open(os.path.join(out, "summary.md"), "w") as f:
    f.write("# Extended run summary\n\nExit: %d\n\n```\n%s\n```\n" % (rc, summary["machine_line"]))
print(summary["machine_line"])
PY
}

overall_rc=0
REAL_CAPTURE_OK=0

if [[ "${FAILED_ONLY}" -eq 1 ]]; then
	if go run ./tests/extended/cmd/validate-ux-bundle -out "${OUT}" 2>/dev/null; then
		REAL_CAPTURE_OK=1
		log "resume: existing ui-real/ bundle valid"
	fi
fi

case "${LAYER}" in
all | functional)
	if [[ "${FAILED_ONLY}" -eq 1 && "${LAYER}" == "all" ]]; then
		log ">>> skip functional (failed-only resume)"
	elif ! run_functional; then
		overall_rc=1
		fail_or_continue "functional" "${overall_rc}" || true
	fi
	;;
esac

case "${LAYER}" in
all | step_ux | capture)
	if [[ "${FAILED_ONLY}" -eq 0 ]]; then
		if run_software_capture; then
			log "software capture ok (Tier A)"
		else
			log "warning: software capture failed (non-blocking)"
		fi
		if run_real_capture; then
			REAL_CAPTURE_OK=1
		else
			overall_rc=1
			fail_or_continue "real-binary-capture" 1 || true
		fi
		if [[ "${REAL_CAPTURE_OK}" -eq 1 ]]; then
			run_gui_linux_smoke || log "warning: gui_linux smoke failed (see logs/gui-linux-journey.txt)"
		fi
	fi
	;;
esac

want_judges=0
if [[ "${RUN_JUDGES}" -eq 1 || "${LAYER}" == "step_ux" || "${LAYER}" == "flow_ux" || "${LAYER}" == "all" ]]; then
	want_judges=1
fi

if [[ "${want_judges}" -eq 1 ]]; then
	if [[ "${REAL_CAPTURE_OK}" -eq 1 ]] && validate_before_judges; then
		run_step_judges || overall_rc=1
		run_flow_judges || overall_rc=1
		issue_out="$(go run ./tests/extended/cmd/build-issues -out "${OUT}" 2>&1)" || true
		log "${issue_out}"
		if echo "${issue_out}" | grep -q 'EXTENDED_ISSUE_QUEUE=open'; then
			overall_rc=1
		fi
	else
		log "error: skipping step_ux/flow_ux judges — real-binary capture or validation failed"
		overall_rc=1
	fi
fi

write_summary "${overall_rc}"
log "Run directory: ${OUT}"
exit "${overall_rc}"
