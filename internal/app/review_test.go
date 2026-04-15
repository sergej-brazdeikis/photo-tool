package app

import (
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"photo-tool/internal/config"
	"photo-tool/internal/store"
)

func TestReviewFilterStripSegmentLabels_order(t *testing.T) {
	want := []string{"Collection", "Minimum rating", "Tags"}
	if got := ReviewFilterStripSegmentLabels(); !reflect.DeepEqual(got, want) {
		t.Fatalf("ReviewFilterStripSegmentLabels: got %q want %q", got, want)
	}
}

// Story 2.2 AC1/AC2 default option strings (MVP English literals; update story if copy changes).
func TestReviewFilterStrip_defaultSentinels_matchStory22(t *testing.T) {
	if reviewCollectionSentinel != "No assigned collection" {
		t.Fatalf("collection sentinel: got %q", reviewCollectionSentinel)
	}
	if reviewRatingAny != "Any rating" {
		t.Fatalf("rating any: got %q", reviewRatingAny)
	}
	if reviewTagAny != "Any tag" {
		t.Fatalf("tag any: got %q", reviewTagAny)
	}
}

func collectLabels(o fyne.CanvasObject) []*widget.Label {
	var out []*widget.Label
	var walk func(fyne.CanvasObject)
	walk = func(x fyne.CanvasObject) {
		if x == nil {
			return
		}
		switch v := x.(type) {
		case *widget.Label:
			out = append(out, v)
		case *fyne.Container:
			for _, ch := range v.Objects {
				walk(ch)
			}
		}
	}
	walk(o)
	return out
}

func matchingAssetsLabelText(t *testing.T, view fyne.CanvasObject) string {
	t.Helper()
	for _, lb := range collectLabels(view) {
		if strings.HasPrefix(lb.Text, "Matching assets:") {
			return lb.Text
		}
	}
	t.Fatal("no Matching assets label in view")
	return ""
}

func collectSelectWidgets(o fyne.CanvasObject) []*widget.Select {
	var out []*widget.Select
	var walk func(fyne.CanvasObject)
	walk = func(x fyne.CanvasObject) {
		if x == nil {
			return
		}
		switch v := x.(type) {
		case *widget.Select:
			out = append(out, v)
		case *container.Scroll:
			walk(v.Content)
		case *fyne.Container:
			for _, ch := range v.Objects {
				walk(ch)
			}
		}
	}
	walk(o)
	return out
}

func collectButtons(o fyne.CanvasObject) []*widget.Button {
	var out []*widget.Button
	var walk func(fyne.CanvasObject)
	walk = func(x fyne.CanvasObject) {
		if x == nil {
			return
		}
		switch v := x.(type) {
		case *widget.Button:
			out = append(out, v)
		case *container.Scroll:
			walk(v.Content)
		case *fyne.Container:
			for _, ch := range v.Objects {
				walk(ch)
			}
		}
	}
	walk(o)
	return out
}

func findButtonByText(t *testing.T, root fyne.CanvasObject, want string) *widget.Button {
	t.Helper()
	for _, b := range collectButtons(root) {
		if b.Text == want {
			return b
		}
	}
	t.Fatalf("button %q not found", want)
	return nil
}

func findLabelByText(t *testing.T, root fyne.CanvasObject, want string) *widget.Label {
	t.Helper()
	for _, lb := range collectLabels(root) {
		if lb.Text == want {
			return lb
		}
	}
	t.Fatalf("label %q not found", want)
	return nil
}

// Story 2.2 AC4: Tab order through the filter strip follows visual order (three strip Selects).
// Story 2.10 adds a separate assign-target Select under the bulk tag row (not part of the strip).
func TestReviewFilterStrip_tabFocusOrder_matchesSelectLayoutOrder(t *testing.T) {
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

	view := NewReviewView(nil, db, root, nil, nil, nil, nil)
	sels := collectSelectWidgets(view)
	if len(sels) < 3 {
		t.Fatalf("expected at least 3 Select widgets in Review view, got %d", len(sels))
	}
	strip := sels[:3]
	if g, w := strip[0].Selected, reviewCollectionSentinel; g != w {
		t.Fatalf("collection select: got %q want %q", g, w)
	}
	if g, w := strip[1].Selected, reviewRatingAny; g != w {
		t.Fatalf("rating select: got %q want %q", g, w)
	}
	if g, w := strip[2].Selected, reviewTagAny; g != w {
		t.Fatalf("tags select: got %q want %q", g, w)
	}

	win := test.NewTempWindow(t, view)
	win.Resize(fyne.NewSize(900, 400))
	applyTestPhotoToolTheme(t, theme.VariantDark)

	c := win.Canvas()
	for i := range strip {
		test.FocusNext(c)
		if f := c.Focused(); f != strip[i] {
			t.Fatalf("after %d FocusNext: focused %T %p want strip select[%d] %p", i+1, f, f, i, strip[i])
		}
	}
}

