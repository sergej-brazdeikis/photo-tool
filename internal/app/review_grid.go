package app

import (
	"database/sql"
	"errors"
	"image"
	"image/color"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"photo-tool/internal/domain"
	"photo-tool/internal/share"
	"photo-tool/internal/store"
)

const (
	reviewGridPageSize = 48
	reviewGridColumns = 4

	// User-facing only — must stay free of driver/SQL fragments (Story 2.3 AC3–AC4).
	reviewGridMsgPageLoadFail = "Can't load this page — library read failed. Try changing the filter or restarting the app."
	reviewGridMsgDecodeFail   = "Can't preview — file missing or unsupported format."
)

// reviewGridListRowCount is Fyne List row count for a paged thumbnail grid (Story 2.3 / UX-DR18).
// When there are no matching assets, the list must report zero rows so the shell empty state is not
// undermined by blank grid chrome.
func reviewGridListRowCount(total int64) int {
	if total <= 0 {
		return 0
	}
	return int((total + reviewGridColumns - 1) / reviewGridColumns)
}

// errReviewGridPageFailed is returned when a paged list query failed for this page.
// It carries no driver/SQL text (Story 2.3 AC4); the first failure is logged once with the real error.
var errReviewGridPageFailed = errors.New("review grid: page load failed")

var uxJourneyGridPendingOnce sync.Once
var uxJourneyGridPendingRaster image.Image

// uxJourneyGridPendingThumbRaster is a decode-safe pending thumbnail for UX capture only (MediaPhotoIcon is SVG).
func uxJourneyGridPendingThumbRaster() image.Image {
	uxJourneyGridPendingOnce.Do(func() {
		const w, h = 56, 42
		rgba := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				rgba.Set(x, y, color.NRGBA{
					R: uint8(x * 255 / max(w-1, 1)),
					G: uint8(y * 255 / max(h-1, 1)),
					B: 105,
					A: 255,
				})
			}
		}
		uxJourneyGridPendingRaster = rgba
	})
	return uxJourneyGridPendingRaster
}

func ratingBadgeText(r *int) string {
	if r == nil {
		return "—"
	}
	return strconv.Itoa(*r) + "★"
}

func rejectBadgeLabel(rejected int) string {
	if rejected == 0 {
		return ""
	}
	return "Hidden"
}

type reviewGridCell struct {
	root        *fyne.Container
	tap         *tapLayer
	bg          *canvas.Rectangle
	img         *canvas.Image
	failIcon    *widget.Icon
	failLbl     *widget.Label
	rating      *widget.Label
	rejectBadge *widget.Label
	restoreBtn  *widget.Button
}

// tapLayer turns a subtree into a tappable region (Story 2.4: open loupe from thumbnail).
type tapLayer struct {
	widget.BaseWidget

	Child fyne.CanvasObject
	// Handler is rebound on each List bind so recycled rows do not keep stale indices.
	// Uses desktop.MouseEvent so Cmd/Ctrl (Story 2.5) is available; do not add fyne.Tapped on this widget
	// or the GLFW driver would invoke both paths and double-fire.
	Handler func(*desktop.MouseEvent)
	// SecondaryHandler: context menu / quick actions (Story 2.10); primary path stays loupe + bulk select.
	SecondaryHandler func(*desktop.MouseEvent)
}

func newTapLayer(child fyne.CanvasObject) *tapLayer {
	t := &tapLayer{Child: child}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tapLayer) MouseDown(*desktop.MouseEvent) {}

func (t *tapLayer) MouseUp(e *desktop.MouseEvent) {
	if e == nil {
		return
	}
	if e.Button == desktop.MouseButtonSecondary {
		if t.SecondaryHandler != nil {
			t.SecondaryHandler(e)
		}
		return
	}
	if e.Button != desktop.MouseButtonPrimary {
		return
	}
	if t.Handler != nil {
		t.Handler(e)
	}
}

// reviewMultiSelectModifier is Cmd (Super) and/or Ctrl (Story 2.5 bulk select).
func reviewMultiSelectModifier(e *desktop.MouseEvent) bool {
	if e == nil {
		return false
	}
	return e.Modifier&(fyne.KeyModifierSuper|fyne.KeyModifierControl) != 0
}

