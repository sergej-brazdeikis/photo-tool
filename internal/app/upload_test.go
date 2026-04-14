package app

import (
	"strings"
	"testing"

	"photo-tool/internal/domain"
)

func TestSummarizeDoneMessage_skipCollection(t *testing.T) {
	sum := domain.OperationSummary{Added: 2, SkippedDuplicate: 0, Failed: 0}
	got := summarizeDoneMessage(sum, false, false)
	if !strings.Contains(got, "No new collection was created") {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(got, "linked") {
		t.Fatalf("unexpected link copy: %q", got)
	}
}

func TestSummarizeDoneMessage_linked(t *testing.T) {
	sum := domain.OperationSummary{Added: 1, SkippedDuplicate: 0, Failed: 0}
	got := summarizeDoneMessage(sum, true, true)
	if !strings.Contains(got, "linked to the new collection") {
		t.Fatalf("got %q", got)
	}
}

func TestSummarizeDoneMessage_wantedAssignButNoIngestedAssets(t *testing.T) {
	sum := domain.OperationSummary{Added: 0, SkippedDuplicate: 0, Failed: 3}
	got := summarizeDoneMessage(sum, true, false)
	if strings.Contains(got, "linked to the new collection") {
		t.Fatalf("must not claim linked: %q", got)
	}
	if !strings.Contains(got, "No collection was created") {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(got, "successfully ingested") {
		t.Fatalf("got %q", got)
	}
}

func TestSummarizeDoneMessage_failedIncludesNextStep(t *testing.T) {
	sum := domain.OperationSummary{Added: 1, SkippedDuplicate: 0, Failed: 2}
	got := summarizeDoneMessage(sum, false, false)
	if !strings.Contains(got, "failed 2") {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(got, "For items that failed") {
		t.Fatalf("missing next step: %q", got)
	}
}
