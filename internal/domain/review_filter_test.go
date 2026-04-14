package domain

import "testing"

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
