package store

import (
	"path/filepath"
	"strings"
	"testing"

	"photo-tool/internal/config"
	"photo-tool/internal/domain"
)

func TestCountAssetsForReview_defaultsAndExclusions(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ins := func(rel, hash string, rejected int, deletedUnix any, rating any) int64 {
		t.Helper()
		res, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, deleted_at_unix, rating)
VALUES (?, ?, 1, 1, ?, ?, ?)`, hash, rel, rejected, deletedUnix, rating)
		if err != nil {
			t.Fatal(err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}

	ins("a/a.jpg", "h1", 0, nil, nil) // active, unrated
	ins("a/b.jpg", "h2", 1, nil, nil) // rejected
	ins("a/c.jpg", "h3", 0, 99, nil)  // soft-deleted
	aidRated := ins("a/d.jpg", "h4", 0, nil, 4)
	_ = ins("a/e.jpg", "h5", 0, nil, 2)

	n, err := CountAssetsForReview(db, domain.ReviewFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("default count: got %d want 3 (active, non-rejected, not deleted)", n)
	}

	three := 3
	n, err = CountAssetsForReview(db, domain.ReviewFilters{MinRating: &three})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("min rating 3: got %d want 1 (only rated 4; unrated and 2 excluded)", n)
	}

	collID, err := CreateCollection(db, "Album", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := LinkAssetsToCollection(db, collID, []int64{aidRated}); err != nil {
		t.Fatal(err)
	}

	n, err = CountAssetsForReview(db, domain.ReviewFilters{CollectionID: &collID})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("collection filter: got %d want 1", n)
	}
}

func TestListCollections_order(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := CreateCollection(db, "zebra", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateCollection(db, "Alpha", ""); err != nil {
		t.Fatal(err)
	}

	rows, err := ListCollections(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Name != "Alpha" || rows[1].Name != "zebra" {
		t.Fatalf("list collections: %+v", rows)
	}
}

func TestReviewBrowseBaseWhere_contract(t *testing.T) {
	const want = `rejected = 0 AND deleted_at_unix IS NULL`
	if ReviewBrowseBaseWhere != want {
		t.Fatalf("browse base WHERE changed (Story 2.3 list must match count): got %q", ReviewBrowseBaseWhere)
	}
}

func TestReviewRejectedBaseWhere_contract(t *testing.T) {
	const want = `rejected = 1 AND deleted_at_unix IS NULL`
	if ReviewRejectedBaseWhere != want {
		t.Fatalf("rejected bucket base WHERE changed (Story 2.6 list must match count): got %q", ReviewRejectedBaseWhere)
	}
}

func TestReviewFilterWhereSuffix_empty(t *testing.T) {
	suf, args, err := ReviewFilterWhereSuffix(domain.ReviewFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if suf != "" || len(args) != 0 {
		t.Fatalf("empty filters: suffix %q args %#v", suf, args)
	}
}

func TestReviewFilterWhereSuffix_invalidMinRating(t *testing.T) {
	bad := 99
	_, _, err := ReviewFilterWhereSuffix(domain.ReviewFilters{MinRating: &bad})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestReviewFilterWhereSuffix_collectionAndRatingArgOrder(t *testing.T) {
	cid := int64(42)
	three := 3
	suf, args, err := ReviewFilterWhereSuffix(domain.ReviewFilters{
		CollectionID: &cid,
		MinRating:    &three,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(suf, "asset_collections") || !strings.Contains(suf, "rating >=") {
		t.Fatalf("unexpected suffix: %q", suf)
	}
	if len(args) != 2 {
		t.Fatalf("args: %#v", args)
	}
	if args[0] != cid {
		t.Fatalf("first arg: got %v want collection id", args[0])
	}
	if args[1] != 3 {
		t.Fatalf("second arg: got %v want min rating", args[1])
	}
}

func TestListAssetsForReview_matchesCountAndPaging(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ins := func(rel, hash string, capUnix int64, rating any) {
		t.Helper()
		if _, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, deleted_at_unix, rating)
VALUES (?, ?, ?, 1, 0, NULL, ?)`, hash, rel, capUnix, rating); err != nil {
			t.Fatal(err)
		}
	}

	ins("a/z.jpg", "hz", 300, nil)
	ins("a/y.jpg", "hy", 200, 3)
	ins("a/x.jpg", "hx", 100, 5)

	f := domain.ReviewFilters{}
	n, err := CountAssetsForReview(db, f)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("count: got %d want 3", n)
	}

	page0, err := ListAssetsForReview(db, f, 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(page0) != 2 {
		t.Fatalf("page0 len: got %d want 2", len(page0))
	}
	if page0[0].RelPath != "a/z.jpg" || page0[1].RelPath != "a/y.jpg" {
		t.Fatalf("order page0: %#v", page0)
	}
	if page0[1].Rating == nil || *page0[1].Rating != 3 {
		t.Fatalf("rating: %#v", page0[1].Rating)
	}

	page1, err := ListAssetsForReview(db, f, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page1) != 1 || page1[0].RelPath != "a/x.jpg" {
		t.Fatalf("page1: %#v", page1)
	}

	empty, err := ListAssetsForReview(db, f, 10, 99)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("empty page: %#v", empty)
	}
}

