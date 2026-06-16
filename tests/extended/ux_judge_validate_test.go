package extended_test

import (
	"os"
	"path/filepath"
	"testing"

	"photo-tool/tests/extended"
)

func TestValidateUXJudgeInputs_rejectsSoftwareDriverAppMode(t *testing.T) {
	dir := t.TempDir()
	realDir := filepath.Join(dir, "ui-real")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realDir, "steps.json"), []byte(`{"app_mode":"software_driver","steps":[{"id":"x","file":"01_x.png"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realDir, "01_x.png"), make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := extended.ValidateUXJudgeInputs(dir); err == nil {
		t.Fatal("expected error for software_driver app_mode")
	}
}

func TestValidateUXJudgeInputs_acceptsRealBinary(t *testing.T) {
	dir := t.TempDir()
	realDir := filepath.Join(dir, "ui-real")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realDir, "steps.json"), []byte(`{"app_mode":"real_binary","steps":[{"id":"upload_empty","file":"01_upload_empty.png"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realDir, "01_upload_empty.png"), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := extended.ValidateUXJudgeInputs(dir); err != nil {
		t.Fatal(err)
	}
}

func TestValidateCaptureDistinct_rejectsDuplicateReview(t *testing.T) {
	dir := t.TempDir()
	realDir := filepath.Join(dir, "ui-real")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	png := make([]byte, 2048)
	if err := os.WriteFile(filepath.Join(realDir, "01_upload_empty.png"), png, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realDir, "02_review_grid_default_filters.png"), png, 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{"app_mode":"real_binary","steps":[{"id":"upload_empty","file":"01_upload_empty.png"},{"id":"review_grid_default_filters","file":"02_review_grid_default_filters.png"}]}`
	if err := os.WriteFile(filepath.Join(realDir, "steps.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := extended.ValidateCaptureDistinct(dir); err == nil {
		t.Fatal("expected duplicate capture error")
	}
}
