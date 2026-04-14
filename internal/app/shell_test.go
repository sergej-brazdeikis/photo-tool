package app

import (
	"reflect"
	"testing"
)

func TestPrimaryNavLabels_UXDR13_order(t *testing.T) {
	want := []string{"Upload", "Review", "Collections", "Rejected"}
	if got := PrimaryNavLabels(); !reflect.DeepEqual(got, want) {
		t.Fatalf("PrimaryNavLabels: got %q want %q", got, want)
	}
}

func TestPrimaryNavItems_keysMatchLabels(t *testing.T) {
	seen := make(map[string]struct{}, len(primaryNavItems))
	for _, it := range primaryNavItems {
		if it.label == "" || it.key == "" {
			t.Fatalf("empty label or key: %#v", it)
		}
		if _, dup := seen[it.key]; dup {
			t.Fatalf("duplicate key %q", it.key)
		}
		seen[it.key] = struct{}{}
	}
}

func TestCollectionsNavShouldResetToList_AC12(t *testing.T) {
	t.Parallel()
	if !collectionsNavShouldResetToList("collections", "collections") {
		t.Fatal("re-tap Collections while on Collections should reset list")
	}
	if collectionsNavShouldResetToList("review", "collections") {
		t.Fatal("first navigation into Collections is not a reselect")
	}
	if collectionsNavShouldResetToList("collections", "review") {
		t.Fatal("leaving Collections must not trigger reset predicate")
	}
}

func TestClearReviewUndoIfLeftReview(t *testing.T) {
	t.Parallel()
	t.Run("stays_on_review", func(t *testing.T) {
		t.Parallel()
		var calls int
		clearReviewUndoIfLeftReview("review", "review", func() { calls++ })
		if calls != 0 {
			t.Fatalf("got %d calls want 0", calls)
		}
	})
	t.Run("enters_review", func(t *testing.T) {
		t.Parallel()
		var calls int
		clearReviewUndoIfLeftReview("upload", "review", func() { calls++ })
		if calls != 0 {
			t.Fatalf("got %d calls want 0", calls)
		}
	})
	t.Run("leaves_review_to_rejected", func(t *testing.T) {
		t.Parallel()
		var calls int
		clear := func() { calls++ }
		clearReviewUndoIfLeftReview("review", "rejected", clear)
		if calls != 1 {
			t.Fatalf("got %d calls want 1", calls)
		}
	})
	t.Run("leaves_review_to_upload", func(t *testing.T) {
		t.Parallel()
		var calls int
		clearReviewUndoIfLeftReview("review", "upload", func() { calls++ })
		if calls != 1 {
			t.Fatalf("got %d calls want 1", calls)
		}
	})
	t.Run("not_from_review_no_clear", func(t *testing.T) {
		t.Parallel()
		var calls int
		clearReviewUndoIfLeftReview("rejected", "upload", func() { calls++ })
		if calls != 0 {
			t.Fatalf("got %d calls want 0", calls)
		}
	})
	t.Run("nil_clear_safe", func(t *testing.T) {
		t.Parallel()
		clearReviewUndoIfLeftReview("review", "collections", nil)
	})
}
