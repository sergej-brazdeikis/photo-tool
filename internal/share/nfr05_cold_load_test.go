package share

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"photo-tool/internal/config"
	"photo-tool/internal/store"
)

// TestNFR05_ShareColdLoadMedian gates PRD NFR-05 for the share path: cold load of the HTML
// document and (recommended) first full image fetch for the same token, median over N trials,
// excluding user network (localhost httptest). Skipped under -short (CI still runs full tests
// without -short per .github/workflows/go.yml).
//
// Methodology: see docs/share-cold-load-nfr05.md.
func TestNFR05_ShareColdLoadMedian(t *testing.T) {
	if testing.Short() {
		t.Skip("NFR-05 median gate skipped in -short; run full go test for cold-load timing")
	}

	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	rel := "2024/nfr05-cold.jpg"
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, minimalJPEG(t), 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Now().Unix()
	if err := store.InsertAsset(db, "nfr05-cold", rel, now, now); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = ?`, "nfr05-cold").Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE assets SET mime = ? WHERE id = ?`, "image/jpeg", id); err != nil {
		t.Fatal(err)
	}
	raw, _, err := store.MintDefaultShareLink(context.Background(), db, id, now+1)
	if err != nil {
		t.Fatal(err)
	}

	const trials = 9
	htmlDurs := make([]time.Duration, trials)
	imgDurs := make([]time.Duration, trials)

	// Each request must open a new connection: httptest.Server.Client() returns distinct
	// *http.Client values but they share one cached Transport on the server, so keep-alive
	// can still reuse the TCP connection from GET /s/ to GET /i/. DisableKeepAlives
	// matches "cold" per-leg semantics for NFR-05 (see docs/share-cold-load-nfr05.md).
	coldClient := &http.Client{
		Transport: &http.Transport{DisableKeepAlives: true},
	}

	for i := 0; i < trials; i++ {
		srv := httptest.NewServer(NewHTTPHandler(db, root))
		base := srv.URL

		t0 := time.Now()
		resp, err := coldClient.Get(base + ShareHTTPPath(raw))
		if err != nil {
			srv.Close()
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != 200 {
			srv.Close()
			t.Fatalf("trial %d HTML status %d", i, resp.StatusCode)
		}
		htmlDurs[i] = time.Since(t0)

		t1 := time.Now()
		respI, err := coldClient.Get(base + ShareImageHTTPPath(raw))
		if err != nil {
			srv.Close()
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, respI.Body)
		_ = respI.Body.Close()
		if respI.StatusCode != 200 {
			srv.Close()
			t.Fatalf("trial %d image status %d", i, respI.StatusCode)
		}
		imgDurs[i] = time.Since(t1)

		srv.Close()
	}

	sort.Slice(htmlDurs, func(a, b int) bool { return htmlDurs[a] < htmlDurs[b] })
	sort.Slice(imgDurs, func(a, b int) bool { return imgDurs[a] < imgDurs[b] })

	medHTML := htmlDurs[len(htmlDurs)/2]
	medImg := imgDurs[len(imgDurs)/2]
	const budget = 3 * time.Second
	if medHTML > budget {
		t.Fatalf("NFR-05 cold HTML median %s exceeds %s (sorted %v)", medHTML, budget, htmlDurs)
	}
	if medImg > budget {
		t.Fatalf("NFR-05 cold image median %s exceeds %s (sorted %v)", medImg, budget, imgDurs)
	}
}
