// Package app holds Fyne UI for photo-tool (architecture §5.1).
package app

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// imageOpenFilter limits the picker to the same extension set as CLI scan ([ingest.PickerFilterExtensions]).
var imageOpenFilter = storage.NewExtensionFileFilter(ingest.PickerFilterExtensions())

// NewUploadView builds the desktop upload flow: multi-file pick (accumulated via repeated open),
// ingest + receipt, optional collection confirm (FR-06). No SQL in widgets — only calls to ingest/store.
func NewUploadView(win fyne.Window, db *sql.DB, libraryRoot string) fyne.CanvasObject {
	root := filepath.Clean(libraryRoot)

	paths := []string{}
	var batchStart time.Time
	var lastSummary domain.OperationSummary
	var lastAssetIDs []int64
	// True while post-import receipt UI is shown; blocks re-entrant batch ingest.
	awaitingPostImportStep := false
	var addBtn, clearBtn, importBtn *widget.Button

	pathList := widget.NewList(
		func() int { return len(paths) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(paths[id])
		},
	)

	addedLab := widget.NewLabel("Added: —")
	dupLab := widget.NewLabel("Skipped duplicate: —")
	failLab := widget.NewLabel("Failed: —")
	updatedLab := widget.NewLabel("Updated: —")
	updatedRow := container.NewHBox(updatedLab)

	assignRadio := widget.NewRadioGroup([]string{"Skip collection", "Assign to collection"}, nil)
	assignRadio.Selected = "Skip collection"
	nameEntry := widget.NewEntry()
	nameEntry.Disable()

	assignRadio.OnChanged = func(s string) {
		if s == "Assign to collection" {
			nameEntry.Enable()
			if strings.TrimSpace(nameEntry.Text) == "" && !batchStart.IsZero() {
				nameEntry.SetText(defaultUploadCollectionName(batchStart))
			}
		} else {
			nameEntry.Disable()
		}
	}

	postImport := container.NewVBox()
	postImport.Hide()

	confirmCollectionBtn := widget.NewButton("Confirm", nil)
	confirmCollectionBtn.Importance = widget.HighImportance
	cancelCollectionBtn := widget.NewButton("Cancel", nil)

	refreshPaths := func() {
		pathList.Refresh()
	}

	addAbsolute := func(abs string) {
		if tryAddUniquePath(&paths, abs) {
			refreshPaths()
		}
	}

	showReceipt := func(sum domain.OperationSummary) {
		addedLab.SetText(fmt.Sprintf("Added: %d", sum.Added))
		dupLab.SetText(fmt.Sprintf("Skipped duplicate: %d", sum.SkippedDuplicate))
		failLab.SetText(fmt.Sprintf("Failed: %d", sum.Failed))
		if sum.Updated != 0 {
			updatedLab.SetText(fmt.Sprintf("Updated: %d", sum.Updated))
			updatedRow.Show()
		} else {
			updatedRow.Hide()
		}
	}

	resetBatchUI := func() {
		awaitingPostImportStep = false
		paths = paths[:0]
		refreshPaths()
		lastAssetIDs = nil
		lastSummary = domain.OperationSummary{}
		batchStart = time.Time{}
		assignRadio.Selected = "Skip collection"
		assignRadio.Refresh()
		nameEntry.SetText("")
		nameEntry.Disable()
		postImport.Hide()
		addedLab.SetText("Added: —")
		dupLab.SetText("Skipped duplicate: —")
		failLab.SetText("Failed: —")
		updatedRow.Hide()
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

	runImportBatch := func() {
		if len(paths) == 0 {
			return
		}
		// Default collection label date: local calendar day when this batch ingest starts (PRD FR-05).
		batchStart = time.Now()
		sum, ids := ingest.IngestWithAssetIDs(db, root, paths)
		lastSummary = sum
		lastAssetIDs = ids
		showReceipt(sum)
		assignRadio.Selected = "Skip collection"
		assignRadio.Refresh()
		nameEntry.SetText("")
		nameEntry.Disable()
		postImport.Show()
		postImport.Refresh()
		awaitingPostImportStep = true
		if addBtn != nil {
			addBtn.Disable()
			clearBtn.Disable()
			importBtn.Disable()
		}
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
		var actuallyLinked bool
		if wantedAssign && len(link) > 0 {
			displayISO := batchStart.In(time.Local).Format("2006-01-02")
			cid, err := store.CreateCollection(db, name, displayISO)
			if err != nil {
				dialog.ShowError(errors.New(userFacingDialogErrText(err)), win)
				return
			}
			if err := store.LinkAssetsToCollection(db, cid, link); err != nil {
				dialog.ShowError(errors.New(userFacingDialogErrText(err)), win)
				return
			}
			actuallyLinked = true
		}

		dialog.ShowInformation("Import complete", summarizeDoneMessage(lastSummary, wantedAssign, actuallyLinked), win)
		resetBatchUI()
	}

	cancelCollectionBtn.OnTapped = func() {
		dialog.ShowInformation("Collection skipped", "Files remain in the library; no collection was created.", win)
		resetBatchUI()
	}

	assignForm := container.NewVBox(
		widget.NewLabel("Collection (after import)"),
		assignRadio,
		widget.NewForm(&widget.FormItem{Text: "Name", Widget: nameEntry}),
		container.NewHBox(confirmCollectionBtn, cancelCollectionBtn),
	)

	postImport.Add(widget.NewSeparator())
	postImport.Add(widget.NewLabelWithStyle("Receipt", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	postImport.Add(addedLab)
	postImport.Add(dupLab)
	postImport.Add(failLab)
	postImport.Add(updatedRow)
	postImport.Add(widget.NewSeparator())
	postImport.Add(assignForm)

	header := widget.NewLabelWithStyle(fmt.Sprintf("Library: %s", root), fyne.TextAlignLeading, fyne.TextStyle{Italic: true})

	dropTitle := widget.NewLabelWithStyle("Drop images here", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	dropBody := widget.NewLabel("Release files on this area to add them to the list and run import immediately — same pipeline as “Add images…” + “Import selected files”.")
	dropBody.Wrapping = fyne.TextWrapWord
	dropHint := widget.NewLabel("Types allowed match the file picker (see ingest rules). Folders and non-file drops are rejected with a message.")
	dropHint.Wrapping = fyne.TextWrapWord
	dropHint.TextStyle = fyne.TextStyle{Italic: true}
	dropPad := container.NewPadded(container.NewVBox(dropTitle, dropBody, dropHint))
	dropBG := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	dropBG.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	dropBG.StrokeWidth = 1
	dropBG.CornerRadius = 4
	dropZone := container.NewStack(dropBG, dropPad)

	top := container.NewVBox(
		header,
		widget.NewLabel("Add one or more images (each pick adds to the list). Then run Import, or drop files on the target below."),
		dropZone,
		container.NewHBox(addBtn, clearBtn, importBtn),
		pathList,
		postImport,
	)

	scroll := container.NewScroll(top)

	// One handler per window; [fyne.Window.SetOnDropped] replaces any previous callback.
	// Hit-test uses the same absolute coordinates as pointer events ([fyne.Driver.AbsolutePositionForObject]).
	// The drop target lives inside a [container.Scroll]; the driver’s absolute position for [dropZone]
	// includes scroll offset on supported platforms (macOS baseline — re-check Windows/Linux in QA).
	win.SetOnDropped(func(absPos fyne.Position, uris []fyne.URI) {
		if len(uris) == 0 {
			return
		}
		if !dropHitTest(absPos, dropZone) {
			return
		}
		if awaitingPostImportStep {
			dialog.ShowInformation("Finish collection step",
				"Confirm or cancel the upload collection step before dropping more files.", win)
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
		runImportBatch()
		if len(res.Unsupported) > 0 {
			dialog.ShowInformation("Some items were skipped", droppedSkipSummaryForDialog(res.Unsupported), win)
		}
	})

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
