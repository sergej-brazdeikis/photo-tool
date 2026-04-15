package app

import (
	"strings"
	"testing"
)

func TestRatingBadgeText(t *testing.T) {
	if got := ratingBadgeText(nil); got != "—" {
		t.Fatalf("nil: %q", got)
	}
	three := 3
	if got := ratingBadgeText(&three); got != "3★" {
		t.Fatalf("rated: %q", got)
	}
}

func TestRejectBadgeLabel(t *testing.T) {
	if got := rejectBadgeLabel(0); got != "" {
		t.Fatalf("0: %q", got)
	}
	if got := rejectBadgeLabel(1); got != "Hidden" {
		t.Fatalf("1: %q", got)
	}
}

func TestReviewGridListRowCount(t *testing.T) {
	tests := []struct {
		total int64
		want  int
	}{
		{total: -3, want: 0},
		{total: 0, want: 0},
		{total: 1, want: 1},
		{total: 4, want: 1},
		{total: 5, want: 2},
		{total: 48, want: 12},
		{total: 49, want: 13},
	}
	for _, tt := range tests {
		if got := reviewGridListRowCount(tt.total); got != tt.want {
			t.Fatalf("total=%d: got %d want %d", tt.total, got, tt.want)
		}
	}
}

func TestReviewGridUserFacingMessagesSanitized(t *testing.T) {
	badSubstr := []string{
		"sqlite", "SQL", "near ", "syntax", "errno", "syscall",
		"database disk", "locked", "0x",
	}
	for _, msg := range []string{reviewGridMsgPageLoadFail, reviewGridMsgDecodeFail} {
		lower := strings.ToLower(msg)
		for _, frag := range badSubstr {
			if strings.Contains(lower, strings.ToLower(frag)) {
				t.Fatalf("user copy must not contain %q: %q", frag, msg)
			}
		}
	}
}
