package extended

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BuildIssueQueue scans verdict files and functional log failures under runDir.
func BuildIssueQueue(runDir string) ([]Issue, error) {
	var issues []Issue
	n := 1

	add := func(iss Issue) {
		if iss.ID == "" {
			iss.ID = fmt.Sprintf("ISSUE-%03d", n)
		}
		n++
		if iss.MachineLine == "" {
			iss.MachineLine = "EXTENDED_ISSUE_RESULT=open"
		}
		if iss.Status == "" {
			iss.Status = "open"
		}
		issues = append(issues, iss)
	}

	_ = filepath.Walk(filepath.Join(runDir, "verdicts", "steps"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		ok, _ := ParseStepUXResult(path)
		if ok {
			return nil
		}
		step := strings.TrimSuffix(filepath.Base(path), ".md")
		add(Issue{
			Flow:          inferFlowFromStep(step),
			Step:          step,
			Layer:         LayerStepUX,
			ParallelGroup: inferParallelGroup(inferFlowFromStep(step)),
			Evidence:      []string{rel(runDir, path), "ui-real/steps.json"},
			Summary:       fmt.Sprintf("Step UX judge failed for %s", step),
			Acceptance:    fmt.Sprintf("STEP_UX_RESULT=pass in verdicts/steps/%s.md after re-run", step),
		})
		return nil
	})

	_ = filepath.Walk(filepath.Join(runDir, "verdicts", "flows"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		ok, _ := ParseFlowUXResult(path)
		if ok {
			return nil
		}
		flow := strings.TrimSuffix(filepath.Base(path), ".md")
		add(Issue{
			Flow:          flow,
			Layer:         LayerFlowUX,
			ParallelGroup: inferParallelGroup(flow),
			Evidence:      []string{rel(runDir, path), "ui-real/"},
			Summary:       fmt.Sprintf("Flow UX judge failed for %s", flow),
			Acceptance:    fmt.Sprintf("FLOW_UX_RESULT=pass in verdicts/flows/%s.md after re-run", flow),
		})
		return nil
	})

	for _, g := range []string{"epic1_foundation", "epic1_cli", "epic1_upload", "epic2_review", "epic2_layout", "epic3_share", "epic4_packages", "root_ci", "go-test", "go-test-ci", "go-test-e2e"} {
		logPath := filepath.Join(runDir, "logs", g+".txt")
		data, err := os.ReadFile(logPath)
		if err != nil {
			continue
		}
		body := string(data)
		if strings.Contains(body, "FAIL") || strings.Contains(body, "--- FAIL:") {
			add(Issue{
				Layer:         LayerFunctional,
				ParallelGroup: inferGroupFromLogName(g),
				Evidence:      []string{"logs/" + g + ".txt"},
				Summary:       fmt.Sprintf("Functional tests failed in %s", g),
				Acceptance:    fmt.Sprintf("logs/%s.txt shows no FAIL after re-run", g),
			})
		}
	}

	if err := os.MkdirAll(filepath.Join(runDir, "issues"), 0o755); err != nil {
		return nil, err
	}
	for _, iss := range issues {
		if err := WriteIssue(filepath.Join(runDir, "issues"), iss); err != nil {
			return nil, err
		}
	}
	if err := WriteIssuesREADME(filepath.Join(runDir, "issues"), issues); err != nil {
		return nil, err
	}
	return issues, nil
}

func rel(runDir, path string) string {
	r, err := filepath.Rel(runDir, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(r)
}

func inferFlowFromStep(step string) string {
	switch {
	case strings.HasPrefix(step, "upload"):
		return "upload"
	case strings.HasPrefix(step, "review"):
		return "review"
	case strings.HasPrefix(step, "collections"):
		return "collections"
	case strings.HasPrefix(step, "rejected"):
		return "rejected"
	case strings.HasPrefix(step, "delete"):
		return "delete"
	case strings.HasPrefix(step, "package"):
		return "packages"
	default:
		return "review"
	}
}

func inferParallelGroup(flow string) string {
	switch flow {
	case "upload", "cli", "foundation", "ingest":
		return "epic-1"
	case "review", "collections", "rejected", "delete":
		return "epic-2"
	case "share_desktop", "share_http":
		return "epic-3"
	case "packages":
		return "epic-4"
	default:
		return "root"
	}
}

func inferGroupFromLogName(name string) string {
	switch {
	case strings.HasPrefix(name, "epic1"):
		return "epic-1"
	case strings.HasPrefix(name, "epic2"):
		return "epic-2"
	case strings.HasPrefix(name, "epic3"):
		return "epic-3"
	case strings.HasPrefix(name, "epic4"):
		return "epic-4"
	default:
		return "root"
	}
}
