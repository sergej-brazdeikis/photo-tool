package app

import (
	"strings"
	"testing"
	"time"

	"photo-tool/internal/config"
	"photo-tool/internal/domain"
	"photo-tool/internal/fixture"
	"photo-tool/internal/store"
)

func TestScale_ReviewGrid_rowCountAt96(t *testing.T) {
	if testing.Short() {
		t.Skip("scale grid row count skipped in -short")
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
	total, err := store.CountAssetsForReview(db, domain.ReviewFilters{})
	if err != nil {
		t.Fatal(err)
	}
	got := reviewGridListRowCount(total)
	want := int((total + reviewGridColumns - 1) / reviewGridColumns)
	if got != want {
		t.Fatalf("rowCount=%d want %d for total=%d", got, want, total)
	}
	if total < 80 {
		t.Fatalf("S4 active total=%d too low", total)
	}
}

func TestScale_FilterPartialSubset_count(t *testing.T) {
	if testing.Short() {
		t.Skip("scale filter subset skipped in -short")
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
	var collID int64
	for _, c := range cols {
		if c.Name == "UXCapAlb" {
			collID = c.ID
			break
		}
	}
	if collID == 0 {
		t.Fatal("UXCapAlb missing")
	}
	tags, err := store.ListTags(db)
	if err != nil {
		t.Fatal(err)
	}
	var tagID int64
	for _, tg := range tags {
		if tg.Label == "UXCapTag" {
			tagID = tg.ID
			break
		}
	}
	if tagID == 0 {
		t.Fatal("UXCapTag missing")
	}
	f := domain.ReviewFilters{CollectionID: &collID, TagID: &tagID}
	n, err := store.CountAssetsForReview(db, f)
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("expected non-empty partial filter subset")
	}
	if n >= 500 {
		t.Fatalf("partial subset should be smaller than full library, got %d", n)
	}
	rows, err := store.ListAssetsForReview(db, f, 48, 0)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(rows)) > n {
		t.Fatalf("page exceeds count")
	}
}

