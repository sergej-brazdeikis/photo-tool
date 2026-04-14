package share

import (
	"context"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"photo-tool/internal/config"
	"photo-tool/internal/store"
)

func pickFreeLoopbackTCPPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func TestLoopback_EnsureRunning_idempotent(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	port := pickFreeLoopbackTCPPort(t)
	lb := NewLoopback(db, root, config.ShareHTTPConfig{Host: "127.0.0.1", Port: port})
	ctx := context.Background()
	a, err := lb.EnsureRunning(ctx)
	if err != nil {
		t.Fatal(err)
	}
	b, err := lb.EnsureRunning(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("base URL drift: %q vs %q", a, b)
	}
	t.Cleanup(func() { _ = lb.Close() })
}

func TestLoopback_EnsureRunning_concurrentSecondCallWaitsForListen(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	port := pickFreeLoopbackTCPPort(t)
	lb := NewLoopback(db, root, config.ShareHTTPConfig{Host: "127.0.0.1", Port: port})
	ctx := context.Background()

	var bases [3]string
	var errs [3]error
	var wg sync.WaitGroup
	for i := range bases {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			bases[i], errs[i] = lb.EnsureRunning(ctx)
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: %v", i, err)
		}
	}
	if bases[0] != bases[1] || bases[1] != bases[2] {
		t.Fatalf("bases differ: %q %q %q", bases[0], bases[1], bases[2])
	}
	t.Cleanup(func() { _ = lb.Close() })
}

func TestLoopback_CloseThenEnsureRunning_canServeAgain(t *testing.T) {
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
	if err := store.InsertAsset(db, "loopback-restart", "2024/lb.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = ?`, "loopback-restart").Scan(&id); err != nil {
		t.Fatal(err)
	}
	raw, _, err := store.MintDefaultShareLink(context.Background(), db, id, now+1)
	if err != nil {
		t.Fatal(err)
	}

	port := pickFreeLoopbackTCPPort(t)
	lb := NewLoopback(db, root, config.ShareHTTPConfig{Host: "127.0.0.1", Port: port})
	ctx := context.Background()

	getStatus := func(base string) int {
		t.Helper()
		c := &http.Client{Timeout: 3 * time.Second}
		resp, err := c.Get(base + ShareHTTPPath(raw))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)
		return resp.StatusCode
	}

	base1, err := lb.EnsureRunning(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(base1, "127.0.0.1:") {
		t.Fatalf("base URL: %q", base1)
	}
	if got := getStatus(base1); got != http.StatusOK {
		t.Fatalf("first run status: %d", got)
	}
	if err := lb.Close(); err != nil {
		t.Fatal(err)
	}
	base2, err := lb.EnsureRunning(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if base1 != base2 {
		t.Fatalf("base changed after restart: %q vs %q", base1, base2)
	}
	if got := getStatus(base2); got != http.StatusOK {
		t.Fatalf("second run status: %d", got)
	}
	t.Cleanup(func() { _ = lb.Close() })
}
