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
	"photo-tool/internal/fixture"
)

type journeyManifest struct {
	AppMode      string            `json:"app_mode"`
	FixtureTier  string            `json:"fixture_tier,omitempty"`
	Flows        []string          `json:"flows"`
	Steps        []JourneyStepMeta `json:"steps"`
	CaptureTool  string            `json:"capture_tool"`
	GoTestTarget string            `json:"go_test_target,omitempty"`
	Omissions    []string          `json:"omissions"`
}

func writeJourneyManifest(outDir string, m journeyManifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "steps.json"), data, 0o644)
}

func newScaleJourneyOptions(win fyne.Window, db *sql.DB, root, outDir, captureTool string) JourneyCaptureOptions {
	appMode := "software_driver"
	if UXJourneyRealBinaryMode() || strings.TrimSpace(os.Getenv(envUXCaptureAppMode)) == "real_binary" {
		appMode = "real_binary"
	}
	return JourneyCaptureOptions{
		Win:           win,
		OutDir:        outDir,
		DB:            db,
		LibraryRoot:   root,
		AppMode:       appMode,
		CaptureTool:   captureTool,
		UseLightTheme: strings.ToLower(os.Getenv("PHOTO_TOOL_TEST_THEME")) != "dark",
	}
}

// RunUXJourneyScaleSpot captures paging/bulk/high-volume UI spots (S4/S5 fixture tier).
func RunUXJourneyScaleSpot(win fyne.Window, db *sql.DB, root string) error {
	o := newScaleJourneyOptions(win, db, root, UXCaptureDir(), "photo-tool-scale-spot-journey")
	return runScaleSpotJourney(o)
}

// RunUXJourneyEdgePack captures empty states, caps, and edge dialogs.
func RunUXJourneyEdgePack(win fyne.Window, db *sql.DB, root string) error {
	o := newScaleJourneyOptions(win, db, root, UXCaptureDir(), "photo-tool-edge-pack-journey")
	return runEdgePackJourney(o)
}

// RunUXJourneyLayoutMatrix captures NFR-01/NFR-07 viewport variants.
func RunUXJourneyLayoutMatrix(win fyne.Window, db *sql.DB, root string) error {
	o := newScaleJourneyOptions(win, db, root, UXCaptureDir(), "photo-tool-layout-matrix-journey")
	return runLayoutMatrixJourney(o)
}

