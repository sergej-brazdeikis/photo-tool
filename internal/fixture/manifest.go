package fixture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Manifest describes a seeded library for scale runs.
type Manifest struct {
	Tier         string   `json:"tier"`
	Assets       int      `json:"assets"`
	Albums       int      `json:"albums"`
	Tags         int      `json:"tags"`
	Rejected     int      `json:"rejected"`
	UploadSeeds  []string `json:"upload_seeds,omitempty"`
	LibraryRoot  string   `json:"library_root"`
	GeneratedAt  string   `json:"generated_at"`
	GenerationMS int64    `json:"generation_ms"`
	DiskBytes    int64    `json:"disk_bytes,omitempty"`
}

// WriteManifest writes fixture-manifest.json under dir.
func WriteManifest(dir string, m Manifest) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if m.GeneratedAt == "" {
		m.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "fixture-manifest.json"), data, 0o644)
}

// ReadManifest loads fixture-manifest.json from dir when present.
func ReadManifest(dir string) (Manifest, bool, error) {
	data, err := os.ReadFile(filepath.Join(dir, "fixture-manifest.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return Manifest{}, false, nil
		}
		return Manifest{}, false, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, false, err
	}
	return m, true, nil
}

// LibrarySeededForTier reports whether dir already has a matching tier seed (skip re-insert).
func LibrarySeededForTier(dir string, tier Tier) bool {
	m, ok, err := ReadManifest(dir)
	if err != nil || !ok {
		return false
	}
	if m.Tier != string(tier) {
		return false
	}
	spec := TierSpec(tier)
	return m.Assets >= spec.Assets
}

// UploadSeedPaths returns upload JPEG paths from manifest or srcDir when seed was skipped.
func UploadSeedPaths(root, srcDir string) (uploadA, uploadB string, err error) {
	if m, ok, err := ReadManifest(root); err != nil {
		return "", "", err
	} else if ok && len(m.UploadSeeds) >= 2 {
		return m.UploadSeeds[0], m.UploadSeeds[1], nil
	} else if ok && len(m.UploadSeeds) == 1 {
		return m.UploadSeeds[0], m.UploadSeeds[0], nil
	}
	if srcDir == "" {
		srcDir = filepath.Join(root, ".fixture-src")
	}
	a := filepath.Join(srcDir, "ux_journey_new_a.jpg")
	b := filepath.Join(srcDir, "ux_journey_new_b.jpg")
	if _, err := os.Stat(a); err == nil {
		if _, err := os.Stat(b); err == nil {
			return a, b, nil
		}
		return a, a, nil
	}
	return "", "", nil
}
