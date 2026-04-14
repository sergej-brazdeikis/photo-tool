package domain

import "testing"

func TestNFR01WindowBandMatchesPRD(t *testing.T) {
	// PRD NFR-01: between 1024×768 and 5120×1440.
	if NFR01WindowMinWidth != 1024 || NFR01WindowMinHeight != 768 {
		t.Fatalf("min: got %d×%d", NFR01WindowMinWidth, NFR01WindowMinHeight)
	}
	if NFR01WindowMaxWidth != 5120 || NFR01WindowMaxHeight != 1440 {
		t.Fatalf("max: got %d×%d", NFR01WindowMaxWidth, NFR01WindowMaxHeight)
	}
}

func TestNFR01Epic2MatrixCells_geometryInBand(t *testing.T) {
	cells := NFR01Epic2MatrixCells()
	if len(cells) != 18 {
		t.Fatalf("matrix row count: got %d want 18 (9 aspect points × Review + Loupe)", len(cells))
	}
	for _, c := range cells {
		if c.Width < NFR01WindowMinWidth || c.Width > NFR01WindowMaxWidth {
			t.Fatalf("%s: width %d out of band", c.CellID, c.Width)
		}
		if c.Height < NFR01WindowMinHeight || c.Height > NFR01WindowMaxHeight {
			t.Fatalf("%s: height %d out of band", c.CellID, c.Height)
		}
	}
}

func TestNFR07Epic2DefaultSubset_refsNFR01Cells(t *testing.T) {
	seen := make(map[string]struct{}, 30)
	for _, c := range NFR01Epic2MatrixCells() {
		seen[c.CellID] = struct{}{}
	}
	for _, id := range NFR07Epic2DefaultSubsetCellIDs() {
		if _, ok := seen[id]; !ok {
			t.Fatalf("subset id %q not in NFR-01 matrix", id)
		}
	}
}

func TestNFR01AC2ResizeSweepPath_inBand(t *testing.T) {
	for i, wh := range NFR01AC2ResizeSweepPath() {
		w, h := wh[0], wh[1]
		if w < NFR01WindowMinWidth || w > NFR01WindowMaxWidth {
			t.Fatalf("step %d: width %d out of band", i, w)
		}
		if h < NFR01WindowMinHeight || h > NFR01WindowMaxHeight {
			t.Fatalf("step %d: height %d out of band", i, h)
		}
	}
}
