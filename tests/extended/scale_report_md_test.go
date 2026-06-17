package extended

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateScaleScrollSettle_budget(t *testing.T) {
	m := stepsManifest{
		Steps: []struct {
			ID       string `json:"id"`
			File     string `json:"file"`
			TimingMS int64  `json:"timing_ms"`
		}{
			{ID: "review_grid_page2_top", TimingMS: 1500},
			{ID: "collections_list_scroll_50", TimingMS: 2500},
		},
	}
	if err := validateScaleScrollSettle("ui-real-scale", m); err == nil {
		t.Fatal("expected fail for timing_ms > 2000")
	}
	m.Steps[1].TimingMS = 800
	if err := validateScaleScrollSettle("ui-real-scale", m); err != nil {
		t.Fatal(err)
	}
}

func TestValidateScaleScrollSettle_ignoresNonScaleDir(t *testing.T) {
	m := stepsManifest{
		Steps: []struct {
			ID       string `json:"id"`
			File     string `json:"file"`
			TimingMS int64  `json:"timing_ms"`
		}{
			{ID: "review_grid_page2_top", TimingMS: 9000},
		},
	}
	if err := validateScaleScrollSettle("ui-real", m); err != nil {
		t.Fatal(err)
	}
}

func TestScaleTraceabilitySummary_readsRunContext(t *testing.T) {
	dir := t.TempDir()
	ctxDir := filepath.Join(dir, "context")
	if err := os.MkdirAll(ctxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "| Story | Status |\n|-------|--------|\n| 1.1 | PASS |\n| 2.4 | PARTIAL |\n| 3.1 | GAP |\n"
	if err := os.WriteFile(filepath.Join(ctxDir, "requirements-trace.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	s := scaleTraceabilitySummary(dir)
	for _, p := range []string{"PASS", "PARTIAL", "GAP", "context/requirements-trace.md"} {
		if !strings.Contains(s, p) {
			t.Fatalf("summary missing %q: %q", p, s)
		}
	}
}
