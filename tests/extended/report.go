package extended

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WriteMatrix writes matrix.json and matrix.md under dir.
func WriteMatrix(dir string, m *Matrix) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "matrix.json"), data, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "matrix.md"), renderMatrixMD(m), 0o644)
}

func renderMatrixMD(m *Matrix) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "# Extended testing matrix\n\nGenerated: %s\n\n", m.GeneratedAt)
	if m.GitShort != "" {
		fmt.Fprintf(&b, "Git: `%s`\n\n", m.GitShort)
	}
	b.WriteString("| Story | Flow | Step | Layer | FR | Status | ID |\n")
	b.WriteString("|-------|------|------|-------|----|--------|----|\n")
	for _, r := range m.Rows {
		fr := strings.Join(r.FR, ", ")
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s |\n",
			r.Story, r.Flow, r.Step, r.Layer, fr, r.Status, r.ID)
	}
	return []byte(b.String())
}

// Issue is a parallel-fix work item.
type Issue struct {
	ID            string   `json:"id"`
	Story         string   `json:"story,omitempty"`
	Flow          string   `json:"flow,omitempty"`
	Step          string   `json:"step,omitempty"`
	Layer         Layer    `json:"layer"`
	Status        string   `json:"status"`
	ParallelGroup string   `json:"parallel_group"`
	FixScopeFiles []string `json:"fix_scope_files,omitempty"`
	Evidence      []string `json:"evidence,omitempty"`
	Acceptance    string   `json:"acceptance,omitempty"`
	Summary       string   `json:"summary,omitempty"`
	MachineLine   string   `json:"machine_line"`
}

// WriteIssue writes ISSUE-*.json and .md under issuesDir.
func WriteIssue(issuesDir string, iss Issue) error {
	if err := os.MkdirAll(issuesDir, 0o755); err != nil {
		return err
	}
	base := filepath.Join(issuesDir, iss.ID)
	data, err := json.MarshalIndent(iss, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(base+".json", data, 0o644); err != nil {
		return err
	}
	var md strings.Builder
	fmt.Fprintf(&md, "# %s\n\n", iss.ID)
	fmt.Fprintf(&md, "**Layer:** %s  \n**Status:** %s  \n**Parallel group:** %s\n\n", iss.Layer, iss.Status, iss.ParallelGroup)
	if iss.Summary != "" {
		fmt.Fprintf(&md, "## Summary\n\n%s\n\n", iss.Summary)
	}
	if iss.Acceptance != "" {
		fmt.Fprintf(&md, "## Acceptance\n\n%s\n\n", iss.Acceptance)
	}
	if len(iss.Evidence) > 0 {
		md.WriteString("## Evidence\n\n")
		for _, e := range iss.Evidence {
			fmt.Fprintf(&md, "- `%s`\n", e)
		}
		md.WriteString("\n")
	}
	fmt.Fprintf(&md, "```\n%s\n```\n", iss.MachineLine)
	return os.WriteFile(base+".md", []byte(md.String()), 0o644)
}

// WriteIssuesREADME indexes open issues for humans and LLM fix agents.
func WriteIssuesREADME(issuesDir string, issues []Issue) error {
	var b strings.Builder
	b.WriteString("# Issue queue (parallel fix)\n\n")
	b.WriteString("| ID | Group | Layer | Flow | Step | Status |\n")
	b.WriteString("|----|-------|-------|------|------|--------|\n")
	for _, iss := range issues {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n",
			iss.ID, iss.ParallelGroup, iss.Layer, iss.Flow, iss.Step, iss.Status)
	}
	return os.WriteFile(filepath.Join(issuesDir, "README.md"), []byte(b.String()), 0o644)
}

// ScaleReport is the machine-readable scale run rollup.
type ScaleReport struct {
	RunID        string   `json:"run_id"`
	GitShort     string   `json:"git_short,omitempty"`
	FinishedAt   string   `json:"finished_at"`
	MachineLine  string   `json:"machine_line"`
	FixtureTiers []string `json:"fixture_tiers,omitempty"`
	Counts       struct {
		FunctionalPass int `json:"functional_pass"`
		FunctionalFail int `json:"functional_fail"`
		ScaleUnitPass  int `json:"scale_unit_pass"`
		ScaleUnitFail  int `json:"scale_unit_fail"`
		UXStepsPass    int `json:"ux_real_binary_steps_pass"`
		UXStepsFail    int `json:"ux_real_binary_steps_fail"`
		IssuesOpen     int `json:"issues_open"`
		UXCaptureDirs  int `json:"ux_capture_dirs"`
	} `json:"counts"`
	Personas        []PersonaRow `json:"personas,omitempty"`
	NFRGates        []NFRGateRow `json:"nfr_gates,omitempty"`
	Recommendations []string     `json:"recommendations,omitempty"`
}

