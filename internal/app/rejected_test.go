package app

import (
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"photo-tool/internal/config"
	"photo-tool/internal/store"
)

// Story 2.12 / UX-DR9: default Rejected scope with no hidden rows — distinct copy + Back to Review.
func TestStory212_Rejected_defaultEmpty_backToReviewCTA(t *testing.T) {
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

	var reviewCalls int32
	view := NewRejectedView(nil, db, root, func() { atomic.AddInt32(&reviewCalls, 1) })
	findLabelContaining(t, view, "Nothing is hidden")
	btn := findButtonByText(t, view, "Back to Review")
	if btn.Importance != widget.HighImportance {
		t.Fatalf("Back to Review importance = %v want High", btn.Importance)
	}
	test.NewTempWindow(t, view)
	test.Tap(btn)
	if atomic.LoadInt32(&reviewCalls) != 1 {
		t.Fatalf("onGotoReview calls = %d want 1", reviewCalls)
	}
}

// Story 2.12: filters exclude all hidden rows — Reset filters primary CTA.
func TestStory212_Rejected_filterEmpty_resetFiltersCTA(t *testing.T) {
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

	cidHas, err := store.CreateCollection(db, "HasPhotos", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateCollection(db, "NoPhotos", ""); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	if err := store.InsertAsset(db, "rej212", "2026/04/13/r.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	aid, ok, err := store.AssetIDByContentHash(db, "rej212")
	if err != nil || !ok {
		t.Fatalf("asset id: ok=%v err=%v", ok, err)
	}
	if err := store.LinkAssetsToCollection(db, cidHas, []int64{aid}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RejectAsset(db, aid, now); err != nil {
		t.Fatal(err)
	}

	view := NewRejectedView(nil, db, root, nil)
	sels := collectSelectWidgets(view)
	if len(sels) < 3 {
		t.Fatalf("expected filter strip selects, got %d", len(sels))
	}
	test.NewTempWindow(t, view)
	sels[0].SetSelected("NoPhotos")

	findLabelContaining(t, view, "No hidden photos match these filters")
	btn := findButtonByText(t, view, "Reset filters")
	if btn.Importance != widget.HighImportance {
		t.Fatalf("Reset filters importance = %v want High", btn.Importance)
	}
	test.Tap(btn)
	if g, w := sels[0].Selected, reviewCollectionSentinel; g != w {
		t.Fatalf("after reset, collection select: got %q want %q", g, w)
	}
}
