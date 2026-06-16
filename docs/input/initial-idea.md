# Initial product idea (historical)

> **Requirements authority:** [`PRD.md`](../_bmad-output/planning-artifacts/PRD.md) since 2026-04-12 BMAD conversion. This document preserves original intent referenced by PRD frontmatter.

## Vision

A **local-first photo management tool** for a single operator — no cloud accounts, no multi-tenancy. Photos live on disk under a configurable library root with SQLite metadata.

## Core capabilities (original scope)

- **Import:** Drag-drop and folder scan; deduplicated storage by capture date and content hash.
- **Review:** Thumbnail grid, loupe, ratings, tags, filters; reject vs delete with distinct semantics.
- **Collections:** Albums with display dates and grouping views.
- **Share:** Mint read-only web links from the desktop app; preview before mint; snapshot semantics at mint time.

## Platform and UI

- **Desktop:** Fyne (Go) — default launch mode.
- **CLI:** `scan` and `import` subcommands with the same ingest pipeline as the GUI.
- **Share web:** Lightweight HTML/CSS served from the desktop process on loopback — not Fyne WASM for MVP.

## UX priorities called out early

- **Image-first layout** — thumbnails and loupe are primary; chrome stays minimal.
- **Dual themes** — dark default, light peer; semantic color roles for primary, destructive, reject-caution.
- **Ultrawide safety** — never pin critical controls off-screen; use flex layouts and safe regions (explicit failure mode from early feedback).
- **OS scaling** — layout must hold at 125% / 150% display scaling on macOS and Windows.

## Non-goals (MVP)

- Cloud sync, accounts, or collaborative editing.
- Hard erase / secure delete (soft-delete + quarantine instead).
- Share link revocation (deferred).

For measurable requirements (FR/NFR IDs), see the PRD.
