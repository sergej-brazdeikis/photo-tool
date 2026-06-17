package extended

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type stepsManifest struct {
	AppMode string `json:"app_mode"`
	Steps   []struct {
		ID       string `json:"id"`
		File     string `json:"file"`
		TimingMS int64  `json:"timing_ms"`
	} `json:"steps"`
}

// ValidateUXJudgeInputs ensures step_ux / flow_ux judges may run (Tier B only) for ui-real/.
func ValidateUXJudgeInputs(runDir string) error {
	return ValidateUXCaptureSubdir(runDir, "ui-real")
}

// ValidateUXCaptureSubdir validates a capture subdirectory (ui-real, ui-real-scale, ui-real-edge).
func ValidateUXCaptureSubdir(runDir, subdir string) error {
	realDir := filepath.Join(runDir, subdir)
	stepsPath := filepath.Join(realDir, "steps.json")
	data, err := os.ReadFile(stepsPath)
	if err != nil {
		return fmt.Errorf("%s/steps.json: %w", subdir, err)
	}
	var m stepsManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("parse %s/steps.json: %w", subdir, err)
	}
	if m.AppMode != "real_binary" {
		return fmt.Errorf("%s steps.json app_mode=%q; require real_binary (reject software_driver)", subdir, m.AppMode)
	}
	if len(m.Steps) == 0 {
		return fmt.Errorf("%s/steps.json has no steps", subdir)
	}
	for _, s := range m.Steps {
		if strings.Contains(s.File, "ui-software") {
			return fmt.Errorf("step %q file %q references ui-software; UX judges reject software-driver PNGs", s.ID, s.File)
		}
		png := filepath.Join(realDir, s.File)
		st, err := os.Stat(png)
		if err != nil {
			return fmt.Errorf("step %q missing PNG %s: %w", s.ID, s.File, err)
		}
		if st.Size() < 512 {
			return fmt.Errorf("step %q PNG %s too small (%d bytes)", s.ID, s.File, st.Size())
		}
	}
	if err := validateScaleScrollSettle(subdir, m); err != nil {
		return err
	}
	if m.AppMode == "real_binary" {
		if err := validateCaptureNotBlank(realDir, m); err != nil {
			return err
		}
	}
	return nil
}

// validateCaptureNotBlank rejects real-binary PNGs that are uniform near-black (unpainted GL frames).
func validateCaptureNotBlank(realDir string, m stepsManifest) error {
	for _, s := range m.Steps {
		pngPath := filepath.Join(realDir, s.File)
		f, err := os.Open(pngPath)
		if err != nil {
			return fmt.Errorf("step %q open PNG %s: %w", s.ID, s.File, err)
		}
		img, _, err := image.Decode(f)
		_ = f.Close()
		if err != nil {
			return fmt.Errorf("step %q decode PNG %s: %w", s.ID, s.File, err)
		}
		if captureLooksBlank(img) {
			return fmt.Errorf("step %q PNG %s is blank/unpainted GL frame", s.ID, s.File)
		}
	}
	return nil
}

const scaleScrollSettleMS = 2000

// validateScaleScrollSettle gates scroll/paging captures that record timing_ms (Part C).
func validateScaleScrollSettle(subdir string, m stepsManifest) error {
	if subdir != "ui-real-scale" && !strings.HasPrefix(subdir, "ui-real-edge") && subdir != "ui-real-layout" {
		return nil
	}
	for _, s := range m.Steps {
		if s.TimingMS <= 0 {
			continue
		}
		if !strings.Contains(s.ID, "page2") && !strings.Contains(s.ID, "scroll") {
			continue
		}
		if s.TimingMS > scaleScrollSettleMS {
			return fmt.Errorf("step %q timing_ms=%d exceeds scroll settle budget %dms", s.ID, s.TimingMS, scaleScrollSettleMS)
		}
	}
	return nil
}

// ValidateAllRealCaptureDirs validates every ui-real* directory present under runDir.
func ValidateAllRealCaptureDirs(runDir string) error {
	entries, err := os.ReadDir(runDir)
	if err != nil {
		return err
	}
	var validated int
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "ui-real") {
			continue
		}
		if _, err := os.Stat(filepath.Join(runDir, e.Name(), "steps.json")); err != nil {
			continue
		}
		if err := ValidateUXCaptureSubdir(runDir, e.Name()); err != nil {
			return err
		}
		validated++
	}
	if validated == 0 {
		return fmt.Errorf("no ui-real* capture dirs with steps.json under %s", runDir)
	}
	return nil
}

// mustDistinctFromUpload lists steps that must not byte-match upload_empty in real-binary capture.
var mustDistinctFromUpload = []string{
	"review_grid_default_filters",
	"collections_album_list",
	"rejected_hidden_grid",
	"review_loupe",
}

