package app

import (
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	"photo-tool/internal/config"
	"photo-tool/internal/domain"
	"photo-tool/internal/store"
)

// UX copy gate: initial Upload surface exposes drop zone, picker guidance, and neutral receipt placeholders.
func TestUX_upload_initialCopy_dropZoneAndReceiptPlaceholders(t *testing.T) {
	test.NewTempApp(t)

	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	win := test.NewTempWindow(t, nil)
	view := NewUploadView(win, db, root)
	for _, sub := range []string{"Drop images here", "Add one or more images"} {
		var ok bool
		for _, lb := range collectLabelsDeep(view) {
			if strings.Contains(lb.Text, sub) {
				ok = true
				break
			}
		}
		if !ok {
			t.Fatalf("no label contains %q", sub)
		}
	}
	var sawAddedDash bool
	for _, lb := range collectLabelsDeep(view) {
		if lb.Text == "Added: —" {
			sawAddedDash = true
			break
		}
	}
	if !sawAddedDash {
		t.Fatal(`want initial receipt label "Added: —"`)
	}
}

func TestSummarizeDoneMessage_skipCollection(t *testing.T) {
	sum := domain.OperationSummary{Added: 2, SkippedDuplicate: 0, Failed: 0}
	got := summarizeDoneMessage(sum, false, false)
	if !strings.Contains(got, "No new collection was created") {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(got, "linked") {
		t.Fatalf("unexpected link copy: %q", got)
	}
}

func TestSummarizeDoneMessage_linked(t *testing.T) {
	sum := domain.OperationSummary{Added: 1, SkippedDuplicate: 0, Failed: 0}
	got := summarizeDoneMessage(sum, true, true)
	if !strings.Contains(got, "linked to the new collection") {
		t.Fatalf("got %q", got)
	}
}

func TestSummarizeDoneMessage_wantedAssignButNoIngestedAssets(t *testing.T) {
	sum := domain.OperationSummary{Added: 0, SkippedDuplicate: 0, Failed: 3}
	got := summarizeDoneMessage(sum, true, false)
	if strings.Contains(got, "linked to the new collection") {
		t.Fatalf("must not claim linked: %q", got)
	}
	if !strings.Contains(got, "No collection was created") {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(got, "successfully ingested") {
		t.Fatalf("got %q", got)
	}
}

func TestSummarizeDoneMessage_failedIncludesNextStep(t *testing.T) {
	sum := domain.OperationSummary{Added: 1, SkippedDuplicate: 0, Failed: 2}
	got := summarizeDoneMessage(sum, false, false)
	if !strings.Contains(got, "failed 2") {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(got, "For items that failed") {
		t.Fatalf("missing next step: %q", got)
	}
}
