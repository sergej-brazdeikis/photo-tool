package filehash

import (
	"io"
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

func TestReaderHex_matchesSumHex(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "payload.bin")
	payload := []byte("ingest-style single read")
	if err := os.WriteFile(p, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	want, err := SumHex(p)
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got, err := ReaderHex(f)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("ReaderHex: got %q want %q (same as SumHex)", got, want)
	}
}

// Ingest opens the file once, hashes with ReaderHex, then seeks to start for copy.
func TestReaderHex_matchesSumHex_afterSeekFromEnd(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "seek.bin")
	if err := os.WriteFile(p, []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	want, err := SumHex(p)
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	got, err := ReaderHex(f)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("after seek rewind: got %q want %q", got, want)
	}
}
