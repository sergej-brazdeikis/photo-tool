package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveLibraryRoot_defaultIsAbsolute(t *testing.T) {
	t.Setenv(EnvLibraryRoot, "")
	root, err := ResolveLibraryRoot()
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(root) {
		t.Fatalf("default library root not absolute: %q", root)
	}
}

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

func TestResolveLibraryRoot_envWhitespaceFallsBackToDefault(t *testing.T) {
	t.Setenv(EnvLibraryRoot, "")
	defaultRoot, err := ResolveLibraryRoot()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvLibraryRoot, " \t ")
	got, err := ResolveLibraryRoot()
	if err != nil {
		t.Fatal(err)
	}
	if got != defaultRoot {
		t.Fatalf("ASCII whitespace-only env should fall back to default root\ngot %q\nwant %q", got, defaultRoot)
	}
}

func TestResolveLibraryRoot_envUnicodeSpaceNotTreatedAsASCIIWhitespace(t *testing.T) {
	t.Setenv(EnvLibraryRoot, "")
	def, err := ResolveLibraryRoot()
	if err != nil {
		t.Fatal(err)
	}
	// NBSP is not trimmed; value is non-empty so we must not silently fall back like " \t ".
	t.Setenv(EnvLibraryRoot, "\u00a0")
	got, err := ResolveLibraryRoot()
	if err != nil {
		return
	}
	if got == def {
		t.Fatalf("got default root %q; NBSP-only env must not be treated as unset", got)
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
