package share

import (
	"strings"
	"testing"

	"photo-tool/internal/store"
)

func TestRatingViewModel(t *testing.T) {
	z := 0
	six := 6
	three := 3
	cases := []struct {
		name      string
		p store.ShareSnapshotPayload
		wantLabel string
		wantStars int // filled count when rated; -1 means unrated (all hollow)
	}{
		{"empty", store.ShareSnapshotPayload{}, "Unrated", -1},
		{"nil", store.ShareSnapshotPayload{Rating: nil}, "Unrated", -1},
		{"zero", store.ShareSnapshotPayload{Rating: &z}, "Unrated", -1},
		{"six", store.ShareSnapshotPayload{Rating: &six}, "Unrated", -1},
		{"three", store.ShareSnapshotPayload{Rating: &three}, "Rating: 3", 3},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			stars, label := ratingViewModel(tc.p)
			if label != tc.wantLabel {
				t.Fatalf("label: %q want %q", label, tc.wantLabel)
			}
			s := string(stars)
			if tc.wantStars < 0 {
				if strings.Contains(s, `star filled`) {
					t.Fatalf("expected no filled stars, got %q", s)
				}
				return
			}
			n := strings.Count(s, `star filled`)
			if n != tc.wantStars {
				t.Fatalf("filled segments: %d want %d in %q", n, tc.wantStars, s)
			}
		})
	}
}
