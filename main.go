package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"

	"photo-tool/internal/config"
	"photo-tool/internal/store"
)

func main() {
	msg := bootstrapMessage()
	a := app.New()
	w := a.NewWindow("Photo Tool")
	w.SetContent(widget.NewLabel(msg))
	w.Resize(fyne.NewSize(520, 140))
	w.ShowAndRun()
}

func bootstrapMessage() string {
	root, err := config.ResolveLibraryRoot()
	if err != nil {
		return fmt.Sprintf("Library root: error\n%v", err)
	}
	if err := config.EnsureLibraryLayout(root); err != nil {
		return fmt.Sprintf("Library layout: %v", err)
	}
	db, err := store.Open(root)
	if err != nil {
		return fmt.Sprintf("Library: %s\nDatabase: %v", root, err)
	}
	_ = db.Close()
	return fmt.Sprintf("Library ready\n%s", root)
}
