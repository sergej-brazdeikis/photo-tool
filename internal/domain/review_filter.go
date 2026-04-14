package domain

import "fmt"

// ReviewFilters is the canonical filter state for the Review surface (FR-15, FR-16).
// It must stay free of Fyne and store imports so Story 2.3 can share the same predicate.
type ReviewFilters struct {
	// CollectionID, when non-nil, restricts to assets linked in asset_collections for that id.
	// Nil means the FR-16 sentinel: no collection constraint (all assets regardless of membership).
	CollectionID *int64
	// MinRating, when non-nil, means "at least N stars" (rating >= N). Assets with rating NULL
	// are excluded for N >= 1; included when MinRating is nil ("Any rating").
	MinRating *int
	// TagID, when non-nil, restricts to assets linked in asset_tags for that tag row id (FR-15).
	// Nil means "Any tag" — no extra predicate (Story 2.2 placeholder semantics).
	TagID *int64
}

// Validate returns an error if filter fields are inconsistent with DB constraints.
func (f ReviewFilters) Validate() error {
	if f.MinRating != nil {
		n := *f.MinRating
		if n < 1 || n > 5 {
			return fmt.Errorf("review filters: min rating must be 1..5, got %d", n)
		}
	}
	if f.TagID != nil && *f.TagID <= 0 {
		return fmt.Errorf("review filters: tag id must be > 0 when set, got %d", *f.TagID)
	}
	return nil
}
