package domain

// OperationSummary aggregates ingest outcomes for CLI/GUI (NFR-04).
// Field names and JSON tags are stable API; use snake_case when serialized (architecture §3.9, §4.1).
// Updated counts metadata corrections (e.g. import backfill of capture_time_unix per Story 1.7).
type OperationSummary struct {
	Added            int `json:"added"`
	SkippedDuplicate int `json:"skipped_duplicate"`
	Updated          int `json:"updated"`
	Failed           int `json:"failed"`
}
