package share

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"photo-tool/internal/config"
	"photo-tool/internal/domain"
	"photo-tool/internal/fixture"
	"photo-tool/internal/store"
)

func TestShareHTTP_scalePackage50Members(t *testing.T) {
	if testing.Short() {
		t.Skip("scale package HTTP test skipped in -short")
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
		t.Fatalf("need ≥50 assets, got %d", len(rows))
	}
	var ids []int64
	for _, r := range rows[:50] {
		ids = append(ids, r.ID)
	}
	now := time.Now().Unix()
	raw, _, err := store.MintPackageShareLink(context.Background(), db, ids, now+1, store.ShareSnapshotPayload{DisplayTitle: "Scale50"})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(NewHTTPHandler(db, root))
	t.Cleanup(srv.Close)
	resp, err := srv.Client().Get(srv.URL + ShareHTTPPath(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "package-grid") || !strings.Contains(string(body), "Scale50") {
		t.Fatalf("expected package page body")
	}
}

func TestShareHTTP_scalePackage500Members(t *testing.T) {
	if testing.Short() {
		t.Skip("scale package HTTP test skipped in -short")
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
		t.Fatalf("need ≥500 assets, got %d", len(rows))
	}
	var ids []int64
	for _, r := range rows[:500] {
		ids = append(ids, r.ID)
	}
	now := time.Now().Unix()
	raw, _, err := store.MintPackageShareLink(context.Background(), db, ids, now+1, store.ShareSnapshotPayload{DisplayTitle: "Scale500"})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(NewHTTPHandler(db, root))
	t.Cleanup(srv.Close)
	resp, err := srv.Client().Get(srv.URL + ShareHTTPPath(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "Scale500") {
		t.Fatalf("expected package title in HTML")
	}
}

func TestShareHTTP_scalePackageRejectExcluded(t *testing.T) {
	if testing.Short() {
		t.Skip("scale package reject exclusion skipped in -short")
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
	rows, err := store.ListAssetsForReview(db, domain.ReviewFilters{}, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 2 {
		t.Fatalf("need ≥2 active assets, got %d", len(rows))
	}
	rejRows, err := store.ListRejectedForReview(db, domain.ReviewFilters{}, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	var ids []int64
	for _, r := range rows[:2] {
		ids = append(ids, r.ID)
	}
	if len(rejRows) > 0 {
		ids = append(ids, rejRows[0].ID)
	} else {
		now := time.Now().Unix()
		if _, err := store.RejectAsset(db, rows[0].ID, now); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, rows[0].ID)
	}
	eligible, err := store.PackagePrepareEligibleForMint(context.Background(), db, ids)
	if err != nil {
		t.Fatal(err)
	}
	if len(eligible) >= len(ids) {
		t.Fatalf("expected rejected ids dropped from package manifest, got %d eligible of %d candidates", len(eligible), len(ids))
	}
}

// TestNFR05_Package500ColdLoadHTML gates NFR-05 stretch: cold HTML load for a 500-member package page.
func TestNFR05_Package500ColdLoadHTML(t *testing.T) {
	if testing.Short() {
		t.Skip("NFR-05 package 500 cold load skipped in -short")
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
		t.Fatalf("need ≥500 assets, got %d", len(rows))
	}
	var ids []int64
	for _, r := range rows[:500] {
		ids = append(ids, r.ID)
	}
	now := time.Now().Unix()
	raw, _, err := store.MintPackageShareLink(context.Background(), db, ids, now+1, store.ShareSnapshotPayload{DisplayTitle: "Scale500Cold"})
	if err != nil {
		t.Fatal(err)
	}

	const trials = 5
	durs := make([]time.Duration, trials)
	coldClient := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}
	for i := 0; i < trials; i++ {
		srv := httptest.NewServer(NewHTTPHandler(db, root))
		t0 := time.Now()
		resp, err := coldClient.Get(srv.URL + ShareHTTPPath(raw))
		if err != nil {
			srv.Close()
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		srv.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("trial %d status %d", i, resp.StatusCode)
		}
		durs[i] = time.Since(t0)
	}
	sort.Slice(durs, func(a, b int) bool { return durs[a] < durs[b] })
	med := durs[len(durs)/2]
	const budget = 3 * time.Second
	if med > budget {
		t.Fatalf("NFR-05 package500 HTML median %s exceeds %s (sorted %v)", med, budget, durs)
	}
}

// TestShareHTTP_package500FocusOrderHTML checks WCAG focus-order landmarks on a 500-member package page (B8).
func TestShareHTTP_package500FocusOrderHTML(t *testing.T) {
	if testing.Short() {
		t.Skip("package500 focus order skipped in -short")
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
		t.Fatalf("need ≥500 assets, got %d", len(rows))
	}
	var ids []int64
	for _, r := range rows[:500] {
		ids = append(ids, r.ID)
	}
	now := time.Now().Unix()
	raw, _, err := store.MintPackageShareLink(context.Background(), db, ids, now+1, store.ShareSnapshotPayload{DisplayTitle: "Focus500"})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(NewHTTPHandler(db, root))
	t.Cleanup(srv.Close)
	resp, err := srv.Client().Get(srv.URL + ShareHTTPPath(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		t.Fatal(err)
	}
	html := string(body)
	for _, want := range []string{
		`<a class="skip-link" href="#share-main">`,
		`<main id="share-main" class="shell" tabindex="-1">`,
		".skip-link:focus:not(:focus-visible)",
		`<ul class="package-grid" role="list">`,
		`<li role="listitem">`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("package500 HTML missing %q", want)
		}
	}
	linkCount := strings.Count(html, `<a href="`)
	if linkCount < 500 {
		t.Fatalf("expected ≥500 thumbnail links, got %d", linkCount)
	}
}
