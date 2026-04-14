package share

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"photo-tool/internal/config"
	"photo-tool/internal/store"
)

// minimalJPEG is a tiny valid JPEG (1×1) for image route tests.
func minimalJPEG(t *testing.T) []byte {
	t.Helper()
	const b64 = "/9j/4AAQSkZJRgABAQEASABIAAD/2wBDAP//////////////////////////////////////////////////////////////////////////////////////2wBDAf//////////////////////////////////////////////////////////////////////////////////////wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIQAxAAAAF//8QAFBABAAAAAAAAAAAAAAAAAAAAAP/aAAwDAQACAAMAAAAA/wD/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAQIRAxEAAAAB/9k="
	b, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestShareHTTP_resolve200_HTML(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	rel := "2024/http-share-1.jpg"
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, minimalJPEG(t), 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Now().Unix()
	if err := store.InsertAsset(db, "http-share-1", rel, now, now); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = ?`, "http-share-1").Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE assets SET mime = ? WHERE id = ?`, "image/jpeg", id); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateAssetRating(db, id, 4); err != nil {
		t.Fatal(err)
	}
	raw, _, err := store.MintDefaultShareLink(context.Background(), db, id, now+1)
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(NewHTTPHandler(db, root))
	t.Cleanup(srv.Close)

	resp, err := srv.Client().Get(srv.URL + ShareHTTPPath(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type: %q", ct)
	}
	if rp := resp.Header.Get("Referrer-Policy"); rp != "no-referrer" {
		t.Fatalf("Referrer-Policy: %q want no-referrer", rp)
	}
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("X-Content-Type-Options: %q want nosniff", resp.Header.Get("X-Content-Type-Options"))
	}
	if g, w := resp.Header.Get("Content-Security-Policy"), ShareHTMLContentSecurityPolicy; g != w {
		t.Fatalf("Content-Security-Policy: %q want %q", g, w)
	}
	b, _ := io.ReadAll(resp.Body)
	body := string(b)
	if !strings.Contains(body, "<!DOCTYPE html>") && !strings.Contains(strings.ToLower(body), "<!doctype html>") {
		t.Fatalf("expected HTML document: %q", truncate(body, 200))
	}
	if !strings.Contains(body, `object-fit: contain`) {
		t.Fatalf("expected layout CSS: %q", truncate(body, 400))
	}
	if !strings.Contains(body, "prefers-reduced-motion") {
		t.Fatalf("expected reduced-motion CSS in inlined stylesheet: %q", truncate(body, 500))
	}
	if !strings.Contains(body, ShareImageHTTPPath(raw)) {
		t.Fatalf("expected image path in HTML: %q", body)
	}
	if !strings.Contains(body, "Rating: 4") {
		t.Fatalf("expected rating label: %q", body)
	}
	if !strings.Contains(body, `id="share-rating-summary"`) {
		t.Fatalf("expected rating summary id for a11y grouping: %q", truncate(body, 400))
	}
	if !strings.Contains(body, `class="skip-link"`) || !strings.Contains(body, `href="#share-main"`) {
		t.Fatalf("expected skip link to main: %q", truncate(body, 400))
	}
	if !strings.Contains(body, ".skip-link:focus:not(:focus-visible)") {
		t.Fatalf("expected skip-link focus-visible pairing in inlined CSS: %q", truncate(body, 500))
	}
	if !strings.Contains(body, `<main id="share-main" class="shell" tabindex="-1">`) {
		t.Fatalf("expected main landmark id+shell+tabindex for skip target: %q", truncate(body, 400))
	}
	if !strings.Contains(body, `alt="Shared photo"`) {
		t.Fatalf("expected neutral alt text: %q", truncate(body, 400))
	}
}

