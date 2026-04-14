package app

import (
	"fmt"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"image/color"

	"photo-tool/internal/store"
)

func TestCollectionStoreErrText(t *testing.T) {
	t.Parallel()
	if got := collectionStoreErrText(fmt.Errorf(`link asset 1 to collection 2: FOREIGN KEY constraint failed`)); got == "" || got == `link asset 1 to collection 2: FOREIGN KEY constraint failed` {
		t.Fatalf("FK: got %q", got)
	}
	if want := "create collection: name is required"; collectionStoreErrText(fmt.Errorf("%s", want)) != want {
		t.Fatal("validation should pass through")
	}
	wrapped := fmt.Errorf("update collection 3: %w", store.ErrCollectionNotFound)
	if got := collectionStoreErrText(wrapped); !strings.Contains(got, "no longer in the library") {
		t.Fatalf("not-found: got %q want user copy", got)
	}
}

func TestLoupeRatingKeyAllowed(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		id   int64
		want bool
	}{
		{0, false},
		{-1, false},
		{1, true},
	} {
		if got := loupeRatingKeyAllowed(tc.id); got != tc.want {
			t.Fatalf("id=%d: got %v want %v", tc.id, got, tc.want)
		}
	}
}

func TestLoupeStepIndex_clampNoWrap(t *testing.T) {
	t.Parallel()
	total := int64(3)
	if got, moved := loupeStepIndex(0, -1, total); got != 0 || moved {
		t.Fatalf("at first prev: got idx=%d moved=%v", got, moved)
	}
	if got, moved := loupeStepIndex(0, 1, total); got != 1 || !moved {
		t.Fatalf("0+1: got idx=%d moved=%v", got, moved)
	}
	if got, moved := loupeStepIndex(2, 1, total); got != 2 || moved {
		t.Fatalf("at last next: got idx=%d moved=%v", got, moved)
	}
	if got, moved := loupeStepIndex(2, -1, total); got != 1 || !moved {
		t.Fatalf("2-1: got idx=%d moved=%v", got, moved)
	}
	if got, moved := loupeStepIndex(1, 0, total); got != 1 || moved {
		t.Fatalf("delta 0: got idx=%d moved=%v", got, moved)
	}
}

func TestLoupeStepIndex_emptyTotal(t *testing.T) {
	t.Parallel()
	if got, moved := loupeStepIndex(3, 1, 0); got != 3 || moved {
		t.Fatalf("empty: got idx=%d moved=%v", got, moved)
	}
}

func TestLoupeImageLayout_reservesNinetyPercent(t *testing.T) {
	t.Parallel()
	var l loupeImageLayout
	img := canvas.NewRectangle(color.NRGBA{A: 255})
	parent := fyne.NewSize(1000, 800)
	l.Layout([]fyne.CanvasObject{img}, parent)
	if g, w := int(img.Size().Width), 900; g != w {
		t.Fatalf("width: got %d want %d", g, w)
	}
	if g, w := int(img.Size().Height), 720; g != w {
		t.Fatalf("height: got %d want %d", g, w)
	}
}

func TestLoupeStepIndex_clampsOOBStartIndex(t *testing.T) {
	t.Parallel()
	total := int64(3)
	// Past end: first normalize to last, then step back.
	if got, moved := loupeStepIndex(10, -1, total); got != 1 || !moved {
		t.Fatalf("oob then prev: got idx=%d moved=%v", got, moved)
	}
	// Past end: at last after normalize, next is no-op.
	if got, moved := loupeStepIndex(10, 1, total); got != 2 || moved {
		t.Fatalf("oob then next: got idx=%d moved=%v", got, moved)
	}
	// Negative start clamps to 0; prev stays put.
	if got, moved := loupeStepIndex(-3, -1, total); got != 0 || moved {
		t.Fatalf("negative then prev: got idx=%d moved=%v", got, moved)
	}
}
