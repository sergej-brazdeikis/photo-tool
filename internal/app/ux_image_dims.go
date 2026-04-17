package app

// UX image/viewport minimums (logical device-independent pixels).
//
// Product bar: _bmad-output/planning-artifacts/ux-design-specification.md — "Core User Experience"
// and anti-patterns (no postage-stamp grids; photos as dominant object).
//
// Tune with: internal/app/nfr01_layout_gate_test.go, TestUXJourneyCapture, and CI OS scale tiers
// (.github/workflows/go.yml FYNE_SCALE / Windows LogPixels).
const (
	// Grid thumbnails: Review, Collections section grids, Rejected — list template MinSize / image floor.
	// Kept at 168 so 4-up rows stay on-canvas with shell nav at NFR-01 1024-wide (layout gates).
	uxImageGridThumbMin = 168

	// Review loupe: main photo and transparent stack floor (hidden image does not contribute MinSize).
	uxImageLoupeMainMin = 280

	// Upload: horizontal batch preview strip on confirm path (FR-06; match grid decode path via thumbnails).
	uxImageUploadBatchPreviewMin = 156

	// Collections album list: cover thumbnail per row (UX image dominance on album list).
	uxImageAlbumListCoverMin = 168

	uxImageSharePackageThumbW = 120
	uxImageSharePackageThumbH = 90
	// Share preview dialog: image must read larger than surrounding copy (UX spec share mint).
	// Sized so the dialog MinSize stays image-forward vs metadata at 1280×800 captures; NFR-01 share step still fits 1024×768.
	uxImageShareLoupeW = 640
	uxImageShareLoupeH = 480
	// Single metadata strip below the image (scroll viewport height cap).
	uxShareLoupeMetaScrollH = 88

	// Review bulk row: HScroll viewports for SelectEntry / assign-target Select must not collapse to
	// scrollbar-only min width; otherwise input surfaces stack on adjacent buttons (judge bulk-row overlap).
	uxReviewBulkTagEntryMinW     = 180
	uxReviewBulkAssignSelectMinW = 200
	// Share-as-package pair: without a floor the viewport collapses at NFR-01 and truncates (“Sha…”).
	uxReviewBulkShareScrollMinW = 300
)