func TestShareHTTP_imageMissingFile_404MatchesUnknown(t *testing.T) {
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
	if err := store.InsertAsset(db, "missing-file-img", "2024/missing.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = ?`, "missing-file-img").Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE assets SET mime = ? WHERE id = ?`, "image/jpeg", id); err != nil {
		t.Fatal(err)
	}
	raw, _, err := store.MintDefaultShareLink(context.Background(), db, id, now+1)
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(NewHTTPHandler(db, root))
	t.Cleanup(srv.Close)
	client := srv.Client()

	respUnk, err := client.Get(srv.URL + ShareImageHTTPPath("bogus-token-not-in-db-xxxxxxxxxxxxxxxxxxxxxxxxxx"))
	if err != nil {
		t.Fatal(err)
	}
	defer respUnk.Body.Close()
	unkBody, _ := io.ReadAll(respUnk.Body)

	respMiss, err := client.Get(srv.URL + ShareImageHTTPPath(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer respMiss.Body.Close()
	if respMiss.StatusCode != http.StatusNotFound {
		t.Fatalf("missing file status: %d", respMiss.StatusCode)
	}
	missBody, _ := io.ReadAll(respMiss.Body)
	if string(missBody) != string(unkBody) {
		t.Fatalf("body mismatch missing file vs unknown: %q vs %q", missBody, unkBody)
	}
}

func TestShareHTTP_image200(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	rel := "2024/img-route.jpg"
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	jpeg := minimalJPEG(t)
	if err := os.WriteFile(full, jpeg, 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Now().Unix()
	if err := store.InsertAsset(db, "img-route", rel, now, now); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = ?`, "img-route").Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE assets SET mime = ? WHERE id = ?`, "image/jpeg", id); err != nil {
		t.Fatal(err)
	}
	raw, _, err := store.MintDefaultShareLink(context.Background(), db, id, now+1)
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(NewHTTPHandler(db, root))
	t.Cleanup(srv.Close)

	resp, err := srv.Client().Get(srv.URL + ShareImageHTTPPath(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "image/jpeg") {
		t.Fatalf("Content-Type: %q", ct)
	}
	if resp.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control: %q", resp.Header.Get("Cache-Control"))
	}
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("nosniff: %q", resp.Header.Get("X-Content-Type-Options"))
	}
	if resp.Header.Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("Referrer-Policy: %q want no-referrer", resp.Header.Get("Referrer-Policy"))
	}
	b, _ := io.ReadAll(resp.Body)
	if string(b) != string(jpeg) {
		t.Fatalf("body len %d want %d", len(b), len(jpeg))
	}
}

