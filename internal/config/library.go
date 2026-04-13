package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnvLibraryRoot is the environment variable overriding the default library location.
const EnvLibraryRoot = "PHOTO_TOOL_LIBRARY"

// ResolveLibraryRoot returns the absolute path to the photo library root.
// If PHOTO_TOOL_LIBRARY is set, it must be a non-empty path (made absolute).
// Otherwise UserConfigDir()/photo-tool/library is used.
func ResolveLibraryRoot() (string, error) {
	if v := os.Getenv(EnvLibraryRoot); v != "" {
		abs, err := filepath.Abs(v)
		if err != nil {
			return "", fmt.Errorf("%s: %w", EnvLibraryRoot, err)
		}
		return abs, nil
	}
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config dir: %w", err)
	}
	return filepath.Join(cfg, "photo-tool", "library"), nil
}

// EnsureLibraryLayout creates the library root and standard subdirectories
// (.phototool metadata, .trash quarantine, .cache/thumbnails).
func EnsureLibraryLayout(root string) error {
	dirs := []string{
		root,
		filepath.Join(root, ".phototool"),
		filepath.Join(root, ".trash"),
		filepath.Join(root, ".cache", "thumbnails"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("mkdir %q: %w", d, err)
		}
	}
	return nil
}
