package cli

import (
	"strings"
	"testing"
)

// assertOperationReceiptLineOrder locks NFR-04 / UX-DR6 parity: scan and import print exactly four
// lines in a stable order so scripts can rely on the last line being "Failed: …".
func assertOperationReceiptLineOrder(t *testing.T, out string) {
	t.Helper()
	s := strings.TrimSpace(out)
	if s == "" {
		t.Fatal("empty stdout")
	}
	lines := strings.Split(s, "\n")
	if len(lines) != 4 {
		t.Fatalf("want exactly 4 summary lines, got %d:\n%s", len(lines), out)
	}
	prefixes := []string{"Added: ", "Skipped duplicate: ", "Updated: ", "Failed: "}
	for i, p := range prefixes {
		if !strings.HasPrefix(lines[i], p) {
			t.Fatalf("line %d: want prefix %q, got %q", i, p, lines[i])
		}
	}
}
