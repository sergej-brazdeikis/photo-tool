package extended_test

import (
	"image"
	"image/color"
	"image/png"
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

func writeTestPNG(t *testing.T, path string, lum uint8) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 512, 512))
	for y := 0; y < 512; y++ {
		for x := 0; x < 512; x++ {
			img.SetRGBA(x, y, color.RGBA{R: lum, G: lum, B: lum, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
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
	writeTestPNG(t, filepath.Join(realDir, "01_upload_empty.png"), 200)
	if err := extended.ValidateUXJudgeInputs(dir); err != nil {
		t.Fatal(err)
	}
}

func TestValidateCaptureNotBlank_rejectsBlackPNG(t *testing.T) {
	dir := t.TempDir()
	realDir := filepath.Join(dir, "ui-real-edge")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	black := image.NewRGBA(image.Rect(0, 0, 1280, 800))
	manifest := `{"app_mode":"real_binary","steps":[{"id":"review_grid_edge_baseline","file":"01_review_grid_edge_baseline.png"}]}`
	if err := os.WriteFile(filepath.Join(realDir, "steps.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(realDir, "01_review_grid_edge_baseline.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, black); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := extended.ValidateUXCaptureSubdir(dir, "ui-real-edge"); err == nil {
		t.Fatal("expected blank PNG rejection")
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
