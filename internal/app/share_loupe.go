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

	"photo-tool/internal/share"
	"photo-tool/internal/store"
)

// loupeShareSelectionMatchesPreview is true when the loupe is still showing the same asset as the share preview (AC8 hardening).
func loupeShareSelectionMatchesPreview(previewAssetID int64, currentLoupeAssetID int64) bool {
	return previewAssetID > 0 && previewAssetID == currentLoupeAssetID
}

// loupeSharePreviewProceedToMint is true when the user confirmed and the loupe still shows the previewed asset (AC7c/AC8).
// selectionDrift is true when the user confirmed but navigated away—caller should show the drift message and must not mint.
// If currentLoupeAssetID is nil, the loupe cannot be verified—caller should surface wiring/error copy and must not mint (fail closed).
func loupeSharePreviewProceedToMint(confirmed bool, previewAssetID int64, currentLoupeAssetID func() int64) (proceed bool, selectionDrift bool) {
	if !confirmed {
		return false, false
	}
	if currentLoupeAssetID == nil || previewAssetID <= 0 {
		return false, false
	}
	if !loupeShareSelectionMatchesPreview(previewAssetID, currentLoupeAssetID()) {
		return false, true
	}
	return true, false
}

func userFacingShareMintErrText(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, store.ErrPackageTooManyAssets) {
		return "This package would include more than 500 photos. Reduce the selection or narrow filters, then try again."
	}
	if errors.Is(err, store.ErrPackageNoEligibleAssets) {
		return "None of the photos can be included in this package. They may be rejected, missing, or in library trash."
	}
	base := userFacingDialogErrText(err)
	if base == "" {
		// Belt-and-suspenders: mint failures must never surface an empty dialog (TEA / AC6).
		base = "Could not create the share link. Check that the library folder is available, then try again."
	}
	return base + " If this keeps happening, check the application's log output for details."
}

func openLoupeShareFlow(win fyne.Window, grid *reviewAssetGrid, idx int, currentLoupeAssetID func() int64) {
	if win == nil || grid == nil {
		return
	}
	grid.mu.Lock()
	f := grid.filters
	grid.mu.Unlock()

	rows, err := store.ListAssetsForReview(grid.db, f, 1, idx)
	if err != nil || len(rows) == 0 {
		slog.Error("share: load row", "err", err, "idx", idx)
		dialog.ShowError(errors.New("could not load this photo for sharing — return to review and try again"), win)
		return
	}
	row := rows[0]

	if block, qerr := store.DefaultShareBlockedUserMessage(grid.db, row.ID); qerr != nil {
		dialog.ShowError(errors.New(userFacingDialogErrText(qerr)), win)
		return
	} else if block != "" {
		dialog.ShowInformation("Share", block, win)
		return
	}

	preview := buildLoupeSharePreview(grid.libraryRoot, row)
	dialog.ShowCustomConfirm(
		"Share preview",
		"Create link",
		"Cancel",
		preview,
		func(confirmed bool) {
			if !confirmed {
				return
			}
			proceed, drift := loupeSharePreviewProceedToMint(true, row.ID, currentLoupeAssetID)
			if !proceed && !drift {
				slog.Error("share: confirm without verifiable loupe selection", "preview_asset_id", row.ID)
				dialog.ShowInformation("Share",
					"Could not verify the photo in the loupe. Close this dialog and tap Share… again.",
					win)
				return
			}
			if drift {
				dialog.ShowInformation("Share",
					"The photo in the loupe changed while this preview was open. Close this dialog and tap Share… again for the photo you want.",
					win)
				return
			}
			tok, _, merr := store.MintDefaultShareLink(context.Background(), grid.db, row.ID, time.Now().Unix())
			if merr != nil {
				if errors.Is(merr, store.ErrShareAssetIneligible) {
					msg, _ := store.DefaultShareBlockedUserMessage(grid.db, row.ID)
					if msg == "" {
						msg = "This photo can't be shared anymore. It may have been rejected or moved to trash."
					}
					dialog.ShowInformation("Share", msg, win)
					return
				}
				slog.Error("share mint", "err", merr, "asset_id", row.ID)
				dialog.ShowError(errors.New(userFacingShareMintErrText(merr)), win)
				return
			}
			showLoupeShareMintSuccess(win, grid.shareLoopback, tok)
		},
		win,
	)
}