// Degraded DB: list + count both fail; user sees em-dash and both errors (Story 2.2 risks table).
func TestNewReviewView_nilDB_honestLabel(t *testing.T) {
	test.NewTempApp(t)

	view := NewReviewView(nil, nil, "", nil, nil, nil, nil)
	got := matchingAssetsLabelText(t, view)
	if !strings.Contains(got, "no database") {
		t.Fatalf("label: %q", got)
	}
	sels := collectSelectWidgets(view)
	if len(sels) != 3 {
		t.Fatalf("expected 3 Select widgets, got %d", len(sels))
	}
}

// Story 2.2 / Story 2.5: tag strip sync failure must not be log-only — count line mirrors collections-unavailable pattern.
// Story 2.2 / party dev 2/2: when count fails and both album + tag reads fail, suffix order matches strip (Collection before Tags).
func TestReviewView_closedDB_degradedSuffix_ordersCollectionsBeforeTags(t *testing.T) {
	test.NewTempApp(t)

	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	view := NewReviewView(nil, db, root, nil, nil, nil, nil)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	sels := collectSelectWidgets(view)
	if len(sels) < 2 {
		t.Fatalf("expected filter selects, got %d", len(sels))
	}
	sels[1].SetSelected("1")
	sels[1].SetSelected(reviewRatingAny)

	got := matchingAssetsLabelText(t, view)
	iCol := strings.Index(got, "collections unavailable")
	iTag := strings.Index(got, "tags unavailable")
	if iCol < 0 || iTag < 0 {
		t.Fatalf("want both degraded hints in label: %q", got)
	}
	if iCol >= iTag {
		t.Fatalf("collections hint should precede tags hint: %q", got)
	}
}

