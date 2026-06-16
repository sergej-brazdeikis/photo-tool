# Share cold load — NFR-05 methodology

**Requirement:** PRD NFR-05 — shared review URL resolves in under **3 seconds** on broadband for cold load of a single photo page (excluding user network variability).

**Gate test:** `internal/share/nfr05_cold_load_test.go` — `TestNFR05_ShareColdLoadMedian`

## Cold load definition

For each trial:

1. Start a **fresh** `httptest.Server` with `NewHTTPHandler`.
2. **HTML leg:** `GET /s/{token}` — read full response body through `io.Copy(io.Discard, ...)`.
3. **Image leg:** `GET /i/{token}` for the same token — read full image bytes.

Both legs must use a **new TCP connection** per request (see keep-alive caveat below).

## Keep-alive caveat

`httptest.Server.Client()` returns distinct `*http.Client` values that **share one cached Transport** on the server. Keep-alive can reuse the TCP connection from `GET /s/` to `GET /i/`, which does **not** match cold per-leg semantics.

The gate uses:

```go
coldClient := &http.Client{
    Transport: &http.Transport{DisableKeepAlives: true},
}
```

Do not replace this with two `srv.Client()` calls without updating this document and the test.

## Fixture

- Small on-disk JPEG under a temp library root (not RAW decode on the share path).
- Asset registered in SQLite; share link minted via `store.MintDefaultShareLink`.

## Gate parameters

| Parameter | Value |
|-----------|-------|
| Trials | 9 |
| Metric | Median of per-trial durations |
| Budget | ≤ **3s** per leg (HTML and image) |
| `-short` | Test skipped (run full `go test` in CI) |

## Out of scope

- User network latency (WAN, Wi-Fi variability)
- RAW decode performance (fixture is a small JPEG)

## CI caveats

- Measurement is localhost `httptest` — low variance but not identical to production.
- Hosted CI contention may occasionally cause flakes; median over 9 trials mitigates this.
- If flakes persist, consider more trials or staging replay (see `_bmad-output/implementation-artifacts/deferred-work.md`).

## Related

- [`internal/share/nfr05_cold_load_test.go`](../internal/share/nfr05_cold_load_test.go)
- [`docs/share-abuse-posture.md`](share-abuse-posture.md) — NFR-06
- [`.github/workflows/go.yml`](../.github/workflows/go.yml) — CI runs full tests without `-short`
