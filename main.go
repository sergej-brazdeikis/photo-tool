package main

import (
	"log/slog"
	"os"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"

	ptapp "photo-tool/internal/app"
	"photo-tool/internal/cli"
	"photo-tool/internal/config"
	"photo-tool/internal/share"
	"photo-tool/internal/store"
)

func main() {
	if len(os.Args) == 1 {
		runGUI()
		return
	}
	cli.MainExit()
}

func runGUI() {
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

	shareCfg, err := config.LoadShareHTTPConfig()
	if err != nil {
		slog.Error("share http config", "err", err)
		os.Exit(1)
	}
	shareLoop := share.NewLoopback(db, root, shareCfg)
	defer func() { _ = shareLoop.Close() }()

	a := fyneapp.NewWithID(ptapp.FyneAppID)
	photoTheme := ptapp.NewPhotoToolTheme(ptapp.LoadThemeVariantFromPrefs(a.Preferences()))
	a.Settings().SetTheme(photoTheme)

	w := a.NewWindow("Photo Tool")
	w.SetMainMenu(fyne.NewMainMenu(
		fyne.NewMenu("View",
			fyne.NewMenuItem("Use dark theme", func() {
				photoTheme.SetVariant(theme.VariantDark)
				ptapp.SaveThemeVariantToPrefs(a.Preferences(), theme.VariantDark)
				a.Settings().SetTheme(photoTheme)
			}),
			fyne.NewMenuItem("Use light theme", func() {
				photoTheme.SetVariant(theme.VariantLight)
				ptapp.SaveThemeVariantToPrefs(a.Preferences(), theme.VariantLight)
				a.Settings().SetTheme(photoTheme)
			}),
		),
	))

	w.SetContent(ptapp.NewMainShell(w, db, root, shareLoop))
	w.Resize(fyne.NewSize(800, 560))
	w.ShowAndRun()
}
