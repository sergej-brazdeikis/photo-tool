package domain

import "testing"

// FR-16 headline covers collection + rating; the third strip slot still defaults to “any tag”
// (nil TagID) per Story 2.2 / epic §2.2 — all-nil means no extra predicates.
func TestReviewFilters_FR16DefaultMeansUnconstrained(t *testing.T) {
	var f ReviewFilters
	if f.CollectionID != nil || f.MinRating != nil || f.TagID != nil {
		t.Fatalf("zero value should mean unconstrained on all dimensions: %#v", f)
	}
	if err := f.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestReviewFilters_Validate(t *testing.T) {
	if err := (ReviewFilters{}).Validate(); err != nil {
		t.Fatal(err)
	}
	bad := 0
	if err := (ReviewFilters{MinRating: &bad}).Validate(); err == nil {
		t.Fatal("expected error for min rating 0")
	}
	ok := 3
	if err := (ReviewFilters{MinRating: &ok}).Validate(); err != nil {
		t.Fatal(err)
	}
	badTag := int64(0)
	if err := (ReviewFilters{TagID: &badTag}).Validate(); err == nil {
		t.Fatal("expected error for tag id 0")
	}
	goodTag := int64(1)
	if err := (ReviewFilters{TagID: &goodTag}).Validate(); err != nil {
		t.Fatal(err)
	}
}