func (t *tapLayer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.Child)
}

// newReviewGridCell builds one thumbnail cell. rejectedListTemplate must be true for Rejected-mode
// grids so widget.List's template row MinSize includes the Restore button; hidden widgets contribute
// no height and the list would fix row height too short, clipping Restore after bind.
// newFilterZeroMatchPhotoPlate is a large neutral “photo-shaped” plate for zero-match filter empty
// states so the viewport stays image-forward (UX spec) without implying a real library asset.
func newFilterZeroMatchPhotoPlate() fyne.CanvasObject {
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	bg.CornerRadius = 4
	bg.SetMinSize(fyne.NewSize(uxImageLoupeMainMin, uxImageLoupeMainMin))
	icon := canvas.NewImageFromResource(theme.MediaPhotoIcon())
	icon.FillMode = canvas.ImageFillContain
	icon.SetMinSize(fyne.NewSize(uxImageLoupeMainMin, uxImageLoupeMainMin))
	return container.NewMax(bg, container.NewCenter(icon))
}

func newReviewGridCell(rejectedListTemplate bool) *reviewGridCell {
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	bg.CornerRadius = 4
	// Reserve thumb footprint even when img is Hidden (clear/empty slots) — hidden Images
	// do not contribute to MinSize, so without this the List row collapses to slivers.
	bg.SetMinSize(fyne.NewSize(uxImageGridThumbMin, uxImageGridThumbMin))
	img := canvas.NewImageFromFile("")
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(uxImageGridThumbMin, uxImageGridThumbMin))
	failIcon := widget.NewIcon(theme.ErrorIcon())
	failIcon.Hide()
	failLbl := widget.NewLabel("")
	// Word wrap can panic in Fyne when the cell has not been laid out yet (async decode path).
	failLbl.Wrapping = fyne.TextWrapOff
	failLbl.Hide()
	rating := widget.NewLabel("")
	rejectBadge := widget.NewLabel("")
	rejectBadge.Importance = widget.WarningImportance
	restoreBtn := widget.NewButton("Restore", nil)
	if rejectedListTemplate {
		restoreBtn.Disable()
	} else {
		restoreBtn.Hide()
	}
	thumb := container.NewStack(bg, img, failIcon, failLbl)
	ratingRow := container.NewHBox(rating, layout.NewSpacer(), rejectBadge)
	// Full cell width so "Restore" is not clipped in narrow Rejected columns (NFR-01 / 4-up grid).
	meta := container.NewVBox(ratingRow, container.NewMax(restoreBtn))
	tap := newTapLayer(thumb)
	c := &reviewGridCell{
		tap:         tap,
		bg:          bg,
		img:         img,
		failIcon:    failIcon,
		failLbl:     failLbl,
		rating:      rating,
		rejectBadge: rejectBadge,
		restoreBtn:  restoreBtn,
	}
	c.root = container.NewVBox(tap, meta)
	return c
}

func (c *reviewGridCell) object() fyne.CanvasObject { return c.root }

type reviewAssetGrid struct {
	win         fyne.Window
	db          *sql.DB
	libraryRoot string
	onLoupeOpen func(index int)
	// onSelectionChange runs on the UI thread after bulk selection changes (Story 2.5).
	onSelectionChange func()
	// rejectedMode lists ReviewRejectedBaseWhere rows; no loupe; Cmd/Ctrl+tap bulk-selects (Story 2.6 / 2.7).
	rejectedMode bool
	// onRestoreAsset is required when rejectedMode (per-cell Restore).
	onRestoreAsset func(assetID int64)
	// openCellAssignMenu: secondary-click assign on a thumbnail (Story 2.10); nil in rejectedMode / DnD-only builds.
	openCellAssignMenu func(anchor fyne.CanvasObject, assetID int64, absPos fyne.Position)
	// shareLoopback serves GET /s/{token} after mint (Story 3.2); nil in tests / Rejected grid.
	shareLoopback *share.Loopback

	mu      sync.Mutex
	filters domain.ReviewFilters
	total   int64
	pages   map[int][]store.ReviewGridRow
	// pageFailed records pages whose list query failed; avoids hammering SQLite and duplicate slog
	// lines while the user scrolls the same broken window (cleared on reset / invalidatePages).
	pageFailed map[int]struct{}
	// selected holds asset ids chosen with Cmd/Ctrl+click for bulk tagging (plain tap clears and opens loupe).
	selected map[int64]struct{}

	// thumbnailBinding maps each cell's *canvas.Image to the asset id last bound there;
	// async thumbnail completion checks this to ignore stale results after scroll/recycle.
	thumbnailBinding sync.Map
	// thumbGen bumps on reset/invalidatePages so in-flight decodes never touch cells after a grid refresh (Fyne safety).
	thumbGen atomic.Uint64

	list *widget.List
}

