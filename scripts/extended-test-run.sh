#!/usr/bin/env bash
# Run extended matrix layers: functional, software capture, real-app capture, optional judges.
# Phased with preflight — stops on first hard failure when EXTENDED_STOP_ON_FAIL=1 (default).
set -euo pipefail

log() { printf '%s\n' "$*" >&2; }

PHOTO_TOOL_CAPTURE_TIMEOUT="${PHOTO_TOOL_UX_CAPTURE_TIMEOUT:-20m}"
PHOTO_TOOL_CAPTURE_CPU_WARN="${PHOTO_TOOL_UX_CAPTURE_CPU_WARN:-95}"

ux_journey_pids() {
	local pid
	for pid in $(pgrep -f 'photo-tool' 2>/dev/null || true); do
		if [[ -r "/proc/${pid}/environ" ]] && tr '\0' '\n' </proc/"${pid}"/environ 2>/dev/null | grep -q '^PHOTO_TOOL_UX_JOURNEY_TEST=1'; then
			echo "${pid}"
		fi
	done
}

kill_photo_tool_orphans() {
	local killed=0 pid
	for pid in $(ux_journey_pids); do
		log "cleanup: terminating UX journey photo-tool pid=${pid}"
		kill -TERM "${pid}" 2>/dev/null || true
		killed=1
	done
	if [[ "${killed}" -eq 1 ]]; then
		sleep 2
		for pid in $(ux_journey_pids); do
			kill -KILL "${pid}" 2>/dev/null || true
		done
	fi
}

ensure_no_photo_tool_orphans() {
	local n=0 pid
	for pid in $(ux_journey_pids); do
		n=$((n + 1))
	done
	if [[ "${n}" -gt 0 ]]; then
		log "warning: ${n} UX journey photo-tool process(es) still running — cleaning up before next capture"
		kill_photo_tool_orphans
	fi
}

run_photo_tool_monitored() {
	local label="$1"
	local logfile="$2"
	shift 2
	ensure_no_photo_tool_orphans
	local -a timeout_cmd=()
	if command -v timeout >/dev/null 2>&1; then
		timeout_cmd=(timeout --foreground --kill-after=30s "${PHOTO_TOOL_CAPTURE_TIMEOUT}")
	fi
	set +e
	"${timeout_cmd[@]}" "$@" >"${logfile}" 2>&1 &
	local pid=$!
	set -e
	local start elapsed cpu high_streak=0
	start=$(date +%s)
	while kill -0 "${pid}" 2>/dev/null; do
		sleep 5
		if ! kill -0 "${pid}" 2>/dev/null; then
			break
		fi
		elapsed=$(( $(date +%s) - start ))
		cpu="$(ps -p "${pid}" -o %cpu= 2>/dev/null | tr -d ' ' || echo 0)"
		log "  [monitor] ${label} pid=${pid} elapsed=${elapsed}s cpu=${cpu}%"
		if awk -v c="${cpu}" 'BEGIN { exit !(c + 0 >= '"${PHOTO_TOOL_CAPTURE_CPU_WARN}"') }'; then
			high_streak=$((high_streak + 1))
			if (( high_streak >= 12 )); then
				log "error: ${label} pid=${pid} sustained >=${PHOTO_TOOL_CAPTURE_CPU_WARN}% CPU — killing"
				kill -TERM "${pid}" 2>/dev/null || true
				sleep 3
				kill -KILL "${pid}" 2>/dev/null || true
				wait "${pid}" 2>/dev/null || true
				kill_photo_tool_orphans
				return 137
			fi
		else
			high_streak=0
		fi
	done
	set +e
	wait "${pid}"
	local rc=$?
	set -e
	if kill -0 "${pid}" 2>/dev/null; then
		log "error: ${label} pid=${pid} still alive after wait — force kill"
		kill -KILL "${pid}" 2>/dev/null || true
		rc=137
	fi
	ensure_no_photo_tool_orphans
	if [[ "${rc}" -eq 124 || "${rc}" -eq 137 ]]; then
		log "error: ${label} timed out or was killed (exit ${rc})"
	fi
	return "${rc}"
}

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
mkdir -p "${OUT}/logs" "${OUT}/ui-software" "${OUT}/ui-real" "${OUT}/ui-real-scale" "${OUT}/ui-real-edge" "${OUT}/ui-real-layout" "${OUT}/context" "${OUT}/verdicts/steps" "${OUT}/verdicts/flows" "${OUT}/issues"
trap 'kill_photo_tool_orphans' EXIT INT TERM
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
	local rc=0
	if [[ "${EXTENDED_USE_XVFB:-}" == "1" ]] || [[ -z "${DISPLAY:-}" && -z "${WAYLAND_DISPLAY:-}" ]]; then
		if ! command -v xvfb-run >/dev/null 2>&1; then
			log "error: need DISPLAY or xvfb-run for real GUI capture"
			return 1
		fi
		run_photo_tool_monitored "ui-real" "${OUT}/logs/ux-capture-real.txt" \
			xvfb-run -a "${runner[@]}" "${bin}" || rc=$?
	else
		run_photo_tool_monitored "ui-real" "${OUT}/logs/ux-capture-real.txt" \
			"${runner[@]}" "${bin}" || rc=$?
	fi
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
		go test ./tests/gui_linux/... -run 'TestLinuxGUIE2E_journeyRealAppCapture|TestLinuxGUIE2E_scaleSpotCapture' -count=1 -timeout=5m \
		>"${OUT}/logs/gui-linux-journey.txt" 2>&1
}