func runScaleSpotJourney(o JourneyCaptureOptions) error {
	setJourneyRealBinary(o.AppMode == "real_binary")
	defer setJourneyRealBinary(false)
	uploadSeeds, err := prepScaleJourney(o)
	if err != nil {
		return err
	}
	jr := &journeyRunner{o: o}
	shell, err := jr.mountShell()
	if err != nil {
		return err
	}
	if err := journeyTapPanel(shell, "Review"); err != nil {
		return err
	}
	journeyWarmRealBinaryWindow(o.Win)
	journeyAfterNav(o.Win)
	journeyWarmRealBinaryWindow(o.Win)
	if err := journeyWaitPainted(o.Win, 5*time.Second); err != nil {
		return err
	}
	if err := jr.capture("scale_spot", "review_grid_page1_full", "Review grid page 1 at scale"); err != nil {
		return err
	}
	journeyOnMain(func() { _ = uxCaptureScrollReviewGridTo(48) })
	journeySettle()
	journeyAfterNav(o.Win)
	if err := jr.capture("scale_spot", "review_grid_page2_top", "Review grid after scroll to page 2"); err != nil {
		return err
	}
	if err := journeySelectReviewGridRange(0, 5); err != nil {
		return err
	}
	journeySyncReviewGridSelection(o.Win)
	if err := jr.capture("scale_spot", "review_grid_bulk_5_selected", "Review bulk bar with 5 selected"); err != nil {
		return err
	}
	journeyClearReviewGridSelection()
	if err := journeySelectReviewGridRange(0, 20); err != nil {
		return err
	}
	journeyOnMain(func() { _ = uxCaptureScrollReviewGridTo(12) })
	journeySyncReviewGridSelection(o.Win)
	if err := jr.capture("scale_spot", "review_grid_bulk_20_selected", "Review bulk bar with 20 selected"); err != nil {
		return err
	}
	journeyClearReviewGridSelection()
	if err := journeyApplyZeroMatchFilters(shell); err != nil {
		return err
	}
	if err := jr.capture("scale_spot", "review_filter_zero_5star", "Review zero-match stacked filters"); err != nil {
		return err
	}
	_ = journeyResetFilters(shell)
	if err := journeySetSelectAt(shell, 0, "UXCapAlb"); err != nil {
		return err
	}
	if err := jr.capture("scale_spot", "review_filter_collection_album", "Review collection filter UXCapAlb"); err != nil {
		return err
	}
	if err := journeySetSelectAt(shell, 0, reviewCollectionSentinel); err != nil {
		return err
	}
	if err := journeySetSelectAt(shell, 2, "UXCapTag"); err != nil {
		return err
	}
	if err := jr.capture("scale_spot", "review_filter_tag_uxcaptag", "Review tag filter UXCapTag"); err != nil {
		return err
	}
	_ = journeySetSelectAt(shell, 2, reviewTagAny)
	_ = journeySetSelectAt(shell, 0, reviewCollectionSentinel)
	if err := journeyOpenReviewLoupeAt(0); err != nil {
		return err
	}
	if err := journeyKeyboardLoupeRating(o.Win, 3); err != nil {
		return err
	}
	if err := jr.capture("scale_spot", "review_loupe_rating", "Review loupe after keyboard rating"); err != nil {
		return err
	}
	if err := journeyLoupeShareRejectedBlock(jr.o.DB, o.Win); err != nil {
		return err
	}
	journeyAfterNav(o.Win)
	if err := jr.capture("scale_spot", "review_loupe_share_rejected_block", "Loupe share blocked on rejected photo"); err != nil {
		return err
	}
	_ = journeyTapButtonInOverlays(o.Win, "OK")
	if err := journeyCloseLoupe(o.Win); err != nil {
		return err
	}
	if err := journeyOpenReviewLoupeAt(0); err != nil {
		return err
	}
	if err := journeyTapLoupeButton(o.Win, "Share…"); err != nil {
		return err
	}
	journeyAfterNav(o.Win)
	if err := jr.capture("scale_spot", "review_loupe_share_preview", "Loupe share preview before mint"); err != nil {
		return err
	}
	if err := journeyTapSharePreviewOverlay(o.Win, "Create link"); err != nil {
		_ = journeyTapSharePreviewOverlay(o.Win, "Cancel")
		jr.omissions = append(jr.omissions, "share_post_mint_copy_url omitted: Create link unavailable (loopback may be off)")
	} else {
		journeyAfterNav(o.Win)
		if err := jr.capture("scale_spot", "share_post_mint_copy_url", "Post-mint share URL with Copy link"); err != nil {
			return err
		}
		_ = journeyTapButtonInOverlays(o.Win, "Close")
	}
	if err := journeyCloseLoupe(o.Win); err != nil {
		return err
	}
	if err := jr.runScaleSpotSharePackageCaptures(shell); err != nil {
		return err
	}
	if err := journeyTapPanel(shell, "Collections"); err != nil {
		return err
	}
	journeyAfterNav(o.Win)
	if err := jr.capture("scale_spot", "collections_album_list_scale", "Collections album list at scale"); err != nil {
		return err
	}
	if UXFixtureScaleTier() == "S5" || UXFixtureScaleTier() == "S6" || UXFixtureScaleTier() == "S7" {
		if err := journeyScrollListAt(shell, 0, 45); err != nil {
			return err
		}
		journeyAfterNav(o.Win)
		if err := jr.capture("scale_spot", "collections_list_scroll_50", "Collections list scrolled at 50 albums"); err != nil {
			return err
		}
	} else {
		jr.omissions = append(jr.omissions, "collections_list_scroll_50 omitted: tier has fewer than 50 albums")
	}
	if err := journeyCollectionSelectNamed(shell, "UXCapNone"); err != nil {
		return err
	}
	journeyAfterNav(o.Win)
	if err := jr.capture("scale_spot", "collections_empty_album", "Collections empty album detail"); err != nil {
		return err
	}
	if err := journeyTapButton(shell, "Back"); err != nil {
		return err
	}
	if err := journeyListSelectAt(shell, 0, 0); err != nil {
		return err
	}
	if err := journeyRadioGroupSelect(shell, "Stars"); err != nil {
		return err
	}
	journeySettle()
	if err := jr.capture("scale_spot", "collections_detail_500_stars", "Collections album detail grouped by stars"); err != nil {
		return err
	}
	if err := journeyRadioGroupSelect(shell, "By day"); err != nil {
		return err
	}
	journeyWaitCollectionThumbnails(o.Win)
	if err := jr.capture("scale_spot", "collections_detail_by_day_dense", "Collections album detail grouped by day"); err != nil {
		return err
	}
	if err := journeyTapButton(shell, "Back"); err != nil {
		return err
	}
	if err := journeyTapPanel(shell, "Upload"); err != nil {
		return err
	}
	if err := jr.capture("scale_spot", "upload_empty_scale", "Upload at scale tier"); err != nil {
		return err
	}
	if len(uploadSeeds) > 0 {
		_ = os.Setenv(envUXUploadSeedPaths, strings.Join(uploadSeeds, "\n"))
		shell, err = jr.mountShell()
		if err != nil {
			return err
		}
		if err := journeyTapPanel(shell, "Upload"); err != nil {
			return err
		}
		if err := jr.capture("scale_spot", "upload_staged_50", "Upload staged paths at scale"); err != nil {
			return err
		}
		if err := journeyTapButton(shell, "Import selected files"); err != nil {
			return err
		}
		if err := journeyWaitForUploadPostImport(shell, 10*time.Second); err != nil {
			return err
		}
		journeyAfterNav(o.Win)
		if len(uploadSeeds) > 0 {
			if err := journeySimulateUploadDrop(uploadSeeds[0]); err != nil {
				return err
			}
			journeyAfterNav(o.Win)
			if err := jr.capture("scale_spot", "upload_drop_during_fr06", "Drop blocked during FR-06 collection step"); err != nil {
				return err
			}
			_ = journeyTapButtonInOverlays(o.Win, "OK")
		}
		if err := jr.capture("scale_spot", "upload_fr06_batch_50", "FR-06 batch import at scale"); err != nil {
			return err
		}
	} else {
		jr.omissions = append(jr.omissions, "upload_staged_50 / upload_fr06_batch_50 omitted: no upload seeds for tier")
	}
	return jr.finish([]string{"scale_spot"}, UXFixtureScaleTier())
}

