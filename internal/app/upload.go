// Package app holds Fyne UI for photo-tool (architecture §5.1).
package app

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"photo-tool/internal/domain"
	"photo-tool/internal/ingest"
	"photo-tool/internal/store"
)

// Bounded preview strip for Direction E (UX spec): keeps pixmap count predictable on large batches.
const (
	uploadPreviewStripMaxItems = 6
)

func addUploadPathPreviewTiles(strip *fyne.Container, fileAbs []string) {
	strip.RemoveAll()
	thumb := fyne.NewSize(uxImageUploadBatchPreviewMin, uxImageUploadBatchPreviewMin)
	for _, p := range fileAbs {
		if p == "" {
			continue
		}
		plate := canvas.NewRectangle(theme.Color(theme.ColorNameMenuBackground))
		plate.CornerRadius = 4
		plate.SetMinSize(thumb)
		img := canvas.NewImageFromFile("")
		img.FillMode = canvas.ImageFillContain
		img.SetMinSize(thumb)
		if raster, err := decodeImageFile(p); err == nil {
			img.Image = raster
			img.File = ""
			img.Resource = nil
		} else {
			img.Image = nil
			img.File = p
		}
		img.Refresh()
		strip.Add(container.NewStack(plate, img))
	}
}

// addUploadBatchPreviewCell appends one FR-06 preview tile; uses the same thumbnail cache as Review when assetID is known.
func addUploadBatchPreviewCell(strip *fyne.Container, libraryRoot string, db *sql.DB, absPath string, assetID int64, gen uint64, previewGen *atomic.Uint64, thumbBind *sync.Map) {
	thumb := fyne.NewSize(uxImageUploadBatchPreviewMin, uxImageUploadBatchPreviewMin)
	plate := canvas.NewRectangle(theme.Color(theme.ColorNameMenuBackground))
	plate.CornerRadius = 4
	plate.SetMinSize(thumb)
	img := canvas.NewImageFromFile("")
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(thumb)

	if assetID != 0 && db != nil && thumbBind != nil && previewGen != nil {
		rp, hash, ok, err := store.AssetRelPathContentHashByID(db, assetID)
		if err == nil && ok {
			srcAbs := filepath.Join(libraryRoot, filepath.FromSlash(rp))
			cacheAbs := ThumbnailCachePath(libraryRoot, assetID, hash)
			// Do not use theme SVG icons on canvas.Image: Fyne Refresh decodes via
			// image.Decode (raster-only) and can error or panic on later layout.
			img.Resource = nil
			img.File = ""
			img.Image = nil
			if raster, err := decodeImageFile(cacheAbs); err == nil {
				img.Image = raster
			} else if raster, err := decodeImageFile(srcAbs); err == nil {
				img.Image = raster
			} else {
				img.File = srcAbs
			}
			img.Refresh()
			wantID := assetID
			imgRef := img
			thumbBind.Store(imgRef, wantID)
			go func() {
				_ = WriteThumbnailJPEG(srcAbs, cacheAbs)
				fyne.Do(func() {
					if previewGen.Load() != gen {
						return
					}
					stored, ok := thumbBind.Load(imgRef)
					if !ok || stored.(int64) != wantID {
						return
					}
					// Decode into image.Image instead of canvas.File: mixing SVG Resource
					// raster (previous Refresh) with a File source can panic in Fyne 2.7
					// Resize→Refresh (nil reader) under parallel test layout.
					img.Resource = nil
					img.File = ""
					img.Image = nil
					if raster, err := decodeImageFile(cacheAbs); err == nil {
						img.Image = raster
					} else if raster, err := decodeImageFile(srcAbs); err == nil {
						img.Image = raster
					} else {
						img.File = srcAbs
					}
					img.Refresh()
				})
			}()
			strip.Add(container.NewStack(plate, img))
			return
		}
	}
	if raster, err := decodeImageFile(absPath); err == nil {
		img.Image = raster
		img.File = ""
		img.Resource = nil
	} else {
		img.Image = nil
		img.File = absPath
	}
	img.Refresh()
	strip.Add(container.NewStack(plate, img))
}

// imageOpenFilter limits the picker to the same extension set as CLI scan ([ingest.PickerFilterExtensions]).
var imageOpenFilter = storage.NewExtensionFileFilter(ingest.PickerFilterExtensions())