run_scale_unit() {
	log ">>> scale_unit layer (go test -short skips heavy rows)"
	local rc=0
	run_go_test "${OUT}/logs/scale_unit.txt" bash -lc "${SHELL_EXPORTS}; cd '${REPO}' && go test ./tests/fixture/... -run TestScale_ -count=1 -short" || rc=1
	return "${rc}"
}

run_scale_functional() {
	log ">>> scale_functional layer (NFR-02, share package volume, ingest)"
	local rc=0
	run_go_test "${OUT}/logs/scale_nfr02.txt" bash -lc "${SHELL_EXPORTS}; cd '${REPO}' && go test ./tests/e2e/... -run TestScale_Scan_dryRun10k -count=1 -timeout=30m" || rc=1
	run_go_test "${OUT}/logs/scale_cli_deep.txt" bash -lc "${SHELL_EXPORTS}; cd '${REPO}' && go test ./tests/e2e/... -run TestScale_CLI_recursiveDeepTree -count=1" || rc=1
	run_go_test "${OUT}/logs/scale_share_http.txt" bash -lc "${SHELL_EXPORTS}; cd '${REPO}' && go test ./internal/share/... -run 'TestShareHTTP_scalePackage|TestShareHTTP_rateLimit_burst80' -count=1" || rc=1
	run_go_test "${OUT}/logs/scale_share_reject.txt" bash -lc "${SHELL_EXPORTS}; cd '${REPO}' && go test ./internal/share/... -run TestShareHTTP_scalePackageRejectExcluded -count=1" || rc=1
	run_go_test "${OUT}/logs/scale_ingest.txt" bash -lc "${SHELL_EXPORTS}; cd '${REPO}' && go test ./tests/fixture/... -run 'TestScale_Ingest|TestScale_Import' -count=1" || rc=1
	run_go_test "${OUT}/logs/scale_app.txt" bash -lc "${SHELL_EXPORTS}; cd '${REPO}' && go test ./internal/app/... -run TestScale_ -count=1" || rc=1
	return "${rc}"
}

