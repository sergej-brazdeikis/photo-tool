package extended

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type stepsManifest struct {
	AppMode string `json:"app_mode"`
	Steps   []struct {
		ID   string `json:"id"`
		File string `json:"file"`
	} `json:"steps"`
}

// ValidateUXJudgeInputs ensures step_ux / flow_ux judges may run (Tier B only).
func ValidateUXJudgeInputs(runDir string) error {
	realDir := filepath.Join(runDir, "ui-real")
	stepsPath := filepath.Join(realDir, "steps.json")
	data, err := os.ReadFile(stepsPath)
	if err != nil {
		return fmt.Errorf("ui-real/steps.json: %w", err)
	}
	var m stepsManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("parse steps.json: %w", err)
	}
	if m.AppMode != "real_binary" {
		return fmt.Errorf("steps.json app_mode=%q; step_ux/flow_ux require real_binary (reject software_driver)", m.AppMode)
	}
	if len(m.Steps) == 0 {
		return fmt.Errorf("steps.json has no steps")
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
	// Reject if judges would accidentally use Tier A manifest.
	swPath := filepath.Join(runDir, "ui-software", "steps.json")
	if swData, err := os.ReadFile(swPath); err == nil {
		var sw stepsManifest
		if json.Unmarshal(swData, &sw) == nil && sw.AppMode == "software_driver" {
			// OK: software tier exists but is not used for UX sign-off.
			_ = sw
		}
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
	realDir := filepath.Join(runDir, "ui-real")
	stepsPath := filepath.Join(realDir, "steps.json")
	data, err := os.ReadFile(stepsPath)
	if err != nil {
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
	// review_grid must not match late upload idle frame (stale GL buffer signature from prior runs).
	if rg, ok := byID["review_grid_default_filters"]; ok {
		if up, ok := byID["upload_after_confirm_idle"]; ok {
			rh, _ := fileMD5(filepath.Join(realDir, rg))
			uh, _ := fileMD5(filepath.Join(realDir, up))
			if rh == uh {
				return fmt.Errorf("capture regression: review_grid_default_filters duplicates upload_after_confirm_idle")
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
