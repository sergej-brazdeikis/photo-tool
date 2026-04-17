package app

import (
	"database/sql"
	"image/color"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"photo-tool/internal/share"
)

// newShellUploadView is the Upload panel for [newMainShell]. When PHOTO_TOOL_UX_JOURNEY_TEST=1 and
// PHOTO_TOOL_UX_UPLOAD_SEED_PATHS is set (newline-separated absolute paths), uses [NewUploadViewWithOptions]
// so [TestUXJourneyCapture] can screenshot the import / FR-06 flow without a file picker.
func newShellUploadView(win fyne.Window, db *sql.DB, libraryRoot string) fyne.CanvasObject {
	if os.Getenv("PHOTO_TOOL_UX_JOURNEY_TEST") != "1" {
		return NewUploadView(win, db, libraryRoot)
	}
	raw := strings.TrimSpace(os.Getenv("PHOTO_TOOL_UX_UPLOAD_SEED_PATHS"))
	if raw == "" {
		return NewUploadView(win, db, libraryRoot)
	}
	var paths []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			paths = append(paths, line)
		}
	}
	if len(paths) == 0 {
		return NewUploadView(win, db, libraryRoot)
	}
	return NewUploadViewWithOptions(win, db, libraryRoot, UploadViewOptions{
		SeedPaths:             paths,
		SkipCompletionDialogs: true,
		SynchronousIngest:     true,
	})
}

// primaryNavItems are UX-DR13 destinations in fixed order (label → internal panel key).
var primaryNavItems = []struct {
	label, key string
}{
	{"Upload", "upload"},
	{"Review", "review"},
	{"Collections", "collections"},
	{"Rejected", "rejected"},
}

// PrimaryNavLabels returns nav labels in UX-DR13 order (exported for regression tests).
func PrimaryNavLabels() []string {
	out := make([]string, len(primaryNavItems))
	for i, it := range primaryNavItems {
		out[i] = it.label
	}
	return out
}

// PrimaryNavKeys returns internal panel keys in the same order as PrimaryNavLabels (exported for regression tests).
func PrimaryNavKeys() []string {
	out := make([]string, len(primaryNavItems))
	for i, it := range primaryNavItems {
		out[i] = it.key
	}
	return out
}

// NewMainShell builds persistent navigation (UX-DR13 order) and a swappable content region.
// Upload embeds the existing ingest flow. Review, Collections, and Rejected mount the real
// Epic 2 panels (Stories 2.2+); Story 2.1 originally shipped placeholders — see story history.
// shareLoop starts the loopback share HTTP server on first successful mint (Story 3.2); nil disables URLs in tests.
//
// The left-rail “Semantic roles (preview)” demo strip is omitted in production builds so the
// shipping UI matches user-facing UX (see ux_layout_invariants_test). Tests that need the strip
// for theme/NFR layout calls should use [newMainShell] with omitSemanticStylePreview false.
func NewMainShell(win fyne.Window, db *sql.DB, libraryRoot string, shareLoop *share.Loopback) fyne.CanvasObject {
	return newMainShell(win, db, libraryRoot, true, shareLoop)
}