run_real_journey_capture() {
	local capture_dir="$1"
	local flow="$2"
	local tier="${3:-S1}"
	local log_name="$4"
	log ">>> real-binary capture (${capture_dir}/) flow=${flow} tier=${tier}"
	local lib="${OUT}/library"
	mkdir -p "${OUT}/bin" "${lib}"
	if [[ ! -x "${OUT}/bin/photo-tool" ]]; then
		if ! go build -o "${OUT}/bin/photo-tool" .; then
			return 1
		fi
	fi
	rm -rf "${lib}/.phototool" "${lib}/.cache" "${lib}/2026" 2>/dev/null || true
	go run ./tests/extended/cmd/seed-library -out "${lib}" -tier "${tier}" -src-dir "${lib}/.fixture-src" \
		>"${OUT}/logs/seed-${tier}.txt" 2>&1 || return 1
	local runner=(env
		PHOTO_TOOL_LIBRARY="${lib}"
		PHOTO_TOOL_UX_JOURNEY_TEST=1
		PHOTO_TOOL_UX_CAPTURE_DIR="${OUT}/${capture_dir}"
		PHOTO_TOOL_UX_FIXTURE_SCALE="${tier}"
		PHOTO_TOOL_GUI_E2E_LINUX=1
		PHOTO_TOOL_UX_CAPTURE_APP_MODE=real_binary
		PHOTO_TOOL_UX_CAPTURE_FLOWS="${flow}"
	)
	local bin="${OUT}/bin/photo-tool"
	local rc=0
	if [[ "${EXTENDED_USE_XVFB:-}" == "1" ]] || [[ -z "${DISPLAY:-}" && -z "${WAYLAND_DISPLAY:-}" ]]; then
		if ! command -v xvfb-run >/dev/null 2>&1; then
			log "error: need DISPLAY or xvfb-run for real GUI capture"
			return 1
		fi
		run_photo_tool_monitored "${capture_dir}" "${OUT}/logs/${log_name}.txt" \
			xvfb-run -a "${runner[@]}" "${bin}" || rc=$?
	else
		run_photo_tool_monitored "${capture_dir}" "${OUT}/logs/${log_name}.txt" \
			"${runner[@]}" "${bin}" || rc=$?
	fi
	if [[ "${rc}" -ne 0 ]]; then
		log "error: capture ${capture_dir} exited ${rc}"
		tail -n 15 "${OUT}/logs/${log_name}.txt" >&2 || true
		return "${rc}"
	fi
	go run ./tests/extended/cmd/validate-ux-bundle -out "${OUT}" -dir "${capture_dir}" || return 1
	return 0
}

run_scale_ux() {
	local tier="${PHOTO_TOOL_SCALE_TIER:-S5}"
	local rc=0
	run_real_journey_capture "ui-real-scale" "scale_spot" "${tier}" "ux-capture-scale" || rc=1
	run_real_journey_capture "ui-real-edge" "edge" "S4" "ux-capture-edge" || rc=1
	run_real_journey_capture "ui-real-edge-s6" "edge" "S6" "ux-capture-edge-s6" || true
	run_real_journey_capture "ui-real-edge-s0" "edge" "S0" "ux-capture-edge-s0" || true
	run_real_journey_capture "ui-real-edge-s5r" "edge" "S5R" "ux-capture-edge-s5r" || true
	run_real_journey_capture "ui-real-layout" "layout" "S4" "ux-capture-layout" || rc=1
	go run ./tests/extended/cmd/validate-ux-bundle -out "${OUT}" -dir all || rc=1
	return "${rc}"
}

run_scale_step_judges() {
	local spec="${REPO}/_bmad-output/test-artifacts/judge-prompt-scale-ux.md"
	[[ -f "${spec}" ]] || { log "missing ${spec}"; return 1; }
	local subdir
	for subdir in "${OUT}"/ui-real*; do
		[[ -d "${subdir}" ]] || continue
		[[ -f "${subdir}/steps.json" ]] || continue
		case "$(basename "${subdir}")" in
		ui-real) continue ;; # baseline journey uses step_ux judges
		esac
		log ">>> scale step judge ($(basename "${subdir}")/)"
		"${EXTENDED_AGENT_BIN}" --print --trust --workspace "${REPO}" --yolo \
			"photo-tool scale UX judge. Read ${spec}. Run dir: ${OUT}. Use ONLY $(basename "${subdir}")/ PNGs where steps.json app_mode is real_binary. Write verdicts/steps/<step_id>.md; last line STEP_UX_RESULT=pass or fail." \
			|| true
	done
}

