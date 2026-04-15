package app

import (
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"fyne.io/fyne/v2/test"

	"photo-tool/internal/config"
	"photo-tool/internal/store"
)

func TestCollectionsView_zeroAlbums_emptyStateHidesList(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	test.NewTempApp(t)
	w := test.NewTempWindow(t, nil)

	v := NewCollectionsView(w, db, root, nil, nil)
	if !strings.Contains(v.listMsg.Text, "No albums yet") {
		t.Fatalf("listMsg: %q", v.listMsg.Text)
	}
	if !v.list.Hidden {
		t.Fatal("expected album list hidden when library has zero collections (AC7)")
	}
}

// Story 2.12: empty album detail exposes Back to albums (primary) and Go to Review.
func TestStory212_CollectionsDetail_emptyAlbum_CTAs(t *testing.T) {
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

	cid, err := store.CreateCollection(db, "EmptyAlbum", "")
	if err != nil {
		t.Fatal(err)
	}

	w := test.NewTempWindow(t, nil)
	var reviewCalls int32
	v := NewCollectionsView(w, db, root, func() { atomic.AddInt32(&reviewCalls, 1) }, nil)
	v.openDetail(cid, "EmptyAlbum")

	rootObj := v.CanvasObject()
	findButtonByText(t, rootObj, "Back to albums")
	reviewBtn := findButtonByText(t, rootObj, "Go to Review")
	test.Tap(reviewBtn)
	if atomic.LoadInt32(&reviewCalls) != 1 {
		t.Fatalf("onGotoReview calls = %d want 1", reviewCalls)
	}
}