func TestShareHTTP_HTML_doesNotLeakIdentifiers(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	rel := "2024/leak-test.jpg"
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, minimalJPEG(t), 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Now().Unix()
	if err := store.InsertAsset(db, "leak-ident-content-hash", rel, now, now); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = ?`, "leak-ident-content-hash").Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE assets SET mime = ? WHERE id = ?`, "image/jpeg", id); err != nil {
		t.Fatal(err)
	}
	raw, _, err := store.MintDefaultShareLink(context.Background(), db, id, now+1)
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(NewHTTPHandler(db, root))
	t.Cleanup(srv.Close)

	resp, err := srv.Client().Get(srv.URL + ShareHTTPPath(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	body := string(b)

	leaks := []string{
		"leak-ident-content-hash",
		strings.ReplaceAll(rel, "/", string(filepath.Separator)),
		rel,
		"rel_path",
		"content_hash",
		"asset_id",
	}
	for _, sub := range leaks {
		if sub != "" && strings.Contains(body, sub) {
			t.Fatalf("HTML must not contain %q", sub)
		}
	}
}

func TestShareHTTP_HTML_extraPayloadJSON_doesNotLeakGeo(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	rel := "2024/geo-payload.jpg"
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, minimalJPEG(t), 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Now().Unix()
	if err := store.InsertAsset(db, "geo-payload-hash", rel, now, now); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = ?`, "geo-payload-hash").Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE assets SET mime = ? WHERE id = ?`, "image/jpeg", id); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateAssetRating(db, id, 3); err != nil {
		t.Fatal(err)
	}
	raw, _, err := store.MintDefaultShareLink(context.Background(), db, id, now+1)
	if err != nil {
		t.Fatal(err)
	}

	// Malicious / forward-looking payload keys must never reach HTML (Story 3.4 AC1).
	bad := `{"rating":3,"latitude":55.7558,"longitude":37.6173,"gps":"55.7558,37.6173","location":"Moscow"}`
	sum := sha256.Sum256([]byte(raw))
	tokenHash := hex.EncodeToString(sum[:])
	if _, err := db.Exec(`UPDATE share_links SET payload = ? WHERE token_hash = ?`, bad, tokenHash); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(NewHTTPHandler(db, root))
	t.Cleanup(srv.Close)

	resp, err := srv.Client().Get(srv.URL + ShareHTTPPath(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	b, _ := io.ReadAll(resp.Body)
	body := string(b)
	for _, sub := range []string{"55.7558", "37.6173", "latitude", "longitude", "Moscow"} {
		if strings.Contains(body, sub) {
			t.Fatalf("HTML must not leak geo payload field %q", sub)
		}
	}
	if !strings.Contains(body, "Rating: 3") {
		t.Fatalf("expected rating from sanitized parse: %q", truncate(body, 300))
	}
}

// TestShareHTTP_404_noReferrerPolicyHeader locks Story 3.4: Referrer-Policy and HTML CSP are only for successful GET/HEAD.
func TestShareHTTP_404_noReferrerPolicyHeader(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	srv := httptest.NewServer(NewHTTPHandler(db, root))
	t.Cleanup(srv.Close)

	badTok := "bogus-token-not-in-db-xxxxxxxxxxxxxxxxxxxxxxxxxx"
	for _, tc := range []struct {
		name string
		path string
	}{
		{"html", ShareHTTPPath(badTok)},
		{"image", ShareImageHTTPPath(badTok)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, method := range []string{http.MethodGet, http.MethodHead} {
				t.Run(method, func(t *testing.T) {
					req, err := http.NewRequest(method, srv.URL+tc.path, nil)
					if err != nil {
						t.Fatal(err)
					}
					resp, err := srv.Client().Do(req)
					if err != nil {
						t.Fatal(err)
					}
					defer resp.Body.Close()
					if resp.StatusCode != http.StatusNotFound {
						t.Fatalf("status: %d", resp.StatusCode)
					}
					if g := resp.Header.Get("Referrer-Policy"); g != "" {
						t.Fatalf("Referrer-Policy on 404 must be absent (got %q)", g)
					}
					if g := resp.Header.Get("Content-Security-Policy"); g != "" {
						t.Fatalf("Content-Security-Policy on 404 must be absent (got %q)", g)
					}
					if g := resp.Header.Get("X-Content-Type-Options"); g != "" {
						t.Fatalf("X-Content-Type-Options on 404 must be absent (got %q)", g)
					}
				})
			}
		})
	}
}

func TestShareHTTP_404Parity(t *testing.T) {
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
	if err := store.InsertAsset(db, "404-parity", "2024/p.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = ?`, "404-parity").Scan(&id); err != nil {
		t.Fatal(err)
	}
	raw, _, err := store.MintDefaultShareLink(context.Background(), db, id, now+1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.RejectAsset(db, id, now+5); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(NewHTTPHandler(db, root))
	t.Cleanup(srv.Close)
	client := srv.Client()

	wantBody := string(NotFoundBody)
	cases := []struct {
		name string
		do   func() (*http.Response, error)
	}{
		{"unknown_token_s", func() (*http.Response, error) {
			return client.Get(srv.URL + ShareHTTPPath("bogus-token-not-in-db-xxxxxxxxxxxxxxxxxxxxxxxxxx"))
		}},
		{"unknown_token_i", func() (*http.Response, error) {
			return client.Get(srv.URL + ShareImageHTTPPath("bogus-token-not-in-db-xxxxxxxxxxxxxxxxxxxxxxxxxx"))
		}},
		{"ineligible_asset_s", func() (*http.Response, error) {
			return client.Get(srv.URL + ShareHTTPPath(raw))
		}},
		{"ineligible_asset_i", func() (*http.Response, error) {
			return client.Get(srv.URL + ShareImageHTTPPath(raw))
		}},
		{"extra_path_segment_s", func() (*http.Response, error) {
			return client.Get(srv.URL + "/s/" + raw + "/extra")
		}},
		{"extra_path_segment_i", func() (*http.Response, error) {
			return client.Get(srv.URL + "/i/" + raw + "/extra")
		}},
		{"post_same_path_s", func() (*http.Response, error) {
			return client.Post(srv.URL+ShareHTTPPath(raw), "text/plain", strings.NewReader("x"))
		}},
		{"post_same_path_i", func() (*http.Response, error) {
			return client.Post(srv.URL+ShareImageHTTPPath(raw), "text/plain", strings.NewReader("x"))
		}},
		{"options_same_path_s", func() (*http.Response, error) {
			req, err := http.NewRequest(http.MethodOptions, srv.URL+ShareHTTPPath(raw), nil)
			if err != nil {
				return nil, err
			}
			return client.Do(req)
		}},
		{"options_same_path_i", func() (*http.Response, error) {
			req, err := http.NewRequest(http.MethodOptions, srv.URL+ShareImageHTTPPath(raw), nil)
			if err != nil {
				return nil, err
			}
			return client.Do(req)
		}},
		{"trace_same_path_s", func() (*http.Response, error) {
			req, err := http.NewRequest(http.MethodTrace, srv.URL+ShareHTTPPath(raw), nil)
			if err != nil {
				return nil, err
			}
			return client.Do(req)
		}},
		{"trace_same_path_i", func() (*http.Response, error) {
			req, err := http.NewRequest(http.MethodTrace, srv.URL+ShareImageHTTPPath(raw), nil)
			if err != nil {
				return nil, err
			}
			return client.Do(req)
		}},
	}
	var first []byte
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resp, err := tc.do()
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("status: %d", resp.StatusCode)
			}
			if resp.Header.Get("Allow") != "" {
				t.Fatalf("Allow header leaks method: %q", resp.Header.Get("Allow"))
			}
			if g, w := resp.Header.Get("Content-Length"), strconv.Itoa(len(NotFoundBody)); g != w {
				t.Fatalf("Content-Length: %q want %q", g, w)
			}
			b, _ := io.ReadAll(resp.Body)
			if string(b) != wantBody {
				t.Fatalf("body %q want %q", b, wantBody)
			}
			if first == nil {
				first = append([]byte(nil), b...)
			} else if string(b) != string(first) {
				t.Fatalf("body differs from first case: %q vs %q", b, first)
			}
		})
	}
}

