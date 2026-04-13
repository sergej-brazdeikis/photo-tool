package main

import (
	"log/slog"
	"os"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"

	ptapp "photo-tool/internal/app"
	"photo-tool/internal/config"
	"photo-tool/internal/store"
)

func main() {
	root, err := config.ResolveLibraryRoot()
	if err != nil {
		slog.Error("library root", "err", err)
		os.Exit(1)
	}
	if err := config.EnsureLibraryLayout(root); err != nil {
		slog.Error("library layout", "err", err)
		os.Exit(1)
	}
	db, err := store.Open(root)
	if err != nil {
		slog.Error("open store", "path", root, "err", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	a := fyneapp.New()
	w := a.NewWindow("Photo Tool")
	w.SetContent(ptapp.NewUploadView(w, db, root))
	w.Resize(fyne.NewSize(640, 520))
	w.ShowAndRun()
}
