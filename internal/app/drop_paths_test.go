package app

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

func TestDropBlockedDialogInfo(t *testing.T) {
	t.Parallel()
	if _, _, blocked := dropBlockedDialogInfo(false, false); blocked {
		t.Fatal("expected not blocked")
	}
	if title, msg, blocked := dropBlockedDialogInfo(true, false); !blocked || title != "Finish collection step" || msg == "" {
		t.Fatalf("post-import: blocked=%v title=%q msg=%q", blocked, title, msg)
	}
	if title, msg, blocked := dropBlockedDialogInfo(false, true); !blocked || title != "Import in progress" || msg == "" {
		t.Fatalf("in-flight: blocked=%v title=%q msg=%q", blocked, title, msg)
	}
	// Post-import wins over import-in-flight (both should not be true in production).
	if title, _, blocked := dropBlockedDialogInfo(true, true); !blocked || title != "Finish collection step" {
		t.Fatalf("both flags: title=%q blocked=%v", title, blocked)
	}
}

func TestTryAddUniquePath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	a := filepath.Join(dir, "a.jpg")
	b := filepath.Join(dir, "b.jpg")
	paths := []string{a}
	if tryAddUniquePath(&paths, a) {
		t.Fatal("duplicate should not append")
	}
	if len(paths) != 1 {
		t.Fatalf("paths: %#v", paths)
	}
	if !tryAddUniquePath(&paths, b) {
		t.Fatal("expected new path to append")
	}
	if len(paths) != 2 {
		t.Fatalf("paths: %#v", paths)
	}
}

func TestTakePendingStringSlice(t *testing.T) {
	t.Parallel()
	var p []string
	if _, ok := takePendingStringSlice(&p); ok {
		t.Fatal("empty pending should not take")
	}
	p = []string{"a: bad", "b: bad"}
	lines, ok := takePendingStringSlice(&p)
	if !ok || len(lines) != 2 || lines[0] != "a: bad" {
		t.Fatalf("first take: ok=%v lines=%#v", ok, lines)
	}
	if p != nil {
		t.Fatalf("expected pending cleared, got %#v", p)
	}
	if _, ok := takePendingStringSlice(&p); ok {
		t.Fatal("second take should be empty")
	}
}

func TestRectContainsPoint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		pos  fyne.Position
		want bool
	}{
		{"inside", fyne.NewPos(5, 5), true},
		{"on_origin", fyne.NewPos(0, 0), true},
		{"past_right", fyne.NewPos(10, 5), false},
		{"past_bottom", fyne.NewPos(5, 10), false},
		{"negative_outside", fyne.NewPos(-1, 0), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rectContainsPoint(tt.pos, fyne.NewPos(0, 0), fyne.NewSize(10, 10))
			if got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}

func TestDropRejectReason(t *testing.T) {
	t.Parallel()
	if got := dropRejectReason(nil); got != "" {
		t.Fatalf("nil: %q", got)
	}
	if got := dropRejectReason(errors.New("nil URI")); got != "A dropped item could not be read." {
		t.Fatalf("got %q", got)
	}
	if got := dropRejectReason(errors.New("empty path")); got != "A dropped item had no file path." {
		t.Fatalf("got %q", got)
	}
	wantNonLocal := "That drop is not a file on this computer (for example a browser or app link). Save or export the image, then drop the saved file or use Add images…"
	if got := dropRejectReason(errors.New("not a local file (https)")); got != wantNonLocal {
		t.Fatalf("got %q", got)
	}
}

func TestDroppedSkipSummaryForDialog(t *testing.T) {
	t.Parallel()
	if got := droppedSkipSummaryForDialog(nil); got != "" {
		t.Fatalf("nil: %q", got)
	}
	short := []string{"a: unsupported type", "b: unsupported type"}
	if got := droppedSkipSummaryForDialog(short); got != "a: unsupported type\nb: unsupported type" {
		t.Fatalf("short: %q", got)
	}
	long := make([]string, 12)
	for i := range long {
		long[i] = "x: unsupported type"
	}
	got := droppedSkipSummaryForDialog(long)
	if !strings.Contains(got, "… and 4 more") || !strings.Contains(got, "Add images…") {
		t.Fatalf("long: %q", got)
	}
}

func TestURILocalPath(t *testing.T) {
	t.Parallel()
	tmp := filepath.Join(t.TempDir(), "x.jpg")
	if err := os.WriteFile(tmp, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	httpsURI, err := storage.ParseURI("https://example.com/x.jpg")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		uri     fyne.URI
		want    string
		wantErr bool
	}{
		{"nil", nil, "", true},
		{"https", httpsURI, "", true},
		{"file_uri", storage.NewFileURI(tmp), tmp, false},
		{"empty_path_file", testFileURI{scheme: "file", path: ""}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := uriLocalPath(tt.uri)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestClassifyDroppedURIs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	okFile := filepath.Join(dir, "a.jpg")
	if err := os.WriteFile(okFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	badExt := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(badExt, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	subDir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	httpsZ, err := storage.ParseURI("https://example.com/z")
	if err != nil {
		t.Fatal(err)
	}

	uris := []fyne.URI{
		storage.NewFileURI(okFile),
		storage.NewFileURI(badExt),
		storage.NewFileURI(subDir),
		httpsZ,
	}
	res := classifyDroppedURIs(uris, os.Stat)
	if len(res.Supported) != 1 || res.Supported[0] != okFile {
		t.Fatalf("supported: %#v", res.Supported)
	}
	if len(res.Unsupported) < 3 {
		t.Fatalf("expected several unsupported, got %#v", res.Unsupported)
	}
}

func TestClassifyDroppedURIs_statErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	okFile := filepath.Join(dir, "a.jpg")
	if err := os.WriteFile(okFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	statErr := func(string) (fs.FileInfo, error) { return nil, errors.New("permission denied") }
	res := classifyDroppedURIs([]fyne.URI{storage.NewFileURI(okFile)}, statErr)
	if len(res.Supported) != 0 {
		t.Fatalf("supported: %#v", res.Supported)
	}
	if len(res.Unsupported) != 1 || !strings.Contains(res.Unsupported[0], "could not be opened") {
		t.Fatalf("unsupported: %#v", res.Unsupported)
	}
}

func TestClassifyDroppedURIs_dedupesPaths(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	okFile := filepath.Join(dir, "a.jpg")
	if err := os.WriteFile(okFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	u := storage.NewFileURI(okFile)
	res := classifyDroppedURIs([]fyne.URI{u, u, u}, os.Stat)
	if len(res.Supported) != 1 || res.Supported[0] != okFile {
		t.Fatalf("supported: %#v", res.Supported)
	}
	if len(res.Unsupported) != 0 {
		t.Fatalf("unexpected unsupported: %#v", res.Unsupported)
	}
}

// testFileURI is a minimal fyne.URI for exercising uriLocalPath edge cases without deprecated storage.NewURI.
type testFileURI struct {
	scheme, path string
}

func (u testFileURI) String() string    { return u.scheme + ":" + u.path }
func (u testFileURI) Extension() string { return "" }
func (u testFileURI) Name() string      { return "" }
func (u testFileURI) MimeType() string  { return "" }
func (u testFileURI) Scheme() string    { return u.scheme }
func (u testFileURI) Authority() string { return "" }
func (u testFileURI) Path() string      { return u.path }
func (u testFileURI) Query() string     { return "" }
func (u testFileURI) Fragment() string  { return "" }
