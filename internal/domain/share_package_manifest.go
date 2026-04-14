package domain

// StableDedupeAssetIDs returns a copy of ids with duplicates removed, preserving
// the order of first occurrence. Non-positive ids are dropped (same invalidity
// rule as store share mint). Empty input yields nil.
//
// Used for Story 4.1 package manifest construction so multi-select or
// filter-merge cannot inflate counts or mint duplicate child rows.
func StableDedupeAssetIDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
