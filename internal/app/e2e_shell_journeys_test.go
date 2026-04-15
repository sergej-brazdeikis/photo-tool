package app

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"photo-tool/internal/config"
	"photo-tool/internal/store"
)

// collectLabelsDeep walks Scroll content so labels inside Review / Collections panels are visible to shell-level assertions.
func collectLabelsDeep(o fyne.CanvasObject) []*widget.Label {
	var out []*widget.Label
	var walk func(fyne.CanvasObject)
	walk = func(x fyne.CanvasObject) {
		if x == nil {
			return
		}
		switch v := x.(type) {
		case *widget.Label:
			out = append(out, v)
		case *widget.Accordion:
			for _, it := range v.Items {
				if it != nil {
					walk(it.Detail)
				}
			}
		case *container.Scroll:
			walk(v.Content)
		case *fyne.Container:
			for _, ch := range v.Objects {
				walk(ch)
			}
		}
	}
	walk(o)
	return out
}

// E2E-style shell journeys: multi-step primary navigation and cross-panel smoke using the Fyne test driver.

func TestE2E_shell_primaryNav_Upload_Review_Collections_Rejected(t *testing.T) {
	test.NewTempApp(t)

	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	win := test.NewTempWindow(t, nil)
	// Wide enough that primary nav buttons (incl. "Rejected") stay on-canvas with semantic preview rail.
	win.Resize(fyne.NewSize(1120, 600))
	applyTestPhotoToolTheme(t, theme.VariantDark)
	shell := newMainShell(win, db, root, false, nil)
	win.SetContent(shell)

	for _, label := range []string{"Upload", "Review", "Collections", "Rejected"} {
		tapPanel(t, shell, label)
		assertPrimaryNavVisible(t, win, shell)
		if label == "Review" {
			assertReviewFilterStripOnScreen(t, win, shell)
		}
	}
}

func TestE2E_shell_crossTab_reviewCount_then_collectionsEmptyState(t *testing.T) {
	test.NewTempApp(t)

	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().Unix()
	if err := store.InsertAsset(db, "e2e-cross-tab-hash", "2026/04/14/e2e.jpg", now, now); err != nil {
		t.Fatal(err)
	}

	win := test.NewTempWindow(t, nil)
	win.Resize(fyne.NewSize(1120, 600))
	test.ApplyTheme(t, NewPhotoToolTheme(theme.VariantLight))
	shell := newMainShell(win, db, root, false, nil)
	win.SetContent(shell)

	tapPanel(t, shell, "Review")
	assertPrimaryNavVisible(t, win, shell)
	assertReviewFilterStripOnScreen(t, win, shell)
	var sawCount bool
	for _, lb := range collectLabelsDeep(shell) {
		if strings.HasPrefix(lb.Text, "Matching assets: ") && strings.Contains(lb.Text, "1") {
			sawCount = true
			break
		}
	}
	if !sawCount {
		t.Fatal(`expected Review to show matching count including "1"`)
	}

	tapPanel(t, shell, "Collections")
	assertPrimaryNavVisible(t, win, shell)
	var sawEmptyCollections bool
	for _, lb := range collectLabelsDeep(shell) {
		if strings.Contains(lb.Text, "No albums yet") {
			sawEmptyCollections = true
			break
		}
	}
	if !sawEmptyCollections {
		t.Fatal(`expected Collections empty state to mention "No albums yet"`)
	}
}
