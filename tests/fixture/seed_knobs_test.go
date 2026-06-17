package fixture_test

import (
	"testing"

	"photo-tool/internal/config"
	"photo-tool/internal/domain"
	"photo-tool/internal/fixture"
	"photo-tool/internal/store"
)

func TestSeedLibrary_distributionKnobs(t *testing.T) {
	root := t.TempDir()
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, _, err = fixture.SeedLibrary(db, fixture.SeedOptions{
		Tier:      fixture.TierS4,
		Root:      root,
		RatedPct:  50,
		DaySpread: 10,
		TagLabels: []string{"KnobTagA", "KnobTagB"},
		Cameras:   [][2]string{{"Fuji", "X-T5"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	tags, err := store.ListTags(db)
	if err != nil {
		t.Fatal(err)
	}
	var foundA, foundB bool
	for _, tg := range tags {
		switch tg.Label {
		case "KnobTagA":
			foundA = true
		case "KnobTagB":
			foundB = true
		}
	}
	if !foundA || !foundB {
		t.Fatalf("expected KnobTagA/KnobTagB tags, got %v", tags)
	}

	rows, err := store.ListAssetsForReview(db, domain.ReviewFilters{}, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	var rated int
	for _, r := range rows {
		if r.Rating != nil {
			rated++
		}
	}
	if rated == 0 || rated == len(rows) {
		t.Fatalf("RatedPct=50: want partial ratings, got %d/%d rated", rated, len(rows))
	}

	var days int
	seen := map[string]bool{}
	for _, r := range rows {
		if len(r.RelPath) < 10 {
			continue
		}
		day := r.RelPath[8:10]
		if !seen[day] {
			seen[day] = true
			days++
		}
	}
	if days > 10 {
		t.Fatalf("DaySpread=10: saw %d distinct day folders, want ≤10", days)
	}
}
