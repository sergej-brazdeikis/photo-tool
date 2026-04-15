package app

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"photo-tool/internal/store"
)

// Keyboard shortcuts while the review loupe is open (Story 2.4 / 2.6 / 2.7, UX-DR5):
//  1–5 — set rating (instant persist)
//   ← / → — prev / next (clamped, no wrap)
//   Esc — close loupe, return focus to grid
// Reject has no letter shortcut: R sits under 4 on QWERTY (adjacent to rating keys). Use the Reject button.
// Cmd/Ctrl+Shift+D — open the same delete confirmation as the “Move to library trash…” button (Story 2.7 AC5).
// Cmd/Ctrl+Shift+S — open Share… (Story 3.1); S is not used for rating (UX-DR5).
// Delete / Backspace alone are not bound to destructive commit (instant delete forbidden).
//
// Focus (UX-DR15): filter strip → thumbnail grid (inside Review scroll) → loupe overlay.
// After close, cleanup focuses the grid list so Tab order returns to the Review surface.
// Loupe chrome is an HBox: Prev, rating1–5, Next, Close — before the letterboxed image (no extra focusable decoration).

// loupeImageLayout reserves ~90% of the loupe body for the image region (FR-09 / UX-DR4).
type loupeImageLayout struct{}

func (loupeImageLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) != 1 {
		return fyne.NewSize(0, 0)
	}
	return objects[0].MinSize()
}

func (loupeImageLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) != 1 {
		return
	}
	img := objects[0]
	mw := size.Width * 9 / 10
	mh := size.Height * 9 / 10
	if mw < 1 {
		mw = 1
	}
	if mh < 1 {
		mh = 1
	}
	img.Resize(fyne.NewSize(mw, mh))
	img.Move(fyne.NewPos((size.Width-mw)/2, (size.Height-mh)/2))
}

// loupeRatingKeyAllowed is true when keyboard1–5 should persist (guards initial load and list errors).
func loupeRatingKeyAllowed(assetID int64) bool {
	return assetID > 0
}

func loupeStepIndex(idx int, delta int, total int64) (newIdx int, moved bool) {
	if total <= 0 {
		return idx, false
	}
	n := int(total)
	if idx < 0 {
		idx = 0
	} else if idx >= n {
		idx = n - 1
	}
	newIdx = idx + delta
	if newIdx < 0 {
		return 0, false
	}
	if newIdx >= n {
		return n - 1, false
	}
	return newIdx, newIdx != idx
}

