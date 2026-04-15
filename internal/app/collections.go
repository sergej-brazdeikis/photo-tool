package app

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"photo-tool/internal/domain"
	"photo-tool/internal/store"
)

// CollectionsView is the full-page collections list + album detail surface (Story 2.8).
type CollectionsView struct {
	win                  fyne.Window
	db                   *sql.DB
	libraryRoot          string
	onGotoReview         func()
	onCollectionsMutated func() // optional: e.g. reload Review filter strip (Story 2.9 AC6)

	stack *fyne.Container

	collRows []store.CollectionRow
	list     *widget.List
	listMsg  *widget.Label

	detailCollectionID   int64
	detailCollectionName string
	grouping             domain.CollectionGrouping

	newAlbumBtn *widget.Button
}

// NewCollectionsView builds list + detail state machine. Call [CollectionsView.ResetToList] from shell when
// the Collections nav item is activated while already on Collections (Story 2.8 AC12).
// onGotoReview switches primary nav to Review (Story 2.12 empty album detail CTA).
// onCollectionsMutated runs after successful album create/edit/delete so other panels can reload cached ListCollections (Story 2.9 AC6).
func NewCollectionsView(win fyne.Window, db *sql.DB, libraryRoot string, onGotoReview func(), onCollectionsMutated func()) *CollectionsView {
	v := &CollectionsView{
		win:                  win,
		db:                   db,
		libraryRoot:          libraryRoot,
		onGotoReview:         onGotoReview,
		onCollectionsMutated: onCollectionsMutated,
		grouping:             domain.CollectionGroupStars,
	}
	v.listMsg = widget.NewLabel("")
	v.listMsg.Wrapping = fyne.TextWrapWord
	v.list = widget.NewList(
		func() int {
			if len(v.collRows) == 0 {
				return 0
			}
			return len(v.collRows)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("collection")
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			lbl, ok := o.(*widget.Label)
			if !ok || id < 0 || int(id) >= len(v.collRows) {
				return
			}
			lbl.SetText(v.collRows[id].Name)
		},
	)
	v.list.HideSeparators = true
	v.list.OnSelected = func(id widget.ListItemID) {
		if id < 0 || int(id) >= len(v.collRows) {
			return
		}
		row := v.collRows[id]
		v.openDetail(row.ID, row.Name)
	}
	v.reloadCollectionRows()

	v.stack = container.NewStack(v.listChrome())
	v.refreshListChrome()
	return v
}

// CanvasObject returns the root canvas object for the shell content region.
func (v *CollectionsView) CanvasObject() fyne.CanvasObject { return v.stack }

func (v *CollectionsView) notifyCollectionsMutated() {
	if v.onCollectionsMutated != nil {
		v.onCollectionsMutated()
	}
}

// ResetToList pops collection detail and returns to the library list (Story 2.8 AC12).
func (v *CollectionsView) ResetToList() {
	v.detailCollectionID = 0
	v.detailCollectionName = ""
	v.stack.RemoveAll()
	v.reloadCollectionRows()
	v.refreshListChrome()
	v.stack.Add(v.listChrome())
	v.stack.Refresh()
}

func (v *CollectionsView) listChrome() fyne.CanvasObject {
	newBtn := widget.NewButton("New album", func() { v.promptNewAlbum() })
	v.newAlbumBtn = newBtn
	renBtn := widget.NewButton("Rename…", func() { v.pickAlbumThenEdit() })
	delBtn := widget.NewButton("Delete…", func() { v.pickAlbumThenDelete() })
	delBtn.Importance = widget.DangerImportance
	if len(v.collRows) == 0 {
		newBtn.Importance = widget.HighImportance
		renBtn.Importance = widget.MediumImportance
	} else {
		newBtn.Importance = widget.MediumImportance
	}
	return container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("Albums", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			container.NewHBox(newBtn, renBtn, delBtn),
		),
		nil, nil, nil,
		container.NewVBox(v.listMsg, v.list),
	)
}