func TestShareHTTP_crossRoute_unknownTokenBytesMatch(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	srv := httptest.NewServer(NewHTTPHandler(db, root))
	t.Cleanup(srv.Close)
	client := srv.Client()
	tok := "bogus-token-not-in-db-xxxxxxxxxxxxxxxxxxxxxxxxxx"

	respS, err := client.Get(srv.URL + ShareHTTPPath(tok))
	if err != nil {
		t.Fatal(err)
	}
	defer respS.Body.Close()
	bodyS, _ := io.ReadAll(respS.Body)

	respI, err := client.Get(srv.URL + ShareImageHTTPPath(tok))
	if err != nil {
		t.Fatal(err)
	}
	defer respI.Body.Close()
	bodyI, _ := io.ReadAll(respI.Body)

	if string(bodyS) != string(bodyI) {
		t.Fatalf("s vs i body: %q vs %q", bodyS, bodyI)
	}
}

func TestShareHTTP_404_rangeDoesNotChangeBody(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	srv := httptest.NewServer(NewHTTPHandler(db, root))
	t.Cleanup(srv.Close)
	tok := "bogus-token-not-in-db-xxxxxxxxxxxxxxxxxxxxxxxxxx"

	for _, name := range []string{"html", "image"} {
		t.Run(name, func(t *testing.T) {
			path := ShareImageHTTPPath(tok)
			if name == "html" {
				path = ShareHTTPPath(tok)
			}
			req, err := http.NewRequest(http.MethodGet, srv.URL+path, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Range", "bytes=0-1")
			resp, err := srv.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("status: %d", resp.StatusCode)
			}
			b, _ := io.ReadAll(resp.Body)
			if string(b) != string(NotFoundBody) {
				t.Fatalf("body with Range: %q", b)
			}
		})
	}
}

func TestShareHTTP_404_HEAD_withRange(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	srv := httptest.NewServer(NewHTTPHandler(db, root))
	t.Cleanup(srv.Close)
	tok := "bogus-token-not-in-db-xxxxxxxxxxxxxxxxxxxxxxxxxx"

	for _, name := range []string{"html", "image"} {
		t.Run(name, func(t *testing.T) {
			path := ShareImageHTTPPath(tok)
			if name == "html" {
				path = ShareHTTPPath(tok)
			}
			req, err := http.NewRequest(http.MethodHead, srv.URL+path, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Range", "bytes=0-1")
			resp, err := srv.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("status: %d", resp.StatusCode)
			}
			b, _ := io.ReadAll(resp.Body)
			if len(b) != 0 {
				t.Fatalf("HEAD 404 body: %q", b)
			}
			if g, w := resp.Header.Get("Content-Length"), strconv.Itoa(len(NotFoundBody)); g != w {
				t.Fatalf("Content-Length: %q want %q", g, w)
			}
		})
	}
}

