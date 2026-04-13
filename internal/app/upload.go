// Package app holds Fyne UI for photo-tool (architecture §5.1).
package app

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"photo-tool/internal/domain"
	"photo-tool/internal/ingest"
	"photo-tool/internal/store"
)

// imageOpenFilter limits the picker to common raster / camera formats the ingest pipeline can read.
var imageOpenFilter = storage.NewExtensionFileFilter([]string{
	".jpg", ".jpeg", ".png", ".gif", ".webp", ".tif", ".tiff",
	".heic", ".HEIC", ".dng", ".DNG",
})

// NewUploadView builds the desktop upload flow: multi-file pick (accumulated via repeated open),
// ingest + receipt, optional collection confirm (FR-06). No SQL in widgets — only calls to ingest/store.
func NewUploadView(win fyne.Window, db *sql.DB, libraryRoot string) fyne.CanvasObject {
	root := filepath.Clean(libraryRoot)

	paths := []string{}
	var batchStart time.Time
	var lastSummary domain.OperationSummary
	var lastAssetIDs []int64

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
		abs = filepath.Clean(abs)
		for _, p := range paths {
			if p == abs {
				return
			}
		}
		paths = append(paths, abs)
		refreshPaths()
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
	}

	openPicker := func() {
		fcd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(fmt.Errorf("open file: %w", err), win)
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

	importBtn := widget.NewButton("Import selected files", func() {
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
	})
	importBtn.Disable()

	clearBtn := widget.NewButton("Clear list", func() {
		resetBatchUI()
		importBtn.Disable()
	})

	addBtn := widget.NewButton("Add images…", func() {
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
				dialog.ShowError(fmt.Errorf("create collection: %w", err), win)
				return
			}
			if err := store.LinkAssetsToCollection(db, cid, link); err != nil {
				dialog.ShowError(fmt.Errorf("link assets: %w", err), win)
				return
			}
			actuallyLinked = true
		}

		dialog.ShowInformation("Import complete", summarizeDoneMessage(lastSummary, wantedAssign, actuallyLinked), win)
		resetBatchUI()
		importBtn.Disable()
	}

	cancelCollectionBtn.OnTapped = func() {
		dialog.ShowInformation("Collection skipped", "Files remain in the library; no collection was created.", win)
		resetBatchUI()
		importBtn.Disable()
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

	top := container.NewVBox(
		header,
		widget.NewLabel("Add one or more images (each pick adds to the list). Then run Import."),
		container.NewHBox(addBtn, clearBtn, importBtn),
		pathList,
		postImport,
	)

	return container.NewScroll(top)
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
	return b.String()
}