// WriteScaleReport writes scale-report.json and scale-report.md under runDir.
func WriteScaleReport(runDir string) (ScaleReport, error) {
	rep := ScaleReport{
		FinishedAt:  time.Now().UTC().Format(time.RFC3339),
		MachineLine: "EXTENDED_SCALE_RESULT=pass",
	}
	if st, err := os.Stat(runDir); err == nil {
		_ = st
	}
	rep.RunID = filepath.Base(runDir)
	if data, err := os.ReadFile(filepath.Join(runDir, "summary.json")); err == nil {
		var sum struct {
			ExitCode int `json:"exit_code"`
		}
		if json.Unmarshal(data, &sum) == nil && sum.ExitCode != 0 {
			rep.MachineLine = "EXTENDED_SCALE_RESULT=fail"
		}
	}
	rep.Counts.FunctionalPass, rep.Counts.FunctionalFail = countLogResults(runDir, "epic")
	rep.Counts.ScaleUnitPass, rep.Counts.ScaleUnitFail = countLogResults(runDir, "scale")
	entries, _ := os.ReadDir(runDir)
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "ui-real") {
			if _, err := os.Stat(filepath.Join(runDir, e.Name(), "steps.json")); err == nil {
				rep.Counts.UXCaptureDirs++
			}
		}
	}
	if manifest, err := os.ReadFile(filepath.Join(runDir, "library", "fixture-manifest.json")); err == nil {
		var mf struct {
			Tier string `json:"tier"`
		}
		if json.Unmarshal(manifest, &mf) == nil && mf.Tier != "" {
			rep.FixtureTiers = append(rep.FixtureTiers, mf.Tier)
		}
	}
	issuesDir := filepath.Join(runDir, "issues")
	if entries, err := os.ReadDir(issuesDir); err == nil {
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), "ISSUE-") && strings.HasSuffix(e.Name(), ".json") {
				rep.Counts.IssuesOpen++
			}
		}
	}
	if rep.Counts.IssuesOpen > 0 {
		rep.MachineLine = "EXTENDED_SCALE_RESULT=fail"
	}
	rep.Counts.UXStepsPass, rep.Counts.UXStepsFail = countUXStepVerdicts(runDir)
	if rep.Counts.UXStepsFail > 0 || rep.Counts.ScaleUnitFail > 0 || rep.Counts.FunctionalFail > 0 {
		rep.MachineLine = "EXTENDED_SCALE_RESULT=fail"
	}
	rep.NFRGates = collectNFRGates(runDir, rep)
	rep.Personas = defaultPersonaRows(rep)
	rep.Recommendations = buildScaleRecommendations(rep, runDir)
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return rep, err
	}
	if err := os.WriteFile(filepath.Join(runDir, "scale-report.json"), data, 0o644); err != nil {
		return rep, err
	}
	var md strings.Builder
	md.WriteString("# Scale & UX limit test report\n\n")
	fmt.Fprintf(&md, "Result: `%s`\n\n", rep.MachineLine)
	fmt.Fprintf(&md, "Run: `%s` · Finished: %s\n\n", rep.RunID, rep.FinishedAt)
	md.WriteString("## Counts\n\n")
	fmt.Fprintf(&md, "| Metric | Pass | Fail |\n|--------|------|------|\n")
	fmt.Fprintf(&md, "| Functional | %d | %d |\n", rep.Counts.FunctionalPass, rep.Counts.FunctionalFail)
	fmt.Fprintf(&md, "| Scale unit | %d | %d |\n", rep.Counts.ScaleUnitPass, rep.Counts.ScaleUnitFail)
	fmt.Fprintf(&md, "| UX step judges | %d | %d |\n", rep.Counts.UXStepsPass, rep.Counts.UXStepsFail)
	fmt.Fprintf(&md, "\n- UX capture dirs: %d\n", rep.Counts.UXCaptureDirs)
	fmt.Fprintf(&md, "- Open issues: %d\n", rep.Counts.IssuesOpen)
	fmt.Fprintf(&md, "- Fixture tiers: %v\n\n", rep.FixtureTiers)
	if len(rep.NFRGates) > 0 {
		md.WriteString("## NFR gates\n\n| ID | Target | Result |\n|----|--------|--------|\n")
		for _, g := range rep.NFRGates {
			fmt.Fprintf(&md, "| %s | %s | %s |\n", g.ID, g.Target, g.Result)
		}
		md.WriteString("\n")
	}
	if len(rep.Personas) > 0 {
		md.WriteString("## Persona coverage\n\n| Persona | Journey | Tier | Result |\n|---------|---------|------|--------|\n")
		for _, p := range rep.Personas {
			fmt.Fprintf(&md, "| %s | %s | %s | %s |\n", p.ID, p.Journey, p.Tier, p.Result)
		}
		md.WriteString("\n")
	}
	steps := collectUXSteps(runDir, firstFixtureTier(rep))
	if len(steps) > 0 {
		md.WriteString("## UX capture volume\n\n| Flow | Steps | Pass | Fail | Unknown |\n|------|-------|------|------|--------|\n")
		type flowStat struct{ total, pass, fail, unknown int }
		byFlow := map[string]*flowStat{}
		for _, s := range steps {
			st := byFlow[s.Flow]
			if st == nil {
				st = &flowStat{}
				byFlow[s.Flow] = st
			}
			st.total++
			switch s.Judge {
			case "pass":
				st.pass++
			case "fail":
				st.fail++
			default:
				st.unknown++
			}
		}
		for flow, st := range byFlow {
			fmt.Fprintf(&md, "| %s | %d | %d | %d | %d |\n", flow, st.total, st.pass, st.fail, st.unknown)
		}
		md.WriteString("\n")
	}
	if trace := scaleTraceabilitySummary(runDir); trace != "" {
		md.WriteString("## Traceability\n\n")
		md.WriteString(trace)
		md.WriteString("\n\n")
	}
	if len(rep.Recommendations) > 0 {
		md.WriteString("## Recommendations\n\n")
		for _, r := range rep.Recommendations {
			fmt.Fprintf(&md, "- %s\n", r)
		}
		md.WriteString("\n")
	}
	md.WriteString("## Reproduce\n\n```\n")
	md.WriteString(scaleReproduceCommands(rep))
	md.WriteString("\n```\n\nOpen `scale-report.html` for interactive filters, persona click-filter, and step verdict excerpts.\n")
	_ = os.WriteFile(filepath.Join(runDir, "scale-report.md"), []byte(md.String()), 0o644)
	if err := WriteScaleReportHTML(runDir, rep); err != nil {
		return rep, err
	}
	return rep, nil
}

