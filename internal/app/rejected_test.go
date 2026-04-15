package app

import (
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"photo-tool/internal/config"
	"photo-tool/internal/store"
)

func hiddenAssetsLabelText(t *testing.T, view fyne.CanvasObject) string {
	t.Helper()
	for _, lb := range collectLabels(view) {
		if strings.HasPrefix(lb.Text, "Hidden assets:") {
			return lb.Text
		}
	}
	t.Fatal("no Hidden assets label in view")
	return ""
}

// Story 2.2 / party dev 2/2: Rejected reloads collections each refresh; dead DB shows collection degradation before tag degradation (strip order).
func TestRejectedView_closedDB_degradedSuffix_ordersCollectionsBeforeTags(t *testing.T) {
	test.NewTempApp(t)

	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	view := NewRejectedView(nil, db, root, nil)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	sels := collectSelectWidgets(view)
	if len(sels) < 2 {
		t.Fatalf("expected filter selects, got %d", len(sels))
	}
	sels[1].SetSelected("1")
	sels[1].SetSelected(reviewRatingAny)

	got := hiddenAssetsLabelText(t, view)
	iCol := strings.Index(got, "collections unavailable")
	iTag := strings.Index(got, "tags unavailable")
	if iCol < 0 || iTag < 0 {
		t.Fatalf("want both degraded hints in label: %q", got)
	}
	if iCol >= iTag {
		t.Fatalf("collections hint should precede tags hint: %q", got)
	}
	// Story 2.3 party dev 2/2: count-query failure must not trap the user — escape hatch stays visible.
	btn := findButtonByText(t, view, "Back to Review")
	if !btn.Visible() {
		t.Fatal("expected Back to Review visible after count failure")
	}
	if btn.Importance != widget.MediumImportance {
		t.Fatalf("Back to Review importance = %v want Medium", btn.Importance)
	}
}

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
