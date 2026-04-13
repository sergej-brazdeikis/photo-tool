package domain

// OperationSummary aggregates ingest outcomes for CLI/GUI (NFR-04).
// Field names and JSON tags are stable API; use snake_case when serialized (architecture §3.9, §4.1).
// Updated remains zero for metadata-only updates until a future story adds that path.
type OperationSummary struct {
	Added            int `json:"added"`
	SkippedDuplicate int `json:"skipped_duplicate"`
	Updated          int `json:"updated"`
	Failed           int `json:"failed"`
}
