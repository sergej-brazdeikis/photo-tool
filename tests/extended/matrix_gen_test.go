package extended_test

import (
	"testing"

	"photo-tool/tests/extended"
)

func TestGenerateMatrix_uniqueIDs(t *testing.T) {
	m, err := extended.GenerateMatrix("test")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Rows) < 40 {
		t.Fatalf("expected at least 40 rows, got %d", len(m.Rows))
	}
	seen := map[string]struct{}{}
	for _, r := range m.Rows {
		if r.ID == "" {
			t.Fatal("empty row id")
		}
		if _, ok := seen[r.ID]; ok {
			t.Fatalf("duplicate id %q", r.ID)
		}
		seen[r.ID] = struct{}{}
	}
}

func TestAllStoriesPresent(t *testing.T) {
	m, err := extended.GenerateMatrix("test")
	if err != nil {
		t.Fatal(err)
	}
	if missing := extended.AllStoriesPresent(m); len(missing) != 0 {
		t.Fatalf("missing stories: %v", missing)
	}
}

func TestStepUXRowsRequireRealBinary(t *testing.T) {
	m, err := extended.GenerateMatrix("test")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range m.Rows {
		if r.Layer != extended.LayerStepUX && r.Layer != extended.LayerFlowUX {
			continue
		}
		if r.UxAppMode != extended.UxAppRealBinary {
			t.Fatalf("row %q ux_app_mode = %q, want real_binary", r.ID, r.UxAppMode)
		}
	}
}
