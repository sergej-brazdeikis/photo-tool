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
		return widget.NewLabel("Hidden assets: — (no database)")
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

	countLabel := widget.NewLabel("Hidden assets: —")
	deleteSelectedBtn := widget.NewButton("Delete selected…", nil)
	deleteSelectedBtn.Importance = widget.DangerImportance
	deleteSelectedBtn.Disable()
	bulkHint := widget.NewLabel("Cmd/Ctrl+click thumbnails to select multiple hidden photos for bulk delete.")
	emptyHint := widget.NewLabel("")
	emptyHint.Wrapping = fyne.TextWrapWord
	emptyHint.Hide()
	backToReview := widget.NewButton("Back to Review", func() {
		if onGotoReview != nil {
			onGotoReview()
		}
	})
	backToReview.Hide()
	emptyRow := container.NewHBox(emptyHint, layout.NewSpacer(), backToReview)

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

	syncBulkDeleteUI := func() {
		if grid == nil {
			return
		}
		if len(grid.SelectedAssetIDs()) > 0 {
			deleteSelectedBtn.Enable()
		} else {
			deleteSelectedBtn.Disable()
		}
	}

	refreshRejectedData := func() {
		tagStripSyncErr := false
		if err := syncTagStrip(); err != nil {
			slog.Error("rejected: sync tag strip", "err", err)
			tagStripSyncErr = true
		}
		f := buildFilters()
		n, qerr := store.CountRejectedForReview(db, f)
		if qerr != nil {
			msg := fmt.Sprintf("Hidden assets: — (%s)", libraryErrText(qerr))
			if tagStripSyncErr {
				msg += "; could not refresh tag list"
			}
			if listErr != nil {
				msg += "; " + libraryErrText(listErr)
			}
			countLabel.SetText(msg)
			emptyHint.Hide()
			backToReview.Hide()
			grid.reset(f, 0)
			return
		}
		msg := fmt.Sprintf("Hidden assets: %d", n)
		if listErr != nil {
			msg += " (collections unavailable — " + libraryErrText(listErr) + ")"
		}
		if tagStripSyncErr {
			msg += " — Could not refresh tag list"
		}
		countLabel.SetText(msg)
		if n == 0 {
			emptyHint.Show()
			backToReview.Show()
			backToReview.Importance = widget.HighImportance
			if reviewFiltersAtFR16Defaults(f) {
				emptyHint.SetText("Nothing is hidden or rejected yet. Items you reject or hide from Review will show up here.")
				backToReview.SetText("Back to Review")
				backToReview.OnTapped = func() {
					if onGotoReview != nil {
						onGotoReview()
					}
				}
			} else {
				emptyHint.SetText("No hidden photos match these filters. Reset filters to list everything that is hidden, or adjust the filters above.")
				backToReview.SetText("Reset filters")
				backToReview.OnTapped = func() {
					if resetRejectedFiltersToFR16 != nil {
						resetRejectedFiltersToFR16()
					}
				}
			}
		} else {
			emptyHint.Hide()
			backToReview.Hide()
			backToReview.Importance = widget.MediumImportance
		}
		grid.reset(f, n)
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
		title := "Move 1 hidden photo to library trash?"
		if len(ids) != 1 {
			title = fmt.Sprintf("Move %d hidden photos to library trash?", len(ids))
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

	segLabels := ReviewFilterStripSegmentLabels()
	strip := container.NewHBox(
		container.NewHBox(widget.NewLabel(segLabels[0]), colSel),
		widget.NewSeparator(),
		container.NewHBox(widget.NewLabel(segLabels[1]), minRatingSel),
		widget.NewSeparator(),
		container.NewHBox(widget.NewLabel(segLabels[2]), tagsSel),
	)

	refreshAll()

	bulkBar := container.NewVBox(
		bulkHint,
		container.NewHBox(layout.NewSpacer(), deleteSelectedBtn),
	)

	body := container.NewBorder(
		countLabel,
		nil, nil, nil,
		container.NewVBox(emptyRow, container.NewScroll(grid.canvasObject())),
	)
	return container.NewPadded(container.NewVBox(strip, widget.NewSeparator(), bulkBar, widget.NewSeparator(), body))
}