// openReviewLoupe shows a modal loupe for the asset at startIdx in grid’s current filtered ordering.
// onReviewDataChanged is invoked after tag mutations so the strip/grid stay aligned without closing the loupe.
// onRejectApplied is called after RejectAsset with (assetID, changed); push session undo only when changed is true (Story 2.6).
// Call the returned dismiss function to close and release canvas keyboard hooks (idempotent).
func openReviewLoupe(win fyne.Window, grid *reviewAssetGrid, startIdx int, onReviewDataChanged func(), onRejectApplied func(assetID int64, changed bool)) (dismiss func()) {
	if win == nil || grid == nil {
		return func() {}
	}
	if onReviewDataChanged == nil {
		onReviewDataChanged = func() {}
	}
	if onRejectApplied == nil {
		onRejectApplied = func(int64, bool) {}
	}
	grid.mu.Lock()
	total := grid.total
	grid.mu.Unlock()
	if total <= 0 || startIdx < 0 || int64(startIdx) >= total {
		return func() {}
	}

	cnv := win.Canvas()
	var closed atomic.Bool
	var imgGen atomic.Uint64

	img := canvas.NewImageFromFile("")
	img.FillMode = canvas.ImageFillContain

	errLbl := widget.NewLabel(reviewGridMsgDecodeFail)
	errLbl.Alignment = fyne.TextAlignCenter
	errLbl.Wrapping = fyne.TextWrapWord
	errLbl.Hide()

	imgStack := container.NewStack(img, container.NewCenter(errLbl))
	imgArea := container.New(&loupeImageLayout{}, imgStack)

	prevBtn := widget.NewButton("← Prev", nil)
	nextBtn := widget.NewButton("Next →", nil)
	closeBtn := widget.NewButton("Close", nil)

	ratingBtns := make([]*widget.Button, 5)
	ratingBox := container.NewHBox()
	for i := range ratingBtns {
		stars := strconv.Itoa(i+1) + "★"
		ratingBtns[i] = widget.NewButton(stars, nil)
		ratingBox.Add(ratingBtns[i])
	}

	tagEntry := widget.NewSelectEntry([]string{})
	tagAdd := widget.NewButton("Add tag", nil)
	tagRem := widget.NewButton("Remove tag", nil)
	tagsLbl := widget.NewLabel("")

	newAlbumLoupeBtn := widget.NewButton("New album…", nil)
	albumChecksBox := container.NewVBox()
	albumScroll := container.NewVScroll(albumChecksBox)
	albumScroll.SetMinSize(fyne.NewSize(80, 100))

	var suppressColl atomic.Bool
	var collWriteMu sync.Mutex

	var rebuildAlbumStrip func(photoID int64)
	rebuildAlbumStrip = func(photoID int64) {
		albumChecksBox.RemoveAll()
		if photoID <= 0 {
			albumChecksBox.Add(widget.NewLabel("—"))
			albumChecksBox.Refresh()
			return
		}
		collRows, err := store.ListCollections(grid.db)
		if err != nil {
			slog.Error("review loupe: list collections", "err", err)
			albumChecksBox.Add(widget.NewLabel("Could not load albums."))
			albumChecksBox.Refresh()
			return
		}
		if len(collRows) == 0 {
			albumChecksBox.Add(widget.NewLabel("No albums yet — use New album…"))
			albumChecksBox.Refresh()
			return
		}
		memberIDs, err := store.ListCollectionIDsForAsset(grid.db, photoID)
		if err != nil {
			slog.Error("review loupe: list memberships", "err", err)
			albumChecksBox.Add(widget.NewLabel("Could not load album membership."))
			albumChecksBox.Refresh()
			return
		}
		member := make(map[int64]struct{}, len(memberIDs))
		for _, id := range memberIDs {
			member[id] = struct{}{}
		}
		for _, row := range collRows {
			collID, title := row.ID, row.Name
			chk := widget.NewCheck(title, nil)
			_, on := member[collID]
			suppressColl.Store(true)
			chk.SetChecked(on)
			suppressColl.Store(false)
			chk.OnChanged = func(want bool) {
				if suppressColl.Load() || closed.Load() {
					return
				}
				go func(want bool, cid int64) {
					collWriteMu.Lock()
					defer collWriteMu.Unlock()
					var opErr error
					if want {
						opErr = store.LinkAssetsToCollection(grid.db, cid, []int64{photoID})
					} else {
						opErr = store.UnlinkAssetFromCollection(grid.db, photoID, cid)
					}
					fyne.Do(func() {
						if opErr != nil {
							slog.Error("review loupe: collection toggle", "err", opErr)
							dialog.ShowError(errors.New(userFacingCollectionWriteErrText(opErr)), win)
							suppressColl.Store(true)
							chk.SetChecked(!want)
							suppressColl.Store(false)
							return
						}
						onReviewDataChanged()
						rebuildAlbumStrip(photoID)
					})
				}(want, collID)
			}
			albumChecksBox.Add(chk)
		}
		albumChecksBox.Refresh()
	}

	rejectBtn := widget.NewButton("Reject photo", nil)
	rejectBtn.Importance = widget.WarningImportance

	shareBtn := widget.NewButton("Share…", nil)

	deleteBtn := widget.NewButton("Move to library trash…", nil)
	deleteBtn.Importance = widget.DangerImportance

	albumHeader := container.NewHBox(widget.NewLabel("Albums"), layout.NewSpacer(), newAlbumLoupeBtn)

	top := container.NewVBox(
		container.NewHBox(prevBtn, layout.NewSpacer(), ratingBox, layout.NewSpacer(), shareBtn, rejectBtn, deleteBtn, nextBtn, closeBtn),
		container.NewHBox(tagEntry, tagAdd, tagRem),
		tagsLbl,
		widget.NewSeparator(),
		albumHeader,
		albumScroll,
	)
	root := container.NewBorder(top, nil, nil, nil, imgArea)

	pop := widget.NewModalPopUp(root, cnv)

	var idx = startIdx
	var currentID int64

	deleteShortcut := &desktop.CustomShortcut{KeyName: fyne.KeyD, Modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift}
	shareShortcut := &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift}

	newAlbumLoupeBtn.OnTapped = func() {
		if !loupeRatingKeyAllowed(currentID) {
			dialog.ShowInformation("New album", "No photo is loaded.", win)
			return
		}
		pid := currentID
		var albumSaving atomic.Bool
		nameEntry := widget.NewEntry()
		dateEntry := widget.NewEntry()
		dateEntry.SetPlaceHolder("YYYY-MM-DD (optional)")
		errLbl := widget.NewLabel("")
		errLbl.Wrapping = fyne.TextWrapWord
		form := container.NewVBox(
			widget.NewForm(
				&widget.FormItem{Text: "Name", Widget: nameEntry},
				&widget.FormItem{Text: "Display date", Widget: dateEntry},
			),
			errLbl,
		)
		d := dialog.NewCustomWithoutButtons("New album", form, win)
		save := widget.NewButton("Save", func() {
			errLbl.SetText("")
			if !albumSaving.CompareAndSwap(false, true) {
				return
			}
			defer albumSaving.Store(false)
			_, err := store.CreateCollectionAndLinkAssets(grid.db, nameEntry.Text, dateEntry.Text, []int64{pid})
			if err != nil {
				errLbl.SetText(userFacingCollectionWriteErrText(err))
				return
			}
			d.Hide()
			onReviewDataChanged()
			rebuildAlbumStrip(pid)
		})
		cancel := widget.NewButton("Cancel", func() { d.Hide() })
		d.SetButtons([]fyne.CanvasObject{cancel, save})
		d.Show()
	}

	cleanup := func() {
		if !closed.CompareAndSwap(false, true) {
			return
		}
		cnv.RemoveShortcut(deleteShortcut)
		cnv.RemoveShortcut(shareShortcut)
		cnv.SetOnTypedRune(nil)
		cnv.SetOnTypedKey(nil)
		pop.Hide()
		cnv.Focus(grid.list)
	}

	applyChrome := func() {
		grid.mu.Lock()
		tot := grid.total
		grid.mu.Unlock()
		prevBtn.Disable()
		nextBtn.Disable()
		if tot > 0 {
			if idx > 0 {
				prevBtn.Enable()
			}
			if int64(idx) < tot-1 {
				nextBtn.Enable()
			}
		}
	}

	var loadRow func()
	loadRow = func() {
		defer applyChrome()
		grid.mu.Lock()
		f := grid.filters
		tot := grid.total
		grid.mu.Unlock()
		if tot > 0 && int64(idx) >= tot {
			idx = int(tot - 1)
			if idx < 0 {
				idx = 0
			}
		}
		rows, err := store.ListAssetsForReview(grid.db, f, 1, idx)
		if err != nil || len(rows) == 0 {
			slog.Error("review loupe: load row", "err", err, "idx", idx)
			currentID = 0
			for _, b := range ratingBtns {
				b.OnTapped = nil
				b.Importance = widget.MediumImportance
				b.Disable()
			}
			errLbl.SetText(reviewGridMsgPageLoadFail)
			errLbl.Show()
			img.Hide()
			rebuildAlbumStrip(0)
			return
		}
		row := rows[0]
		currentID = row.ID

		allTags, terr := store.ListTags(grid.db)
		if terr != nil {
			slog.Error("review loupe: list tags", "err", terr)
		} else {
			opts := make([]string, len(allTags))
			for i := range allTags {
				opts[i] = allTags[i].Label
			}
			tagEntry.SetOptions(opts)
		}
		utags, uerr := store.ListTagsUnionForAssets(grid.db, []int64{row.ID})
		if uerr != nil {
			slog.Error("review loupe: asset tags", "err", uerr)
			tagsLbl.SetText("Tags: —")
		} else {
			lbls := make([]string, len(utags))
			for i := range utags {
				lbls[i] = utags[i].Label
			}
			tagsLbl.SetText("Tags: " + strings.Join(lbls, ", "))
		}

		tagAdd.OnTapped = func() {
			if !loupeRatingKeyAllowed(currentID) {
				dialog.ShowInformation("Add tag", "No photo is loaded.", win)
				return
			}
			tid, err := store.FindOrCreateTagByLabel(grid.db, tagEntry.Text)
			if err != nil {
				slog.Error("review loupe: add tag", "err", err)
				dialog.ShowError(errors.New(userFacingDialogErrText(err)), win)
				return
			}
			if err := store.LinkTagToAssets(grid.db, tid, []int64{currentID}); err != nil {
				slog.Error("review loupe: link tag", "err", err)
				dialog.ShowError(errors.New(userFacingDialogErrText(err)), win)
				return
			}
			onReviewDataChanged()
			loadRow()
		}
		tagRem.OnTapped = func() {
			if !loupeRatingKeyAllowed(currentID) {
				dialog.ShowInformation("Remove tag", "No photo is loaded.", win)
				return
			}
			tid, ok, err := store.FindTagByLabel(grid.db, tagEntry.Text)
			if err != nil {
				slog.Error("review loupe: remove tag", "err", err)
				dialog.ShowError(errors.New(userFacingDialogErrText(err)), win)
				return
			}
			if !ok {
				dialog.ShowInformation("Remove tag", "No tag matches that label in the library.", win)
				return
			}
			if err := store.UnlinkTagFromAssets(grid.db, tid, []int64{currentID}); err != nil {
				slog.Error("review loupe: unlink tag", "err", err)
				dialog.ShowError(errors.New(userFacingDialogErrText(err)), win)
				return
			}
			onReviewDataChanged()
			loadRow()
		}

		for _, b := range ratingBtns {
			b.Enable()
		}

		for i, b := range ratingBtns {
			want := i + 1
			b.OnTapped = func() {
				if err := store.UpdateAssetRating(grid.db, currentID, want); err != nil {
					slog.Error("review loupe: rating", "err", err)
					return
				}
				grid.invalidatePages()
				loadRow()
			}
			if row.Rating != nil && *row.Rating == want {
				b.Importance = widget.HighImportance
			} else {
				b.Importance = widget.MediumImportance
			}
		}

		abs := filepath.Join(grid.libraryRoot, filepath.FromSlash(row.RelPath))
		g := imgGen.Add(1)
		img.Show()
		errLbl.Hide()
		img.File = ""
		img.Resource = nil
		img.Refresh()

		go func() {
			if _, statErr := os.Stat(abs); statErr != nil {
				fyne.Do(func() {
					if imgGen.Load() != g {
						return
					}
					img.Hide()
					errLbl.SetText(reviewGridMsgDecodeFail)
					errLbl.Show()
				})
				return
			}
			fyne.Do(func() {
				if imgGen.Load() != g {
					return
				}
				img.File = abs
				img.Show()
				img.Refresh()
			})
		}()

		rebuildAlbumStrip(row.ID)
	}

	nav := func(delta int) {
		grid.mu.Lock()
		tot := grid.total
		grid.mu.Unlock()
		if tot <= 0 {
			return
		}
		// Defensive: if total shrank without going through refreshAll (future callers), stay in range.
		if int64(idx) >= tot {
			idx = int(tot - 1)
			if idx < 0 {
				idx = 0
			}
		}
		next, moved := loupeStepIndex(idx, delta, tot)
		if !moved {
			return
		}
		idx = next
		loadRow()
	}

	prevBtn.OnTapped = func() { nav(-1) }
	nextBtn.OnTapped = func() { nav(1) }
	closeBtn.OnTapped = cleanup

	applyReject := func() {
		if !loupeRatingKeyAllowed(currentID) {
			return
		}
		changed, err := store.RejectAsset(grid.db, currentID, time.Now().Unix())
		if err != nil {
			slog.Error("review loupe: reject", "err", err)
			if win != nil {
				dialog.ShowError(errors.New(userFacingDialogErrText(err)), win)
			}
			return
		}
		onRejectApplied(currentID, changed)
		grid.invalidatePages()
		onReviewDataChanged()
		grid.mu.Lock()
		tot := grid.total
		grid.mu.Unlock()
		if tot <= 0 {
			cleanup()
			return
		}
		if int64(idx) >= tot {
			idx = int(tot - 1)
			if idx < 0 {
				idx = 0
			}
		}
		loadRow()
	}
	rejectBtn.OnTapped = applyReject

	openShare := func() {
		if closed.Load() {
			return
		}
		if !loupeRatingKeyAllowed(currentID) {
			return
		}
		openLoupeShareFlow(win, grid, idx, func() int64 { return currentID })
	}
	shareBtn.OnTapped = openShare

	promptDelete := func() {
		if !loupeRatingKeyAllowed(currentID) {
			return
		}
		dialog.ShowConfirm(
			"Move to library trash?",
			"This removes the photo from your library (stronger than Reject). The file is moved under .trash in your library folder.",
			func(ok bool) {
				if !ok {
					return
				}
				fyne.Do(func() {
					aid := currentID
					_, err := store.DeleteAssetToTrash(grid.db, grid.libraryRoot, aid, time.Now().Unix())
					if err != nil {
						slog.Error("review loupe: delete", "err", err)
						if win != nil {
							dialog.ShowError(errors.New(userFacingDialogErrText(err)), win)
						}
						return
					}
					// Resync even when changed==false (idempotent recall, race with bulk delete) so the loupe
					// does not keep showing a row that default queries already exclude.
					grid.invalidatePages()
					onReviewDataChanged()
					grid.mu.Lock()
					tot := grid.total
					grid.mu.Unlock()
					if tot <= 0 {
						cleanup()
						return
					}
					if int64(idx) >= tot {
						idx = int(tot - 1)
						if idx < 0 {
							idx = 0
						}
					}
					loadRow()
				})
			},
			win,
		)
	}
	deleteBtn.OnTapped = promptDelete

	cnv.AddShortcut(deleteShortcut, func(fyne.Shortcut) {
		if closed.Load() {
			return
		}
		promptDelete()
	})
	cnv.AddShortcut(shareShortcut, func(fyne.Shortcut) {
		openShare()
	})

	cnv.SetOnTypedRune(func(r rune) {
		if closed.Load() {
			return
		}
		if !loupeRatingKeyAllowed(currentID) {
			return
		}
		if r < '1' || r > '5' {
			return
		}
		want := int(r - '0')
		if err := store.UpdateAssetRating(grid.db, currentID, want); err != nil {
			slog.Error("review loupe: rating key", "err", err)
			return
		}
		grid.invalidatePages()
		loadRow()
	})

	cnv.SetOnTypedKey(func(ev *fyne.KeyEvent) {
		if closed.Load() {
			return
		}
		switch ev.Name {
		case fyne.KeyEscape:
			cleanup()
		case fyne.KeyLeft:
			nav(-1)
		case fyne.KeyRight:
			nav(1)
		}
	})

	pop.Resize(cnv.Size())
	pop.Show()
	loadRow()

	return cleanup
}