func (jr *journeyRunner) runScaleSpotSharePackageCaptures(shell fyne.CanvasObject) error {
	if err := journeyTapPanel(shell, "Review"); err != nil {
		return err
	}
	journeyAfterNav(jr.o.Win)
	journeyClearReviewGridSelection()
	spec := fixture.TierSpec(fixture.ParseTier(UXFixtureScaleTier()))
	if spec.Assets >= 150 {
		_ = journeyResetFilters(shell)
		if err := journeyTapButton(shell, "Share (filtered)…"); err != nil {
			return err
		}
		journeyAfterNav(jr.o.Win)
		if err := jr.capture("scale_spot", "review_package_preview_100_truncated", "Package preview truncated at 100 rows"); err != nil {
			return err
		}
		_ = journeyTapSharePreviewOverlay(jr.o.Win, "Cancel")
		journeyClearReviewGridSelection()
	} else {
		jr.omissions = append(jr.omissions, "review_package_preview_100_truncated omitted: tier has fewer than 150 assets")
	}
	if err := journeySelectReviewGridRange(0, 20); err != nil {
		return err
	}
	journeySyncReviewGridSelection(jr.o.Win)
	if err := journeyTapButton(shell, "Share (selection)…"); err != nil {
		return err
	}
	journeyAfterNav(jr.o.Win)
	if err := jr.capture("scale_spot", "review_share_selection_20", "Share package preview from 20 selected"); err != nil {
		return err
	}
	_ = journeyTapSharePreviewOverlay(jr.o.Win, "Cancel")
	journeyClearReviewGridSelection()
	_ = journeyResetFilters(shell)
	journeyAfterNav(jr.o.Win)
	if err := journeyTapButton(shell, "Share (filtered)…"); err != nil {
		return err
	}
	journeyAfterNav(jr.o.Win)
	if err := jr.capture("scale_spot", "review_package_share_preview", "Package share preview from filtered selection"); err != nil {
		return err
	}
	_ = journeyTapSharePreviewOverlay(jr.o.Win, "Cancel")
	return nil
}

