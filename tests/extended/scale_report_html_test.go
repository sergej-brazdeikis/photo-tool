package extended

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteScaleReportHTML_embedsDataAndFilters(t *testing.T) {
	dir := t.TempDir()
	uxDir := filepath.Join(dir, "ui-real-scale")
	if err := os.MkdirAll(uxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	steps := `{"app_mode":"real_binary","fixture_tier":"S4","steps":[{"id":"review_grid_page1_full","flow":"scale_spot","file":"01_review_grid_page1_full.png","intent":"page1"}]}`
	if err := os.WriteFile(filepath.Join(uxDir, "steps.json"), []byte(steps), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uxDir, "01_review_grid_page1_full.png"), []byte(strings.Repeat("x", 600)), 0o644); err != nil {
		t.Fatal(err)
	}
	rep := ScaleReport{
		RunID:       "test-run",
		FinishedAt:  "2026-06-16T12:00:00Z",
		MachineLine: "EXTENDED_SCALE_RESULT=pass",
	}
	if err := WriteScaleReportHTML(dir, rep); err != nil {
		t.Fatal(err)
	}
	html, err := os.ReadFile(filepath.Join(dir, "scale-report.html"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(html)
	for _, want := range []string{
		"id=\"scale-report-data\"",
		"review_grid_page1_full",
		"id=\"q\"",
		"id=\"issues\"",
		"id=\"step-detail\"",
		"ui-real-scale/01_review_grid_page1_full.png",
		"Persona coverage",
		"id=\"functional\"",
		"id=\"edge-matrix\"",
		"id=\"reproduce\"",
		"make extended-test-scale",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("html missing %q", want)
		}
	}
}