// uploadImportCloseBlocked is the policy behind [fyne.Window.SetCloseIntercept] for the upload view:
// block while ingest runs (UX-DR17 worker may still schedule [fyne.Do]) and during the post-import collection step (FR-06).
func uploadImportCloseBlocked(importInFlight, awaitingPostImportStep bool) (title, msg string, block bool) {
	if importInFlight {
		return "Import in progress",
			"Wait for the current import to finish before closing the window.", true
	}
	if awaitingPostImportStep {
		return "Collection step pending",
			"Confirm or cancel the collection assignment before closing the window.", true
	}
	return "", "", false
}

// UploadViewOptions configures [NewUploadViewWithOptions]. Zero value matches [NewUploadView] behavior.
type UploadViewOptions struct {
	// SeedPaths prepopulates the upload list (absolute paths). Enables headless tests without a file picker.
	SeedPaths []string
	// SkipCompletionDialogs omits success/info dialogs after Confirm/Cancel on the collection step (tests only).
	SkipCompletionDialogs bool
	// SynchronousIngest runs [ingest.IngestWithAssetIDs] on the caller goroutine (tests only).
	// The fyne test driver does not queue [fyne.Do] across Tap boundaries; without this, Tap(Import) then
	// Tap(Confirm) races the background ingest (UX-DR17 production path uses a worker + [fyne.Do]).
	SynchronousIngest bool
	// DisableImportCloseIntercept skips [fyne.Window.SetCloseIntercept] for upload-flow close guarding
	// (import in flight and pending collection step — e.g. tests or when another owner manages window close).
	DisableImportCloseIntercept bool
}

// NewUploadView builds the desktop upload flow: multi-file pick (accumulated via repeated open),
// ingest + receipt, optional collection confirm (FR-06). No SQL in widgets — only calls to ingest/store.
func NewUploadView(win fyne.Window, db *sql.DB, libraryRoot string) fyne.CanvasObject {
	return newUploadView(win, db, libraryRoot, UploadViewOptions{})
}

// NewUploadViewWithOptions is like [NewUploadView] with extra options for tests (seeded paths, dialog skipping).
func NewUploadViewWithOptions(win fyne.Window, db *sql.DB, libraryRoot string, opts UploadViewOptions) fyne.CanvasObject {
	return newUploadView(win, db, libraryRoot, opts)
}