func TestScale_BulkTag_50Selection(t *testing.T) {
	if testing.Short() {
		t.Skip("scale bulk tag skipped in -short")
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
	rows, err := store.ListAssetsForReview(db, domain.ReviewFilters{}, 60, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 50 {
		t.Fatalf("need 50 ids, got %d", len(rows))
	}
	var ids []int64
	for _, r := range rows[:50] {
		ids = append(ids, r.ID)
	}
	tid, err := store.FindOrCreateTagByLabel(db, "ScaleBulk50")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.LinkTagToAssets(db, tid, ids); err != nil {
		t.Fatal(err)
	}
	tagID := tid
	f := domain.ReviewFilters{TagID: &tagID}
	n, err := store.CountAssetsForReview(db, f)
	if err != nil {
		t.Fatal(err)
	}
	if n < 50 {
		t.Fatalf("tag filter count=%d want ≥50", n)
	}
}

func TestScale_BulkTag_500Selection(t *testing.T) {
	if testing.Short() {
		t.Skip("scale bulk tag 500 skipped in -short")
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
	rows, err := store.ListAssetsForReview(db, domain.ReviewFilters{}, 600, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 500 {
		t.Fatalf("need 500 ids, got %d", len(rows))
	}
	var ids []int64
	for _, r := range rows[:500] {
		ids = append(ids, r.ID)
	}
	tid, err := store.FindOrCreateTagByLabel(db, "ScaleBulk500")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.LinkTagToAssets(db, tid, ids); err != nil {
		t.Fatal(err)
	}
	tagID := tid
	f := domain.ReviewFilters{TagID: &tagID}
	n, err := store.CountAssetsForReview(db, f)
	if err != nil {
		t.Fatal(err)
	}
	if n < 500 {
		t.Fatalf("tag filter count=%d want ≥500", n)
	}
}

func TestScale_Upload_batchReceipt50(t *testing.T) {
	sum := domain.OperationSummary{Added: 50, SkippedDuplicate: 2, Failed: 1}
	got := summarizeDoneMessage(sum, false, false)
	if !strings.Contains(got, "Added 50") {
		t.Fatalf("batch receipt missing Added 50: %q", got)
	}
	if !strings.Contains(got, "skipped duplicate 2") {
		t.Fatalf("batch receipt missing duplicate count: %q", got)
	}
}

func TestScale_Upload_batchReceiptAllDuplicate(t *testing.T) {
	sum := domain.OperationSummary{Added: 0, SkippedDuplicate: 50, Failed: 0}
	got := summarizeDoneMessage(sum, false, false)
	if !strings.Contains(got, "Added 0") || !strings.Contains(got, "skipped duplicate 50") {
		t.Fatalf("all-duplicate receipt: %q", got)
	}
}

func TestScale_Upload_batchReceiptAllFailed(t *testing.T) {
	sum := domain.OperationSummary{Added: 0, SkippedDuplicate: 0, Failed: 50}
	got := summarizeDoneMessage(sum, false, false)
	if !strings.Contains(got, "failed 50") {
		t.Fatalf("all-failed receipt: %q", got)
	}
	if !strings.Contains(got, "For items that failed") {
		t.Fatalf("all-failed receipt missing next step: %q", got)
	}
}

// TestScale_SC3_loupeRatingPersistUnder1s checks SC-3: rating persist completes within 1s (software driver).
func TestScale_SC3_loupeRatingPersistUnder1s(t *testing.T) {
	if testing.Short() {
		t.Skip("SC-3 timing skipped in -short")
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
	rows, err := store.ListAssetsForReview(db, domain.ReviewFilters{}, 1, 0)
	if err != nil || len(rows) == 0 {
		t.Fatal(err)
	}
	start := time.Now()
	if err := store.UpdateAssetRating(db, rows[0].ID, 4); err != nil {
		t.Fatal(err)
	}
	got, err := store.ListAssetsForReview(db, domain.ReviewFilters{}, 1, 0)
	if err != nil || len(got) == 0 {
		t.Fatal(err)
	}
	if got[0].Rating == nil || *got[0].Rating != 4 {
		t.Fatal("rating not persisted")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("SC-3 rating persist took %v, want ≤1s", elapsed)
	}
}

// TestScale_RejectUndoStackCap documents AC8 undo stack cap at scale (unit; no 129-reject PNG journey).
func TestScale_RejectUndoStackCap(t *testing.T) {
	var s reviewRejectUndoStack
	for i := int64(1); i <= maxReviewRejectUndoIDs+5; i++ {
		s.Push(i)
	}
	if g := s.Len(); g != maxReviewRejectUndoIDs {
		t.Fatalf("Len=%d want cap %d", g, maxReviewRejectUndoIDs)
	}
	id, ok := s.Pop()
	if !ok || id != maxReviewRejectUndoIDs+5 {
		t.Fatalf("pop after cap: got id=%d ok=%v want %d", id, ok, maxReviewRejectUndoIDs+5)
	}
}

func TestScale_PackageEligibleMatchesFilteredSelection(t *testing.T) {
	if testing.Short() {
		t.Skip("package eligible parity skipped in -short")
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
	var collID int64
	for _, c := range cols {
		if c.Name == "UXCapAlb" {
			collID = c.ID
			break
		}
	}
	if collID == 0 {
		t.Fatal("UXCapAlb missing")
	}
	f := domain.ReviewFilters{CollectionID: &collID}
	total, err := store.CountAssetsForReview(db, f)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := store.ListAssetsForReview(db, f, int(total), 0)
	if err != nil {
		t.Fatal(err)
	}
	var ids []int64
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	eligible, err := store.PackagePrepareEligibleForMint(t.Context(), db, ids)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(eligible)) != total {
		t.Fatalf("eligible=%d filter count=%d ids=%d", len(eligible), total, len(ids))
	}
}