func TestReviewView_tagStripSyncFailure_showsActionableSuffix(t *testing.T) {
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

	view := NewReviewView(nil, db, root, nil, nil, nil, nil)
	if _, err := db.Exec("DROP TABLE IF EXISTS asset_tags"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("DROP TABLE IF EXISTS tags"); err != nil {
		t.Fatal(err)
	}
	sels := collectSelectWidgets(view)
	if len(sels) < 2 {
		t.Fatalf("expected filter selects, got %d", len(sels))
	}
	sels[1].SetSelected("1")
	sels[1].SetSelected(reviewRatingAny)

	got := matchingAssetsLabelText(t, view)
	if !strings.Contains(got, "tags unavailable") {
		t.Fatalf("want tags unavailable hint in %q", got)
	}
}

func TestReviewView_closedDB_matchingLabelHonest(t *testing.T) {
	test.NewTempApp(t)

	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	view := NewReviewView(nil, db, root, nil, nil, nil, nil)
	got := matchingAssetsLabelText(t, view)
	if !strings.HasPrefix(got, "Matching assets: —") {
		t.Fatalf("label: %q", got)
	}
	if !strings.Contains(got, "Could not read the library") && !strings.Contains(got, "list collections") && !strings.Contains(got, "count assets for review") {
		t.Fatalf("expected mapped read error or list/count detail in label: %q", got)
	}
}

func TestReviewFilterStrip_tabFocusOrder_lightTheme(t *testing.T) {
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

	view := NewReviewView(nil, db, root, nil, nil, nil, nil)
	sels := collectSelectWidgets(view)
	if len(sels) < 3 {
		t.Fatalf("expected at least 3 Select widgets, got %d", len(sels))
	}
	strip := sels[:3]

	win := test.NewTempWindow(t, view)
	win.Resize(fyne.NewSize(900, 400))
	test.ApplyTheme(t, NewPhotoToolTheme(theme.VariantLight))

	c := win.Canvas()
	for i := range strip {
		test.FocusNext(c)
		if f := c.Focused(); f != strip[i] {
			t.Fatalf("light theme: after %d FocusNext: focused %T want strip select[%d]", i+1, f, i)
		}
	}
}

// Story 2.10 AC5: Rejected (hidden) grid must not expose collection quick-assign or bulk assign strip.
func TestStory210_RejectedViewOmitsCollectionAssignChrome(t *testing.T) {
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

	view := NewRejectedView(nil, db, root, nil)
	for _, lb := range collectLabels(view) {
		if lb.Text == "Assign selection" {
			t.Fatalf("rejected view must not show bulk assign label: %q", lb.Text)
		}
	}
	for _, b := range collectButtons(view) {
		if strings.Contains(b.Text, "Assign selection") {
			t.Fatalf("rejected view must not include assign button, got %q", b.Text)
		}
	}
}

// Story 2.10 AC3/AC7: Review exposes bulk assign; with zero albums the assign action stays disabled.
func TestStory210_ReviewViewBulkAssignPresentAndDisabledWithoutAlbums(t *testing.T) {
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

	view := NewReviewView(nil, db, root, nil, nil, nil, nil)
	var sawAssignLabel bool
	for _, lb := range collectLabels(view) {
		if lb.Text == "Assign selection" {
			sawAssignLabel = true
			break
		}
	}
	if !sawAssignLabel {
		t.Fatal("expected Assign selection label in Review chrome")
	}
	assignBtn := findButtonByText(t, view, "Assign selection to album")
	if !assignBtn.Disabled() {
		t.Fatal("with zero albums, bulk assign button should be disabled")
	}
}

// Story 2.10: fourth Select is assign-target (in addition to three filter-strip selects).
func TestStory210_ReviewViewHasAssignTargetSelect(t *testing.T) {
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

	view := NewReviewView(nil, db, root, nil, nil, nil, nil)
	sels := collectSelectWidgets(view)
	if len(sels) < 4 {
		t.Fatalf("expected at least 4 Select widgets (strip + assign target), got %d", len(sels))
	}
}

// Story 2.10 AC3: assign-target dropdown lists real albums only — never the Collection browse sentinel.
func TestStory210_AssignTargetOptionsExcludeBrowsePredicate(t *testing.T) {
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

	if _, err := store.CreateCollection(db, "PartyAlbum", ""); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	if err := store.InsertAsset(db, "deadbeefcafebabe", "2026/04/13/a.jpg", now, now); err != nil {
		t.Fatal(err)
	}

	view := NewReviewView(nil, db, root, nil, nil, nil, nil)
	sels := collectSelectWidgets(view)
	if len(sels) < 4 {
		t.Fatalf("expected strip + assign Selects, got %d", len(sels))
	}
	assignSel := sels[3]
	var sawAlbum bool
	for _, o := range assignSel.Options {
		if o == reviewCollectionSentinel {
			t.Fatalf("assign target must not include browse predicate %q", reviewCollectionSentinel)
		}
		if o == "PartyAlbum" {
			sawAlbum = true
		}
	}
	if !sawAlbum {
		t.Fatalf("assign target options %v missing album name", assignSel.Options)
	}
}

func findLabelContaining(t *testing.T, root fyne.CanvasObject, sub string) {
	t.Helper()
	for _, lb := range collectLabels(root) {
		if strings.Contains(lb.Text, sub) {
			return
		}
	}
	t.Fatalf("no label contains %q", sub)
}

// Story 2.12 / UX-DR9: library-empty Review surface offers a primary Upload CTA.
func TestStory212_ReviewEmptyLibrary_goToUploadCTA(t *testing.T) {
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

	var uploadCalls int32
	view := NewReviewView(nil, db, root, nil, func() { atomic.AddInt32(&uploadCalls, 1) }, nil, nil)
	findLabelContaining(t, view, "Your library has no photos")
	uploadBtn := findButtonByText(t, view, "Go to Upload")
	test.NewTempWindow(t, view)
	test.Tap(uploadBtn)
	if atomic.LoadInt32(&uploadCalls) != 1 {
		t.Fatalf("onGotoUpload calls = %d want 1", uploadCalls)
	}
}

// Story 2.12: non-default filters with zero rows show Reset filters CTA.
func TestStory212_ReviewFiltersEmpty_resetFiltersCTA(t *testing.T) {
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

	now := time.Now().Unix()
	if err := store.InsertAsset(db, "story212unrated", "2026/04/13/u.jpg", now, now); err != nil {
		t.Fatal(err)
	}

	view := NewReviewView(nil, db, root, nil, nil, nil, nil)
	sels := collectSelectWidgets(view)
	if len(sels) < 3 {
		t.Fatalf("expected filter strip selects, got %d", len(sels))
	}
	test.NewTempWindow(t, view)
	sels[1].SetSelected("5")

	findLabelContaining(t, view, "No photos match the current filters")
	resetBtn := findButtonByText(t, view, "Reset filters")
	test.Tap(resetBtn)
	if g, w := sels[1].Selected, reviewRatingAny; g != w {
		t.Fatalf("after reset, rating select: got %q want %q", g, w)
	}
}
