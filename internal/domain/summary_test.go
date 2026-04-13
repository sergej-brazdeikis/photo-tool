package domain

import (
	"encoding/json"
	"testing"
)

func TestOperationSummary_JSON_snakeCase(t *testing.T) {
	b, err := json.Marshal(OperationSummary{
		Added:            1,
		SkippedDuplicate: 2,
		Updated:          0,
		Failed:           3,
	})
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"added":1,"skipped_duplicate":2,"updated":0,"failed":3}`
	if string(b) != want {
		t.Fatalf("JSON: got %s want %s", b, want)
	}
}
