package store

import (
	"testing"
)

func TestParseShareSnapshotPayloadJSON_goldenRatings(t *testing.T) {
	cases := []struct {
		name    string
		json    string
		wantNil bool
		wantVal int
	}{
		{"absent", `{}`, true, 0},
		{"null", `{"rating":null}`, true, 0},
		{"one", `{"rating":1}`, false, 1},
		{"five", `{"rating":5}`, false, 5},
		{"zero", `{"rating":0}`, false, 0},
		{"whitespace", `  {"rating": 3}  `, false, 3},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			p, err := ParseShareSnapshotPayloadJSON(tc.json)
			if err != nil {
				t.Fatal(err)
			}
			if tc.wantNil {
				if p.Rating != nil {
					t.Fatalf("Rating: got %v want nil", p.Rating)
				}
				return
			}
			if p.Rating == nil {
				t.Fatal("Rating: nil")
			}
			if *p.Rating != tc.wantVal {
				t.Fatalf("Rating: got %d want %d", *p.Rating, tc.wantVal)
			}
		})
	}
}

func TestParseShareSnapshotPayloadJSON_emptyString(t *testing.T) {
	p, err := ParseShareSnapshotPayloadJSON("")
	if err != nil {
		t.Fatal(err)
	}
	if p.Rating != nil {
		t.Fatalf("want nil rating, got %v", *p.Rating)
	}
}

func TestParseShareSnapshotPayloadJSON_invalidJSON(t *testing.T) {
	_, err := ParseShareSnapshotPayloadJSON("{")
	if err == nil {
		t.Fatal("expected error")
	}
}
