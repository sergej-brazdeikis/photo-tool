package filehash

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSumHex(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.dat")
	if err := os.WriteFile(p, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := SumHex(p)
	if err != nil {
		t.Fatal(err)
	}
	// echo -n hello | shasum -a 256
	const want = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