func newReviewAssetGrid(win fyne.Window, db *sql.DB, libraryRoot string, onLoupeOpen func(index int), onSelectionChange func(), rejectedMode bool, onRestore func(assetID int64), openCellAssign func(anchor fyne.CanvasObject, assetID int64, absPos fyne.Position), shareLoopback *share.Loopback) *reviewAssetGrid {
	g := &reviewAssetGrid{
		win:                win,
		db:                 db,
		libraryRoot:        libraryRoot,
		onLoupeOpen:        onLoupeOpen,
		onSelectionChange:  onSelectionChange,
		rejectedMode:       rejectedMode,
		onRestoreAsset:     onRestore,
		openCellAssignMenu: openCellAssign,
		shareLoopback:      shareLoopback,
		pages:              make(map[int][]store.ReviewGridRow),
	}
	g.list = widget.NewList(
		func() int {
			g.mu.Lock()
			defer g.mu.Unlock()
			return reviewGridListRowCount(g.total)
		},
		func() fyne.CanvasObject {
			cells := make([]fyne.CanvasObject, reviewGridColumns)
			for i := range cells {
				cells[i] = newReviewGridCell(g.rejectedMode).object()
			}
			return container.NewHBox(cells...)
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			g.bindGridRow(int(id), o)
		},
	)
	g.list.HideSeparators = true
	// PHOTO_TOOL_UX_JOURNEY_TEST=1 scopes registration to the capture test / bundle subprocess only
	// (avoid parallel package tests seeing PHOTO_TOOL_UX_CAPTURE_DIR alone and clobbering this pointer).
	if os.Getenv("PHOTO_TOOL_UX_CAPTURE_DIR") != "" && os.Getenv("PHOTO_TOOL_UX_JOURNEY_TEST") == "1" && g.onLoupeOpen != nil {
		registerUXCaptureReviewGrid(g)
	}
	return g
}

func (g *reviewAssetGrid) canvasObject() fyne.CanvasObject { return g.list }

// syncGridScrollVisible shows or hides the shell scroll around the thumbnail list (Story 2.3 UX-DR18).
func (g *reviewAssetGrid) syncGridScrollVisible(scroll *container.Scroll, show bool) {
	if g == nil || scroll == nil {
		return
	}
	if show {
		scroll.Show()
		return
	}
	scroll.Hide()
}

func (g *reviewAssetGrid) invalidatePages() {
	g.thumbGen.Add(1)
	g.mu.Lock()
	g.pages = make(map[int][]store.ReviewGridRow)
	g.pageFailed = nil
	g.mu.Unlock()
	fyne.Do(func() { g.list.Refresh() })
}

func (g *reviewAssetGrid) reset(f domain.ReviewFilters, total int64) {
	g.thumbGen.Add(1)
	g.mu.Lock()
	g.filters = f
	g.total = total
	g.pages = make(map[int][]store.ReviewGridRow)
	g.pageFailed = nil
	g.selected = nil
	fn := g.onSelectionChange
	g.mu.Unlock()
	if fn != nil {
		fn()
	}
	g.list.Refresh()
}

func (g *reviewAssetGrid) toggleSelected(assetID int64) {
	g.mu.Lock()
	if g.selected == nil {
		g.selected = make(map[int64]struct{})
	}
	if _, ok := g.selected[assetID]; ok {
		delete(g.selected, assetID)
	} else {
		g.selected[assetID] = struct{}{}
	}
	cb := g.onSelectionChange
	g.mu.Unlock()
	if cb != nil {
		cb()
	}
	g.list.Refresh()
}