// ValidateCaptureDistinct rejects real-binary runs where key steps duplicate other steps (GLX stale capture).
func ValidateCaptureDistinct(runDir string) error {
	if _, err := os.Stat(filepath.Join(runDir, "ui-real", "steps.json")); err == nil {
		if err := validateCaptureDistinctSubdir(runDir, "ui-real"); err != nil {
			return err
		}
	}
	if err := validateScaleCaptureDistinct(runDir); err != nil {
		return err
	}
	return validateEdgeCaptureDistinct(runDir)
}

func validateCaptureDistinctSubdir(runDir, subdir string) error {
	realDir := filepath.Join(runDir, subdir)
	stepsPath := filepath.Join(realDir, "steps.json")
	data, err := os.ReadFile(stepsPath)
	if err != nil {
		if subdir != "ui-real" {
			return nil
		}
		return err
	}
	var m stepsManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	if m.AppMode != "real_binary" {
		return nil
	}
	byID := map[string]string{}
	for _, s := range m.Steps {
		byID[s.ID] = s.File
	}
	if subdir == "ui-real" {
		uploadFile, ok := byID["upload_empty"]
		if !ok {
			return fmt.Errorf("steps.json missing upload_empty")
		}
		uploadHash, err := fileMD5(filepath.Join(realDir, uploadFile))
		if err != nil {
			return err
		}
		var dupes []string
		for _, id := range mustDistinctFromUpload {
			f, ok := byID[id]
			if !ok {
				continue
			}
			h, err := fileMD5(filepath.Join(realDir, f))
			if err != nil {
				return err
			}
			if h == uploadHash {
				dupes = append(dupes, id)
			}
		}
		if len(dupes) > 0 {
			return fmt.Errorf("capture regression: real-binary PNGs duplicate upload_empty for steps: %s", strings.Join(dupes, ", "))
		}
		if rg, ok := byID["review_grid_default_filters"]; ok {
			if up, ok := byID["upload_after_confirm_idle"]; ok {
				rh, _ := fileMD5(filepath.Join(realDir, rg))
				uh, _ := fileMD5(filepath.Join(realDir, up))
				if rh == uh {
					return fmt.Errorf("capture regression: review_grid_default_filters duplicates upload_after_confirm_idle")
				}
			}
		}
	}
	return nil
}

func validateScaleCaptureDistinct(runDir string) error {
	realDir := filepath.Join(runDir, "ui-real-scale")
	stepsPath := filepath.Join(realDir, "steps.json")
	data, err := os.ReadFile(stepsPath)
	if err != nil {
		return nil
	}
	var m stepsManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	if m.AppMode != "real_binary" {
		return nil
	}
	byID := map[string]string{}
	for _, s := range m.Steps {
		byID[s.ID] = s.File
	}
	p1, ok1 := byID["review_grid_page1_full"]
	p2, ok2 := byID["review_grid_page2_top"]
	if ok1 && ok2 {
		h1, _ := fileMD5(filepath.Join(realDir, p1))
		h2, _ := fileMD5(filepath.Join(realDir, p2))
		if h1 == h2 {
			return fmt.Errorf("scale capture regression: review_grid_page2_top duplicates review_grid_page1_full")
		}
	}
	// Partial-filter steps must not duplicate zero-match empty captures.
	emptyIDs := []string{"review_filter_zero_5star", "review_filter_empty_edge"}
	partialIDs := []string{"review_filter_collection_album", "review_filter_tag_uxcaptag"}
	for _, partial := range partialIDs {
		pf, okP := byID[partial]
		if !okP {
			continue
		}
		ph, _ := fileMD5(filepath.Join(realDir, pf))
		for _, empty := range emptyIDs {
			ef, okE := byID[empty]
			if !okE {
				continue
			}
			eh, _ := fileMD5(filepath.Join(realDir, ef))
			if ph == eh {
				return fmt.Errorf("scale capture regression: %s duplicates empty filter step %s", partial, empty)
			}
		}
	}
	return nil
}

func validateEdgeCaptureDistinct(runDir string) error {
	for _, sub := range []string{"ui-real-edge", "ui-real-edge-s6"} {
		realDir := filepath.Join(runDir, sub)
		stepsPath := filepath.Join(realDir, "steps.json")
		data, err := os.ReadFile(stepsPath)
		if err != nil {
			continue
		}
		var m stepsManifest
		if err := json.Unmarshal(data, &m); err != nil {
			return err
		}
		byID := map[string]string{}
		for _, s := range m.Steps {
			byID[s.ID] = s.File
		}
		if f, ok := byID["review_package_blocked_501"]; ok {
			st, err := os.Stat(filepath.Join(realDir, f))
			if err != nil || st.Size() < 512 {
				return fmt.Errorf("edge capture: review_package_blocked_501 PNG missing or too small")
			}
		}
	}
	return nil
}

func fileMD5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	sum := md5.New()
	if _, err := io.Copy(sum, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}
