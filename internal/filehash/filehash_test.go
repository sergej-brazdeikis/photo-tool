package filehash

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSumHex_openErrorWraps(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "nope.dat")
	_, err := SumHex(missing)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want errors.Is(..., os.ErrNotExist); got %v", err)
	}
}

func TestSumHex_emptyFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "empty.dat")
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := SumHex(p)
	if err != nil {
		t.Fatal(err)
	}
	// echo -n | shasum -a 256
	const want = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

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

type errReader struct{ err error }

func (e errReader) Read(p []byte) (int, error) { return 0, e.err }

func TestReaderHex_readErrorWraps(t *testing.T) {
	wantErr := errors.New("boom")
	_, err := ReaderHex(errReader{err: wantErr})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("errors.Is: got %v", err)
	}
	if !strings.Contains(err.Error(), "filehash: read:") {
		t.Fatalf("want filehash read prefix, got %v", err)
	}
}