func (g *reviewAssetGrid) clearSelected() {
	g.mu.Lock()
	g.selected = nil
	cb := g.onSelectionChange
	g.mu.Unlock()
	if cb != nil {
		cb()
	}
	g.list.Refresh()
}

func (g *reviewAssetGrid) isSelected(assetID int64) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.selected == nil {
		return false
	}
	_, ok := g.selected[assetID]
	return ok
}

// SelectedAssetIDs returns the current bulk selection (unordered).
func (g *reviewAssetGrid) SelectedAssetIDs() []int64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.selected) == 0 {
		return nil
	}
	out := make([]int64, 0, len(g.selected))
	for id := range g.selected {
		out = append(out, id)
	}
	return out
}

func (g *reviewAssetGrid) ensurePageLocked(pageIdx int) error {
	// Runs during widget.List item updates (UI thread). LIMIT is small (reviewGridPageSize);
	// Story 2.3 trades a bit of main-thread SQLite latency for simpler paging vs async prefetch.
	if g.pages[pageIdx] != nil {
		return nil
	}
	if g.pageFailed != nil {
		if _, ok := g.pageFailed[pageIdx]; ok {
			return errReviewGridPageFailed
		}
	}
	offset := pageIdx * reviewGridPageSize
	var rows []store.ReviewGridRow
	var err error
	if g.rejectedMode {
		rows, err = store.ListRejectedForReview(g.db, g.filters, reviewGridPageSize, offset)
	} else {
		rows, err = store.ListAssetsForReview(g.db, g.filters, reviewGridPageSize, offset)
	}
	if err != nil {
		if g.pageFailed == nil {
			g.pageFailed = make(map[int]struct{})
		}
		if _, dup := g.pageFailed[pageIdx]; !dup {
			g.pageFailed[pageIdx] = struct{}{}
			slog.Error("review grid: page query", "page", pageIdx, "rejected_mode", g.rejectedMode, "err", err)
		}
		return errReviewGridPageFailed
	}
	g.pages[pageIdx] = rows
	return nil
}

// gridMetaRestoreButton finds the per-cell Restore control under optional layout wrappers (e.g. Max).
func gridMetaRestoreButton(o fyne.CanvasObject) (*widget.Button, bool) {
	switch v := o.(type) {
	case *widget.Button:
		return v, true
	case *fyne.Container:
		if len(v.Objects) == 1 {
			return gridMetaRestoreButton(v.Objects[0])
		}
	}
	return nil, false
}

func (g *reviewAssetGrid) rowAt(i int) (store.ReviewGridRow, bool, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if i < 0 || int64(i) >= g.total {
		return store.ReviewGridRow{}, false, nil
	}
	pageIdx := i / reviewGridPageSize
	if err := g.ensurePageLocked(pageIdx); err != nil {
		return store.ReviewGridRow{}, false, err
	}
	rows := g.pages[pageIdx]
	slot := i % reviewGridPageSize
	if slot >= len(rows) {
		return store.ReviewGridRow{}, false, nil
	}
	return rows[slot], true, nil
}

func parseReviewGridCell(root *fyne.Container) (*reviewGridCell, bool) {
	if root == nil || len(root.Objects) != 2 {
		return nil, false
	}
	tapW, ok := root.Objects[0].(*tapLayer)
	if !ok {
		return nil, false
	}
	thumbStack, ok := tapW.Child.(*fyne.Container)
	if !ok || len(thumbStack.Objects) < 4 {
		return nil, false
	}
	meta, ok := root.Objects[1].(*fyne.Container)
	if !ok || len(meta.Objects) != 2 {
		return nil, false
	}
	ratingRow, ok := meta.Objects[0].(*fyne.Container)
	if !ok || len(ratingRow.Objects) < 3 {
		return nil, false
	}
	ratingLbl, ok := ratingRow.Objects[0].(*widget.Label)
	if !ok {
		return nil, false
	}
	rejectLbl, ok := ratingRow.Objects[2].(*widget.Label)
	if !ok {
		return nil, false
	}
	restore, ok := gridMetaRestoreButton(meta.Objects[1])
	if !ok {
		return nil, false
	}
	return &reviewGridCell{
		root:        root,
		tap:         tapW,
		bg:          thumbStack.Objects[0].(*canvas.Rectangle),
		img:         thumbStack.Objects[1].(*canvas.Image),
		failIcon:    thumbStack.Objects[2].(*widget.Icon),
		failLbl:     thumbStack.Objects[3].(*widget.Label),
		rating:      ratingLbl,
		rejectBadge: rejectLbl,
		restoreBtn:  restore,
	}, true
}

