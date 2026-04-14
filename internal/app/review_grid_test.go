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