func TestListAssetsForReview_invalidLimitOrOffset(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := ListAssetsForReview(db, domain.ReviewFilters{}, 0, 0); err == nil {
		t.Fatal("limit0: expected error")
	}
	if _, err := ListAssetsForReview(db, domain.ReviewFilters{}, 10, -1); err == nil {
		t.Fatal("negative offset: expected error")
	}
}

func TestListAssetsForReview_excludesRejectedAndDeletedLikeCount(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, deleted_at_unix, rating)
VALUES ('h1', 'ok.jpg', 1, 1, 0, NULL, NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, deleted_at_unix, rating)
VALUES ('h2', 'rej.jpg', 1, 1, 1, NULL, NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, deleted_at_unix, rating)
VALUES ('h3', 'del.jpg', 1, 1, 0, 99, NULL)`); err != nil {
		t.Fatal(err)
	}

	n, err := CountAssetsForReview(db, domain.ReviewFilters{})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := ListAssetsForReview(db, domain.ReviewFilters{}, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(rows)) || n != 1 || rows[0].RelPath != "ok.jpg" {
		t.Fatalf("parity: count=%d rows=%#v", n, rows)
	}
}

func TestCountAndListAssetsForReview_tagFilter_andStaleTagID(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := ForeignKeyCheck(db); err != nil {
		t.Fatal(err)
	}

	ins := func(rel, hash string) int64 {
		t.Helper()
		res, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, deleted_at_unix, rating)
VALUES (?, ?, 1, 1, 0, NULL, NULL)`, hash, rel)
		if err != nil {
			t.Fatal(err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	a1 := ins("a/1.jpg", "h1")
	a2 := ins("a/2.jpg", "h2")
	t1, err := FindOrCreateTagByLabel(db, "vacation")
	if err != nil {
		t.Fatal(err)
	}
	if err := LinkTagToAssets(db, t1, []int64{a1}); err != nil {
		t.Fatal(err)
	}

	fAny := domain.ReviewFilters{}
	nAny, err := CountAssetsForReview(db, fAny)
	if err != nil {
		t.Fatal(err)
	}
	if nAny != 2 {
		t.Fatalf("any tag count: got %d want 2", nAny)
	}

	fTag := domain.ReviewFilters{TagID: &t1}
	nTag, err := CountAssetsForReview(db, fTag)
	if err != nil {
		t.Fatal(err)
	}
	if nTag != 1 {
		t.Fatalf("tag filter count: got %d want 1", nTag)
	}
	rows, err := ListAssetsForReview(db, fTag, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ID != a1 {
		t.Fatalf("tag filter list: %#v", rows)
	}

	stale := int64(999999)
	fStale := domain.ReviewFilters{TagID: &stale}
	nStale, err := CountAssetsForReview(db, fStale)
	if err != nil {
		t.Fatal(err)
	}
	if nStale != 0 {
		t.Fatalf("stale tag id count: got %d want 0", nStale)
	}
	rowsStale, err := ListAssetsForReview(db, fStale, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rowsStale) != 0 {
		t.Fatalf("stale tag list: %#v", rowsStale)
	}

	// Tag with zero linked assets: empty grid, count0
	t2, err := FindOrCreateTagByLabel(db, "unused")
	if err != nil {
		t.Fatal(err)
	}
	fUnused := domain.ReviewFilters{TagID: &t2}
	nUnused, err := CountAssetsForReview(db, fUnused)
	if err != nil {
		t.Fatal(err)
	}
	if nUnused != 0 {
		t.Fatalf("unused tag count: got %d want 0", nUnused)
	}
	_ = a2
}

func TestCountAndListRejectedForReview_matchesBrowseFiltersAndSort(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ins := func(rel, hash string, capUnix int64, rejected int, rating any) int64 {
		t.Helper()
		res, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, deleted_at_unix, rating)
VALUES (?, ?, ?, 1, ?, NULL, ?)`, hash, rel, capUnix, rejected, rating)
		if err != nil {
			t.Fatal(err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}

	ins("browse.jpg", "hb", 50, 0, nil)
	rLow := ins("rej-low.jpg", "hr1", 200, 1, 2)
	rHigh := ins("rej-high.jpg", "hr2", 300, 1, 5)

	f := domain.ReviewFilters{}
	n, err := CountRejectedForReview(db, f)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("rejected count: got %d want 2", n)
	}
	rows, err := ListRejectedForReview(db, f, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(rows)) != n {
		t.Fatalf("count/list parity: count=%d len=%d", n, len(rows))
	}
	if rows[0].ID != rHigh || rows[1].ID != rLow {
		t.Fatalf("sort: %#v", rows)
	}
	if rows[0].Rejected != 1 {
		t.Fatalf("rejected flag: %#v", rows[0])
	}

	three := 3
	nMin, err := CountRejectedForReview(db, domain.ReviewFilters{MinRating: &three})
	if err != nil {
		t.Fatal(err)
	}
	if nMin != 1 {
		t.Fatalf("min rating on rejected bucket: got %d want 1", nMin)
	}
	rowsMin, err := ListRejectedForReview(db, domain.ReviewFilters{MinRating: &three}, 5, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rowsMin) != 1 || rowsMin[0].ID != rHigh {
		t.Fatalf("filtered rejected list: %#v", rowsMin)
	}

	collID, err := CreateCollection(db, "Bin", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := LinkAssetsToCollection(db, collID, []int64{rLow}); err != nil {
		t.Fatal(err)
	}
	nColl, err := CountRejectedForReview(db, domain.ReviewFilters{CollectionID: &collID})
	if err != nil {
		t.Fatal(err)
	}
	if nColl != 1 {
		t.Fatalf("collection filter rejected: got %d want 1", nColl)
	}
}

func TestListRejectedForReview_invalidLimitOrOffset(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := ListRejectedForReview(db, domain.ReviewFilters{}, 0, 0); err == nil {
		t.Fatal("limit0: expected error")
	}
	if _, err := ListRejectedForReview(db, domain.ReviewFilters{}, 10, -1); err == nil {
		t.Fatal("negative offset: expected error")
	}
}

func TestCountRejectedForReview_excludesDeletedAndNonRejected(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, deleted_at_unix)
VALUES ('h1', 'ok-rej.jpg', 1, 1, 1, NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, deleted_at_unix)
VALUES ('h2', 'browse.jpg', 1, 1, 0, NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, deleted_at_unix)
VALUES ('h3', 'rej-del.jpg', 1, 1, 1, 99)`); err != nil {
		t.Fatal(err)
	}

	n, err := CountRejectedForReview(db, domain.ReviewFilters{})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := ListRejectedForReview(db, domain.ReviewFilters{}, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 || len(rows) != 1 || rows[0].RelPath != "ok-rej.jpg" {
		t.Fatalf("parity: count=%d rows=%#v", n, rows)
	}
}

func TestReviewFilterWhereSuffix_tagArgOrderWithCollection(t *testing.T) {
	cid := int64(7)
	tid := int64(99)
	suf, args, err := ReviewFilterWhereSuffix(domain.ReviewFilters{
		CollectionID: &cid,
		TagID:        &tid,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(suf, "asset_collections") || !strings.Contains(suf, "asset_tags") {
		t.Fatalf("suffix: %q", suf)
	}
	if len(args) != 2 || args[0] != cid || args[1] != tid {
		t.Fatalf("args: %#v", args)
	}
}

// Locks placeholder order for Count/List/ListIDs/Rejected: collection id, min rating, tag id.
// A refactor that reorders SQL fragments without reordering args breaks every review query at once.
func TestReviewFilterWhereSuffix_collectionMinRatingTagArgOrder(t *testing.T) {
	cid := int64(11)
	three := 3
	tid := int64(17)
	suf, args, err := ReviewFilterWhereSuffix(domain.ReviewFilters{
		CollectionID: &cid,
		MinRating:    &three,
		TagID:        &tid,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(suf, "asset_collections") || !strings.Contains(suf, "rating >=") || !strings.Contains(suf, "asset_tags") {
		t.Fatalf("suffix: %q", suf)
	}
	if len(args) != 3 {
		t.Fatalf("args: %#v", args)
	}
	if args[0] != cid || args[1] != three || args[2] != tid {
		t.Fatalf("arg order: %#v", args)
	}
}

// ListAssetIDsForReview backs filtered package/share selection; it must stay cardinality- and order-locked
// to CountAssetsForReview and ListAssetsForReview (same WHERE + ORDER BY), or multi-select flows silently disagree with the grid count.
func TestListAssetIDsForReview_matchesCountAndListAssetsForReviewOrder(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ins := func(rel, hash string, capUnix int64, rating int) int64 {
		t.Helper()
		res, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, deleted_at_unix, rating)
VALUES (?, ?, ?, 1, 0, NULL, ?)`, hash, rel, capUnix, rating)
		if err != nil {
			t.Fatal(err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}

	aNewer := ins("a/newer.jpg", "h1", 200, 4)
	aOlder := ins("a/older.jpg", "h2", 100, 5)
	aExcluded := ins("a/low.jpg", "h3", 50, 2)

	collID, err := CreateCollection(db, "Trip", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{aNewer, aOlder, aExcluded} {
		if err := LinkAssetsToCollection(db, collID, []int64{id}); err != nil {
			t.Fatal(err)
		}
	}

	t1, err := FindOrCreateTagByLabel(db, "keep")
	if err != nil {
		t.Fatal(err)
	}
	if err := LinkTagToAssets(db, t1, []int64{aNewer, aOlder}); err != nil {
		t.Fatal(err)
	}

	three := 3
	f := domain.ReviewFilters{
		CollectionID: &collID,
		MinRating:    &three,
		TagID:        &t1,
	}

	n, err := CountAssetsForReview(db, f)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("count: got %d want 2 (older excluded by min rating)", n)
	}

	ids, err := ListAssetIDsForReview(db, f)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(ids)) != n {
		t.Fatalf("ids len: got %d want %d", len(ids), n)
	}

	rows, err := ListAssetsForReview(db, f, int(n), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != int(n) {
		t.Fatalf("rows len: got %d want %d", len(rows), n)
	}
	for i := range ids {
		if rows[i].ID != ids[i] {
			t.Fatalf("order mismatch at %d: list row id=%d ids id=%d", i, rows[i].ID, ids[i])
		}
	}
	if ids[0] != aNewer || ids[1] != aOlder {
		t.Fatalf("expected capture_time_unix DESC: got %#v", ids)
	}
	_ = aExcluded
}