func (g *reviewAssetGrid) bindGridRow(rowIdx int, o fyne.CanvasObject) {
	rowBox, ok := o.(*fyne.Container)
	if !ok || len(rowBox.Objects) != reviewGridColumns {
		return
	}
	cells := make([]*reviewGridCell, reviewGridColumns)
	for col := 0; col < reviewGridColumns; col++ {
		cellRoot, ok := rowBox.Objects[col].(*fyne.Container)
		if !ok {
			return
		}
		gc, ok := parseReviewGridCell(cellRoot)
		if !ok {
			return
		}
		cells[col] = gc
	}

	// List rows are recycled: when the last row shrinks from 2 assets to 1, the second cell
	// may not get another bind pass if Fyne keeps a single list row — clear every slot first
	// so stale thumbnails and InputBackground tiles cannot sit beside an updated count.
	for col := 0; col < reviewGridColumns; col++ {
		cells[col].clear(&g.thumbnailBinding)
	}

	for col := 0; col < reviewGridColumns; col++ {
		idx := rowIdx*reviewGridColumns + col
		assetRow, have, err := g.rowAt(idx)
		if err != nil {
			for _, c := range cells {
				c.showUserFailure(&g.thumbnailBinding, reviewGridMsgPageLoadFail)
			}
			return
		}
		if !have {
			for j := col; j < reviewGridColumns; j++ {
				cells[j].clear(&g.thumbnailBinding)
			}
			return
		}
		cells[col].tap.SecondaryHandler = nil
		if !g.rejectedMode && g.openCellAssignMenu != nil {
			aid := assetRow.ID
			cells[col].tap.SecondaryHandler = func(me *desktop.MouseEvent) {
				if me == nil || me.Button != desktop.MouseButtonSecondary {
					return
				}
				g.openCellAssignMenu(cells[col].tap, aid, me.AbsolutePosition)
			}
		}
		// Rejected bucket: Restore is per-row; Cmd/Ctrl+tap toggles bulk selection for delete (Story 2.7).
		cells[col].tap.Handler = func(me *desktop.MouseEvent) {
			if g.rejectedMode {
				if reviewMultiSelectModifier(me) {
					g.toggleSelected(assetRow.ID)
				} else {
					g.clearSelected()
				}
				return
			}
			// Plain tap: clear multi-select and open loupe. Cmd/Ctrl+tap toggles bulk selection (same cell hit target as loupe).
			if reviewMultiSelectModifier(me) {
				g.toggleSelected(assetRow.ID)
				return
			}
			g.clearSelected()
			if g.win != nil && g.onLoupeOpen != nil {
				g.onLoupeOpen(idx)
			}
		}
		cells[col].bindRow(g, assetRow)
	}
}

func (c *reviewGridCell) clear(thumbBind *sync.Map) {
	if thumbBind != nil {
		thumbBind.Delete(c.img)
	}
	if c.tap != nil {
		c.tap.Handler = nil
		c.tap.SecondaryHandler = nil
	}
	// Slightly dimmer than a real thumbnail cell so empty grid slots do not read as “filled”.
	c.bg.FillColor = theme.Color(theme.ColorNameBackground)
	c.bg.Refresh()
	c.img.File = ""
	c.img.Resource = nil
	c.img.Image = nil
	c.failIcon.Hide()
	c.failLbl.Hide()
	c.failLbl.SetText("")
	c.rating.SetText("")
	c.rejectBadge.SetText("")
	c.rejectBadge.Hide()
	c.restoreBtn.Hide()
	c.restoreBtn.Disable()
	c.restoreBtn.OnTapped = nil
	// Hide empty image (no placeholder tile); bg.SetMinSize(uxImageGridThumbMin) holds list row height.
	c.img.Hide()
	c.img.Refresh()
}

