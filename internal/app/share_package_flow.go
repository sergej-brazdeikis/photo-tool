package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"photo-tool/internal/domain"
	"photo-tool/internal/store"
)

const packageSharePreviewMaxRows = 100

// PackageSharePreviewMandatoryBeforeMint documents Story 4.1 AC2: audience presets never skip the manifest preview step.
func PackageSharePreviewMandatoryBeforeMint() bool { return true }

func audiencePresetShareMetadata(preset string) (displayTitle, audienceLabel string) {
	switch strings.TrimSpace(preset) {
	case "Close friends":
		return "Shared photos", "Close friends"
	case "Family":
		return "Family photos", "Family"
	case "Wider circle":
		return "Photo set", "Wider circle"
	default:
		return "", ""
	}
}

func openPackageShareFromReview(win fyne.Window, grid *reviewAssetGrid, candidateIDs []int64, listErr error) {
	if win == nil || grid == nil {
		return
	}
	if listErr != nil {
		dialog.ShowError(errors.New(userFacingDialogErrText(listErr)), win)
		return
	}
	dedupe := domain.StableDedupeAssetIDs(candidateIDs)
	if len(dedupe) == 0 {
		dialog.ShowInformation("Share package",
			"Select photos first (Cmd/Ctrl+click in Review), or use “Share (filtered)…” when your filters match the photos you want.",
			win)
		return
	}

	eligible, err := store.PackagePrepareEligibleForMint(context.Background(), grid.db, dedupe)
	if err != nil {
		if errors.Is(err, store.ErrPackageNoEligibleAssets) {
			dialog.ShowInformation("Share package",
				"None of the selected photos can be included in a package. Rejected photos and items missing from the library are excluded.",
				win)
			return
		}
		if errors.Is(err, store.ErrPackageTooManyAssets) {
			dialog.ShowInformation("Share package",
				"After excluding rejected or missing items, this set still has more than 500 photos. Narrow the selection or filters (MVP limit), then try again.",
				win)
			return
		}
		slog.Error("share package prepare", "err", err)
		dialog.ShowError(errors.New(userFacingDialogErrText(err)), win)
		return
	}

	summary := widget.NewLabel("")
	summary.Wrapping = fyne.TextWrapWord

	previewIDs := eligible
	truncNote := ""
	if len(eligible) > packageSharePreviewMaxRows {
		previewIDs = eligible[:packageSharePreviewMaxRows]
		truncNote = fmt.Sprintf("Showing the first %d rows in preview. “Create package link” still includes all %d eligible photos.",
			packageSharePreviewMaxRows, len(eligible))
	}
	summaryText := fmt.Sprintf("%d photos in this manifest · %d eligible after excluding rejected or missing items.", len(dedupe), len(eligible))
	if len(dedupe) != len(eligible) {
		summaryText += " Some selected ids were omitted."
	}
	if truncNote != "" {
		summaryText += "\n" + truncNote
	}
	summary.SetText(summaryText)

	previewParts := []fyne.CanvasObject{}
	gridRows, qerr := store.ListReviewGridRowsByIDsInOrder(grid.db, previewIDs)
	if qerr != nil {
		slog.Error("share package preview rows", "err", qerr)
		dialog.ShowError(errors.New(userFacingDialogErrText(qerr)), win)
		return
	}
	for _, row := range gridRows {
		img := canvas.NewImageFromFile(filepath.Join(grid.libraryRoot, filepath.FromSlash(row.RelPath)))
		img.FillMode = canvas.ImageFillContain
		img.SetMinSize(fyne.NewSize(uxImageSharePackageThumbW, uxImageSharePackageThumbH))
		when := time.Unix(row.CaptureTimeUnix, 0).UTC().Format("2006-01-02 15:04")
		lbl := widget.NewLabel(fmt.Sprintf("Library id %d · %s · %s", row.ID, row.RelPath, when))
		lbl.Wrapping = fyne.TextWrapWord
		previewParts = append(previewParts, container.NewVBox(img, lbl, widget.NewSeparator()))
	}
	if len(previewParts) == 0 {
		previewParts = []fyne.CanvasObject{widget.NewLabel("No preview rows loaded.")}
	}

	scroll := container.NewScroll(container.NewVBox(previewParts...))
	scroll.SetMinSize(fyne.NewSize(440, 300))

	presetSel := widget.NewSelect([]string{"No preset", "Close friends", "Family", "Wider circle"}, func(string) {})
	presetSel.SetSelected("No preset")

	hint := widget.NewLabel("A package link is created only after you confirm. Cancel leaves the library unchanged.")
	hint.Wrapping = fyne.TextWrapWord

	body := container.NewVBox(
		summary,
		widget.NewLabel("Audience label (optional, on this computer only):"),
		presetSel,
		widget.NewLabel("Preview:"),
		scroll,
		hint,
	)

	eligibleSnap := append([]int64(nil), eligible...)
	confirmLabel := fmt.Sprintf("Create package link (%d eligible)", len(eligible))
	dialog.ShowCustomConfirm(
		"Share package — preview",
		confirmLabel,
		"Cancel",
		body,
		func(confirmed bool) {
			if !confirmed {
				return
			}
			if !PackageSharePreviewMandatoryBeforeMint() {
				return
			}
			p := store.ShareSnapshotPayload{}
			dt, al := audiencePresetShareMetadata(presetSel.Selected)
			p.DisplayTitle = dt
			p.AudienceLabel = al
			tok, _, merr := store.MintPackageShareLink(context.Background(), grid.db, eligibleSnap, time.Now().Unix(), p)
			if merr != nil {
				if errors.Is(merr, store.ErrShareAssetIneligible) {
					dialog.ShowInformation("Share package",
						"One or more photos are no longer eligible (rejected or moved to trash). Refresh Review and try again.",
						win)
					return
				}
				if errors.Is(merr, store.ErrPackageNoEligibleAssets) {
					dialog.ShowInformation("Share package",
						"No eligible photos remain for this package. Refresh Review and try again.",
						win)
					return
				}
				if errors.Is(merr, store.ErrPackageTooManyAssets) {
					dialog.ShowInformation("Share package",
						"This package would exceed 500 photos. Reduce the selection, then try again.",
						win)
					return
				}
				slog.Error("share package mint", "err", merr)
				dialog.ShowError(errors.New(userFacingShareMintErrText(merr)), win)
				return
			}
			showLoupeShareMintSuccess(win, grid.shareLoopback, tok)
		},
		win,
	)
}