func TestShareHTTP_HEAD_success_and_404(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	rel := "2024/hd.jpg"
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, minimalJPEG(t), 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Now().Unix()
	if err := store.InsertAsset(db, "head-1", rel, now, now); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = ?`, "head-1").Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE assets SET mime = ? WHERE id = ?`, "image/jpeg", id); err != nil {
		t.Fatal(err)
	}
	raw, _, err := store.MintDefaultShareLink(context.Background(), db, id, now+1)
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(NewHTTPHandler(db, root))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodHead, srv.URL+ShareHTTPPath(raw), nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("HEAD ok status: %d", resp.StatusCode)
	}
	if rp := resp.Header.Get("Referrer-Policy"); rp != "no-referrer" {
		t.Fatalf("HEAD Referrer-Policy: %q want no-referrer", rp)
	}
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("HEAD X-Content-Type-Options: %q want nosniff", resp.Header.Get("X-Content-Type-Options"))
	}
	if g, w := resp.Header.Get("Content-Security-Policy"), ShareHTMLContentSecurityPolicy; g != w {
		t.Fatalf("HEAD Content-Security-Policy: %q want %q", g, w)
	}
	b, _ := io.ReadAll(resp.Body)
	if len(b) != 0 {
		t.Fatalf("HEAD body: %q", b)
	}
	cl := resp.Header.Get("Content-Length")
	if cl == "" || cl == "0" {
		t.Fatalf("HEAD Content-Length: %q", cl)
	}

	respGet, err := srv.Client().Get(srv.URL + ShareHTTPPath(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer respGet.Body.Close()
	if respGet.StatusCode != http.StatusOK {
		t.Fatalf("GET HTML status: %d", respGet.StatusCode)
	}
	getBody, _ := io.ReadAll(respGet.Body)
	if cl != strconv.Itoa(len(getBody)) {
		t.Fatalf("HEAD Content-Length %q != GET body len %d", cl, len(getBody))
	}
	if g, w := respGet.Header.Get("Content-Security-Policy"), ShareHTMLContentSecurityPolicy; g != w {
		t.Fatalf("GET Content-Security-Policy: %q want %q", g, w)
	}

	req404, _ := http.NewRequest(http.MethodHead, srv.URL+ShareHTTPPath("nope-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"), nil)
	resp404, err := srv.Client().Do(req404)
	if err != nil {
		t.Fatal(err)
	}
	defer resp404.Body.Close()
	if resp404.StatusCode != http.StatusNotFound {
		t.Fatalf("HEAD 404 status: %d", resp404.StatusCode)
	}
	b404, _ := io.ReadAll(resp404.Body)
	if len(b404) != 0 {
		t.Fatalf("HEAD 404 body: %q", b404)
	}
	if g, w := resp404.Header.Get("Content-Length"), strconv.Itoa(len(NotFoundBody)); g != w {
		t.Fatalf("Content-Length: %q want %q", g, w)
	}

	reqI, _ := http.NewRequest(http.MethodHead, srv.URL+ShareImageHTTPPath(raw), nil)
	respI, err := srv.Client().Do(reqI)
	if err != nil {
		t.Fatal(err)
	}
	defer respI.Body.Close()
	if respI.StatusCode != http.StatusOK {
		t.Fatalf("HEAD image status: %d", respI.StatusCode)
	}
	if rp := respI.Header.Get("Referrer-Policy"); rp != "no-referrer" {
		t.Fatalf("HEAD image Referrer-Policy: %q want no-referrer", rp)
	}
	if respI.Header.Get("Content-Length") == "" {
		t.Fatal("HEAD image Content-Length missing")
	}
}

func TestShareHTTP_softDeleteAfterMint_404MatchesUnknown(t *testing.T) {
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
	if err := store.InsertAsset(db, "softdel-http", "2024/sd.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = ?`, "softdel-http").Scan(&id); err != nil {
		t.Fatal(err)
	}
	raw, _, err := store.MintDefaultShareLink(context.Background(), db, id, now+1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE assets SET deleted_at_unix = ? WHERE id = ?`, now+9, id); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(NewHTTPHandler(db, root))
	t.Cleanup(srv.Close)
	client := srv.Client()

	respUnk, err := client.Get(srv.URL + ShareHTTPPath("bogus-token-not-in-db-xxxxxxxxxxxxxxxxxxxxxxxxxx"))
	if err != nil {
		t.Fatal(err)
	}
	defer respUnk.Body.Close()
	unkBody, _ := io.ReadAll(respUnk.Body)

	for _, path := range []string{ShareHTTPPath(raw), ShareImageHTTPPath(raw)} {
		respDel, err := client.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		func() {
			defer respDel.Body.Close()
			if respDel.StatusCode != http.StatusNotFound {
				t.Fatalf("soft-deleted token status %s: %d", path, respDel.StatusCode)
			}
			delBody, _ := io.ReadAll(respDel.Body)
			if string(delBody) != string(unkBody) {
				t.Fatalf("body mismatch soft-delete vs unknown %s: %q vs %q", path, delBody, unkBody)
			}
			if g, w := respDel.Header.Get("Content-Length"), strconv.Itoa(len(NotFoundBody)); g != w {
				t.Fatalf("Content-Length: %q want %q", g, w)
			}
		}()
	}
}

func TestShareHTTP_queryStringDoesNotAffectResolution(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	rel := "2024/q.jpg"
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, minimalJPEG(t), 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Now().Unix()
	if err := store.InsertAsset(db, "qs-share", rel, now, now); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = ?`, "qs-share").Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE assets SET mime = ? WHERE id = ?`, "image/jpeg", id); err != nil {
		t.Fatal(err)
	}
	raw, _, err := store.MintDefaultShareLink(context.Background(), db, id, now+1)
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(NewHTTPHandler(db, root))
	t.Cleanup(srv.Close)
	client := srv.Client()

	getPlain := func(url string) (int, []byte, error) {
		resp, err := client.Get(url)
		if err != nil {
			return 0, nil, err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, b, nil
	}

	st, body, err := getPlain(srv.URL + ShareHTTPPath(raw) + "?utm_source=test&x=1")
	if err != nil {
		t.Fatal(err)
	}
	if st != http.StatusOK {
		t.Fatalf("status with query: %d", st)
	}
	if !strings.Contains(string(body), "<!DOCTYPE") {
		t.Fatalf("body with query: %q", truncate(string(body), 120))
	}

	st2, body2, err := getPlain(srv.URL + ShareHTTPPath(raw))
	if err != nil {
		t.Fatal(err)
	}
	if st2 != st || string(body2) != string(body) {
		t.Fatalf("plain vs query mismatch: %d/%q vs %d/%q", st2, truncate(string(body2), 80), st, truncate(string(body), 80))
	}

	jpeg := minimalJPEG(t)
	sti, bodyi, err := getPlain(srv.URL + ShareImageHTTPPath(raw) + "?cb=1")
	if err != nil {
		t.Fatal(err)
	}
	if sti != http.StatusOK {
		t.Fatalf("image status with query: %d", sti)
	}
	if string(bodyi) != string(jpeg) {
		t.Fatalf("image body mismatch with query")
	}
	sti2, bodyi2, err := getPlain(srv.URL + ShareImageHTTPPath(raw))
	if err != nil {
		t.Fatal(err)
	}
	if sti2 != sti || string(bodyi2) != string(bodyi) {
		t.Fatalf("image plain vs query mismatch")
	}

	st404, b404, err := getPlain(srv.URL + ShareHTTPPath("bogus-token-not-in-db-xxxxxxxxxxxxxxxxxxxxxxxxxx") + "?foo=bar")
	if err != nil {
		t.Fatal(err)
	}
	if st404 != http.StatusNotFound {
		t.Fatalf("404 with query: %d", st404)
	}
	if string(b404) != string(NotFoundBody) {
		t.Fatalf("404 body with query: %q", b404)
	}
	st404i, b404i, err := getPlain(srv.URL + ShareImageHTTPPath("bogus-token-not-in-db-xxxxxxxxxxxxxxxxxxxxxxxxxx") + "?x=1")
	if err != nil {
		t.Fatal(err)
	}
	if st404i != http.StatusNotFound {
		t.Fatalf("404 i with query: %d", st404i)
	}
	if string(b404i) != string(b404) {
		t.Fatalf("404 i vs s with query: %q vs %q", b404i, b404)
	}
}

func TestShareHTTP_packageHTMLAndSnapshotImage404AfterReject(t *testing.T) {
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
	for _, tc := range []struct {
		hash, rel string
	}{
		{"pkg-http-a", "2024/pkg-a.jpg"},
		{"pkg-http-b", "2024/pkg-b.jpg"},
	} {
		full := filepath.Join(root, filepath.FromSlash(tc.rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, minimalJPEG(t), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := store.InsertAsset(db, tc.hash, tc.rel, now, now); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`UPDATE assets SET mime = ? WHERE content_hash = ?`, "image/jpeg", tc.hash); err != nil {
			t.Fatal(err)
		}
	}
	var idA, idB int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = 'pkg-http-a'`).Scan(&idA); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = 'pkg-http-b'`).Scan(&idB); err != nil {
		t.Fatal(err)
	}

	raw, _, err := store.MintPackageShareLink(context.Background(), db, []int64{idA, idB}, now+1, store.ShareSnapshotPayload{DisplayTitle: "Trip"})
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(NewHTTPHandler(db, root))
	t.Cleanup(srv.Close)
	client := srv.Client()

	resp, err := client.Get(srv.URL + ShareHTTPPath(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("package html: %d", resp.StatusCode)
	}
	b, _ := io.ReadAll(resp.Body)
	body := string(b)
	if !strings.Contains(body, "package-grid") || !strings.Contains(body, "Trip") {
		t.Fatalf("expected package page: %q", truncate(body, 400))
	}
	if !strings.Contains(body, SharePackageMemberImagePath(raw, 0)) {
		t.Fatalf("expected member 0 path in html: %q", body)
	}

	resp0, err := client.Get(srv.URL + SharePackageMemberImagePath(raw, 0))
	if err != nil {
		t.Fatal(err)
	}
	defer resp0.Body.Close()
	if resp0.StatusCode != http.StatusOK {
		t.Fatalf("member 0 image: %d", resp0.StatusCode)
	}

	if _, err := store.RejectAsset(db, idB, now+99); err != nil {
		t.Fatal(err)
	}

	respAfter, err := client.Get(srv.URL + ShareHTTPPath(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer respAfter.Body.Close()
	if respAfter.StatusCode != http.StatusOK {
		t.Fatalf("package html after reject: %d", respAfter.StatusCode)
	}
	bodyAfter, _ := io.ReadAll(respAfter.Body)
	// Snapshot index still lists both minted slots (AC5); bytes route404s for ineligible member.
	if !strings.Contains(string(bodyAfter), SharePackageMemberImagePath(raw, 1)) {
		t.Fatalf("expected post-reject package HTML to still reference member 1: %q", truncate(string(bodyAfter), 500))
	}
	if !strings.Contains(string(bodyAfter), fmt.Sprintf("id %d", idB)) {
		t.Fatalf("expected caption to retain rejected asset id in snapshot list: %q", truncate(string(bodyAfter), 500))
	}

	resp1, err := client.Get(srv.URL + SharePackageMemberImagePath(raw, 1))
	if err != nil {
		t.Fatal(err)
	}
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusNotFound {
		t.Fatalf("rejected member image: %d", resp1.StatusCode)
	}
}

func TestShareHTTP_invalidToken404Parity_packageVsSingle(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	srv := httptest.NewServer(NewHTTPHandler(db, root))
	t.Cleanup(srv.Close)
	client := srv.Client()

	bogus := "bogus-token-not-in-db-xxxxxxxxxxxxxxxxxxxxxxxxxx"
	urls := []string{
		srv.URL + ShareHTTPPath(bogus),
		srv.URL + SharePackageMemberImagePath(bogus, 0),
	}
	var firstHdr http.Header
	for i, u := range urls {
		resp, err := client.Get(u)
		if err != nil {
			t.Fatal(err)
		}
		func() {
			defer resp.Body.Close()
			b, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("GET %s: status %d", u, resp.StatusCode)
			}
			if string(b) != string(NotFoundBody) {
				t.Fatalf("GET body: %q", b)
			}
			if i == 0 {
				firstHdr = resp.Header.Clone()
			} else {
				for _, k := range []string{"Content-Type", "Cache-Control", "Content-Length"} {
					if g, w := firstHdr.Get(k), resp.Header.Get(k); g != w {
						t.Fatalf("header %q: %q vs %q (enumeration parity)", k, g, w)
					}
				}
			}
		}()
	}
	for _, u := range urls {
		req, err := http.NewRequest(http.MethodHead, u, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		func() {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("HEAD %s: status %d", u, resp.StatusCode)
			}
			if cl := resp.Header.Get("Content-Length"); cl != strconv.Itoa(len(NotFoundBody)) {
				t.Fatalf("HEAD Content-Length %s want %d", cl, len(NotFoundBody))
			}
			if resp.ContentLength >= 0 && resp.ContentLength != int64(len(NotFoundBody)) {
				t.Fatalf("HEAD reported length %d", resp.ContentLength)
			}
		}()
	}
}

// Story 4.1 AC5 / NFR-06: out-of-range package member index must not return a distinct404 shape
// that reveals token validity vs unknown-token failures.
func TestShareHTTP_packageMemberIndexOutOfRange404MatchesUnknown(t *testing.T) {
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
	for _, tc := range []struct {
		hash, rel string
	}{
		{"pkg-oor-a", "2024/oor-a.jpg"},
		{"pkg-oor-b", "2024/oor-b.jpg"},
	} {
		full := filepath.Join(root, filepath.FromSlash(tc.rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, minimalJPEG(t), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := store.InsertAsset(db, tc.hash, tc.rel, now, now); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`UPDATE assets SET mime = ? WHERE content_hash = ?`, "image/jpeg", tc.hash); err != nil {
			t.Fatal(err)
		}
	}
	var idA, idB int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = 'pkg-oor-a'`).Scan(&idA); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = 'pkg-oor-b'`).Scan(&idB); err != nil {
		t.Fatal(err)
	}

	raw, _, err := store.MintPackageShareLink(context.Background(), db, []int64{idA, idB}, now+1, store.ShareSnapshotPayload{})
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(NewHTTPHandler(db, root))
	t.Cleanup(srv.Close)
	client := srv.Client()

	bogus := "bogus-token-not-in-db-xxxxxxxxxxxxxxxxxxxxxxxxxx"
	refURL := srv.URL + SharePackageMemberImagePath(bogus, 0)
	oorURL := srv.URL + SharePackageMemberImagePath(raw, 2) // positions 0,1 only

	respRef, err := client.Get(refURL)
	if err != nil {
		t.Fatal(err)
	}
	refBody, err := func() ([]byte, error) {
		defer respRef.Body.Close()
		if respRef.StatusCode != http.StatusNotFound {
			t.Fatalf("ref status: %d", respRef.StatusCode)
		}
		return io.ReadAll(respRef.Body)
	}()
	if err != nil {
		t.Fatal(err)
	}
	refHdr := respRef.Header.Clone()

	respOOR, err := client.Get(oorURL)
	if err != nil {
		t.Fatal(err)
	}
	defer respOOR.Body.Close()
	oorBody, _ := io.ReadAll(respOOR.Body)
	if respOOR.StatusCode != http.StatusNotFound {
		t.Fatalf("out-of-range status: %d", respOOR.StatusCode)
	}
	if string(oorBody) != string(refBody) {
		t.Fatalf("body oor vs ref: %q vs %q", oorBody, refBody)
	}
	for _, k := range []string{"Content-Type", "Cache-Control", "Content-Length"} {
		if g, w := refHdr.Get(k), respOOR.Header.Get(k); g != w {
			t.Fatalf("header %q: ref %q vs oor %q", k, g, w)
		}
	}

	reqH, err := http.NewRequest(http.MethodHead, oorURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	respHead, err := client.Do(reqH)
	if err != nil {
		t.Fatal(err)
	}
	defer respHead.Body.Close()
	if respHead.StatusCode != http.StatusNotFound {
		t.Fatalf("HEAD oor: %d", respHead.StatusCode)
	}
	if cl := respHead.Header.Get("Content-Length"); cl != strconv.Itoa(len(NotFoundBody)) {
		t.Fatalf("HEAD Content-Length: %s", cl)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
