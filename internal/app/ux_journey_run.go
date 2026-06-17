package app

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"

	"photo-tool/internal/domain"
	"photo-tool/internal/share"
)

// JourneyStepMeta is one row in steps.json.
type JourneyStepMeta struct {
	ID       string `json:"id"`
	Flow     string `json:"flow"`
	File     string `json:"file"`
	Intent   string `json:"intent"`
	TimingMs int64  `json:"timing_ms,omitempty"`
}

// JourneyCaptureOptions configures a UX journey capture run.
type JourneyCaptureOptions struct {
	Win           fyne.Window
	OutDir        string
	DB            *sql.DB
	LibraryRoot   string
	AppMode       string // software_driver | real_binary
	CaptureTool   string
	GoTestTarget  string
	UploadSeeds   []string
	FlowFilter    []string
	UseLightTheme bool
	ShareLoopback *share.Loopback // production real_binary: loopback HTTP for share mint URL UI
}

func (o JourneyCaptureOptions) flowAllowed(flow string) bool {
	if len(o.FlowFilter) == 0 {
		return true
	}
	for _, f := range o.FlowFilter {
		if f == flow {
			return true
		}
	}
	return false
}

// RunUXJourneyCapture executes the primary-shell UX journey and writes PNGs + steps.json.
func RunUXJourneyCapture(o JourneyCaptureOptions) error {
	if o.Win == nil || o.OutDir == "" || o.DB == nil || o.LibraryRoot == "" {
		return fmt.Errorf("journey: missing win, dir, db, or library root")
	}
	if err := os.MkdirAll(o.OutDir, 0o755); err != nil {
		return err
	}
	if o.AppMode == "" {
		o.AppMode = "software_driver"
	}
	if o.CaptureTool == "" {
		o.CaptureTool = "RunUXJourneyCapture"
	}
	setJourneyRealBinary(o.AppMode == "real_binary")
	defer setJourneyRealBinary(false)

	srcDir := filepath.Join(o.OutDir, ".fixture-src")
	uploadA, uploadB, err := SeedUXJourneyFixture(o.DB, o.LibraryRoot, srcDir)
	if err != nil {
		return fmt.Errorf("seed fixture: %w", err)
	}

	var steps []JourneyStepMeta
	stepN := 1

	captureAt := func(flow, id, intent string, w, h float32) error {
		if !o.flowAllowed(flow) {
			return nil
		}
		var capErr error
		journeyOnMain(func() {
			o.Win.Resize(fyne.NewSize(w, h))
			if c := o.Win.Content(); c != nil {
				c.Refresh()
			}
			if o.AppMode == "real_binary" {
				journeyRealBinaryRepaintDirect(o.Win)
			}
			journeySettle()
			var img image.Image
			for attempt := 0; attempt < 3; attempt++ {
				if c := o.Win.Content(); c != nil {
					o.Win.Canvas().Refresh(c)
				}
				var captured image.Image
				var panicked any
				func() {
					defer func() {
						if r := recover(); r != nil {
							panicked = r
						}
					}()
					captured = o.Win.Canvas().Capture()
				}()
				if panicked == nil && captured != nil {
					img = captured
					break
				}
				time.Sleep(120 * time.Millisecond)
			}
			if img == nil {
				capErr = fmt.Errorf("capture %s: Canvas().Capture() failed", id)
				return
			}
			file := fmt.Sprintf("%02d_%s.png", stepN, id)
			stepN++
			path := filepath.Join(o.OutDir, file)
			f, err := os.Create(path)
			if err != nil {
				capErr = err
				return
			}
			if err := png.Encode(f, img); err != nil {
				_ = f.Close()
				capErr = err
				return
			}
			if err := f.Close(); err != nil {
				capErr = err
				return
			}
			steps = append(steps, JourneyStepMeta{ID: id, Flow: flow, File: file, Intent: intent})
		})
		return capErr
	}
	capture := func(flow, id, intent string) error {
		return captureAt(flow, id, intent, 1280, 800)
	}

	mountShell := func() (fyne.CanvasObject, error) {
		var sh fyne.CanvasObject
		journeyOnMain(func() {
			clearUXCaptureReviewGrid()
			sh = NewMainShell(o.Win, o.DB, o.LibraryRoot, o.ShareLoopback)
			o.Win.SetContent(sh)
			journeySettle()
		})
		return sh, nil
	}

	shell, err := mountShell()
	if err != nil {
		return err
	}

	if err := capture("upload", "upload_empty", "Upload: empty list, drop zone, Add/Clear/Import (import disabled)"); err != nil {
		return err
	}
	if err := journeyTapPanel(shell, "Review"); err != nil {
		return err
	}
	journeyAfterNav(o.Win)
	if err := capture("review", "review_grid_default_filters", "Review: filter strip, bulk row, grid with ≥1 asset"); err != nil {
		return err
	}

	if err := journeyOpenReviewLoupeAt(0); err != nil {
		return err
	}
	if err := capture("review", "review_loupe", "Review loupe: image band + controls"); err != nil {
		return err
	}

	if err := journeyTapLoupeShare(o.Win); err != nil {
		return err
	}
	if err := capture("review", "review_loupe_share_preview", "Share preview dialog"); err != nil {
		return err
	}
	if err := journeyTapSharePreviewOverlay(o.Win, "Cancel"); err != nil {
		return err
	}

	if err := journeyCloseLoupe(o.Win); err != nil {
		return err
	}

	if err := journeySetSelectAt(shell, 0, "UXCapAlb"); err != nil {
		return err
	}
	if err := capture("review", "review_filter_collection_album", "Review: Collection filter UXCapAlb"); err != nil {
		return err
	}
	if err := journeySetSelectAt(shell, 1, "5"); err != nil {
		return err
	}
	if err := capture("review", "review_filter_min_rating_no_matches", "Review: Min rating 5 empty"); err != nil {
		return err
	}

	if err := journeySetSelectAt(shell, 0, reviewCollectionSentinel); err != nil {
		return err
	}
	if err := journeySetSelectAt(shell, 1, reviewRatingAny); err != nil {
		return err
	}
	if err := journeySetSelectAt(shell, 2, "UXCapTag"); err != nil {
		return err
	}
	if err := capture("review", "review_filter_tag_uxcaptag", "Review: tag filter UXCapTag"); err != nil {
		return err
	}
	if err := journeySetSelectAt(shell, 2, reviewTagAny); err != nil {
		return err
	}
	if err := journeySetSelectAt(shell, 0, reviewCollectionSentinel); err != nil {
		return err
	}
	if err := journeySetSelectAt(shell, 1, reviewRatingAny); err != nil {
		return err
	}
	if err := capture("review", "review_filters_fr16_reset", "Review: filters reset defaults"); err != nil {
		return err
	}

	// Story 2.10 — best-effort quick assign affordance capture
	if err := capture("review", "review_quick_assign_menu", "Review: quick collection assign affordance visible"); err != nil {
		return err
	}

	if err := journeyTapPanel(shell, "Collections"); err != nil {
		return err
	}
	journeyAfterNav(o.Win)
	if err := capture("collections", "collections_album_list", "Collections: album list"); err != nil {
		return err
	}

	if err := journeyTapButton(shell, "New album"); err != nil {
		return err
	}
	if err := capture("collections", "collections_new_album_form", "Collections: new album dialog"); err != nil {
		return err
	}
	if err := journeyTapButtonInOverlays(o.Win, "Cancel"); err != nil {
		return err
	}

	if err := journeyListSelectAt(shell, 0, 0); err != nil {
		return err
	}
	journeyAfterNav(o.Win)
	journeyWaitCollectionThumbnails(o.Win)
	if err := capture("collections", "collections_album_detail_stars", "Collections: album detail stars"); err != nil {
		return err
	}

	for _, opt := range []struct {
		id     string
		intent string
		value  string
	}{
		{"collections_album_group_by_day", "Collections: group by day", "By day"},
		{"collections_album_group_by_camera", "Collections: group by camera", "By camera"},
	} {
		if err := journeyRadioGroupSelect(shell, opt.value); err != nil {
			return err
		}
		if err := capture("collections", opt.id, opt.intent); err != nil {
			return err
		}
	}
	if err := journeyRadioGroupSelect(shell, "Stars"); err != nil {
		return err
	}
	journeySettle()

	if err := journeyTapButton(shell, "Back"); err != nil {
		return err
	}
	if err := capture("collections", "collections_back_to_album_list", "Collections: back to list"); err != nil {
		return err
	}

	if err := journeyTapPanel(shell, "Rejected"); err != nil {
		return err
	}
	journeyAfterNav(o.Win)
	if err := capture("rejected", "rejected_hidden_grid", "Rejected: grid"); err != nil {
		return err
	}
	if err := journeySetSelectAt(shell, 0, "UXCapNone"); err != nil {
		return err
	}
	journeyAfterNav(o.Win)
	if err := capture("rejected", "rejected_filter_min_rating_empty", "Rejected: narrow filter empty"); err != nil {
		return err
	}
	if err := journeySetSelectAt(shell, 0, reviewCollectionSentinel); err != nil {
		return err
	}
	journeyAfterNav(o.Win)

	nfrW := float32(domain.NFR01WindowMinWidth)
	nfrH := float32(domain.NFR01WindowMinHeight)
	if err := captureAt("rejected", "rejected_nfr01_min_window", "Rejected NFR min window", nfrW, nfrH); err != nil {
		return err
	}
	if err := journeyTapPanel(shell, "Upload"); err != nil {
		return err
	}
	if err := captureAt("upload", "upload_empty_nfr01_min_window", "Upload NFR min window", nfrW, nfrH); err != nil {
		return err
	}
	journeyResizeWin(o.Win, 1280, 800)

	// Phase 2 upload FR-06
	seeds := o.UploadSeeds
	if len(seeds) == 0 {
		seeds = []string{uploadA, uploadB}
	}
	_ = os.Setenv(envUXUploadSeedPaths, strings.Join(seeds, "\n"))
	shell, err = mountShell()
	if err != nil {
		return err
	}
	if err := capture("upload", "upload_paths_staged", "Upload: staged paths"); err != nil {
		return err
	}
	if err := journeyTapButton(shell, "Import selected files"); err != nil {
		return err
	}
	if err := journeyWaitForUploadPostImport(shell, 10*time.Second); err != nil {
		return err
	}
	journeyAfterNav(o.Win)
	if err := capture("upload", "upload_fr06_collection_assign", "FR-06 collection assign"); err != nil {
		return err
	}
	if err := journeyTapButton(shell, "Confirm"); err != nil {
		return err
	}
	if err := capture("upload", "upload_after_confirm_idle", "Upload: after confirm"); err != nil {
		return err
	}

	// Story 2.7 — delete confirm with grid selection
	if err := journeyTapPanel(shell, "Review"); err != nil {
		return err
	}
	journeyAfterNav(o.Win)
	if err := journeySelectReviewGridAt(0); err != nil {
		return err
	}
	if err := journeyTapButton(shell, "Delete selected…"); err != nil {
		return err
	}
	journeyAfterNav(o.Win)
	if err := capture("delete", "review_delete_confirm_dialog", "Delete confirm dialog"); err != nil {
		return err
	}
	_ = journeyTapButtonInOverlays(o.Win, "No")

	// Story 4.1 — package share preview via filtered manifest
	if err := journeyTapButton(shell, "Share (filtered)…"); err != nil {
		return err
	}
	journeyAfterNav(o.Win)
	if err := capture("packages", "review_package_share_preview", "Package share preview"); err != nil {
		return err
	}
	_ = journeyTapSharePreviewOverlay(o.Win, "Cancel")

	manifest := struct {
		AppMode      string            `json:"app_mode"`
		Flows        []string          `json:"flows"`
		Steps        []JourneyStepMeta `json:"steps"`
		CaptureTool  string            `json:"capture_tool"`
		GoTestTarget string            `json:"go_test_target,omitempty"`
		Omissions    []string          `json:"omissions"`
	}{
		AppMode:      o.AppMode,
		Flows:        []string{"upload", "review", "collections", "rejected", "delete", "packages"},
		Steps:        steps,
		CaptureTool:  o.CaptureTool,
		GoTestTarget: o.GoTestTarget,
		Omissions: []string{
			"Native file picker and real OS DPI not captured",
			"CLI scan/import and browser share URL not captured in GUI journey",
			"Theme switch mid-session not captured",
		},
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(o.OutDir, "steps.json"), data, 0o644)
}