// showUserFailure is decode/page failure UX (AC3–AC4): icon + short copy, no raw errors.
func (c *reviewGridCell) showUserFailure(thumbBind *sync.Map, msg string) {
	if thumbBind != nil {
		thumbBind.Delete(c.img)
	}
	c.bg.FillColor = theme.Color(theme.ColorNameInputBackground)
	c.bg.Refresh()
	c.img.File = ""
	c.img.Resource = nil
	c.img.Image = nil
	c.img.Hide()
	c.failIcon.SetResource(theme.ErrorIcon())
	c.failIcon.Show()
	c.failLbl.SetText(msg)
	c.failLbl.Show()
	c.rating.SetText("")
	c.rejectBadge.SetText("")
	c.rejectBadge.Hide()
	c.restoreBtn.Hide()
	c.restoreBtn.Disable()
	c.restoreBtn.OnTapped = nil
	c.img.Refresh()
}

func (c *reviewGridCell) bindRow(g *reviewAssetGrid, row store.ReviewGridRow) {
	if g.isSelected(row.ID) {
		c.bg.FillColor = theme.Color(theme.ColorNameSelection)
	} else {
		c.bg.FillColor = theme.Color(theme.ColorNameInputBackground)
	}
	c.bg.Refresh()
	c.img.Show()
	c.failIcon.Hide()
	c.failLbl.Hide()
	c.failLbl.SetText("")
	c.rating.SetText(ratingBadgeText(row.Rating))
	if g.rejectedMode {
		c.rejectBadge.Hide()
		c.restoreBtn.Show()
		c.restoreBtn.Enable()
		c.restoreBtn.Importance = widget.MediumImportance
		c.restoreBtn.OnTapped = func() {
			if g.onRestoreAsset != nil {
				g.onRestoreAsset(row.ID)
			}
		}
	} else {
		c.restoreBtn.Hide()
		c.restoreBtn.Disable()
		c.restoreBtn.OnTapped = nil
		if lbl := rejectBadgeLabel(row.Rejected); lbl != "" {
			c.rejectBadge.SetText(lbl)
			c.rejectBadge.Show()
		} else {
			c.rejectBadge.SetText("")
			c.rejectBadge.Hide()
		}
	}

	// Pending decode (UX-DR3): distinguish from final image and from failed-decode (ErrorIcon path).
	c.img.File = ""
	c.img.Resource = nil
	c.img.Image = nil
	c.img.Refresh()
	if os.Getenv("PHOTO_TOOL_UX_JOURNEY_TEST") == "1" {
		// MediaPhotoIcon is SVG; Fyne's image path can panic during journey grid rebind — use a raster.
		c.img.Image = uxJourneyGridPendingThumbRaster()
	} else {
		c.img.Resource = theme.MediaPhotoIcon()
	}
	c.img.Refresh()

	srcAbs := filepath.Join(g.libraryRoot, filepath.FromSlash(row.RelPath))
	cacheAbs := ThumbnailCachePath(g.libraryRoot, row.ID, row.ContentHash)
	wantID := row.ID
	imgRef := c.img
	g.thumbnailBinding.Store(imgRef, wantID)
	gen := g.thumbGen.Load()

	go func() {
		err := WriteThumbnailJPEG(srcAbs, cacheAbs)
		fyne.Do(func() {
			if g.thumbGen.Load() != gen {
				return
			}
			v, ok := g.thumbnailBinding.Load(imgRef)
			if !ok || v.(int64) != wantID {
				return
			}
			if err != nil {
				c.showUserFailure(&g.thumbnailBinding, reviewGridMsgDecodeFail)
				return
			}
			c.failIcon.Hide()
			c.failLbl.Hide()
			c.img.Show()
			c.img.Resource = nil
			c.img.Image = nil
			c.img.File = cacheAbs
			c.img.Refresh()
		})
	}()
}
