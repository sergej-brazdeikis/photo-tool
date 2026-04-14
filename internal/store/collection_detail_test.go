package store

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"photo-tool/internal/config"
)

func TestCameraLabelFromParts_whitespaceAndJoin(t *testing.T) {
	lbl, ok := CameraLabelFromParts("  Canon  ", " EOS R8 ")
	if !ok || lbl != "Canon EOS R8" {
		t.Fatalf("got %q ok=%v", lbl, ok)
	}
	lbl, ok = CameraLabelFromParts("   ", "Model")
	if !ok || lbl != "Model" {
		t.Fatalf("whitespace make: got %q ok=%v", lbl, ok)
	}
	lbl, ok = CameraLabelFromParts("Make", " \t\n ")
	if !ok || lbl != "Make" {
		t.Fatalf("whitespace model: got %q ok=%v", lbl, ok)
	}
	_, ok = CameraLabelFromParts(" ", " ")
	if ok {
		t.Fatal("expected unknown when both empty after trim")
	}
}

func TestListCollectionStarSections_orderAndOmitEmpty(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	collID, err := CreateCollection(db, "Trip", "")
	if err != nil {
		t.Fatal(err)
	}

	ins := func(rel, hash string, capUnix int64, rating any, rejected int, deleted any, camMake, camModel any) int64 {
		t.Helper()
		res, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rating, rejected, deleted_at_unix, camera_make, camera_model, camera_label)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			hash, rel, capUnix, 1, rating, rejected, deleted, camMake, camModel, nil)
		if err != nil {
			t.Fatal(err)
		}
		id, _ := res.LastInsertId()
		if lbl, ok := CameraLabelFromParts(strPtr(camMake), strPtr(camModel)); ok {
			if _, err := db.Exec(`UPDATE assets SET camera_label = ? WHERE id = ?`, lbl, id); err != nil {
				t.Fatal(err)
			}
		}
		return id
	}

	r3 := 3
	r5 := 5
	id5a := ins("a/a.jpg", "h1", 300, r5, 0, nil, nil, nil)
	id5b := ins("a/b.jpg", "h2", 200, r5, 0, nil, nil, nil)
	id3 := ins("a/c.jpg", "h3", 400, r3, 0, nil, nil, nil)
	idUnrated := ins("a/d.jpg", "h4", 100, nil, 0, nil, nil, nil)
	_ = ins("a/e.jpg", "h5", 500, r5, 1, nil, nil, nil) // rejected — invisible
	_ = ins("a/f.jpg", "h6", 600, r5, 0, 99, nil, nil)  // deleted — invisible

	if err := LinkAssetsToCollection(db, collID, []int64{id5a, id5b, id3, idUnrated}); err != nil {
		t.Fatal(err)
	}

	secs, err := ListCollectionStarSections(db, collID)
	if err != nil {
		t.Fatal(err)
	}
	if len(secs) != 3 {
		t.Fatalf("sections: %+v", secs)
	}
	if secs[0].Rating == nil || *secs[0].Rating != 5 || secs[0].Count != 2 {
		t.Fatalf("first bucket want 5★ x2: %+v", secs[0])
	}
	if secs[1].Rating == nil || *secs[1].Rating != 3 || secs[1].Count != 1 {
		t.Fatalf("second bucket: %+v", secs[1])
	}
	if secs[2].Rating != nil || secs[2].Count != 1 {
		t.Fatalf("unrated: %+v", secs[2])
	}

	rows, err := ListCollectionStarSectionPage(db, collID, intPtr(5), 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].ID != id5a || rows[1].ID != id5b {
		t.Fatalf("5★ time order: got ids %v %v", rows[0].ID, rows[1].ID)
	}
	rows, err = ListCollectionStarSectionPage(db, collID, intPtr(3), 10, 0)
	if err != nil || len(rows) != 1 || rows[0].ID != id3 {
		t.Fatalf("3★: %+v err %v", rows, err)
	}
}

func TestListCollectionDetail_allMembersInvisibleIsEmpty(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	collID, err := CreateCollection(db, "Emptyish", "")
	if err != nil {
		t.Fatal(err)
	}
	res, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, deleted_at_unix)
VALUES ('hx', 'x/x.jpg', 1, 1, 1, NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	aid, _ := res.LastInsertId()
	if err := LinkAssetsToCollection(db, collID, []int64{aid}); err != nil {
		t.Fatal(err)
	}
	n, err := CountCollectionVisibleAssets(db, collID)
	if err != nil || n != 0 {
		t.Fatalf("count=%d err=%v", n, err)
	}
	secs, err := ListCollectionStarSections(db, collID)
	if err != nil || len(secs) != 0 {
		t.Fatalf("secs=%v err=%v", secs, err)
	}
}

