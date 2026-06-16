# Share abuse posture — NFR-06

**Requirement:** PRD NFR-06 — share links use non-guessable tokens; rate-limit or abuse posture documented before public deployment.

**Implementation:** `internal/share/ratelimit.go`, `internal/config/share_http.go`, `internal/share/handler.go`

## MVP defaults (loopback desktop)

| Control | Default |
|---------|---------|
| Bind address | `127.0.0.1` (loopback only) |
| Port | `8765` (up to 10 successors on busy port) |
| Token | 32 random bytes, URL-safe encoding |
| Storage | SHA-256 hash only — never raw token in DB |
| Rate limit | 12 req/s refill, burst **80** per IP |
| Visitor map cap | 4096 distinct IP keys; single-entry eviction when exceeded |
| 429 body | Fixed `Too Many Requests\n` — no token existence oracle |

Environment overrides: see [`docs/env.md`](env.md) (`PHOTO_TOOL_SHARE_HTTP_*`).

## Loopback-first security model

- Default bind is loopback — share URLs are intended for same-machine or explicitly forwarded access.
- **LAN exposure** requires `PHOTO_TOOL_SHARE_HTTP_BIND_ALL=1` — binds `0.0.0.0` with a code comment security warning. Clipboard URLs still prefer `127.0.0.1` for same-machine use.
- No authentication in MVP — security relies on **secret URL** entropy.

## In-process limiter limits

The default burst of **80** is tuned for loopback browsing (HTML + image fetches + parallel gallery loads) without false 429s. It is **not** a public-edge policy.

- **Single process:** limiter state is in-memory; restart clears counters.
- **Loopback MVP:** typically one visitor IP key.
- **Many distinct IPs:** visitor map eviction prevents unbounded memory growth but is not a substitute for edge rate limiting.

## Public deployment guidance

If the share listener is ever exposed beyond loopback:

1. **Reverse proxy** — enforce rate limits, connection limits, and TLS at the edge.
2. **`X-Forwarded-For`** — do not trust forwarded headers unless behind a known trusted proxy; spoofing can bypass or mis-target per-IP limits.
3. **Token entropy** — keep 32-byte random tokens; consider shorter link lifetime or revocation if product requires it (out of MVP).
4. **Restart** — in-process state resets; plan for graceful restarts or external rate-limit stores if needed.

## Package / multi-image shares

Large share packages may trigger parallel browser fetches to `/i/{token}` paths. Default burst 80 accommodates typical loopback gallery loads. If the listener is off-machine, operators should tune edge limits for expected gallery sizes.

Open follow-up: document parallel fetch behavior vs burst for very large packages (Story 3.5 review item).

## Related

- [`docs/share-cold-load-nfr05.md`](share-cold-load-nfr05.md) — NFR-05
- [`internal/share/http_test.go`](../internal/share/http_test.go) — rate limit and 404 discipline tests
- [`_bmad-output/planning-artifacts/architecture.md`](../_bmad-output/planning-artifacts/architecture.md) §3.5