func (v *CollectionsView) reloadCollectionRows() {
	rows, err := store.ListCollections(v.db)
	if err != nil {
		slog.Error("collections: list", "err", err)
		v.collRows = nil
		v.listMsg.SetText("Can't load albums — could not read the library. Check the folder is available, then try again.")
		if v.list != nil {
			v.list.Hide()
		}
		return
	}
	v.collRows = rows
	if len(v.collRows) == 0 {
		v.listMsg.SetText("No albums yet. Use “New album” to create one, or assign the current photo to albums from Review (loupe).")
		if v.list != nil {
			v.list.Hide()
		}
	} else {
		v.listMsg.SetText("")
		if v.list != nil {
			v.list.Show()
		}
	}
	if v.list != nil {
		v.list.UnselectAll()
	}
	if v.detailCollectionID == 0 && v.newAlbumBtn != nil {
		if len(v.collRows) == 0 {
			v.newAlbumBtn.Importance = widget.HighImportance
		} else {
			v.newAlbumBtn.Importance = widget.MediumImportance
		}
		v.newAlbumBtn.Refresh()
	}
}

func (v *CollectionsView) refreshListChrome() {
	v.list.Refresh()
}

func (v *CollectionsView) openDetail(collectionID int64, name string) {
	exists, err := store.CollectionExists(v.db, collectionID)
	if err != nil {
		slog.Error("collections: exists", "err", err)
		v.showTransientListMessage("Can't open this album — library read failed.")
		return
	}
	if !exists {
		v.showTransientListMessage("This album is no longer available.")
		return
	}

	v.detailCollectionID = collectionID
	v.detailCollectionName = name
	v.grouping = domain.CollectionGroupStars

	detail := v.buildDetailView()
	v.stack.RemoveAll()
	v.stack.Add(detail)
	v.stack.Refresh()
}

func (v *CollectionsView) showTransientListMessage(msg string) {
	v.listMsg.SetText(msg)
	v.list.Refresh()
}