// RunUXJourneyRealApp runs capture from the production GUI binary (native driver).
func RunUXJourneyRealApp(win fyne.Window, db *sql.DB, root string, shareLoop *share.Loopback) error {
	flows := UXCaptureFlowFilter()
	if len(flows) == 1 {
		switch flows[0] {
		case "scale_spot":
			o := newScaleJourneyOptions(win, db, root, UXCaptureDir(), "photo-tool-scale-spot-journey")
			o.ShareLoopback = shareLoop
			return runScaleSpotJourney(o)
		case "edge":
			o := newScaleJourneyOptions(win, db, root, UXCaptureDir(), "photo-tool-edge-pack-journey")
			o.ShareLoopback = shareLoop
			return runEdgePackJourney(o)
		case "layout":
			o := newScaleJourneyOptions(win, db, root, UXCaptureDir(), "photo-tool-layout-matrix-journey")
			o.ShareLoopback = shareLoop
			return runLayoutMatrixJourney(o)
		}
	}
	return RunUXJourneyCapture(JourneyCaptureOptions{
		Win:           win,
		OutDir:        UXCaptureDir(),
		DB:            db,
		LibraryRoot:   root,
		AppMode:       "real_binary",
		CaptureTool:   "photo-tool-gui-journey",
		UploadSeeds:   uxUploadSeedPathsFromEnv(),
		FlowFilter:    flows,
		UseLightTheme: strings.ToLower(os.Getenv("PHOTO_TOOL_TEST_THEME")) != "dark",
		ShareLoopback: shareLoop,
	})
}
