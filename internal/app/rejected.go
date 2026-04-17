package app

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"photo-tool/internal/domain"
	"photo-tool/internal/store"
)

// NewRejectedView lists rejected (hidden) assets with the same filter suffix as default Review (Story 2.6 AC3).
// onGotoReview switches primary nav to Review for the empty-state CTA (UX-DR9).
func NewRejectedView(win fyne.Window, db *sql.DB, libraryRoot string, onGotoReview func()) fyne.CanvasObject {
	if db == nil {
		return widget.NewLabel("Rejected: — (no database)")
	}

	collectionIDs := []*int64{nil}
	collectionOpts := []string{reviewCollectionSentinel}

	var listErr error
	cols, err := store.ListCollections(db)
	if err != nil {
		listErr = err
	} else {
		for i := range cols {
			c := cols[i]
			id := c.ID
			collectionIDs = append(collectionIDs, &id)
			collectionOpts = append(collectionOpts, c.Name)
		}
	}

	ratingOpts := []string{reviewRatingAny, "1", "2", "3", "4", "5"}

	tagIDs := []*int64{nil}
	tagOpts := []string{reviewTagAny}

	countLabel := widget.NewLabel("Rejected: —")
	deleteSelectedBtn := widget.NewButton("Delete selected…", nil)
	deleteSelectedBtn.Importance = widget.MediumImportance
	deleteSelectedBtn.Disable()
	bulkHint := widget.NewLabel("Cmd/Ctrl+click thumbnails to select multiple rejected photos for bulk delete.")
	emptyHint := widget.NewLabel("")
	emptyHint.Wrapping = fyne.TextWrapWord
	emptyHint.Hide()
	backToReview := widget.NewButton("Back to Review", func() {
		if onGotoReview != nil {
			onGotoReview()
		}
	})
	backToReview.Hide()
	resetFiltersBtn := widget.NewButton("Reset filters", nil)
	resetFiltersBtn.Hide()

	var colSel, minRatingSel, tagsSel *widget.Select
	var suspendTagSelectRefresh bool
	var suspendColSelectRefresh bool
	var suspendMinRatingSelectRefresh bool
	var resetRejectedFiltersToFR16 func()

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

	var grid *reviewAssetGrid
	var gridScroll *container.Scroll
	var zeroMatchPlate fyne.CanvasObject
	var bulkBar *fyne.Container
	var rejectedBulkCue *widget.Label

	rejectedBulkAllowed := true
	var rejectedAssetCount int64

	syncRejectedBulkStrip := func() {
		if bulkBar == nil || rejectedBulkCue == nil || grid == nil {
			return
		}
		if !rejectedBulkAllowed || rejectedAssetCount <= 0 {
			bulkBar.Hide()
			rejectedBulkCue.Hide()
			return
		}
		if len(grid.SelectedAssetIDs()) > 0 {
			bulkBar.Show()
			rejectedBulkCue.Hide()
		} else {
			bulkBar.Hide()
			rejectedBulkCue.Show()
		}
	}

	syncBulkDeleteUI := func() {
		if grid == nil {
			return
		}
		if len(grid.SelectedAssetIDs()) > 0 {
			deleteSelectedBtn.Enable()
			deleteSelectedBtn.Importance = widget.DangerImportance
		} else {
			deleteSelectedBtn.Disable()
			deleteSelectedBtn.Importance = widget.MediumImportance
		}
		deleteSelectedBtn.Refresh()
		syncRejectedBulkStrip()
	}

	refreshRejectedData := func() {
		rejectedBulkAllowed = true
		var tagStripSyncErr error
		if err := syncTagStrip(); err != nil {
			slog.Error("rejected: sync tag strip", "err", err)
			tagStripSyncErr = err
		}
		cols, colErr := store.ListCollections(db)
		if colErr != nil {
			listErr = colErr
			collectionIDs = []*int64{nil}
			collectionOpts = []string{reviewCollectionSentinel}
			if colSel != nil {
				suspendColSelectRefresh = true
				colSel.Options = collectionOpts
				colSel.SetSelected(reviewCollectionSentinel)
				suspendColSelectRefresh = false
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
			collectionIDs = []*int64{nil}
			collectionOpts = []string{reviewCollectionSentinel}
			for i := range cols {
				c := cols[i]
				id := c.ID
				collectionIDs = append(collectionIDs, &id)
				collectionOpts = append(collectionOpts, c.Name)
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
		}
		f := buildFilters()
		n, qerr := store.CountRejectedForReview(db, f)
		if qerr != nil {
			msg := fmt.Sprintf("Rejected: — (%s)", libraryErrText(qerr))
			if listErr != nil {
				msg += "; collections unavailable — " + libraryErrText(listErr)
			}
			if tagStripSyncErr != nil {
				msg += "; tags unavailable — " + libraryErrText(tagStripSyncErr)
			}
			countLabel.SetText(msg)
			emptyHint.Hide()
			backToReview.Show()
			backToReview.Importance = widget.MediumImportance
			backToReview.SetText("Back to Review")
			backToReview.OnTapped = func() {
				if onGotoReview != nil {
					onGotoReview()
				}
			}
			resetFiltersBtn.Hide()
			if zeroMatchPlate != nil {
				zeroMatchPlate.Hide()
			}
			grid.reset(f, 0)
			grid.syncGridScrollVisible(gridScroll, false)
			if gridScroll != nil {
				gridScroll.ScrollToTop()
			}
			rejectedAssetCount = 0
			syncRejectedBulkStrip()
			return
		}
		msg := fmt.Sprintf("Rejected: %d", n)
		if listErr != nil {
			msg += " (collections unavailable — " + libraryErrText(listErr) + ")"
		}
		if tagStripSyncErr != nil {
			msg += " (tags unavailable — " + libraryErrText(tagStripSyncErr) + ")"
		}
		countLabel.SetText(msg)
		if n == 0 {
			resetFiltersBtn.Hide()
			emptyHint.Show()
			backToReview.Show()
			backToReview.Importance = widget.HighImportance
			if reviewFiltersAtFR16Defaults(f) {
				emptyHint.SetText("Nothing rejected yet. Photos you reject from Review appear here.")
				backToReview.SetText("Back to Review")
				backToReview.OnTapped = func() {
					if onGotoReview != nil {
						onGotoReview()
					}
				}
			} else {
				emptyHint.SetText("No rejected photos match these filters. Reset filters to list everything in Rejected, or adjust the filters above.")
				backToReview.SetText("Reset filters")
				backToReview.OnTapped = func() {
					if resetRejectedFiltersToFR16 != nil {
						resetRejectedFiltersToFR16()
					}
				}
			}
		} else {
			emptyHint.Hide()
			backToReview.Show()
			backToReview.Importance = widget.MediumImportance
			backToReview.SetText("Back to Review")
			backToReview.OnTapped = func() {
				if onGotoReview != nil {
					onGotoReview()
				}
			}
			if reviewFiltersAtFR16Defaults(f) {
				resetFiltersBtn.Hide()
			} else {
				resetFiltersBtn.Show()
			}
		}
		if n == 0 && !reviewFiltersAtFR16Defaults(f) {
			rejectedBulkAllowed = false
			if zeroMatchPlate != nil {
				zeroMatchPlate.Show()
			}
		} else {
			if zeroMatchPlate != nil {
				zeroMatchPlate.Hide()
			}
		}
		grid.reset(f, n)
		grid.syncGridScrollVisible(gridScroll, n > 0)
		if gridScroll != nil {
			gridScroll.ScrollToTop()
		}
		rejectedAssetCount = n
		syncRejectedBulkStrip()
	}

	refreshAll := func() {
		refreshRejectedData()
	}

	grid = newReviewAssetGrid(win, db, libraryRoot, nil, syncBulkDeleteUI, true, func(assetID int64) {
		changed, err := store.RestoreAsset(db, assetID)
		if err != nil {
			slog.Error("rejected: restore", "err", err)
			if win != nil {
				dialog.ShowError(errors.New(userFacingDialogErrText(err)), win)
			}
			return
		}
		if !changed {
			slog.Debug("rejected: restore no-op", "asset_id", assetID)
		}
		refreshRejectedData()
	}, nil, nil)

	deleteSelectedBtn.OnTapped = func() {
		if grid == nil {
			return
		}
		ids := grid.SelectedAssetIDs()
		if len(ids) == 0 {
			return
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		title := "Move 1 rejected photo to library trash?"
		if len(ids) != 1 {
			title = fmt.Sprintf("Move %d rejected photos to library trash?", len(ids))
		}
		dialog.ShowConfirm(
			title,
			"This removes the selected photos from your library (stronger than Reject). Files are moved under .trash in your library folder.",
			func(ok bool) {
				if !ok {
					return
				}
				fyne.Do(func() {
					at := time.Now().Unix()
					for _, id := range ids {
						_, err := store.DeleteAssetToTrash(db, libraryRoot, id, at)
						if err != nil {
							slog.Error("rejected: delete asset", "err", err)
							if win != nil {
								dialog.ShowError(errors.New(userFacingDialogErrText(err)), win)
							}
							refreshRejectedData()
							return
						}
					}
					refreshRejectedData()
				})
			},
			win,
		)
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

	resetRejectedFiltersToFR16 = func() {
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
	resetFiltersBtn.OnTapped = func() {
		resetRejectedFiltersToFR16()
	}

	segLabels := ReviewFilterStripSegmentLabels()
	strip := container.NewHBox(
		container.NewHBox(widget.NewLabel(segLabels[0]), colSel),
		widget.NewSeparator(),
		container.NewHBox(widget.NewLabel(segLabels[1]), minRatingSel),
		widget.NewSeparator(),
		container.NewHBox(widget.NewLabel(segLabels[2]), tagsSel),
	)
	// Horizontal scroll so strip min-width does not widen the whole panel past the viewport (shell
	// uses vertical [container.NewScroll] only — extra width was clipped, chopping right-edge CTAs).
	stripScroll := container.NewHScroll(strip)

	gridScroll = container.NewScroll(grid.canvasObject())
	// Match Review grid floor so hidden/rejected thumbnails keep vertical budget vs chrome (image dominance).
	gridScroll.SetMinSize(fyne.NewSize(120, 400))
	plate := newFilterZeroMatchPhotoPlate()
	plate.Hide()
	zeroMatchPlate = plate
	gridArea := container.NewStack(gridScroll, plate)

	refreshAll()

	bulkHintAccordion := widget.NewAccordion()
	bulkHintAccordion.Append(widget.NewAccordionItem("Bulk delete tips", bulkHint))
	bulkHintAccordion.CloseAll()
	// One band so hidden-asset grids keep more vertical room vs full-width stacked chrome (image dominance).
	bulkBar = container.NewHBox(bulkHintAccordion, layout.NewSpacer(), deleteSelectedBtn)

	rejectedBulkCue = widget.NewLabel("Cmd/Ctrl+click thumbnails to select for bulk delete.")
	rejectedBulkCue.Wrapping = fyne.TextWrapWord
	rejectedBulkCue.Hide()
	syncRejectedBulkStrip()

	// Back before Reset in traversal order: when the empty-filter CTA relabels Back to "Reset filters",
	// tests and users must not hit the hidden resetFiltersBtn (same label, default importance).
	rejectedHeaderRow := container.NewHBox(countLabel, layout.NewSpacer(), backToReview, resetFiltersBtn)
	topChrome := container.NewVBox(
		stripScroll,
		widget.NewSeparator(),
		rejectedHeaderRow,
		emptyHint,
		rejectedBulkCue,
		widget.NewSeparator(),
	)
	bottomChrome := container.NewVBox(widget.NewSeparator(), bulkBar)
	main := container.NewBorder(topChrome, bottomChrome, nil, nil, gridArea)
	return container.NewPadded(main)
}