func (v *CollectionsView) buildDetailView() fyne.CanvasObject {
	back := widget.NewButton("Back", func() { v.ResetToList() })
	editBtn := widget.NewButton("Edit album", func() { v.promptEditDetailAlbum() })
	delBtn := widget.NewButton("Delete album…", func() {
		v.confirmDeleteAlbum(v.detailCollectionID, v.detailCollectionName)
	})
	delBtn.Importance = widget.DangerImportance

	detailBody := container.NewVBox()

	groupLabels := []string{"Stars", "By day", "By camera"}
	group := widget.NewRadioGroup(groupLabels, func(selected string) {
		switch selected {
		case "By day":
			v.grouping = domain.CollectionGroupDay
		case "By camera":
			v.grouping = domain.CollectionGroupCamera
		default:
			v.grouping = domain.CollectionGroupStars
		}
		v.replaceDetailBody(detailBody)
	})
	group.Horizontal = true
	switch v.grouping {
	case domain.CollectionGroupDay:
		group.Selected = "By day"
	case domain.CollectionGroupCamera:
		group.Selected = "By camera"
	default:
		group.Selected = "Stars"
	}

	title := widget.NewLabelWithStyle(v.detailCollectionName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	v.replaceDetailBody(detailBody)

	top := container.NewVBox(
		container.NewHBox(back, layout.NewSpacer(), editBtn, delBtn),
		title,
		widget.NewLabelWithStyle("Group photos", fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
		group,
		widget.NewSeparator(),
	)
	return container.NewBorder(top, nil, nil, nil, container.NewScroll(detailBody))
}

func (v *CollectionsView) replaceDetailBody(host *fyne.Container) {
	host.RemoveAll()
	n, err := store.CountCollectionVisibleAssets(v.db, v.detailCollectionID)
	if err != nil {
		if errors.Is(err, store.ErrCollectionNotFound) {
			v.ResetToList()
			v.showTransientListMessage("This album is no longer available.")
			return
		}
		slog.Error("collections: count", "err", err)
		host.Add(widget.NewLabel("Can't load this album — could not read the library. Try again, or return to the album list."))
		return
	}
	if n == 0 {
		msg := widget.NewLabel("No photos from this album appear here yet. Hidden or removed photos stay out of the album view.")
		msg.Wrapping = fyne.TextWrapWord
		backAlbums := widget.NewButton("Back to albums", func() { v.ResetToList() })
		backAlbums.Importance = widget.HighImportance
		reviewBtn := widget.NewButton("Go to Review", func() {
			if v.onGotoReview != nil {
				v.onGotoReview()
			}
		})
		reviewBtn.Importance = widget.MediumImportance
		host.Add(container.NewVBox(msg, container.NewHBox(backAlbums, reviewBtn)))
		return
	}

	switch v.grouping {
	case domain.CollectionGroupStars:
		secs, err := store.ListCollectionStarSections(v.db, v.detailCollectionID)
		if err != nil {
			v.hostDetailErr(host, err)
			return
		}
		for _, sec := range secs {
			sec := sec
			rating := sec.Rating
			hdr := starSectionHeader(rating)
			host.Add(widget.NewLabelWithStyle(hdr, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
			fetch := func(pageIdx int) ([]store.ReviewGridRow, error) {
				off := pageIdx * reviewGridPageSize
				return store.ListCollectionStarSectionPage(v.db, v.detailCollectionID, rating, reviewGridPageSize, off)
			}
			host.Add(v.newSectionThumbnailGrid(sec.Count, fetch))
		}
	case domain.CollectionGroupDay:
		secs, err := store.ListCollectionDaySections(v.db, v.detailCollectionID)
		if err != nil {
			v.hostDetailErr(host, err)
			return
		}
		for _, sec := range secs {
			sec := sec
			day := sec.DayKey
			host.Add(widget.NewLabelWithStyle(day, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
			fetch := func(pageIdx int) ([]store.ReviewGridRow, error) {
				off := pageIdx * reviewGridPageSize
				return store.ListCollectionDaySectionPage(v.db, v.detailCollectionID, day, reviewGridPageSize, off)
			}
			host.Add(v.newSectionThumbnailGrid(sec.Count, fetch))
		}
	case domain.CollectionGroupCamera:
		secs, err := store.ListCollectionCameraSections(v.db, v.detailCollectionID)
		if err != nil {
			v.hostDetailErr(host, err)
			return
		}
		for _, sec := range secs {
			sec := sec
			lbl := sec.Label
			hdr := store.UnknownCameraLabel
			if lbl != nil {
				hdr = *lbl
			}
			host.Add(widget.NewLabelWithStyle(hdr, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
			fetch := func(pageIdx int) ([]store.ReviewGridRow, error) {
				off := pageIdx * reviewGridPageSize
				return store.ListCollectionCameraSectionPage(v.db, v.detailCollectionID, lbl, reviewGridPageSize, off)
			}
			host.Add(v.newSectionThumbnailGrid(sec.Count, fetch))
		}
	}
	host.Refresh()
}

func (v *CollectionsView) hostDetailErr(host *fyne.Container, err error) {
	if errors.Is(err, store.ErrCollectionNotFound) {
		v.ResetToList()
		v.showTransientListMessage("This album is no longer available.")
		return
	}
	slog.Error("collections: detail", "err", err)
	host.Add(widget.NewLabel("Can't load this album — could not read the library. Try again, or return to the album list."))
}

func starSectionHeader(rating *int) string {
	if rating == nil {
		return "Unrated"
	}
	return fmt.Sprintf("%d★", *rating)
}

func (v *CollectionsView) promptNewAlbum() {
	v.showAlbumForm(0)
}

func (v *CollectionsView) pickAlbumThenEdit() {
	if len(v.collRows) == 0 {
		dialog.ShowInformation("Rename album", "No albums yet.", v.win)
		return
	}
	opts := make([]string, len(v.collRows))
	for i, r := range v.collRows {
		opts[i] = r.Name
	}
	sel := widget.NewSelect(opts, nil)
	sel.SetSelected(opts[0])
	body := container.NewVBox(
		widget.NewLabel("Album to rename"),
		sel,
	)
	d := dialog.NewCustomConfirm("Rename album", "Continue", "Cancel", body, func(ok bool) {
		if !ok {
			return
		}
		idx := sel.SelectedIndex()
		if idx < 0 || idx >= len(v.collRows) {
			return
		}
		v.showAlbumForm(v.collRows[idx].ID)
	}, v.win)
	d.Show()
}

func (v *CollectionsView) pickAlbumThenDelete() {
	if len(v.collRows) == 0 {
		dialog.ShowInformation("Delete album", "No albums yet.", v.win)
		return
	}
	opts := make([]string, len(v.collRows))
	for i, r := range v.collRows {
		opts[i] = r.Name
	}
	sel := widget.NewSelect(opts, nil)
	sel.SetSelected(opts[0])
	body := container.NewVBox(
		widget.NewLabel("Album to delete"),
		sel,
	)
	d := dialog.NewCustomConfirm("Delete album", "Continue", "Cancel", body, func(ok bool) {
		if !ok {
			return
		}
		idx := sel.SelectedIndex()
		if idx < 0 || idx >= len(v.collRows) {
			return
		}
		row := v.collRows[idx]
		v.confirmDeleteAlbum(row.ID, row.Name)
	}, v.win)
	d.Show()
}

func (v *CollectionsView) promptEditDetailAlbum() {
	if v.detailCollectionID == 0 {
		return
	}
	v.showAlbumForm(v.detailCollectionID)
}

func (v *CollectionsView) showAlbumForm(collectionID int64) {
	var saving atomic.Bool
	nameEntry := widget.NewEntry()
	dateEntry := widget.NewEntry()
	dateEntry.SetPlaceHolder("YYYY-MM-DD (optional)")
	errLbl := widget.NewLabel("")
	errLbl.Wrapping = fyne.TextWrapWord

	isEdit := collectionID != 0
	title := "New album"
	if isEdit {
		title = "Edit album"
		d, err := store.GetCollection(v.db, collectionID)
		if err != nil {
			if errors.Is(err, store.ErrCollectionNotFound) {
				dialog.ShowInformation("Album", "This album is no longer available.", v.win)
				if v.detailCollectionID == collectionID {
					v.ResetToList()
				}
				v.reloadCollectionRows()
				v.refreshListChrome()
				v.notifyCollectionsMutated()
				return
			}
			dialog.ShowError(errors.New(userFacingDialogErrText(err)), v.win)
			return
		}
		nameEntry.SetText(d.Name)
		dateEntry.SetText(d.DisplayDate)
	}

	save := widget.NewButton("Save", nil)
	cancel := widget.NewButton("Cancel", nil)

	// Full-width rows read better than a narrow Form inside small dialogs (UX: new album ergonomics).
	body := container.NewVBox(
		widget.NewLabelWithStyle("Name", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nameEntry,
		widget.NewLabelWithStyle("Display date", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		dateEntry,
		errLbl,
		container.NewHBox(layout.NewSpacer(), cancel, save),
	)

	pop := dialog.NewCustomWithoutButtons(title, body, v.win)
	pop.Resize(fyne.NewSize(520, 420))

	save.OnTapped = func() {
		errLbl.SetText("")
		if !saving.CompareAndSwap(false, true) {
			return
		}
		defer saving.Store(false)
		if isEdit {
			if err := store.UpdateCollection(v.db, collectionID, nameEntry.Text, dateEntry.Text); err != nil {
				if errors.Is(err, store.ErrCollectionNotFound) {
					pop.Hide()
					dialog.ShowInformation("Album", "This album is no longer available.", v.win)
					if v.detailCollectionID == collectionID {
						v.ResetToList()
					}
					v.reloadCollectionRows()
					v.refreshListChrome()
					v.notifyCollectionsMutated()
					return
				}
				errLbl.SetText(userFacingCollectionWriteErrText(err))
				return
			}
		} else {
			if _, err := store.CreateCollection(v.db, nameEntry.Text, dateEntry.Text); err != nil {
				errLbl.SetText(userFacingCollectionWriteErrText(err))
				return
			}
		}
		pop.Hide()
		v.reloadCollectionRows()
		v.refreshListChrome()
		v.notifyCollectionsMutated()
		if v.detailCollectionID != 0 && v.detailCollectionID == collectionID {
			d, err := store.GetCollection(v.db, collectionID)
			if err == nil {
				v.detailCollectionName = d.Name
			}
			v.refreshDetailInPlace()
		}
	}
	cancel.OnTapped = func() { pop.Hide() }
	pop.Show()
}

func (v *CollectionsView) refreshDetailInPlace() {
	if v.detailCollectionID == 0 {
		return
	}
	detail := v.buildDetailView()
	v.stack.RemoveAll()
	v.stack.Add(detail)
	v.stack.Refresh()
}

func (v *CollectionsView) confirmDeleteAlbum(id int64, name string) {
	msg := fmt.Sprintf("“%s” will be removed. Photos stay in the library; only this album and its assignments are removed. This cannot be reversed.", name)
	cd := dialog.NewConfirm("Delete album?", msg, func(ok bool) {
		if !ok {
			return
		}
		err := store.DeleteCollection(v.db, id)
		if err != nil {
			if errors.Is(err, store.ErrCollectionNotFound) {
				dialog.ShowInformation("Album", "This album is no longer available.", v.win)
			} else {
				dialog.ShowError(errors.New(userFacingDialogErrText(err)), v.win)
			}
			v.reloadCollectionRows()
			v.refreshListChrome()
			if v.detailCollectionID == id {
				v.ResetToList()
				v.showTransientListMessage("This album is no longer available.")
			}
			v.notifyCollectionsMutated()
			return
		}
		v.reloadCollectionRows()
		v.refreshListChrome()
		if v.detailCollectionID == id {
			v.ResetToList()
		}
		v.showTransientListMessage("Album removed.")
		v.notifyCollectionsMutated()
	}, v.win)
	cd.SetConfirmText("Delete album")
	cd.SetConfirmImportance(widget.DangerImportance)
	cd.Show()
}

// --- Per-section thumbnail grid (Story 2.8 AC10 strategy B: per-section LIMIT/OFFSET).

type collectionSectionGrid struct {
	win         fyne.Window
	libraryRoot string
	total       int64
	fetch       func(pageIdx int) ([]store.ReviewGridRow, error)
	// onStaleCollection runs once if a page fetch finds the album row gone (AC9 while scrolling).
	onStaleCollection func()

	mu               sync.Mutex
	pages            map[int][]store.ReviewGridRow
	staleOnce        sync.Once
	thumbnailBinding sync.Map

	list *widget.List
}

func (v *CollectionsView) newSectionThumbnailGrid(total int64, fetch func(pageIdx int) ([]store.ReviewGridRow, error)) fyne.CanvasObject {
	g := &collectionSectionGrid{
		win:         v.win,
		libraryRoot: v.libraryRoot,
		total:       total,
		fetch:       fetch,
		onStaleCollection: func() {
			fyne.Do(func() {
				v.ResetToList()
				v.showTransientListMessage("This album is no longer available.")
			})
		},
		pages: make(map[int][]store.ReviewGridRow),
	}
	g.list = widget.NewList(
		func() int {
			g.mu.Lock()
			defer g.mu.Unlock()
			if g.total == 0 {
				return 0
			}
			return int((g.total + reviewGridColumns - 1) / reviewGridColumns)
		},
		func() fyne.CanvasObject {
			cells := make([]fyne.CanvasObject, reviewGridColumns)
			for i := range cells {
				cells[i] = newReviewGridCell().object()
			}
			return container.NewHBox(cells...)
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			g.bindGridRow(int(id), o)
		},
	)
	g.list.HideSeparators = true
	return g.list
}

func (g *collectionSectionGrid) ensurePageLocked(pageIdx int) error {
	if g.pages[pageIdx] != nil {
		return nil
	}
	rows, err := g.fetch(pageIdx)
	if err != nil {
		return err
	}
	g.pages[pageIdx] = rows
	return nil
}

func (g *collectionSectionGrid) rowAt(i int) (store.ReviewGridRow, bool, error) {
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

func (g *collectionSectionGrid) bindGridRow(rowIdx int, o fyne.CanvasObject) {
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

	for col := 0; col < reviewGridColumns; col++ {
		idx := rowIdx*reviewGridColumns + col
		assetRow, have, err := g.rowAt(idx)
		if err != nil {
			if errors.Is(err, store.ErrCollectionNotFound) && g.onStaleCollection != nil {
				g.staleOnce.Do(g.onStaleCollection)
				return
			}
			slog.Error("collection grid: page query", "err", err)
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
		cells[col].tap.Handler = nil
		cells[col].bindCollectionThumbnail(g, assetRow)
	}
}

func (c *reviewGridCell) bindCollectionThumbnail(g *collectionSectionGrid, row store.ReviewGridRow) {
	c.bg.FillColor = theme.Color(theme.ColorNameInputBackground)
	c.img.Show()
	c.failIcon.Hide()
	c.failLbl.Hide()
	c.failLbl.SetText("")
	c.rating.SetText(ratingBadgeText(row.Rating))
	c.rejectBadge.SetText("")
	c.rejectBadge.Hide()
	c.restoreBtn.Hide()
	c.restoreBtn.OnTapped = nil

	c.img.File = ""
	c.img.Resource = nil
	c.img.Refresh()

	srcAbs := filepath.Join(g.libraryRoot, filepath.FromSlash(row.RelPath))
	cacheAbs := ThumbnailCachePath(g.libraryRoot, row.ID, row.ContentHash)
	wantID := row.ID
	imgRef := c.img
	g.thumbnailBinding.Store(imgRef, wantID)

	go func() {
		err := WriteThumbnailJPEG(srcAbs, cacheAbs)
		fyne.Do(func() {
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
			c.img.File = cacheAbs
			c.img.Refresh()
		})
	}()
}
