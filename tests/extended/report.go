package extended

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	ID             string   `json:"id"`
	Story          string   `json:"story,omitempty"`
	Flow           string   `json:"flow,omitempty"`
	Step           string   `json:"step,omitempty"`
	Layer          Layer    `json:"layer"`
	Status         string   `json:"status"`
	ParallelGroup  string   `json:"parallel_group"`
	FixScopeFiles  []string `json:"fix_scope_files,omitempty"`
	Evidence       []string `json:"evidence,omitempty"`
	Acceptance     string   `json:"acceptance,omitempty"`
	Summary        string   `json:"summary,omitempty"`
	MachineLine    string   `json:"machine_line"`
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
