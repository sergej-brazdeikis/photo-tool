package app

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"photo-tool/internal/store"
)

func TestLoupeSharePreviewProceedToMint(t *testing.T) {
	t.Parallel()
	t.Run("cancel_leaves_mint_path_inactive", func(t *testing.T) {
		t.Parallel()
		proceed, drift := loupeSharePreviewProceedToMint(false, 1, func() int64 { return 1 })
		if proceed || drift {
			t.Fatalf("cancel: proceed=%v drift=%v", proceed, drift)
		}
	})
	t.Run("confirm_and_match_proceeds", func(t *testing.T) {
		t.Parallel()
		proceed, drift := loupeSharePreviewProceedToMint(true, 7, func() int64 { return 7 })
		if !proceed || drift {
			t.Fatalf("match: proceed=%v drift=%v", proceed, drift)
		}
	})
	t.Run("confirm_but_selection_drift_no_mint", func(t *testing.T) {
		t.Parallel()
		proceed, drift := loupeSharePreviewProceedToMint(true, 1, func() int64 { return 2 })
		if proceed || !drift {
			t.Fatalf("drift: proceed=%v drift=%v", proceed, drift)
		}
	})
	t.Run("nil_current_fn_after_confirm_fails_closed", func(t *testing.T) {
		t.Parallel()
		proceed, drift := loupeSharePreviewProceedToMint(true, 3, nil)
		if proceed || drift {
			t.Fatalf("nil fn: proceed=%v drift=%v (want no mint, no drift flag)", proceed, drift)
		}
	})
	t.Run("confirm_invalid_preview_id_fails_closed", func(t *testing.T) {
		t.Parallel()
		proceed, drift := loupeSharePreviewProceedToMint(true, 0, func() int64 { return 0 })
		if proceed || drift {
			t.Fatalf("preview id 0: proceed=%v drift=%v", proceed, drift)
		}
	})
}

func TestUserFacingShareMintErrText_logsHint(t *testing.T) {
	t.Parallel()
	s := userFacingShareMintErrText(errors.New("some sqlite failure"))
	if !strings.Contains(s, "log output") {
		t.Fatalf("expected log hint suffix, got %q", s)
	}
	if s == "" {
		t.Fatal("expected non-empty copy for mint failure")
	}
}

func TestUserFacingShareMintErrText_nilIsEmpty(t *testing.T) {
	t.Parallel()
	if s := userFacingShareMintErrText(nil); s != "" {
		t.Fatalf("nil err: got %q want empty", s)
	}
}

func TestUserFacingShareMintErrText_wrappedPackageSentinels(t *testing.T) {
	t.Parallel()
	// Story 4.1 AC7h: errors.Is stability through fmt.Errorf chains (TOCTOU mint paths may wrap).
	tooMany := fmt.Errorf("mint: %w", store.ErrPackageTooManyAssets)
	if s := userFacingShareMintErrText(tooMany); !strings.Contains(s, "500") {
		t.Fatalf("wrapped ErrPackageTooManyAssets: %q", s)
	}
	none := fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", store.ErrPackageNoEligibleAssets))
	if s := userFacingShareMintErrText(none); !strings.Contains(s, "None of the photos") {
		t.Fatalf("wrapped ErrPackageNoEligibleAssets: %q", s)
	}
}

func TestLoupeShareSelectionMatchesPreview(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		preview, current int64
		want             bool
	}{
		{1, 1, true},
		{1, 2, false},
		{0, 1, false},
		{-1, -1, false},
	} {
		if got := loupeShareSelectionMatchesPreview(tc.preview, tc.current); got != tc.want {
			t.Fatalf("preview=%d current=%d: got %v want %v", tc.preview, tc.current, got, tc.want)
		}
	}
}