func shareLoupeBreakableRelPath(rel string) string {
	// Zero-width spaces after slashes so wrapped labels don't force the dialog wider than the image min width.
	return strings.ReplaceAll(filepath.ToSlash(rel), "/", "/\u200b")
}

func buildLoupeSharePreview(libraryRoot string, row store.ReviewGridRow) fyne.CanvasObject {
	abs := filepath.Join(libraryRoot, filepath.FromSlash(row.RelPath))
	img := canvas.NewImageFromFile("")
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(uxImageShareLoupeW, uxImageShareLoupeH))
	if raster, err := decodeImageFile(abs); err == nil {
		img.Image = raster
		img.File = ""
		img.Resource = nil
	} else {
		img.Image = nil
		img.File = abs
		img.Resource = nil
	}
	img.Refresh()
	when := time.Unix(row.CaptureTimeUnix, 0).UTC().Format("2006-01-02 15:04 MST")
	rel := shareLoupeBreakableRelPath(row.RelPath)
	hint := "A link is created only after you tap Create link. Nothing is saved before that."
	lbl := widget.NewLabel(fmt.Sprintf("File: %s\nCaptured: %s\nLibrary ID: %d\n\n%s", rel, when, row.ID, hint))
	lbl.Wrapping = fyne.TextWrapWord
	metaScroll := container.NewVScroll(lbl)
	metaScroll.SetMinSize(fyne.NewSize(uxImageShareLoupeW, uxShareLoupeMetaScrollH))
	// Normative share hierarchy: the photo is the largest element; metadata stays below the image band in one short strip.
	imgBand := container.NewMax(container.NewCenter(img))
	return container.NewVBox(imgBand, metaScroll)
}

func showLoupeShareMintSuccess(win fyne.Window, loop *share.Loopback, rawToken string) {
	entry := widget.NewMultiLineEntry()
	entry.SetText(rawToken)
	entry.Disable()

	var fullURL string
	if loop != nil {
		if base, err := loop.EnsureRunning(context.Background()); err != nil {
			slog.Error("share http ensure", "err", err)
		} else {
			fullURL = base + share.ShareHTTPPath(rawToken)
		}
	}

	note := widget.NewLabel("Copy the loopback link to open this shared photo in a browser on this machine, or copy the token. The clipboard is not changed automatically.")
	if fullURL == "" {
		note.SetText("Copy the token if you need it elsewhere. Loopback viewing was unavailable (check the log). The clipboard is not changed automatically.")
	}
	note.Wrapping = fyne.TextWrapWord

	copyTok := widget.NewButton("Copy token", func() {
		fyne.CurrentApp().Clipboard().SetContent(rawToken)
	})

	buttons := []fyne.CanvasObject{copyTok}
	if fullURL != "" {
		urlEntry := widget.NewEntry()
		urlEntry.SetText(fullURL)
		urlEntry.Disable()
		copyLink := widget.NewButton("Copy link", func() {
			fyne.CurrentApp().Clipboard().SetContent(fullURL)
		})
		buttons = append([]fyne.CanvasObject{copyLink}, buttons...)
		note.SetText("Copy the loopback link to open this shared photo in a browser on this machine. You can still copy the raw token if needed. The clipboard is not changed automatically.")
		bodyTop := container.NewVBox(note, widget.NewLabel("URL:"), urlEntry, widget.NewLabel("Token:"), entry)
		d := dialog.NewCustomWithoutButtons("Share link created", bodyTop, win)
		d.SetButtons(append(buttons, widget.NewButton("Close", func() { d.Hide() })))
		d.Show()
		return
	}

	body := container.NewVBox(note, entry)
	d := dialog.NewCustomWithoutButtons("Share link created", body, win)
	d.SetButtons([]fyne.CanvasObject{
		copyTok,
		widget.NewButton("Close", func() { d.Hide() }),
	})
	d.Show()
}
