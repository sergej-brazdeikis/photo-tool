package share

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"photo-tool/internal/config"
	"photo-tool/internal/store"
)

func TestShareHTTP_rateLimit_exhausted_429NoTokenOracle(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().Unix()
	if err := store.InsertAsset(db, "rl-oracle", "2024/rl.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = ?`, "rl-oracle").Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE assets SET mime = ? WHERE id = ?`, "image/jpeg", id); err != nil {
		t.Fatal(err)
	}
	validRaw, _, err := store.MintDefaultShareLink(context.Background(), db, id, now+1)
	if err != nil {
		t.Fatal(err)
	}

	// ~1 token per day, burst 2 → third rapid request is rate limited.
	tight := rateLimitConfig{r: rate.Every(24 * time.Hour), burst: 2}
	srv := httptest.NewServer(wrapRateLimitedHandler(newShareMuxHandler(db, root), tight))
	t.Cleanup(srv.Close)
	client := srv.Client()

	for i := 0; i < 2; i++ {
		resp, err := client.Get(srv.URL + ShareHTTPPath(validRaw))
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("request %d: status %d", i, resp.StatusCode)
		}
	}

	resp429Valid, err := client.Get(srv.URL + ShareHTTPPath(validRaw))
	if err != nil {
		t.Fatal(err)
	}
	defer resp429Valid.Body.Close()
	if resp429Valid.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("valid token want429, got %d", resp429Valid.StatusCode)
	}
	if resp429Valid.Header.Get("Referrer-Policy") != "" || resp429Valid.Header.Get("Content-Security-Policy") != "" {
		t.Fatalf("429 must not add success-only headers")
	}
	bValid, err := io.ReadAll(resp429Valid.Body)
	if err != nil {
		t.Fatal(err)
	}

	resp429Unknown, err := client.Get(srv.URL + ShareHTTPPath("bogus-token-not-in-db-xxxxxxxxxxxxxxxxxxxxxxxxxx"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp429Unknown.Body.Close()
	if resp429Unknown.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("unknown token want 429, got %d", resp429Unknown.StatusCode)
	}

	bUnknown, err := io.ReadAll(resp429Unknown.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(bValid) != string(bUnknown) {
		t.Fatalf("429 body must match for valid vs unknown: %q vs %q", bValid, bUnknown)
	}
}

func TestIPRateLimiter_visitorsCapEviction(t *testing.T) {
	lim := newIPRateLimiter()
	cfg := rateLimitConfig{r: rate.Every(time.Second), burst: 10}
	for i := 0; i < maxRateLimitVisitorEntries; i++ {
		if !lim.allow(fmt.Sprintf("h%d", i), cfg) {
			t.Fatalf("allow h%d: unexpected false before cap", i)
		}
	}
	if got := len(lim.visitors); got != maxRateLimitVisitorEntries {
		t.Fatalf("visitor count: got %d want %d", got, maxRateLimitVisitorEntries)
	}
	if !lim.allow("h-new", cfg) {
		t.Fatal("allow after cap insert: want true")
	}
	if got := len(lim.visitors); got != maxRateLimitVisitorEntries {
		t.Fatalf("after eviction+insert: got %d want %d", got, maxRateLimitVisitorEntries)
	}
}

func TestShareHTTP_rateLimit_HEAD429(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	tight := rateLimitConfig{r: rate.Every(24 * time.Hour), burst: 1}
	srv := httptest.NewServer(wrapRateLimitedHandler(newShareMuxHandler(db, root), tight))
	t.Cleanup(srv.Close)
	client := srv.Client()
	path := ShareHTTPPath("bogus-token-not-in-db-xxxxxxxxxxxxxxxxxxxxxxxxxx")

	req1, err := http.NewRequest(http.MethodHead, srv.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp1, err := client.Do(req1)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp1.Body.Close()
	if resp1.StatusCode != http.StatusNotFound {
		t.Fatalf("first HEAD want 404, got %d", resp1.StatusCode)
	}

	req2, err := http.NewRequest(http.MethodHead, srv.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("second HEAD want 429, got %d", resp2.StatusCode)
	}
}