func runEdgePackJourney(o JourneyCaptureOptions) error {
	setJourneyRealBinary(o.AppMode == "real_binary")
	defer setJourneyRealBinary(false)
	if _, err := prepScaleJourney(o); err != nil {
		return err
	}
	tier := UXFixtureScaleTier()
	spec := fixture.TierSpec(fixture.ParseTier(tier))
	jr := &journeyRunner{o: o}
	shell, err := jr.mountShell()
	if err != nil {
		return err
	}

	if tier == "S0" || spec.Assets == 0 {
		return runEdgePackS0(jr, shell, o, tier)
	}

	if err := journeyTapPanel(shell, "Review"); err != nil {
		return err
	}
	journeyWarmRealBinaryWindow(o.Win)
	journeyAfterNav(o.Win)
	journeyWarmRealBinaryWindow(o.Win)
	if err := journeyWaitPainted(o.Win, 5*time.Second); err != nil {
		return err
	}
	if err := jr.capture("edge", "review_grid_edge_baseline", "Review grid edge baseline"); err != nil {
		return err
	}
	if tier != "S5R" {
		journeyClearReviewGridSelection()
		if err := journeySelectReviewGridAt(0); err != nil {
			return err
		}
		if err := journeyTapButton(shell, "Reject selected"); err != nil {
			return err
		}
		journeyAfterNav(o.Win)
		if err := jr.capture("edge", "review_undo_reject_single", "Session undo visible after single reject"); err != nil {
			return err
		}
		_ = journeyTapButton(shell, "Undo reject")
		journeyAfterNav(o.Win)
	} else {
		jr.omissions = append(jr.omissions, "review_undo_reject_single omitted: S5R has no active review assets")
	}
	journeyClearReviewGridSelection()
	if err := journeyApplyZeroMatchFilters(shell); err != nil {
		return err
	}
	if err := jr.capture("edge", "review_filter_empty_edge", "Review zero-match filter"); err != nil {
		return err
	}
	_ = journeyResetFilters(shell)

	if tier != "S5R" {
		if err := journeySelectReviewGridAt(0); err != nil {
			return err
		}
		if err := journeyTapButton(shell, "Delete selected…"); err != nil {
			return err
		}
		journeyAfterNav(o.Win)
		if err := jr.capture("edge", "review_delete_confirm_edge", "Delete confirm dialog edge"); err != nil {
			return err
		}
		_ = journeyTapButtonInOverlays(o.Win, "No")
	} else {
		jr.omissions = append(jr.omissions, "review_delete_confirm_edge omitted: S5R has no active review assets")
	}

	if err := journeyTapPanel(shell, "Rejected"); err != nil {
		return err
	}
	journeyAfterNav(o.Win)
	if tier == "S5R" {
		if err := jr.capture("edge", "rejected_grid_500", "Rejected grid at 500 assets"); err != nil {
			return err
		}
	} else if err := jr.capture("edge", "rejected_grid_edge", "Rejected grid edge"); err != nil {
		return err
	}
	if tier != "S5R" {
		if err := journeySetSelectAt(shell, 0, "UXCapNone"); err != nil {
			return err
		}
		journeyAfterNav(o.Win)
		if err := jr.capture("edge", "rejected_filter_min_rating_empty", "Rejected zero-match filter on empty collection"); err != nil {
			return err
		}
		if err := journeySetSelectAt(shell, 0, reviewCollectionSentinel); err != nil {
			return err
		}
		journeyAfterNav(o.Win)
	}

	jr.omissions = append(jr.omissions, "review_library_empty omitted: tier is not S0 and library is seeded")

	if err := journeyTapPanel(shell, "Review"); err != nil {
		return err
	}
	journeyAfterNav(o.Win)

	if spec.Assets >= 50 && tier != "S5R" {
		if err := journeySelectReviewGridRange(0, 50); err != nil {
			return err
		}
		if err := journeyTapButton(shell, "Delete selected…"); err != nil {
			return err
		}
		journeyAfterNav(o.Win)
		if err := jr.capture("edge", "review_delete_confirm_50", "Delete confirm with 50 selected"); err != nil {
			return err
		}
		_ = journeyTapButtonInOverlays(o.Win, "No")
		if err := journeyTapPanel(shell, "Upload"); err != nil {
			return err
		}
		if err := journeyTapPanel(shell, "Review"); err != nil {
			return err
		}
		journeyAfterNav(o.Win)
	} else if tier == "S5R" {
		jr.omissions = append(jr.omissions, "review_delete_confirm_50 omitted: S5R uses rejected bulk delete instead")
	} else {
		jr.omissions = append(jr.omissions, "review_delete_confirm_50 omitted: tier has fewer than 50 assets")
	}

	journeyClearReviewGridSelection()
	if err := journeyApplyZeroMatchDeadlockFilters(shell); err != nil {
		return err
	}
	if err := jr.capture("edge", "review_filter_deadlock_reset", "Review zero-match stacked filters before reset"); err != nil {
		return err
	}
	if err := journeyResetFilters(shell); err != nil {
		return err
	}
	journeyAfterNav(o.Win)
	if tier == "S5R" {
		jr.omissions = append(jr.omissions, "review_filters_fr16_reset omitted: S5R has no active review assets")
	} else if err := jr.capture("edge", "review_filters_fr16_reset", "Review filters reset to FR-16 defaults"); err != nil {
		return err
	}

	if tier == "S6" {
		_ = journeyResetFilters(shell)
		journeyClearReviewGridSelection()
		if err := journeyTapShareFiltered(shell); err != nil {
			return err
		}
		journeyAfterNav(o.Win)
		if err := jr.capture("edge", "review_package_blocked_501", "Share package blocked over 500 eligible"); err != nil {
			return err
		}
		_ = journeyTapButtonInOverlays(o.Win, "OK")
	} else {
		jr.omissions = append(jr.omissions, "review_package_blocked_501 omitted: requires PHOTO_TOOL_UX_FIXTURE_SCALE=S6")
	}

	if tier == "S5R" {
		if err := journeyTapPanel(shell, "Rejected"); err != nil {
			return err
		}
		journeyAfterNav(o.Win)
		if err := journeySelectReviewGridRange(0, 5); err != nil {
			return err
		}
		if err := journeyTapButton(shell, "Delete selected…"); err != nil {
			return err
		}
		journeyAfterNav(o.Win)
		if err := jr.capture("edge", "rejected_bulk_delete_confirm", "Rejected bulk delete confirm"); err != nil {
			return err
		}
		_ = journeyTapButtonInOverlays(o.Win, "No")
	} else {
		jr.omissions = append(jr.omissions, "rejected_bulk_delete_confirm omitted: requires PHOTO_TOOL_UX_FIXTURE_SCALE=S5R")
	}

	if tier == "S4" {
		if err := journeyTapPanel(shell, "Collections"); err != nil {
			return err
		}
		journeyAfterNav(o.Win)
		if err := journeyCollectionSelectNamed(shell, "UXCapAlb"); err != nil {
			return err
		}
		journeyWaitCollectionThumbnails(o.Win)
		if err := journeyTapPanel(shell, "Collections"); err != nil {
			return err
		}
		journeyAfterNav(o.Win)
		if err := jr.capture("edge", "collections_re_tap_nav_reset", "Collections tab re-tap returns to album list"); err != nil {
			return err
		}
		if err := journeyTapPanel(shell, "Review"); err != nil {
			return err
		}
		journeyAfterNav(o.Win)
	} else {
		jr.omissions = append(jr.omissions, "collections_re_tap_nav_reset omitted: requires PHOTO_TOOL_UX_FIXTURE_SCALE=S4")
	}

	if tier != "S5R" {
		if err := journeyOpenReviewLoupeAt(0); err != nil {
			return err
		}
		if err := jr.capture("edge", "review_loupe_edge", "Review loupe at scale tier"); err != nil {
			return err
		}
		if err := journeyCloseLoupe(o.Win); err != nil {
			return err
		}
	} else {
		jr.omissions = append(jr.omissions, "review_loupe_edge omitted: S5R has no active review assets")
	}

	if err := journeyTapPanel(shell, "Upload"); err != nil {
		return err
	}
	journeyAfterNav(o.Win)
	if err := jr.capture("edge", "upload_empty_at_scale", "Upload panel at seeded tier"); err != nil {
		return err
	}

	if tier == "S4" || tier == "S5" || tier == "S6" {
		if err := journeyTapPanel(shell, "Collections"); err != nil {
			return err
		}
		journeyAfterNav(o.Win)
		if err := journeyCollectionSelectNamed(shell, "UXCapNone"); err != nil {
			return err
		}
		journeyWaitCollectionThumbnails(o.Win)
		if err := jr.capture("edge", "collections_empty_album_edge", "Collections empty album at scale"); err != nil {
			return err
		}
		_ = journeyTapButton(shell, "Back")
	} else {
		jr.omissions = append(jr.omissions, "collections_empty_album_edge omitted: requires S4/S5/S6 for UXCapNone")
	}

	if tier != "S5R" {
		if err := journeyTapPanel(shell, "Review"); err != nil {
			return err
		}
		journeyAfterNav(o.Win)
		if err := journeyOpenReviewLoupeAt(0); err != nil {
			return err
		}
		if err := journeyLoupeShareRejectedBlock(o.DB, o.Win); err != nil {
			return err
		}
		journeyAfterNav(o.Win)
		if err := jr.capture("edge", "review_loupe_share_rejected_block", "Loupe share blocked on rejected photo"); err != nil {
			return err
		}
		_ = journeyTapButtonInOverlays(o.Win, "OK")
		if err := journeyCloseLoupe(o.Win); err != nil {
			return err
		}
	} else {
		jr.omissions = append(jr.omissions, "review_loupe_share_rejected_block omitted: S5R has no active review assets")
	}

	return jr.finish([]string{"edge"}, tier)
}