validate_scale_before_judges() {
	log ">>> validate scale UX judge inputs (ui-real-scale/edge/layout)"
	if ! go run ./tests/extended/cmd/validate-ux-bundle -out "${OUT}" -dir all; then
		log "error: scale UX validation failed"
		return 1
	fi
	return 0
}

is_scale_only_layer() {
	case "${LAYER}" in
	scale | scale_ux | scale_unit | scale_functional) return 0 ;;
	*) return 1 ;;
	esac
}

run_scale_report() {
	log ">>> scale report"
	local out
	out="$(go run ./tests/extended/cmd/build-scale-report -out "${OUT}" 2>&1)" || true
	log "${out}"
	log "Scale report: ${OUT}/scale-report.html"
	log "Scale report (json): ${OUT}/scale-report.json"
	if echo "${out}" | grep -q 'EXTENDED_SCALE_RESULT=fail'; then
		return 1
	fi
	return 0
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

run_scale_flow_judges() {
	local spec="${REPO}/_bmad-output/test-artifacts/judge-prompt-scale-ux.md"
	[[ -f "${spec}" ]] || { log "missing ${spec}"; return 1; }
	for flow in scale_spot edge layout; do
		log ">>> scale flow judge flow=${flow}"
		"${EXTENDED_AGENT_BIN}" --print --trust --workspace "${REPO}" --yolo \
			"photo-tool scale flow UX judge. Read ${spec}. Run dir: ${OUT}. Flow: ${flow}. Roll up all steps in ui-real-scale/, ui-real-edge/, ui-real-layout/ for this flow. Write ${OUT}/verdicts/flows/${flow}.md ending FLOW_UX_RESULT=pass or fail." \
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

case "${LAYER}" in
all | scale | scale_unit)
	if [[ "${LAYER}" == "all" || "${LAYER}" == "scale" || "${LAYER}" == "scale_unit" ]]; then
		if [[ "${FAILED_ONLY}" -eq 0 ]]; then
			run_scale_unit || overall_rc=1
		fi
	fi
	;;
esac

case "${LAYER}" in
all | scale | scale_functional)
	if [[ "${FAILED_ONLY}" -eq 0 ]]; then
		run_scale_functional || overall_rc=1
	fi
	;;
esac

case "${LAYER}" in
all | scale_ux | scale)
	if [[ "${FAILED_ONLY}" -eq 0 ]]; then
		if run_scale_ux; then
			REAL_CAPTURE_OK=1
		else
			overall_rc=1
		fi
	fi
	;;
esac

if [[ "${want_judges}" -eq 1 ]]; then
	if [[ "${LAYER}" == "all" || "${LAYER}" == "scale" || "${LAYER}" == "scale_ux" ]]; then
		if [[ "${REAL_CAPTURE_OK}" -eq 1 ]] && validate_scale_before_judges; then
			run_scale_step_judges || overall_rc=1
			run_scale_flow_judges || overall_rc=1
		elif [[ "${LAYER}" == "all" || "${LAYER}" == "scale" || "${LAYER}" == "scale_ux" ]]; then
			log "error: skipping scale UX judges — capture or validation failed"
			overall_rc=1
		fi
	fi
	if [[ "${LAYER}" == "all" || "${LAYER}" == "step_ux" || "${LAYER}" == "flow_ux" ]]; then
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
	elif [[ "${RUN_JUDGES}" -eq 1 ]] && is_scale_only_layer; then
		log "scale layer: baseline ui-real/ judges not required"
	fi
fi

if [[ "${LAYER}" == "all" || "${LAYER}" == "scale" || "${LAYER}" == "scale_ux" ]]; then
	run_scale_report || overall_rc=1
fi

write_summary "${overall_rc}"
log "Run directory: ${OUT}"
exit "${overall_rc}"
