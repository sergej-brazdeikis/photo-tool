package domain

// NFR-01 desktop window band (PRD NFR-01; Story 2.11).
// Manual QA matrices in _bmad-output should stay consistent with these bounds.
const (
	NFR01WindowMinWidth  = 1024
	NFR01WindowMinHeight = 768
	NFR01WindowMaxWidth  = 5120
	NFR01WindowMaxHeight = 1440
)

// NFR01MatrixCell is one row of the Epic 2 NFR-01 evidence matrix (cell IDs in
// nfr-01-layout-matrix-evidence.md). IsLoupe distinguishes Review vs Loupe (-L) rows.
type NFR01MatrixCell struct {
	CellID  string
	Width   int
	Height  int
	IsLoupe bool
}

// NFR01Epic2MatrixCells returns the full tier-1 representative matrix (min/mid/max
// per aspect family × Review + Loupe). Sizes match _bmad-output/.../nfr-01-layout-matrix-evidence.md.
func NFR01Epic2MatrixCells() []NFR01MatrixCell {
	add := func(out *[]NFR01MatrixCell, id string, w, h int) {
		*out = append(*out, NFR01MatrixCell{CellID: id, Width: w, Height: h, IsLoupe: false})
		*out = append(*out, NFR01MatrixCell{CellID: id + "-L", Width: w, Height: h, IsLoupe: true})
	}
	var out []NFR01MatrixCell
	add(&out, "S-min", 1024, 1024)
	add(&out, "S-mid", 1280, 1280)
	add(&out, "S-max", 1440, 1440)
	add(&out, "169-min", 1366, 768)
	add(&out, "169-mid", 1920, 1080)
	add(&out, "169-max", 2560, 1440)
	add(&out, "219-min", 1792, 768)
	add(&out, "219-mid", 2560, 1080)
	add(&out, "219-max", 5120, 1440)
	return out
}

// NFR07Epic2DefaultSubsetCellIDs is the documented subset for NFR-07 when the full
// matrix is time-boxed (Story 2.11 / nfr-07-os-scaling-checklist.md).
func NFR07Epic2DefaultSubsetCellIDs() []string {
	return []string{
		"S-mid", "S-mid-L",
		"169-mid", "169-mid-L",
		"219-mid", "219-mid-L",
	}
}

// NFR01AC2ResizeSweepPath is the canonical corner-touching order for AC2-style
// continuous-resize checks (Story 2.11 / nfr-01-layout-matrix-evidence.md).
// It is not exhaustive of every matrix cell; it stresses small square, ultrawide
// max extent, short 16:9, and mid21:9 before returning to a mid16:9 idle point.
func NFR01AC2ResizeSweepPath() [][2]int {
	return [][2]int{
		{1920, 1080}, // 169-mid — typical start
		{1024, 1024}, // S-min
		{5120, 1440}, // 219-max — PRD max extent
		{1366, 768},  // 169-min
		{2560, 1080}, // 219-mid
		{1920, 1080}, // 169-mid — idle endpoint
	}
}
