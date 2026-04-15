package app

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"photo-tool/internal/domain"
	"photo-tool/internal/share"
	"photo-tool/internal/store"
)

const (
	reviewCollectionSentinel = "No assigned collection"
	reviewRatingAny          = "Any rating"
	reviewTagAny             = "Any tag"
)

func reviewFiltersAtFR16Defaults(f domain.ReviewFilters) bool {
	return f.CollectionID == nil && f.MinRating == nil && f.TagID == nil
}

// ReviewFilterStripSegmentLabels is UX-DR2 / Story 2.2 AC1 order (exported for regression tests).
func ReviewFilterStripSegmentLabels() []string {
	return []string{"Collection", "Minimum rating", "Tags"}
}

// NewReviewView builds the Review surface filter strip, live count, and paged thumbnail grid (Story 2.3).
// registerUndoClear, when non-nil, receives a function that clears the session reject-undo stack when the user
// leaves Review via primary nav (Story 2.6 AC6).
// onGotoUpload switches primary nav to Upload for the library-empty CTA (UX-DR9 / Story 2.12).
// shareLoop enables loopback share URLs after mint (Story 3.2); nil keeps token-only success UI (tests).
// registerCollectionsStripReload, when non-nil, receives refreshReviewData so other panels can reload cached album lists (Story 2.9 AC6).
func NewReviewView(win fyne.Window, db *sql.DB, libraryRoot string, registerUndoClear func(clear func()), onGotoUpload func(), shareLoop *share.Loopback, registerCollectionsStripReload func(reload func())) fyne.CanvasObject {
	if db == nil {
		return newReviewViewWithoutDB()
	}

	undoStack := &reviewRejectUndoStack{}

	collectionIDs := []*int64{nil}
	collectionOpts := []string{reviewCollectionSentinel}
	assignTargetIDs := []*int64(nil)
	assignTargetOpts := []string(nil)

	var listErr error

	ratingOpts := []string{reviewRatingAny, "1", "2", "3", "4", "5"}

	tagIDs := []*int64{nil}
	tagOpts := []string{reviewTagAny}

	countLabel := widget.NewLabel("Matching assets: —")
	emptyExplain := widget.NewLabel("")
	emptyExplain.Wrapping = fyne.TextWrapWord
	emptyPrimary := widget.NewButton("", nil)
	emptyPrimary.Importance = widget.HighImportance
	emptyBlock := container.NewVBox(emptyExplain, emptyPrimary)
	emptyBlock.Hide()
	undoRejectBtn := widget.NewButton("Undo reject", nil)
	undoRejectBtn.Hide()
	undoSessionHint := widget.NewLabel("")
	undoSessionHint.TextStyle = fyne.TextStyle{Italic: true}
	undoSessionHint.Wrapping = fyne.TextWrapWord
	undoSessionHint.Hide()
	// Bulk reject matches bulk tag semantics: every selected asset is rejected (Story 2.6). Deterministic undo order: sorted ascending id.
	rejectSelectedBtn := widget.NewButton("Reject selected photos", nil)
	rejectSelectedBtn.Importance = widget.WarningImportance
	deleteSelectedBtn := widget.NewButton("Delete selected…", nil)
	deleteSelectedBtn.Importance = widget.DangerImportance
	sharePkgSelBtn := widget.NewButton("Share selection as package…", nil)
	sharePkgFilterBtn := widget.NewButton("Share filtered set as package…", nil)

	var colSel, minRatingSel, tagsSel *widget.Select
	// tagsSel.OnChanged must not call refreshAll while we rebuild options (SetSelected would recurse).
	var suspendTagSelectRefresh bool
	var suspendColSelectRefresh bool
	var suspendMinRatingSelectRefresh bool
	var suspendAssignTargetRefresh bool
	var resetFiltersToFR16 func()
	tagEntry := widget.NewSelectEntry([]string{})
	tagAddBtn := widget.NewButton("Add tag to selection", nil)
	tagRemBtn := widget.NewButton("Remove tag from selection", nil)
	tagSummaryLabel := widget.NewLabel("")
	tagSummaryLabel.Wrapping = fyne.TextWrapWord
	bulkHint := widget.NewLabel("Cmd/Ctrl+click thumbnails to select multiple photos for bulk tagging.")
	bulkHint.Wrapping = fyne.TextWrapWord
	assignTargetSel := widget.NewSelect([]string{}, func(string) {
		if suspendAssignTargetRefresh {
			return
		}
	})
	assignToColBtn := widget.NewButton("Assign selection to album", nil)
	assignBulkHint := widget.NewLabel("")
	assignBulkHint.TextStyle = fyne.TextStyle{Italic: true}
	assignBulkHint.Wrapping = fyne.TextWrapWord // long album/assign hints must not force NFR-01 min width overflow

	// Fyne Select resolves by visible label. Duplicate collection or tag names map to the first
	// matching option (ambiguous; treat as data-quality until names are disambiguated in store/UI).
	buildFilters := func() domain.ReviewFilters {
		var f domain.ReviewFilters
		if colSel != nil && colSel.Selected != "" {
			for i, opt := range collectionOpts {
				if opt == colSel.Selected && i < len(collectionIDs) {
					f.CollectionID = collectionIDs[i]
					break
				}
			}
		}
		if minRatingSel != nil {
			switch minRatingSel.Selected {
			case "", reviewRatingAny:
				// nil
			default:
				if n, aerr := strconv.Atoi(minRatingSel.Selected); aerr == nil {
					f.MinRating = &n
				}
			}
		}
		if tagsSel != nil && tagsSel.Selected != "" {
			for i, opt := range tagOpts {
				if opt == tagsSel.Selected && i < len(tagIDs) {
					f.TagID = tagIDs[i]
					break
				}
			}
		}
		return f
	}

	syncTagStrip := func() error {
		tags, err := store.ListTags(db)
		if err != nil {
			return err
		}
		sel := ""
		if tagsSel != nil {
			sel = tagsSel.Selected
		}
		var idBefore *int64
		if sel != "" && sel != reviewTagAny {
			for i, o := range tagOpts {
				if o == sel && i < len(tagIDs) {
					idBefore = tagIDs[i]
					break
				}
			}
		}
		nextIDs := []*int64{nil}
		nextOpts := []string{reviewTagAny}
		for _, t := range tags {
			tid := t.ID
			nextIDs = append(nextIDs, &tid)
			nextOpts = append(nextOpts, t.Label)
		}
		tagIDs = nextIDs
		tagOpts = nextOpts
		if tagsSel != nil {
			tagsSel.Options = nextOpts
			newSel := reviewTagAny
			// Stale TagID (e.g. tag row removed): id no longer in ListTags → fall back to "Any tag" + refresh via caller.
			if idBefore != nil {
				for i, idPtr := range tagIDs {
					if idPtr != nil && *idPtr == *idBefore {
						newSel = tagOpts[i]
						break
					}
				}
			}
			suspendTagSelectRefresh = true
			defer func() { suspendTagSelectRefresh = false }()
			tagsSel.SetSelected(newSel)
		}
		return nil
	}

	var dismissLoupe func()
	var grid *reviewAssetGrid
	var refreshBulkTagUI func()

	syncUndoUI := func() {
		n := undoStack.Len()
		if n <= 0 {
			undoRejectBtn.Hide()
			undoSessionHint.Hide()
			return
		}
		undoRejectBtn.Show()
		// UX-DR8 / AC8: transient undo; cap is maxReviewRejectUndoIDs (see reject_undo_stack.go).
		undoSessionHint.SetText("Session undo resets when you leave Review. In very long sessions, the oldest steps may drop off the undo list.")
		undoSessionHint.Show()
		if n == 1 {
			undoRejectBtn.SetText("Undo reject")
		} else {
			undoRejectBtn.SetText(fmt.Sprintf("Undo reject (%d)", n))
		}
	}
	if registerUndoClear != nil {
		registerUndoClear(func() {
			undoStack.Clear()
			syncUndoUI()
		})
	}

	refreshReviewData := func() {
		tagStripSyncErr := false
		if err := syncTagStrip(); err != nil {
			slog.Error("review: sync tag strip", "err", err)
			tagStripSyncErr = true
		}
		cols, colErr := store.ListCollections(db)
		if colErr != nil {
			listErr = colErr
			assignTargetIDs = nil
			assignTargetOpts = nil
			if assignTargetSel != nil {
				suspendAssignTargetRefresh = true
				assignTargetSel.Options = []string{}
				assignTargetSel.ClearSelected()
				suspendAssignTargetRefresh = false
				assignTargetSel.Disable()
			}
		} else {
			listErr = nil
			var prevID *int64
			if colSel != nil && colSel.Selected != "" {
				for i, opt := range collectionOpts {
					if opt == colSel.Selected && i < len(collectionIDs) {
						prevID = collectionIDs[i]
						break
					}
				}
			}
			var prevAssignID *int64
			if assignTargetSel != nil && assignTargetSel.Selected != "" {
				for i, o := range assignTargetOpts {
					if o == assignTargetSel.Selected && i < len(assignTargetIDs) {
						if assignTargetIDs[i] != nil {
							v := *assignTargetIDs[i]
							prevAssignID = &v
						}
						break
					}
				}
			}
			collectionIDs = []*int64{nil}
			collectionOpts = []string{reviewCollectionSentinel}
			assignTargetIDs = nil
			assignTargetOpts = nil
			for i := range cols {
				c := cols[i]
				id := c.ID
				collectionIDs = append(collectionIDs, &id)
				collectionOpts = append(collectionOpts, c.Name)
				assignTargetIDs = append(assignTargetIDs, &id)
				assignTargetOpts = append(assignTargetOpts, c.Name)
			}
			if colSel != nil {
				colSel.Options = collectionOpts
				newSel := reviewCollectionSentinel
				if prevID != nil {
					found := false
					for i, idPtr := range collectionIDs {
						if idPtr != nil && *idPtr == *prevID {
							newSel = collectionOpts[i]
							found = true
							break
						}
					}
					if !found {
						newSel = reviewCollectionSentinel
					}
				}
				suspendColSelectRefresh = true
				colSel.SetSelected(newSel)
				suspendColSelectRefresh = false
			}
			if assignTargetSel != nil {
				assignTargetSel.Options = assignTargetOpts
				newPick := ""
				if len(assignTargetOpts) > 0 {
					newPick = assignTargetOpts[0]
					if prevAssignID != nil {
						for i, idPtr := range assignTargetIDs {
							if idPtr != nil && *idPtr == *prevAssignID {
								newPick = assignTargetOpts[i]
								break
							}
						}
					}
				}
				suspendAssignTargetRefresh = true
				assignTargetSel.SetSelected(newPick)
				suspendAssignTargetRefresh = false
				if len(assignTargetOpts) == 0 {
					assignTargetSel.Disable()
				} else {
					assignTargetSel.Enable()
				}
			}
		}
		f := buildFilters()
		n, qerr := store.CountAssetsForReview(db, f)
		if qerr != nil {
			msg := fmt.Sprintf("Matching assets: — (%s)", libraryErrText(qerr))
			if tagStripSyncErr {
				msg += "; could not refresh tag list"
			}
			if listErr != nil {
				msg += "; " + libraryErrText(listErr)
			}
			countLabel.SetText(msg)
			emptyBlock.Hide()
			grid.reset(f, 0)
			if refreshBulkTagUI != nil {
				refreshBulkTagUI()
			}
			return
		}
		msg := fmt.Sprintf("Matching assets: %d", n)
		if listErr != nil {
			msg += " (collections unavailable — " + libraryErrText(listErr) + ")"
		}
		if tagStripSyncErr {
			msg += " — Could not refresh tag list"
		}
		if f.TagID != nil && n == 0 {
			msg += " — No photos with this tag"
		}
		countLabel.SetText(msg)
		if n == 0 {
			emptyBlock.Show()
			if reviewFiltersAtFR16Defaults(f) {
				emptyExplain.SetText("Your library has no photos to show yet. Add photos from your computer to start reviewing.")
				emptyPrimary.SetText("Go to Upload")
				emptyPrimary.OnTapped = func() {
					if onGotoUpload != nil {
						onGotoUpload()
					}
				}
			} else {
				emptyExplain.SetText("No photos match the current filters. Your library may still have photos — the choices above are hiding them.")
				emptyPrimary.SetText("Reset filters")
				emptyPrimary.OnTapped = func() {
					if resetFiltersToFR16 != nil {
						resetFiltersToFR16()
					}
				}
			}
		} else {
			emptyBlock.Hide()
		}
		grid.reset(f, n)
		if refreshBulkTagUI != nil {
			refreshBulkTagUI()
		}
	}

	if registerCollectionsStripReload != nil {
		registerCollectionsStripReload(refreshReviewData)
	}

	refreshAll := func() {
		if dismissLoupe != nil {
			dismissLoupe()
			dismissLoupe = nil
		}
		refreshReviewData()
	}

	var assignCollMu sync.Mutex
	openCellAssign := func(anchor fyne.CanvasObject, assetID int64, at fyne.Position) {
		if assetID <= 0 || win == nil {
			return
		}
		collRows, err := store.ListCollections(db)
		if err != nil {
			slog.Error("review: list collections for quick assign", "err", err)
			dialog.ShowError(errors.New("album list is unavailable — try again after the library loads"), win)
			return
		}
		if len(collRows) == 0 {
			dialog.ShowInformation("Assign to album", "Create an album first (Collections or the review loupe).", win)
			return
		}
		cvs := fyne.CurrentApp().Driver().CanvasForObject(anchor)
		if cvs == nil {
			return
		}
		items := make([]*fyne.MenuItem, 0, len(collRows))
		for _, row := range collRows {
			cid := row.ID
			lbl := row.Name
			items = append(items, fyne.NewMenuItem(lbl, func() {
				// AC8: same as bulk assign / reject — dismiss loupe before persistence, not after success.
				if dismissLoupe != nil {
					dismissLoupe()
					dismissLoupe = nil
				}
				go func(collectionID int64, aid int64) {
					assignCollMu.Lock()
					defer assignCollMu.Unlock()
					linkErr := store.LinkAssetsToCollection(db, collectionID, []int64{aid})
					fyne.Do(func() {
						if linkErr != nil {
							slog.Error("review: quick assign link", "err", linkErr)
							dialog.ShowError(errors.New(userFacingCollectionWriteErrText(linkErr)), win)
							return
						}
						refreshReviewData()
					})
				}(cid, assetID)
			}))
		}
		widget.ShowPopUpMenuAtPosition(fyne.NewMenu("", items...), cvs, at)
	}

	grid = newReviewAssetGrid(win, db, libraryRoot, func(idx int) {
		if win == nil || grid == nil {
			return
		}
		if dismissLoupe != nil {
			dismissLoupe()
			dismissLoupe = nil
		}
		dismissLoupe = openReviewLoupe(win, grid, idx, refreshReviewData, func(assetID int64, changed bool) {
			if changed {
				undoStack.Push(assetID)
			}
			syncUndoUI()
		})
	}, func() {
		if refreshBulkTagUI != nil {
			refreshBulkTagUI()
		}
	}, false, nil, openCellAssign, shareLoop)

	assignToColBtn.OnTapped = func() {
		if grid == nil {
			return
		}
		ids := grid.SelectedAssetIDs()
		if len(ids) == 0 {
			return
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		var targetID int64
		found := false
		if assignTargetSel != nil && assignTargetSel.Selected != "" {
			for i, o := range assignTargetOpts {
				if o == assignTargetSel.Selected && i < len(assignTargetIDs) && assignTargetIDs[i] != nil {
					targetID = *assignTargetIDs[i]
					found = true
					break
				}
			}
		}
		if !found {
			if win != nil {
				dialog.ShowInformation("Assign to album", "Choose which album to add the selection to.", win)
			}
			return
		}
		if dismissLoupe != nil {
			dismissLoupe()
			dismissLoupe = nil
		}
		go func(sel []int64, tid int64) {
			assignCollMu.Lock()
			defer assignCollMu.Unlock()
			linkErr := store.LinkAssetsToCollection(db, tid, sel)
			fyne.Do(func() {
				if linkErr != nil {
					slog.Error("review: bulk assign link", "err", linkErr)
					dialog.ShowError(errors.New(userFacingCollectionWriteErrText(linkErr)), win)
					return
				}
				refreshReviewData()
			})
		}(ids, targetID)
	}

	rejectSelectedBtn.OnTapped = func() {
		if grid == nil {
			return
		}
		ids := grid.SelectedAssetIDs()
		if len(ids) == 0 {
			return
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		at := time.Now().Unix()
		// Best-effort batch: first DB error aborts; rejects already applied stay undoable (no transaction wrapping).
		// Dismiss loupe only — avoid refreshAll before writes (extra grid reset / double fetch).
		if dismissLoupe != nil {
			dismissLoupe()
			dismissLoupe = nil
		}
		for _, id := range ids {
			changed, err := store.RejectAsset(db, id, at)
			if err != nil {
				slog.Error("review: reject asset", "err", err)
				if win != nil {
					dialog.ShowError(errors.New(userFacingDialogErrText(err)), win)
				}
				refreshReviewData()
				syncUndoUI()
				return
			}
			if changed {
				undoStack.Push(id)
			}
		}
		refreshReviewData()
		syncUndoUI()
	}

	deleteSelectedBtn.OnTapped = func() {
		if grid == nil {
			return
		}
		ids := grid.SelectedAssetIDs()
		if len(ids) == 0 {
			return
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		title := "Move 1 photo to library trash?"
		if len(ids) != 1 {
			title = fmt.Sprintf("Move %d photos to library trash?", len(ids))
		}
		dialog.ShowConfirm(
			title,
			"This removes the selected photos from your library (stronger than Reject). Files are moved under .trash in your library folder.",
			func(ok bool) {
				if !ok {
					return
				}
				fyne.Do(func() {
					if dismissLoupe != nil {
						dismissLoupe()
						dismissLoupe = nil
					}
					at := time.Now().Unix()
					for _, id := range ids {
						_, err := store.DeleteAssetToTrash(db, libraryRoot, id, at)
						if err != nil {
							slog.Error("review: delete asset", "err", err)
							if win != nil {
								dialog.ShowError(errors.New(userFacingDialogErrText(err)), win)
							}
							refreshReviewData()
							return
						}
					}
					refreshReviewData()
				})
			},
			win,
		)
	}

	undoRejectBtn.OnTapped = func() {
		id, ok := undoStack.Pop()
		if !ok {
			syncUndoUI()
			return
		}
		changed, err := store.RestoreAsset(db, id)
		if err != nil {
			slog.Error("review: undo restore", "err", err)
			if win != nil {
				dialog.ShowError(errors.New(userFacingDialogErrText(err)), win)
			}
			undoStack.Push(id)
			syncUndoUI()
			return
		}
		if !changed {
			slog.Debug("review: undo restore no-op", "asset_id", id)
		}
		// changed==false is OK (e.g. asset already restored elsewhere): stack entry still consumed.
		refreshReviewData()
		syncUndoUI()
	}

	refreshBulkTagUI = func() {
		defer func() {
			if grid != nil {
				if len(grid.SelectedAssetIDs()) == 0 {
					sharePkgSelBtn.Disable()
				} else {
					sharePkgSelBtn.Enable()
				}
			}
			if assignTargetSel == nil || grid == nil {
				return
			}
			if listErr != nil {
				assignTargetSel.Disable()
				assignToColBtn.Disable()
				assignBulkHint.SetText("Album list is unavailable — try again after the library loads.")
				return
			}
			if len(assignTargetOpts) == 0 {
				assignTargetSel.Disable()
				assignToColBtn.Disable()
				assignBulkHint.SetText("No albums yet — create one in Collections or the review loupe before assigning.")
				return
			}
			assignTargetSel.Enable()
			if len(grid.SelectedAssetIDs()) == 0 {
				assignToColBtn.Disable()
				assignBulkHint.SetText("Select photos (Cmd/Ctrl+click), choose an album, then assign — or right-click a thumbnail.")
			} else {
				assignToColBtn.Enable()
				assignBulkHint.SetText("")
			}
		}()
		entryOpts := make([]string, 0, len(tagOpts)-1)
		for _, o := range tagOpts {
			if o != reviewTagAny {
				entryOpts = append(entryOpts, o)
			}
		}
		tagEntry.SetOptions(entryOpts)

		ids := grid.SelectedAssetIDs()
		if len(ids) == 0 {
			tagSummaryLabel.SetText("No photos selected for bulk tagging. Cmd/Ctrl+click thumbnails to select, or open the loupe to tag the current photo.")
			tagAddBtn.Disable()
			tagRemBtn.Disable()
			return
		}
		u, err := store.ListTagsUnionForAssets(db, ids)
		if err != nil {
			slog.Error("review: list tags for selection", "err", err)
			tagSummaryLabel.SetText("Could not load tags for the current selection.")
			tagAddBtn.Disable()
			tagRemBtn.Disable()
			return
		}
		lbls := make([]string, len(u))
		for i := range u {
			lbls[i] = u[i].Label
		}
		tagSummaryLabel.SetText("Tags on selection (union): " + strings.Join(lbls, ", "))
		tagAddBtn.Enable()
		tagRemBtn.Enable()
	}

	tagAddBtn.OnTapped = func() {
		tid, err := store.FindOrCreateTagByLabel(db, tagEntry.Text)
		if err != nil {
			slog.Error("review: add tag", "err", err)
			if win != nil {
				dialog.ShowError(errors.New(userFacingDialogErrText(err)), win)
			}
			return
		}
		ids := grid.SelectedAssetIDs()
		if len(ids) == 0 {
			return
		}
		// Bulk semantics (Story 2.5): Add links this tag to every selected asset if not already linked (idempotent).
		if err := store.LinkTagToAssets(db, tid, ids); err != nil {
			slog.Error("review: link tag to assets", "err", err)
			if win != nil {
				dialog.ShowError(errors.New(userFacingDialogErrText(err)), win)
			}
			return
		}
		refreshReviewData()
	}

	sharePkgSelBtn.OnTapped = func() {
		if grid == nil || win == nil {
			return
		}
		openPackageShareFromReview(win, grid, grid.SelectedAssetIDs(), nil)
	}
	sharePkgFilterBtn.OnTapped = func() {
		if grid == nil || win == nil {
			return
		}
		f := buildFilters()
		ids, err := store.ListAssetIDsForReview(db, f)
		openPackageShareFromReview(win, grid, ids, err)
	}

	tagRemBtn.OnTapped = func() {
		tid, ok, err := store.FindTagByLabel(db, tagEntry.Text)
		if err != nil {
			slog.Error("review: remove tag lookup", "err", err)
			if win != nil {
				dialog.ShowError(errors.New(userFacingDialogErrText(err)), win)
			}
			return
		}
		if !ok {
			if win != nil {
				dialog.ShowInformation("Remove tag", "No tag matches that label in the library.", win)
			}
			return
		}
		ids := grid.SelectedAssetIDs()
		if len(ids) == 0 {
			return
		}
		// Remove unlinks this tag from every selected asset (idempotent for assets without the link).
		if err := store.UnlinkTagFromAssets(db, tid, ids); err != nil {
			slog.Error("review: unlink tag from assets", "err", err)
			if win != nil {
				dialog.ShowError(errors.New(userFacingDialogErrText(err)), win)
			}
			return
		}
		refreshReviewData()
	}

	colSel = widget.NewSelect(collectionOpts, func(string) {
		if suspendColSelectRefresh {
			return
		}
		refreshAll()
	})
	colSel.SetSelected(reviewCollectionSentinel)

	minRatingSel = widget.NewSelect(ratingOpts, func(string) {
		if suspendMinRatingSelectRefresh {
			return
		}
		refreshAll()
	})
	minRatingSel.SetSelected(reviewRatingAny)

	tagsSel = widget.NewSelect(tagOpts, func(string) {
		if suspendTagSelectRefresh {
			return
		}
		refreshAll()
	})
	tagsSel.SetSelected(reviewTagAny)

	resetFiltersToFR16 = func() {
		suspendColSelectRefresh = true
		suspendMinRatingSelectRefresh = true
		suspendTagSelectRefresh = true
		colSel.SetSelected(reviewCollectionSentinel)
		minRatingSel.SetSelected(reviewRatingAny)
		tagsSel.SetSelected(reviewTagAny)
		suspendColSelectRefresh = false
		suspendMinRatingSelectRefresh = false
		suspendTagSelectRefresh = false
		refreshAll()
	}

	segLabels := ReviewFilterStripSegmentLabels()
	strip := container.NewHBox(
		container.NewHBox(widget.NewLabel(segLabels[0]), colSel),
		widget.NewSeparator(),
		container.NewHBox(widget.NewLabel(segLabels[1]), minRatingSel),
		widget.NewSeparator(),
		container.NewHBox(widget.NewLabel(segLabels[2]), tagsSel),
	)

	// Stack assign controls vertically so NFR-01 min width (1024) keeps album target + button on-screen
	// without horizontal shell scroll (Story 2.11).
	assignTargetScroll := container.NewHScroll(assignTargetSel)
	assignBar := container.NewVBox(
		widget.NewLabel("Assign selection"),
		assignTargetScroll,
		assignToColBtn,
	)
	// Break bulk/tag/share rows so 1024px windows do not clip primary actions (NFR-01 floor).
	pkgShareRow := container.NewVBox(
		container.NewHBox(sharePkgSelBtn),
		container.NewHBox(sharePkgFilterBtn),
	)
	tagEntryScroll := container.NewHScroll(tagEntry)
	tagRow := container.NewVBox(
		container.NewHBox(tagAddBtn, tagRemBtn),
		tagEntryScroll,
		container.NewHBox(rejectSelectedBtn, deleteSelectedBtn),
	)
	tagBar := container.NewVBox(
		bulkHint,
		pkgShareRow,
		tagRow,
		assignBar,
		assignBulkHint,
		tagSummaryLabel,
	)

	refreshAll()

	undoCluster := container.NewVBox(undoRejectBtn, undoSessionHint)
	countRow := container.NewHBox(countLabel, layout.NewSpacer(), undoCluster)
	body := container.NewBorder(
		countRow,
		nil, nil, nil,
		container.NewVBox(emptyBlock, container.NewScroll(grid.canvasObject())),
	)
	return container.NewPadded(container.NewVBox(strip, widget.NewSeparator(), tagBar, widget.NewSeparator(), body))
}

func newReviewViewWithoutDB() fyne.CanvasObject {
	const nilDBMsg = "Matching assets: — (no database)"
	countLabel := widget.NewLabel(nilDBMsg)
	gridHint := widget.NewLabel("Thumbnail grid needs an open library database.")
	gridHint.Wrapping = fyne.TextWrapWord

	ratingOpts := []string{reviewRatingAny, "1", "2", "3", "4", "5"}
	tagOpts := []string{reviewTagAny}
	collectionOpts := []string{reviewCollectionSentinel}

	noop := func() { countLabel.SetText(nilDBMsg) }

	colSel := widget.NewSelect(collectionOpts, func(string) { noop() })
	colSel.SetSelected(reviewCollectionSentinel)

	minRatingSel := widget.NewSelect(ratingOpts, func(string) { noop() })
	minRatingSel.SetSelected(reviewRatingAny)

	tagsSel := widget.NewSelect(tagOpts, func(string) { noop() })
	tagsSel.SetSelected(reviewTagAny)

	segLabels := ReviewFilterStripSegmentLabels()
	strip := container.NewHBox(
		container.NewHBox(widget.NewLabel(segLabels[0]), colSel),
		widget.NewSeparator(),
		container.NewHBox(widget.NewLabel(segLabels[1]), minRatingSel),
		widget.NewSeparator(),
		container.NewHBox(widget.NewLabel(segLabels[2]), tagsSel),
	)

	body := container.NewVBox(countLabel, gridHint)
	return container.NewPadded(container.NewVBox(strip, widget.NewSeparator(), body))
}