func TestListCollectionDetail_ErrCollectionNotFound(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = CountCollectionVisibleAssets(db, 99999)
	if !errors.Is(err, ErrCollectionNotFound) {
		t.Fatalf("got %v", err)
	}
	_, err = ListCollectionStarSections(db, 99999)
	if !errors.Is(err, ErrCollectionNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestListCollectionDaySections_sameLocalDay(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	collID, err := CreateCollection(db, "Days", "")
	if err != nil {
		t.Fatal(err)
	}
	loc := time.Local
	day := time.Date(2024, 3, 15, 10, 0, 0, 0, loc)
	t1 := day.Unix()
	t2 := day.Add(3 * time.Hour).Unix()

	id1 := insertBareAsset(t, db, "h1", "a/1.jpg", t1)
	id2 := insertBareAsset(t, db, "h2", "a/2.jpg", t2)
	if err := LinkAssetsToCollection(db, collID, []int64{id1, id2}); err != nil {
		t.Fatal(err)
	}

	secs, err := ListCollectionDaySections(db, collID)
	if err != nil {
		t.Fatal(err)
	}
	if len(secs) != 1 || secs[0].Count != 2 {
		t.Fatalf("day sections: %+v", secs)
	}
	wantDay := day.Format("2006-01-02")
	if secs[0].DayKey != wantDay {
		t.Fatalf("day key: got %q want %q", secs[0].DayKey, wantDay)
	}
}

func TestListCollectionCameraSections_unknownLast(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	collID, err := CreateCollection(db, "Cam", "")
	if err != nil {
		t.Fatal(err)
	}
	idA := insertAssetCamera(t, db, "ha", "a/a.jpg", 1, "Zebra", "Z1")
	idB := insertAssetCamera(t, db, "hb", "a/b.jpg", 2, "Alpha", "A1")
	idU := insertBareAsset(t, db, "hc", "a/c.jpg", 3)

	if err := LinkAssetsToCollection(db, collID, []int64{idA, idB, idU}); err != nil {
		t.Fatal(err)
	}

	secs, err := ListCollectionCameraSections(db, collID)
	if err != nil {
		t.Fatal(err)
	}
	if len(secs) != 3 {
		t.Fatalf("got %+v", secs)
	}
	if secs[0].Label == nil || *secs[0].Label != "Alpha A1" {
		t.Fatalf("first: %+v", secs[0])
	}
	if secs[1].Label == nil || *secs[1].Label != "Zebra Z1" {
		t.Fatalf("second: %+v", secs[1])
	}
	if secs[2].Label != nil {
		t.Fatalf("unknown last: %+v", secs[2])
	}
}

// TestCollectionDetail_perSectionPagingDoesNotMixBuckets guards AC10: per-section OFFSET must not
// pull rows from a different star bucket (global flat OFFSET would).
func TestCollectionDetail_perSectionPagingDoesNotMixBuckets(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	collID, err := CreateCollection(db, "Paging", "")
	if err != nil {
		t.Fatal(err)
	}
	r5 := 5
	r4 := 4
	var fiveStarIDs []int64
	for i := 0; i < 7; i++ {
		id := insertBareAsset(t, db, hashOf(i), filepath.Join("a", string(rune('a'+i))+".jpg"), int64(100+i))
		if _, err := db.Exec(`UPDATE assets SET rating = ? WHERE id = ?`, r5, id); err != nil {
			t.Fatal(err)
		}
		fiveStarIDs = append(fiveStarIDs, id)
	}
	id4 := insertBareAsset(t, db, "h4star", "a/z.jpg", 999)
	if _, err := db.Exec(`UPDATE assets SET rating = ? WHERE id = ?`, r4, id4); err != nil {
		t.Fatal(err)
	}
	if err := LinkAssetsToCollection(db, collID, append(fiveStarIDs, id4)); err != nil {
		t.Fatal(err)
	}

	const page = 5
	var p0, p1 []ReviewGridRow
	for _, off := range []int{0, page} {
		rows, err := ListCollectionStarSectionPage(db, collID, &r5, page, off)
		if err != nil {
			t.Fatal(err)
		}
		if off == 0 {
			p0 = rows
		} else {
			p1 = rows
		}
	}
	all, err := ListCollectionStarSectionPage(db, collID, &r5, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(p0)+len(p1) != len(all) {
		t.Fatalf("page sizes %d+%d vs all %d", len(p0), len(p1), len(all))
	}
	for _, r := range append(append([]ReviewGridRow{}, p0...), p1...) {
		if r.Rating == nil || *r.Rating != 5 {
			t.Fatalf("non-5★ leaked into 5★ pages: %+v", r)
		}
	}
}

// TestCollectionDetail_daySectionPagingDoesNotMixDays mirrors AC10 for calendar-day buckets.
func TestCollectionDetail_daySectionPagingDoesNotMixDays(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	collID, err := CreateCollection(db, "DayPaging", "")
	if err != nil {
		t.Fatal(err)
	}
	loc := time.Local
	dayA := time.Date(2024, 5, 1, 8, 0, 0, 0, loc)
	dayB := time.Date(2024, 5, 2, 8, 0, 0, 0, loc)
	keyA := dayA.Format("2006-01-02")
	_ = dayB // second day ensures query filters by keyA only (regression guard)

	var dayAIDs []int64
	for i := 0; i < 6; i++ {
		id := insertBareAsset(t, db, hashOf(i+10), filepath.Join("d", string(rune('a'+i))+".jpg"), dayA.Add(time.Duration(i)*time.Minute).Unix())
		dayAIDs = append(dayAIDs, id)
	}
	idB := insertBareAsset(t, db, "dayB1", "d/z.jpg", dayB.Unix())
	if err := LinkAssetsToCollection(db, collID, append(dayAIDs, idB)); err != nil {
		t.Fatal(err)
	}

	const page = 5
	p0, err := ListCollectionDaySectionPage(db, collID, keyA, page, 0)
	if err != nil {
		t.Fatal(err)
	}
	p1, err := ListCollectionDaySectionPage(db, collID, keyA, page, page)
	if err != nil {
		t.Fatal(err)
	}
	allA, err := ListCollectionDaySectionPage(db, collID, keyA, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(p0)+len(p1) != len(allA) {
		t.Fatalf("page sizes %d+%d vs all %d", len(p0), len(p1), len(allA))
	}
	for _, r := range append(append([]ReviewGridRow{}, p0...), p1...) {
		got := time.Unix(r.CaptureTimeUnix, 0).In(loc).Format("2006-01-02")
		if got != keyA {
			t.Fatalf("day B (or other) leaked into day A pages: id=%d cap=%d got %q", r.ID, r.CaptureTimeUnix, got)
		}
	}
}

// TestCollectionDetail_cameraSectionPagingDoesNotMixLabels mirrors AC10 for camera_label buckets.
func TestCollectionDetail_cameraSectionPagingDoesNotMixLabels(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	collID, err := CreateCollection(db, "CamPaging", "")
	if err != nil {
		t.Fatal(err)
	}
	labelA := "Brand Alpha"
	labelB := "Brand Beta"
	var alphaIDs []int64
	for i := 0; i < 7; i++ {
		id := insertAssetCamera(t, db, hashOf(i+40), filepath.Join("cam", string(rune('a'+i))+".jpg"), int64(300+i), "Brand", "Alpha")
		alphaIDs = append(alphaIDs, id)
	}
	idBeta := insertAssetCamera(t, db, "hBeta", "cam/z.jpg", 999, "Brand", "Beta")
	if err := LinkAssetsToCollection(db, collID, append(alphaIDs, idBeta)); err != nil {
		t.Fatal(err)
	}

	const page = 5
	p0, err := ListCollectionCameraSectionPage(db, collID, &labelA, page, 0)
	if err != nil {
		t.Fatal(err)
	}
	p1, err := ListCollectionCameraSectionPage(db, collID, &labelA, page, page)
	if err != nil {
		t.Fatal(err)
	}
	allA, err := ListCollectionCameraSectionPage(db, collID, &labelA, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(p0)+len(p1) != len(allA) {
		t.Fatalf("page sizes %d+%d vs all %d", len(p0), len(p1), len(allA))
	}
	betaLeak := false
	for _, r := range append(append([]ReviewGridRow{}, p0...), p1...) {
		if r.ID == idBeta {
			betaLeak = true
			break
		}
	}
	if betaLeak {
		t.Fatal("camera B row leaked into camera A pages")
	}
	rowsB, err := ListCollectionCameraSectionPage(db, collID, &labelB, 50, 0)
	if err != nil || len(rowsB) != 1 || rowsB[0].ID != idBeta {
		t.Fatalf("beta bucket: %+v err %v", rowsB, err)
	}
}

func insertBareAsset(t *testing.T, db *sql.DB, hash, rel string, capUnix int64) int64 {
	t.Helper()
	res, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, deleted_at_unix)
VALUES (?, ?, ?, 1, 0, NULL)`, hash, rel, capUnix)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	return id
}

func insertAssetCamera(t *testing.T, db *sql.DB, hash, rel string, capUnix int64, makeStr, modelStr string) int64 {
	t.Helper()
	lbl := CameraLabelForStorage(makeStr, modelStr)
	res, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, deleted_at_unix, camera_make, camera_model, camera_label)
VALUES (?, ?, ?, 1, 0, NULL, ?, ?, ?)`,
		hash, rel, capUnix, strOrNil(makeStr), strOrNil(modelStr), ScanStringPtr(lbl))
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	return id
}

func strOrNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func hashOf(i int) string {
	return string(rune('A'+i)) + "hash"
}

func intPtr(v int) *int { return &v }

func strPtr(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	default:
		return ""
	}
}
