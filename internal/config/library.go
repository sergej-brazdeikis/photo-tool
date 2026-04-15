package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const asciiWhitespaceCut = " \t\n\r\v\f"

func trimLeadingTrailingASCIIWhitespace(s string) string {
	return strings.Trim(s, asciiWhitespaceCut)
}

// EnvLibraryRoot is the environment variable overriding the default library location.
const EnvLibraryRoot = "PHOTO_TOOL_LIBRARY"

// ResolveLibraryRoot returns the absolute path to the photo library root.
// If PHOTO_TOOL_LIBRARY is set to a non-empty value after trimming leading/trailing ASCII whitespace, that path is made absolute. Empty, unset, or ASCII-whitespace-only
// values fall back to UserConfigDir()/photo-tool/library. (Unicode space characters such as NBSP are not trimmed.)
func ResolveLibraryRoot() (string, error) {
	if v := trimLeadingTrailingASCIIWhitespace(os.Getenv(EnvLibraryRoot)); v != "" {
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
	p := filepath.Join(cfg, "photo-tool", "library")
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("library root: %w", err)
	}
	return abs, nil
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