func runEdgePackS0(jr *journeyRunner, shell fyne.CanvasObject, o JourneyCaptureOptions, tier string) error {
	if err := journeyTapPanel(shell, "Review"); err != nil {
		return err
	}
	journeyWarmRealBinaryWindow(o.Win)
	journeyAfterNav(o.Win)
	journeyWarmRealBinaryWindow(o.Win)
	if err := journeyWaitPainted(o.Win, 5*time.Second); err != nil {
		return err
	}
	if err := jr.capture("edge", "review_library_empty", "Review empty library at S0"); err != nil {
		return err
	}
	if err := journeyTapPanel(shell, "Upload"); err != nil {
		return err
	}
	journeyAfterNav(o.Win)
	if err := jr.capture("edge", "upload_empty_s0", "Upload empty at S0"); err != nil {
		return err
	}
	jr.omissions = append(jr.omissions,
		"review_delete_confirm_edge omitted: S0 empty library",
		"rejected_grid_edge omitted: S0 empty library",
		"review_filter_deadlock_reset omitted: S0 empty library",
	)
	return jr.finish([]string{"edge"}, tier)
}

func runLayoutMatrixJourney(o JourneyCaptureOptions) error {
	setJourneyRealBinary(o.AppMode == "real_binary")
	defer setJourneyRealBinary(false)
	if _, err := prepScaleJourney(o); err != nil {
		return err
	}
	jr := &journeyRunner{o: o}
	shell, err := jr.mountShell()
	if err != nil {
		return err
	}
	nfrW := float32(domain.NFR01WindowMinWidth)
	nfrH := float32(domain.NFR01WindowMinHeight)
	if err := journeyTapPanel(shell, "Review"); err != nil {
		return err
	}
	journeyWarmRealBinaryWindow(o.Win)
	journeyAfterNav(o.Win)
	journeyWarmRealBinaryWindow(o.Win)
	if err := journeyWaitPainted(o.Win, 5*time.Second); err != nil {
		return err
	}
	if err := jr.captureAt("layout", "review_grid_nfr01_min_window", "Review grid NFR-01 min", nfrW, nfrH); err != nil {
		return err
	}
	journeyToggleAppTheme(o.Win, true)
	journeyAfterNav(o.Win)
	if err := jr.captureAt("layout", "review_theme_switch_full_grid", "Review grid after light theme switch", nfrW, nfrH); err != nil {
		return err
	}
	journeyToggleAppTheme(o.Win, false)
	if err := journeySelectReviewGridRange(0, 20); err != nil {
		return err
	}
	if err := jr.captureAt("layout", "review_bulk_bar_nfr01_min_window", "Review bulk bar NFR-01 min with 20 selected", nfrW, nfrH); err != nil {
		return err
	}
	if err := journeyOpenReviewLoupeAt(0); err != nil {
		return err
	}
	if err := jr.captureAt("layout", "review_loupe_nfr01_min_window", "Review loupe NFR-01 min", nfrW, nfrH); err != nil {
		return err
	}
	if err := journeyCloseLoupe(o.Win); err != nil {
		return err
	}
	if err := journeyTapPanel(shell, "Collections"); err != nil {
		return err
	}
	journeyAfterNav(o.Win)
	if err := jr.captureAt("layout", "collections_list_nfr01_min_window", "Collections NFR-01 min", nfrW, nfrH); err != nil {
		return err
	}
	if err := journeyListSelectAt(shell, 0, 0); err != nil {
		return err
	}
	if err := jr.captureAt("layout", "collections_detail_nfr01_min_window", "Collections detail NFR-01 min", nfrW, nfrH); err != nil {
		return err
	}
	if err := journeyTapButton(shell, "Back"); err != nil {
		return err
	}
	if err := journeyTapPanel(shell, "Review"); err != nil {
		return err
	}
	journeyAfterNav(o.Win)
	if strings.TrimSpace(os.Getenv("PHOTO_TOOL_SCALE_SKIP_ULTRAWIDE")) != "1" {
		journeyOnMain(func() {
			o.Win.Resize(fyne.NewSize(5120, 1440))
			if c := o.Win.Content(); c != nil {
				c.Refresh()
			}
		})
		journeyWarmRealBinaryWindow(o.Win)
		journeyAfterNav(o.Win)
		if err := jr.captureAt("layout", "review_grid_ultrawide", "Review grid ultrawide 5120×1440", 5120, 1440); err != nil {
			return err
		}
	} else {
		jr.omissions = append(jr.omissions, "review_grid_ultrawide omitted: PHOTO_TOOL_SCALE_SKIP_ULTRAWIDE=1")
	}
	if strings.TrimSpace(os.Getenv("FYNE_SCALE")) == "1.5" {
		if err := jr.capture("layout", "review_filter_strip_fyne_scale_15", "Review filter strip at FYNE_SCALE=1.5"); err != nil {
			return err
		}
	} else {
		jr.omissions = append(jr.omissions, "review_filter_strip_fyne_scale_15 omitted: FYNE_SCALE is not 1.5")
	}
	journeyResizeWin(o.Win, 1280, 800)
	return jr.finish([]string{"layout"}, UXFixtureScaleTier())
}

