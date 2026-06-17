package fixture_test

import (
	"errors"
	"testing"

	"photo-tool/internal/config"
	"photo-tool/internal/domain"
	"photo-tool/internal/fixture"
	"photo-tool/internal/store"
)

func TestScale_ListAssetsForReview_pagingAt96(t *testing.T) {
	if testing.Short() {
		t.Skip("scale paging test skipped in -short")
	}
	root := t.TempDir()
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, _, err = fixture.SeedLibrary(db, fixture.SeedOptions{Tier: fixture.TierS4, Root: root})
	if err != nil {
		t.Fatal(err)
	}
	f := domain.ReviewFilters{}
	total, err := store.CountAssetsForReview(db, f)
	if err != nil {
		t.Fatal(err)
	}
	// S4: 96 assets, ~8% rejected → active count < 96
	if total < 80 {
		t.Fatalf("expected at least 80 reviewable assets, got %d", total)
	}
	page0, err := store.ListAssetsForReview(db, f, 48, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(page0) != 48 {
		t.Fatalf("page0 len=%d want 48", len(page0))
	}
	page1, err := store.ListAssetsForReview(db, f, 48, 48)
	if err != nil {
		t.Fatal(err)
	}
	if len(page0)+len(page1) != int(total) && len(page1) > 0 {
		// sum of full pages should match total when total > 48
		if int(total) > 48 && len(page1) == 0 {
			t.Fatalf("expected page1 rows, total=%d", total)
		}
	}
	var sum int
	for off := 0; off < int(total); off += 48 {
		page, err := store.ListAssetsForReview(db, f, 48, off)
		if err != nil {
			t.Fatal(err)
		}
		sum += len(page)
	}
	if int64(sum) != total {
		t.Fatalf("paging sum=%d total=%d", sum, total)
	}
}

func TestScale_PackagePrepareEligible_cap501(t *testing.T) {
	if testing.Short() {
		t.Skip("scale package cap test skipped in -short")
	}
	root := t.TempDir()
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, _, err = fixture.SeedLibrary(db, fixture.SeedOptions{Tier: fixture.TierS6, Root: root})
	if err != nil {
		t.Fatal(err)
	}
	var ids []int64
	rows, err := store.ListAssetsForReview(db, domain.ReviewFilters{}, 600, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	if len(ids) < 501 {
		t.Fatalf("need ≥501 eligible ids, got %d", len(ids))
	}
	_, err = store.PackagePrepareEligibleForMint(t.Context(), db, ids[:501])
	if !errors.Is(err, store.ErrPackageTooManyAssets) {
		t.Fatalf("501 eligible: want ErrPackageTooManyAssets, got %v", err)
	}
	got500, err := store.PackagePrepareEligibleForMint(t.Context(), db, ids[:500])
	if err != nil {
		t.Fatalf("500 eligible: %v", err)
	}
	if len(got500) != 500 {
		t.Fatalf("got %d eligible", len(got500))
	}
}

func TestScale_TagBulk_link600(t *testing.T) {
	if testing.Short() {
		t.Skip("scale tag bulk skipped in -short")
	}
	root := t.TempDir()
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, _, err = fixture.SeedLibrary(db, fixture.SeedOptions{Tier: fixture.TierS7, Root: root})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := store.ListAssetsForReview(db, domain.ReviewFilters{}, 600, 0)
	if err != nil {
		t.Fatal(err)
	}
	var ids []int64
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	if len(ids) < 600 {
		t.Fatalf("need 600 assets, got %d", len(ids))
	}
	tid, err := store.FindOrCreateTagByLabel(db, "ScaleBulkTag")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.LinkTagToAssets(db, tid, ids[:600]); err != nil {
		t.Fatal(err)
	}
}

func TestScale_ListAssetsForReview_underOnePage(t *testing.T) {
	if testing.Short() {
		t.Skip("scale S2 test skipped in -short")
	}
	root := t.TempDir()
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_, _, err = fixture.SeedLibrary(db, fixture.SeedOptions{Tier: fixture.TierS2, Root: root})
	if err != nil {
		t.Fatal(err)
	}
	f := domain.ReviewFilters{}
	total, err := store.CountAssetsForReview(db, f)
	if err != nil {
		t.Fatal(err)
	}
	if total >= 48 {
		t.Fatalf("S2 active count should be under one page, got %d", total)
	}
	page, err := store.ListAssetsForReview(db, f, 48, 0)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(page)) != total {
		t.Fatalf("single page len=%d total=%d", len(page), total)
	}
}

func TestScale_ListAssetsForReview_onePageBoundary(t *testing.T) {
	if testing.Short() {
		t.Skip("scale S3 test skipped in -short")
	}
	root := t.TempDir()
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_, _, err = fixture.SeedLibrary(db, fixture.SeedOptions{Tier: fixture.TierS3, Root: root})
	if err != nil {
		t.Fatal(err)
	}
	f := domain.ReviewFilters{}
	total, err := store.CountAssetsForReview(db, f)
	if err != nil {
		t.Fatal(err)
	}
	page0, err := store.ListAssetsForReview(db, f, 48, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(page0) != 48 && int64(len(page0)) != total {
		t.Fatalf("S3 page0 len=%d total=%d", len(page0), total)
	}
}

func TestScale_ListRejectedForReview_paging(t *testing.T) {
	if testing.Short() {
		t.Skip("scale rejected paging skipped in -short")
	}
	root := t.TempDir()
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_, _, err = fixture.SeedLibrary(db, fixture.SeedOptions{Tier: fixture.TierS5R, Root: root})
	if err != nil {
		t.Fatal(err)
	}
	f := domain.ReviewFilters{}
	total, err := store.CountRejectedForReview(db, f)
	if err != nil {
		t.Fatal(err)
	}
	if total < 400 {
		t.Fatalf("S5R rejected count=%d want many rejected", total)
	}
	var sum int
	for off := 0; off < int(total); off += 48 {
		page, err := store.ListRejectedForReview(db, f, 48, off)
		if err != nil {
			t.Fatal(err)
		}
		sum += len(page)
	}
	if int64(sum) != total {
		t.Fatalf("rejected paging sum=%d total=%d", sum, total)
	}
}

func TestScale_CollectionDetail_sectionPaging(t *testing.T) {
	if testing.Short() {
		t.Skip("scale collection detail skipped in -short")
	}
	root := t.TempDir()
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_, _, err = fixture.SeedLibrary(db, fixture.SeedOptions{Tier: fixture.TierS5, Root: root})
	if err != nil {
		t.Fatal(err)
	}
	cols, err := store.ListCollections(db)
	if err != nil {
		t.Fatal(err)
	}
	var uxCapID int64
	for _, c := range cols {
		if c.Name == "UXCapAlb" {
			uxCapID = c.ID
			break
		}
	}
	if uxCapID == 0 {
		t.Fatal("UXCapAlb not found")
	}
	secs, err := store.ListCollectionStarSections(db, uxCapID)
	if err != nil {
		t.Fatal(err)
	}
	if len(secs) == 0 {
		t.Fatal("expected star sections in UXCapAlb")
	}
	var totalInSections int
	for _, s := range secs {
		rating := s.Rating
		var rows []store.ReviewGridRow
		var err error
		if rating != nil {
			rows, err = store.ListCollectionStarSectionPage(db, uxCapID, rating, 48, 0)
		} else {
			rows, err = store.ListCollectionStarSectionPage(db, uxCapID, nil, 48, 0)
		}
		if err != nil {
			t.Fatal(err)
		}
		totalInSections += len(rows)
	}
	if totalInSections == 0 {
		t.Fatal("expected assets in star sections")
	}
	daySecs, err := store.ListCollectionDaySections(db, uxCapID)
	if err != nil {
		t.Fatal(err)
	}
	if len(daySecs) == 0 {
		t.Fatal("expected day sections")
	}
}

func TestScale_ListAssetsForReview_pagingParity(t *testing.T) {
	if testing.Short() {
		t.Skip("scale paging parity skipped in -short")
	}
	root := t.TempDir()
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_, _, err = fixture.SeedLibrary(db, fixture.SeedOptions{Tier: fixture.TierS5, Root: root})
	if err != nil {
		t.Fatal(err)
	}
	f := domain.ReviewFilters{}
	total, err := store.CountAssetsForReview(db, f)
	if err != nil {
		t.Fatal(err)
	}
	if total <= 96 {
		t.Fatalf("S5 active total=%d want >96 for parity offsets", total)
	}
	seen := map[int64]struct{}{}
	for _, off := range []int{0, 48, 96} {
		rows, err := store.ListAssetsForReview(db, f, 48, off)
		if err != nil {
			t.Fatalf("offset %d: %v", off, err)
		}
		if len(rows) == 0 {
			t.Fatalf("offset %d: empty page", off)
		}
		for _, r := range rows {
			if _, dup := seen[r.ID]; dup {
				t.Fatalf("duplicate asset id %d at offset %d", r.ID, off)
			}
			seen[r.ID] = struct{}{}
		}
	}
}