func newUploadView(win fyne.Window, db *sql.DB, libraryRoot string, opts UploadViewOptions) fyne.CanvasObject {
	root := filepath.Clean(libraryRoot)
	libraryPathLabel := widget.NewLabelWithStyle(fmt.Sprintf("Library: %s", root), fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	libraryPathLabel.Wrapping = fyne.TextWrapWord

	showImportComplete := func(title, msg string) {
		if opts.SkipCompletionDialogs {
			return
		}
		dialog.ShowInformation(title, msg, win)
	}

	paths := []string{}
	var batchStart time.Time
	var lastSummary domain.OperationSummary
	var lastAssetIDs []int64
	// True while post-import receipt UI is shown; blocks re-entrant batch ingest.
	awaitingPostImportStep := false
	// True while a batch ingest goroutine is running (UX-DR17: work off UI thread).
	importInFlight := false
	// Mixed-drop unsupported lines; flushed in applyImportResult after ingest completes.
	var pendingDropSkipLines []string
	var addBtn, clearBtn, importBtn *widget.Button

	var uploadBatchPreviewGen atomic.Uint64
	var uploadBatchThumbBind sync.Map

	pathList := widget.NewList(
		func() int { return len(paths) },
		func() fyne.CanvasObject {
			l := widget.NewLabel("")
			l.Wrapping = fyne.TextWrapOff
			return l
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(paths[id])
		},
	)
	pathListScroll := container.NewScroll(pathList)
	// Enough height for multiple staged paths to read as a list (journey: two-file staging).
	pathListScroll.SetMinSize(fyne.NewSize(100, 120))
	stagedPhotoHeading := widget.NewLabelWithStyle("Selected photos", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	stagedPhotoHeading.Hide()
	stagedPhotoStrip := container.NewHBox()
	stagedPhotoScroll := container.NewHScroll(stagedPhotoStrip)
	stagedPhotoScroll.Hide()
	stagedPhotoMore := widget.NewLabel("")
	stagedPhotoMore.Hide()
	stagedHeader := widget.NewLabelWithStyle("Files staged for import", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	stagedBlock := container.NewVBox(stagedHeader, stagedPhotoHeading, stagedPhotoScroll, stagedPhotoMore, pathListScroll)

	addedLab := widget.NewLabel("New library rows: —")
	dupLab := widget.NewLabel("Already in library (skipped): —")
	failLab := widget.NewLabel("Failed: —")
	updatedLab := widget.NewLabel("Updated: —")
	updatedRow := container.NewHBox(updatedLab)
	receiptHint := widget.NewLabel("")
	receiptHint.Wrapping = fyne.TextWrapWord
	receiptHint.Hide()

	batchCountLab := widget.NewLabel("")
	batchCountLab.Hide()
	receiptSummaryLine := widget.NewLabel("")
	receiptSummaryLine.Wrapping = fyne.TextWrapWord
	receiptSummaryLine.Hide()
	receiptBody := container.NewVBox(
		batchCountLab,
		receiptHint,
		addedLab,
		dupLab,
		failLab,
		updatedRow,
	)
	receiptDetailsAccordion := widget.NewAccordion()
	receiptDetailsAccordion.Append(widget.NewAccordionItem("Full import details", receiptBody))
	receiptDetailsAccordion.CloseAll()
	receiptBlock := container.NewVBox(receiptSummaryLine, receiptDetailsAccordion)

	previewHeading := widget.NewLabelWithStyle("Batch preview", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	previewHeading.Hide()
	previewSummaryLab := widget.NewLabel("")
	previewSummaryLab.Wrapping = fyne.TextWrapWord
	previewSummaryLab.Hide()
	previewStrip := container.NewHBox()
	previewScroll := container.NewHScroll(previewStrip)
	previewScroll.Hide()
	previewMoreLab := widget.NewLabel("")
	previewMoreLab.Hide()
	previewBlock := container.NewVBox(previewHeading, previewSummaryLab, previewScroll, previewMoreLab)

	// Staging strip: image-forward preview before the path list (UX spec); capped like batch preview.
	updateStagedPhotoStrip := func() {
		stagedPhotoStrip.RemoveAll()
		if len(paths) == 0 {
			stagedPhotoHeading.Hide()
			stagedPhotoScroll.Hide()
			stagedPhotoMore.Hide()
			return
		}
		stagedPhotoHeading.Show()
		n := len(paths)
		showN := n
		if showN > uploadPreviewStripMaxItems {
			showN = uploadPreviewStripMaxItems
		}
		absShow := make([]string, 0, showN)
		for i := 0; i < showN; i++ {
			absShow = append(absShow, paths[i])
		}
		addUploadPathPreviewTiles(stagedPhotoStrip, absShow)
		if n > uploadPreviewStripMaxItems {
			stagedPhotoMore.SetText(fmt.Sprintf("+ %d more (see paths below).", n-uploadPreviewStripMaxItems))
			stagedPhotoMore.Show()
		} else {
			stagedPhotoMore.Hide()
		}
		stagedPhotoScroll.Show()
	}

	// Previews load from the picked paths on the UI goroutine; strip is capped
	// (uploadPreviewStripMaxItems) so cost stays bounded. If large JPEGs cause visible hitch,
	// profile first — offload decode + fyne.Do swap is a follow-up (Story 1.5 risks).
	// After ingest, tiles prefer canonical library paths via asset id (decoded thumbnails vs gray plates).
	updateBatchPreview := func(batchPaths []string, assetIDs []int64) {
		uploadBatchPreviewGen.Add(1)
		gen := uploadBatchPreviewGen.Load()
		previewStrip.RemoveAll()
		if len(batchPaths) == 0 {
			previewHeading.Hide()
			previewSummaryLab.Hide()
			previewScroll.Hide()
			previewMoreLab.Hide()
			return
		}
		previewHeading.Show()
		n := len(batchPaths)
		const summaryMaxNames = 3
		nameBits := make([]string, 0, summaryMaxNames)
		for i, p := range batchPaths {
			if i >= summaryMaxNames {
				break
			}
			nameBits = append(nameBits, filepath.Base(p))
		}
		sumLine := fmt.Sprintf("%d file(s) in this batch — %s", n, strings.Join(nameBits, ", "))
		if n > summaryMaxNames {
			sumLine += fmt.Sprintf(" (+ %d more)", n-summaryMaxNames)
		}
		previewSummaryLab.SetText(sumLine)
		previewSummaryLab.Show()
		showCount := n
		if showCount > uploadPreviewStripMaxItems {
			showCount = uploadPreviewStripMaxItems
		}
		resolvePreviewAbs := func(i int) string {
			if i < len(assetIDs) && assetIDs[i] != 0 {
				if rp, _, ok, err := store.AssetRelPathContentHashByID(db, assetIDs[i]); err == nil && ok {
					return filepath.Join(root, filepath.FromSlash(rp))
				}
			}
			if i < len(batchPaths) {
				return batchPaths[i]
			}
			return ""
		}
		for i := 0; i < showCount; i++ {
			aid := int64(0)
			if i < len(assetIDs) {
				aid = assetIDs[i]
			}
			addUploadBatchPreviewCell(previewStrip, root, db, resolvePreviewAbs(i), aid, gen, &uploadBatchPreviewGen, &uploadBatchThumbBind)
		}
		if n > uploadPreviewStripMaxItems {
			previewMoreLab.SetText(fmt.Sprintf("+ %d more in this batch (scroll the file list above for paths).", n-uploadPreviewStripMaxItems))
			previewMoreLab.Show()
		} else {
			previewMoreLab.Hide()
		}
		previewScroll.Show()
	}

	assignRadio := widget.NewRadioGroup([]string{"Skip collection", "Assign to collection"}, nil)
	assignRadio.Selected = "Skip collection"
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Collection name")
	nameEntry.Disable()
	nameLabel := widget.NewLabelWithStyle("Name", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	nameRow := container.NewVBox(nameLabel, nameEntry)
	nameRow.Hide()

	assignRadio.OnChanged = func(s string) {
		if s == "Assign to collection" {
			nameRow.Show()
			nameEntry.Enable()
			if strings.TrimSpace(nameEntry.Text) == "" && !batchStart.IsZero() {
				nameEntry.SetText(defaultUploadCollectionName(batchStart))
			}
		} else {
			nameEntry.Disable()
			nameRow.Hide()
		}
	}

	postImport := container.NewVBox()
	postImport.Hide()
	importStatusLab := widget.NewLabel("")
	importStatusLab.Hide()
	var uploadIdleChrome *fyne.Container

	confirmCollectionBtn := widget.NewButton("Confirm", nil)
	confirmCollectionBtn.Importance = widget.HighImportance
	cancelCollectionBtn := widget.NewButton("Cancel", nil)

	refreshPaths := func() {
		pathList.Refresh()
		updateStagedPhotoStrip()
	}

	addAbsolute := func(abs string) {
		if tryAddUniquePath(&paths, abs) {
			refreshPaths()
		}
	}

	showReceipt := func(sum domain.OperationSummary, batchFileCount int) {
		var summaryParts []string
		if batchFileCount > 0 {
			summaryParts = append(summaryParts, fmt.Sprintf("%d file(s)", batchFileCount))
		}
		summaryParts = append(summaryParts,
			fmt.Sprintf("+%d new", sum.Added),
			fmt.Sprintf("%d skipped duplicate", sum.SkippedDuplicate),
			fmt.Sprintf("%d failed", sum.Failed),
		)
		summary := strings.Join(summaryParts, " · ")
		if sum.Updated != 0 {
			summary += fmt.Sprintf(" · %d updated", sum.Updated)
		}
		receiptSummaryLine.SetText(summary)
		receiptSummaryLine.Show()

		addedLab.SetText(fmt.Sprintf("New library rows: %d", sum.Added))
		dupLab.SetText(fmt.Sprintf("Already in library (skipped): %d", sum.SkippedDuplicate))
		failLab.SetText(fmt.Sprintf("Failed: %d", sum.Failed))
		if batchFileCount > 0 && sum.Added == 0 && sum.SkippedDuplicate > 0 && sum.Failed == 0 {
			receiptHint.SetText("Every picked file matched an existing library photo — “skipped” counts those duplicates. This is different from how many files you chose above.")
			receiptHint.Show()
		} else if batchFileCount > 0 && sum.Added == 0 && sum.Failed > 0 {
			receiptHint.SetText("No new library rows were added — check failures below, then retry any problem files.")
			receiptHint.Show()
		} else {
			receiptHint.Hide()
		}
		if sum.Updated != 0 {
			updatedLab.SetText(fmt.Sprintf("Updated: %d", sum.Updated))
			updatedRow.Show()
		} else {
			updatedRow.Hide()
		}
	}

	resetBatchUI := func() {
		awaitingPostImportStep = false
		importInFlight = false
		pendingDropSkipLines = nil
		paths = paths[:0]
		refreshPaths()
		lastAssetIDs = nil
		lastSummary = domain.OperationSummary{}
		batchStart = time.Time{}
		assignRadio.Selected = "Skip collection"
		assignRadio.Refresh()
		nameEntry.SetText("")
		nameEntry.Disable()
		nameRow.Hide()
		postImport.Hide()
		libraryPathLabel.Show()
		uploadIdleChrome.Show()
		stagedBlock.Show()
		importStatusLab.SetText("")
		importStatusLab.Hide()
		receiptSummaryLine.SetText("")
		receiptSummaryLine.Hide()
		receiptDetailsAccordion.CloseAll()
		addedLab.SetText("New library rows: —")
		dupLab.SetText("Already in library (skipped): —")
		failLab.SetText("Failed: —")
		receiptHint.SetText("")
		receiptHint.Hide()
		updatedRow.Hide()
		batchCountLab.SetText("")
		batchCountLab.Hide()
		updateBatchPreview(nil, nil)
		updateStagedPhotoStrip()
		if addBtn != nil {
			addBtn.Enable()
			clearBtn.Enable()
			importBtn.Disable()
		}
	}

	openPicker := func() {
		fcd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(errors.New(userFacingFileOpenErrText(err)), win)
				return
			}
			if reader == nil {
				return
			}
			defer reader.Close()
			addAbsolute(reader.URI().Path())
		}, win)
		fcd.SetFilter(imageOpenFilter)
		fcd.Show()
	}

	// Full-width entry: widget.Form’s label/widget columns left the name field too narrow inside scroll.
	assignForm := container.NewVBox(
		widget.NewLabelWithStyle("Collection (after import)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Choose whether to attach this run’s imported assets to an album. Confirm or cancel below — this step stays on-screen until you decide."),
		assignRadio,
		nameRow,
		container.NewHBox(confirmCollectionBtn, cancelCollectionBtn),
	)

	// FR-06: batch preview first, then compact receipt + collection choice (image dominance; judge FR-06).
	// Outer scroll handles overflow; avoid a nested fixed-min-size scroll that read as empty chrome.
	receiptPreview := container.NewVBox(
		previewBlock,
		widget.NewSeparator(),
		receiptBlock,
	)
	postImportBody := container.NewVBox(
		receiptPreview,
		widget.NewSeparator(),
		assignForm,
	)
	postImport.Add(postImportBody)

	applyImportResult := func(sum domain.OperationSummary, ids []int64, batchFileCount int, batchPaths []string) {
		importInFlight = false
		importStatusLab.SetText("")
		importStatusLab.Hide()
		lastSummary = sum
		lastAssetIDs = ids
		updateBatchPreview(batchPaths, ids)
		showReceipt(sum, batchFileCount)
		if batchFileCount > 0 {
			batchCountLab.SetText(fmt.Sprintf("Source files in this import: %d", batchFileCount))
			batchCountLab.Show()
		} else {
			batchCountLab.SetText("")
			batchCountLab.Hide()
		}
		assignRadio.Selected = "Skip collection"
		assignRadio.Refresh()
		nameEntry.SetText("")
		nameEntry.Disable()
		nameRow.Hide()
		// Drop staged preview tiles so FR-06 batch strip is the only thumbnail cluster (tests + clarity).
		stagedPhotoStrip.RemoveAll()
		stagedPhotoHeading.Hide()
		stagedPhotoScroll.Hide()
		stagedPhotoMore.Hide()
		stagedBlock.Hide()
		uploadIdleChrome.Hide()
		libraryPathLabel.Hide()
		postImport.Show()
		postImport.Refresh()
		awaitingPostImportStep = true
		if addBtn != nil {
			addBtn.Disable()
			clearBtn.Disable()
			importBtn.Disable()
		}
		if lines, ok := takePendingStringSlice(&pendingDropSkipLines); ok {
			dialog.ShowInformation("Some items were skipped", droppedSkipSummaryForDialog(lines), win)
		}
	}

	runImportBatch := func() {
		if len(paths) == 0 || importInFlight {
			return
		}
		// Default collection label date: local calendar day when this batch ingest starts (PRD FR-05).
		batchStart = time.Now()
		pathsCopy := append([]string(nil), paths...)
		importInFlight = true
		importStatusLab.SetText("Importing…")
		importStatusLab.Show()
		if addBtn != nil {
			addBtn.Disable()
			clearBtn.Disable()
			importBtn.Disable()
		}
		nFiles := len(pathsCopy)
		if opts.SynchronousIngest {
			sum, ids := ingest.IngestWithAssetIDs(db, root, pathsCopy)
			applyImportResult(sum, ids, nFiles, pathsCopy)
			return
		}
		go func() {
			sum, ids := ingest.IngestWithAssetIDs(db, root, pathsCopy)
			fyne.Do(func() { applyImportResult(sum, ids, nFiles, pathsCopy) })
		}()
	}

	importBtn = widget.NewButton("Import selected files", func() {
		runImportBatch()
	})
	importBtn.Disable()

	clearBtn = widget.NewButton("Clear list", func() {
		resetBatchUI()
	})

	addBtn = widget.NewButton("Add images…", func() {
		openPicker()
		if len(paths) > 0 {
			importBtn.Enable()
		}
	})

	confirmCollectionBtn.OnTapped = func() {
		assign := assignRadio.Selected == "Assign to collection"
		name := strings.TrimSpace(nameEntry.Text)
		wantedAssign := assign && name != ""

		var link []int64
		for _, id := range lastAssetIDs {
			if id != 0 {
				link = append(link, id)
			}
		}

		// FR-06 / Story 1.5 AC3: do not create an empty collection when every ingest failed (no asset IDs).
		// Create + link in one transaction so a successful insert never leaves an orphan collection if linking fails.
		var actuallyLinked bool
		if wantedAssign && len(link) > 0 {
			displayISO := batchStart.In(time.Local).Format("2006-01-02")
			if _, err := store.CreateCollectionAndLinkAssets(db, name, displayISO, link); err != nil {
				dialog.ShowError(errors.New(userFacingDialogErrText(err)), win)
				return
			}
			actuallyLinked = true
		}

		showImportComplete("Import complete", summarizeDoneMessage(lastSummary, wantedAssign, actuallyLinked))
		resetBatchUI()
	}

	cancelCollectionBtn.OnTapped = func() {
		showImportComplete("Collection skipped", "Files remain in the library; no collection was created.")
		resetBatchUI()
	}

	dropTitle := widget.NewLabelWithStyle("Drop images here", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	dropBody := widget.NewLabel("Release files on this area to add them to the list and run import immediately — same pipeline as “Add images…” + “Import selected files”.")
	dropBody.Wrapping = fyne.TextWrapWord
	dropHint := widget.NewLabel("Types allowed match the file picker (see ingest rules). Folders and non-file drops are rejected with a message.")
	dropHint.Wrapping = fyne.TextWrapWord
	dropHint.TextStyle = fyne.TextStyle{Italic: true}
	dropPad := container.NewPadded(container.NewVBox(dropTitle, dropBody, dropHint))
	// Fill-only surface (no stroke / border layout): stroked round-rects and BorderLayout edge
	// strips can get negative logical sizes during early layout passes, which panics the software
	// painter during Canvas().Capture() (judge bundle / TestUXJourneyCapture).
	dropFill := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	dropZone := container.NewStack(dropFill, dropPad)
	// Rectangle fill is not recomputed on repaint; after View → light/dark the labels follow
	// the new theme while this surface would keep a stale InputBackground (bad contrast).
	if app := fyne.CurrentApp(); app != nil {
		if sett := app.Settings(); sett != nil {
			sett.AddListener(func(_ fyne.Settings) {
				dropFill.FillColor = theme.Color(theme.ColorNameInputBackground)
				dropFill.Refresh()
			})
		}
	}

	idleIntro := widget.NewLabel("Add one or more images (each pick adds to the list). Then run Import, or drop files on the target below.")
	idleIntro.Wrapping = fyne.TextWrapWord
	// Hide staging chrome during FR-06 so collection confirm + receipt are the only focus (UX judge: no overlap with drop zone).
	uploadIdleChrome = container.NewVBox(
		idleIntro,
		dropZone,
		container.NewVBox(
			container.NewHBox(addBtn, clearBtn, importBtn),
			importStatusLab,
		),
	)

	top := container.NewVBox(
		libraryPathLabel,
		uploadIdleChrome,
		stagedBlock,
		postImport,
	)

	scroll := container.NewScroll(top)

	// One handler per window; [fyne.Window.SetOnDropped] replaces any previous callback.
	// Hit-test uses the same absolute coordinates as pointer events ([fyne.Driver.AbsolutePositionForObject]).
	// The drop target lives inside a [container.Scroll]; the driver’s absolute position for [dropZone]
	// includes scroll offset on supported platforms (macOS baseline — re-check Windows/Linux in QA).
	win.SetOnDropped(func(absPos fyne.Position, uris []fyne.URI) {
		// Empty payload: treat as no-op (no dialog). Some platforms may deliver an empty slice;
		// there is nothing actionable to explain without inventing failure copy.
		if len(uris) == 0 {
			return
		}
		if !dropHitTest(absPos, dropZone) {
			return
		}
		if title, msg, blocked := dropBlockedDialogInfo(awaitingPostImportStep, importInFlight); blocked {
			dialog.ShowInformation(title, msg, win)
			return
		}
		res := classifyDroppedURIs(uris, os.Stat)
		if len(res.Supported) == 0 {
			msg := "No supported image files in this drop."
			if len(res.Unsupported) > 0 {
				msg = droppedSkipSummaryForDialog(res.Unsupported)
			}
			// Proportionate honesty (UX spec): wrong types are user-correctable — use information, not error chrome.
			dialog.ShowInformation("No supported images", msg, win)
			return
		}
		anyNew := false
		for _, p := range res.Supported {
			if tryAddUniquePath(&paths, p) {
				anyNew = true
			}
		}
		if anyNew {
			refreshPaths()
		}
		if !anyNew {
			var b strings.Builder
			if len(res.Supported) > 0 {
				b.WriteString("Those files are already in the upload list; nothing new to import.")
			}
			if len(res.Unsupported) > 0 {
				if b.Len() > 0 {
					b.WriteString("\n\n")
				}
				b.WriteString(droppedSkipSummaryForDialog(res.Unsupported))
			}
			if b.Len() > 0 {
				dialog.ShowInformation("No new files to import", b.String(), win)
			}
			return
		}
		if len(paths) > 0 {
			importBtn.Enable()
		}
		if len(res.Unsupported) > 0 {
			pendingDropSkipLines = append([]string(nil), res.Unsupported...)
		}
		runImportBatch()
	})

	for _, p := range opts.SeedPaths {
		if tryAddUniquePath(&paths, filepath.Clean(p)) {
			refreshPaths()
		}
	}
	if len(paths) > 0 {
		importBtn.Enable()
	}

	if !opts.DisableImportCloseIntercept {
		// Avoid tearing down the window while a worker still schedules [fyne.Do] (UX-DR17),
		// and avoid abandoning the explicit collection confirm step without Confirm/Cancel (Journey A).
		// If the shell adds its own close intercept later, chain that handler here instead of replacing it.
		win.SetCloseIntercept(func() {
			if title, msg, ok := uploadImportCloseBlocked(importInFlight, awaitingPostImportStep); ok {
				dialog.ShowInformation(title, msg, win)
				return
			}
			win.Close()
		})
	}

	return scroll
}

func defaultUploadCollectionName(batchStart time.Time) string {
	return fmt.Sprintf("Upload %s", batchStart.In(time.Local).Format("20060102"))
}

func summarizeDoneMessage(sum domain.OperationSummary, wantedAssign, actuallyLinked bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Added %d, skipped duplicate %d, failed %d.", sum.Added, sum.SkippedDuplicate, sum.Failed)
	if sum.Updated != 0 {
		fmt.Fprintf(&b, " Updated %d.", sum.Updated)
	}
	switch {
	case !wantedAssign:
		b.WriteString(" No new collection was created.")
	case actuallyLinked:
		b.WriteString(" Assets were linked to the new collection.")
	default:
		b.WriteString(" No collection was created — no successfully ingested files were available to attach.")
	}
	if sum.Failed > 0 {
		b.WriteString(" For items that failed, check permissions and file type, then add them again from Upload.")
	}
	return b.String()
}