func countUXStepVerdicts(runDir string) (pass, fail int) {
	steps := collectUXSteps(runDir, "")
	for _, s := range steps {
		switch s.Judge {
		case "pass":
			pass++
		case "fail":
			fail++
		}
	}
	return pass, fail
}

func countLogResults(runDir, prefix string) (pass, fail int) {
	logsDir := filepath.Join(runDir, "logs")
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		return 0, 0
	}
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), prefix) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(logsDir, e.Name()))
		if err != nil {
			continue
		}
		body := string(data)
		if strings.Contains(body, "FAIL") || strings.Contains(body, "--- FAIL:") {
			fail++
		} else {
			pass++
		}
	}
	return pass, fail
}

func buildScaleRecommendations(rep ScaleReport, runDir string) []string {
	var recs []string
	if rep.Counts.FunctionalFail > 0 {
		recs = append(recs, fmt.Sprintf("Re-run failing scale functional logs (%d fail); see logs/scale_*.txt", rep.Counts.FunctionalFail))
	}
	if rep.Counts.ScaleUnitFail > 0 {
		recs = append(recs, fmt.Sprintf("Fix scale unit tests (%d fail) in tests/fixture and internal/app scale_test.go", rep.Counts.ScaleUnitFail))
	}
	if rep.Counts.UXStepsFail > 0 {
		recs = append(recs, fmt.Sprintf("Re-judge failing UX steps (%d fail): EXTENDED_STOP_ON_FAIL=0 ./scripts/extended-test-run.sh --layer=scale --judges", rep.Counts.UXStepsFail))
	}
	if rep.Counts.IssuesOpen > 0 {
		recs = append(recs, fmt.Sprintf("Triage %d open issues in issues/README.md", rep.Counts.IssuesOpen))
	}
	if rep.MachineLine == "EXTENDED_SCALE_RESULT=pass" && len(recs) == 0 {
		recs = append(recs, "Scale layer green — schedule periodic --layer=scale --judges on release candidates")
	}
	if strings.Contains(scaleTraceabilitySummary(runDir), "GAP") {
		recs = append(recs, "Close GAP rows in bundle-requirements-trace.md before release sign-off")
	}
	return recs
}

func scaleTraceabilitySummary(runDir string) string {
	for _, rel := range []string{
		filepath.Join("context", "requirements-trace.md"),
		filepath.Join("..", "..", "..", "_bmad-output", "test-artifacts", "bundle-requirements-trace.md"),
	} {
		data, err := os.ReadFile(filepath.Join(runDir, rel))
		if err != nil {
			continue
		}
		body := string(data)
		gap := strings.Count(body, "| GAP")
		partial := strings.Count(body, "| PARTIAL")
		pass := strings.Count(body, "| PASS")
		return fmt.Sprintf("Snapshot from `%s`: %d PASS, %d PARTIAL, %d GAP rows in requirements trace table.", filepath.ToSlash(rel), pass, partial, gap)
	}
	return ""
}