func prepScaleJourney(o JourneyCaptureOptions) ([]string, error) {
	if err := os.MkdirAll(o.OutDir, 0o755); err != nil {
		return nil, err
	}
	srcDir := filepath.Join(o.OutDir, ".fixture-src")
	if _, _, err := SeedUXJourneyFixture(o.DB, o.LibraryRoot, srcDir); err != nil {
		return nil, fmt.Errorf("seed fixture: %w", err)
	}
	return scaleUploadSeeds(o.LibraryRoot, srcDir), nil
}

func scaleUploadSeeds(root, srcDir string) []string {
	if m, ok, err := fixture.ReadManifest(root); err == nil && ok && len(m.UploadSeeds) > 0 {
		return m.UploadSeeds
	}
	if srcDir == "" {
		srcDir = filepath.Join(root, ".fixture-src")
	}
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "upload_seed_") && strings.HasSuffix(name, ".jpg") {
			out = append(out, filepath.Join(srcDir, name))
		}
	}
	return out
}

type journeyRunner struct {
	o         JourneyCaptureOptions
	steps     []JourneyStepMeta
	stepN     int
	omissions []string
}

func (jr *journeyRunner) mountShell() (fyne.CanvasObject, error) {
	var sh fyne.CanvasObject
	journeyOnMain(func() {
		clearUXCaptureReviewGrid()
		sh = NewMainShell(jr.o.Win, jr.o.DB, jr.o.LibraryRoot, jr.o.ShareLoopback)
		jr.o.Win.SetContent(sh)
		journeySettle()
		if jr.o.AppMode == "real_binary" {
			journeyRealBinaryRepaintDirect(jr.o.Win)
		}
	})
	if jr.o.AppMode == "real_binary" {
		journeyWarmRealBinaryWindow(jr.o.Win)
	}
	return sh, nil
}

