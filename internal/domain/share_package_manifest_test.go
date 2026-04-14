package domain

import (
	"reflect"
	"testing"
)

func TestStableDedupeAssetIDs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   []int64
		want []int64
	}{
		{name: "nil", in: nil, want: nil},
		{name: "empty", in: []int64{}, want: nil},
		{name: "dedupe_preserves_first", in: []int64{3, 1, 3, 2, 1}, want: []int64{3, 1, 2}},
		{name: "drops_non_positive", in: []int64{0, -1, 5, 0}, want: []int64{5}},
		{name: "all_invalid", in: []int64{0, -3}, want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := StableDedupeAssetIDs(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("StableDedupeAssetIDs(%v) = %v; want %v", tt.in, got, tt.want)
			}
		})
	}
}