// newMainShell is the shell constructor. When omitSemanticStylePreview is true, the
// left-rail “Semantic roles (preview)” demo block is omitted so NFR-01 structural
// tests can stress 1024px width without non-shipping chrome crowding Review.
func newMainShell(win fyne.Window, db *sql.DB, libraryRoot string, omitSemanticStylePreview bool, shareLoop *share.Loopback) fyne.CanvasObject {
	upload := newShellUploadView(win, db, libraryRoot)

	var clearReviewUndo func()

	keyByLabel := make(map[string]string, len(primaryNavItems))
	for _, it := range primaryNavItems {
		keyByLabel[it.label] = it.key
	}

	labels := PrimaryNavLabels()
	prevNavKey := keyByLabel[labels[0]]
	center := container.NewStack()
	var selectPanel func(string)
	var navButtons []*widget.Button
	var setNavSelection func(int)

	gotoReview := func() {
		nextKey := keyByLabel[labels[1]]
		// Match nav button prelude order: undo-clear uses prior prevNavKey; then commit prevNavKey
		// before selectPanel so synchronous panel code never sees a stale transition (party create 2/2).
		// collectionsNavShouldResetToList(_, nextKey) is always false when nextKey=="review" (AC12).
		clearReviewUndoIfLeftReview(prevNavKey, nextKey, clearReviewUndo)
		prevNavKey = nextKey
		if setNavSelection != nil {
			setNavSelection(1)
		}
		if selectPanel != nil {
			selectPanel(nextKey)
		}
	}

	var reloadReviewCollectionsFromCollectionsTab func()
	collectionsView := NewCollectionsView(win, db, libraryRoot, gotoReview, func() {
		if reloadReviewCollectionsFromCollectionsTab != nil {
			reloadReviewCollectionsFromCollectionsTab()
		}
	})

	gotoUpload := func() {
		nextKey := keyByLabel[labels[0]]
		// Same prelude order as nav OnTapped: Collections AC12 reset, then undo-clear, then commit selection/panel.
		if collectionsNavShouldResetToList(prevNavKey, nextKey) {
			collectionsView.ResetToList()
		}
		clearReviewUndoIfLeftReview(prevNavKey, nextKey, clearReviewUndo)
		prevNavKey = nextKey
		if setNavSelection != nil {
			setNavSelection(0)
		}
		if selectPanel != nil {
			selectPanel(nextKey)
		}
	}

	review := NewReviewView(win, db, libraryRoot, func(clear func()) {
		clearReviewUndo = clear
	}, gotoUpload, shareLoop, func(reload func()) {
		reloadReviewCollectionsFromCollectionsTab = reload
	})

	rejected := NewRejectedView(win, db, libraryRoot, gotoReview)

	panels := map[string]fyne.CanvasObject{
		"upload":      upload,
		"review":      review,
		"collections": collectionsView.CanvasObject(),
		"rejected":    rejected,
	}
	for _, it := range primaryNavItems {
		if panels[it.key] == nil {
			panic("app: shell missing panel for primary nav key " + it.key)
		}
	}

	selectPanel = func(key string) {
		center.RemoveAll()
		center.Add(panels[key])
		center.Refresh()
	}

	navBox := container.NewVBox()
	for i, it := range primaryNavItems {
		idx, item := i, it
		b := widget.NewButton(it.label, nil)
		b.Importance = widget.MediumImportance
		navButtons = append(navButtons, b)
		b.OnTapped = func() {
			nextKey := item.key
			// Story 2.8 AC12: re-selecting Collections returns to the album list from detail.
			if collectionsNavShouldResetToList(prevNavKey, nextKey) {
				collectionsView.ResetToList()
			}
			clearReviewUndoIfLeftReview(prevNavKey, nextKey, clearReviewUndo)
			prevNavKey = nextKey
			if setNavSelection != nil {
				setNavSelection(idx)
			}
			selectPanel(nextKey)
		}
		navBox.Add(b)
	}
	setNavSelection = func(selectedIdx int) {
		for j, b := range navButtons {
			if j == selectedIdx {
				b.Importance = widget.HighImportance
			} else {
				b.Importance = widget.MediumImportance
			}
			b.Refresh()
		}
	}
	setNavSelection(0)
	selectPanel(keyByLabel[labels[0]])

	left := container.NewVBox(
		widget.NewLabelWithStyle("Photo Tool", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		navBox,
		widget.NewSeparator(),
	)
	if !omitSemanticStylePreview {
		left.Add(widget.NewLabelWithStyle("Semantic roles (preview)", fyne.TextAlignLeading, fyne.TextStyle{Italic: true}))
		left.Add(widget.NewLabelWithStyle("Non-functional — demonstrates theme colors only.", fyne.TextAlignLeading, fyne.TextStyle{Italic: true}))
		left.Add(newSemanticStylePreviewStrip())
	}

	// Minimum rail width so primary nav (including Upload) stays visible beside wide panels (Collections list/detail).
	railFloor := canvas.NewRectangle(color.Transparent)
	railFloor.SetMinSize(fyne.NewSize(168, 1))
	leftRail := container.NewStack(railFloor, left)

	// Border: persistent chrome + content (AC1–2). Epic §2.1 “compact row / rail” is this **vertical** rail
	// (Direction A); buttons always fire OnTapped (vs RadioGroup same-item no-op), which unlocks Collections
	// list reset on re-tap (Story 2.8 AC12).
	return container.NewBorder(nil, nil, leftRail, nil, center)
}

// clearReviewUndoIfLeftReview invokes clear when primary navigation leaves Review (Story 2.6 AC6).
func clearReviewUndoIfLeftReview(prevPanelKey, nextPanelKey string, clear func()) {
	if clear == nil {
		return
	}
	if prevPanelKey == "review" && nextPanelKey != "review" {
		clear()
	}
}

// collectionsNavShouldResetToList is true when the user re-activated Collections while
// already on that section (Story 2.8 AC12). Kept as a pure function for tests.
func collectionsNavShouldResetToList(prevPanelKey, nextPanelKey string) bool {
	return prevPanelKey == "collections" && nextPanelKey == "collections"
}

func newSemanticStylePreviewStrip() fyne.CanvasObject {
	// Enabled, no-op taps so Fyne renders true Danger/Warning chrome (AC9). Disabled
	// buttons reuse disabled styling and can collapse the distinction we need to prove.
	destructive := widget.NewButton("Destructive (preview)", func() {})
	destructive.Importance = widget.DangerImportance

	caution := widget.NewButton("Reject / caution (preview)", func() {})
	caution.Importance = widget.WarningImportance

	// Vertical stack keeps the left rail narrow at NFR-01 min width (1024) so Review
	// filter strip stays on-screen with full primary nav (Story 2.11 AC1).
	return container.NewVBox(destructive, caution)
}