func (jr *journeyRunner) capture(flow, id, intent string) error {
	return jr.captureAt(flow, id, intent, 1280, 800)
}

func (jr *journeyRunner) captureAt(flow, id, intent string, w, h float32) error {
	var capErr error
	journeyOnMain(func() {
		start := time.Now()
		jr.o.Win.Resize(fyne.NewSize(w, h))
		if c := jr.o.Win.Content(); c != nil {
			c.Refresh()
		}
		if jr.o.AppMode == "real_binary" {
			journeyRealBinaryRepaintDirect(jr.o.Win)
		}
		journeySettle()
		const captureAttempts = 8
		var img image.Image
		var fallback image.Image
		for attempt := 0; attempt < captureAttempts; attempt++ {
			if c := jr.o.Win.Content(); c != nil {
				jr.o.Win.Canvas().Refresh(c)
			}
			if jr.o.AppMode == "real_binary" {
				journeyRealBinaryRepaintDirect(jr.o.Win)
			}
			var captured image.Image
			var panicked any
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicked = r
					}
				}()
				captured = jr.o.Win.Canvas().Capture()
			}()
			if panicked == nil && captured != nil {
				fallback = captured
				if jr.o.AppMode != "real_binary" || !journeyCaptureLooksBlank(captured) {
					img = captured
					break
				}
			}
			time.Sleep(time.Duration(120+attempt*60) * time.Millisecond)
		}
		if img == nil {
			if fallback != nil && (jr.o.AppMode != "real_binary" || !journeyCaptureLooksBlank(fallback)) {
				img = fallback
			}
		}
		if img == nil {
			capErr = fmt.Errorf("capture %s: Canvas().Capture() failed after %d attempts", id, captureAttempts)
			return
		}
		if jr.o.AppMode == "real_binary" && journeyCaptureLooksBlank(img) {
			capErr = fmt.Errorf("capture %s: unpainted GL frame after %d attempts", id, captureAttempts)
			return
		}
		file := fmt.Sprintf("%02d_%s.png", jr.stepN, id)
		jr.stepN++
		path := filepath.Join(jr.o.OutDir, file)
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
		jr.steps = append(jr.steps, JourneyStepMeta{
			ID:       id,
			Flow:     flow,
			File:     file,
			Intent:   intent,
			TimingMs: time.Since(start).Milliseconds(),
		})
	})
	return capErr
}

func (jr *journeyRunner) finish(flows []string, tier string) error {
	omissions := []string{
		"Native file picker not captured",
		"CLI scan at S8 not captured in GUI journey",
	}
	omissions = append(omissions, jr.omissions...)
	return writeJourneyManifest(jr.o.OutDir, journeyManifest{
		AppMode:     jr.o.AppMode,
		FixtureTier: tier,
		Flows:       flows,
		Steps:       jr.steps,
		CaptureTool: jr.o.CaptureTool,
		Omissions:   omissions,
	})
}

func journeySelectReviewGridRange(from, to int) error {
	var err error
	journeyOnMain(func() {
		err = uxCaptureSelectReviewGridRange(from, to)
		if err == nil {
			journeySettle()
		}
	})
	return err
}
