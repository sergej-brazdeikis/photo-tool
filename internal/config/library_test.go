package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveLibraryRoot_envOverride(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv(EnvLibraryRoot, tmp)
	root, err := ResolveLibraryRoot()
	if err != nil {
		t.Fatal(err)
	}
	absTmp, _ := filepath.Abs(tmp)
	if root != absTmp {
		t.Fatalf("got %q want %q", root, absTmp)
	}
}

func TestEnsureLibraryLayout_createsDirs(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{".phototool", ".trash", ".cache/thumbnails"} {
		st, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		if !st.IsDir() {
			t.Fatalf("%s: not a directory", rel)
		}
	}
}
