package fixture

// Tier is a named scale preset (S0–S8) for volume and UX-limit testing.
type Tier string

const (
	TierS0  Tier = "S0"
	TierS1  Tier = "S1"
	TierS2  Tier = "S2"
	TierS3  Tier = "S3"
	TierS4  Tier = "S4"
	TierS5  Tier = "S5"
	TierS5R Tier = "S5R" // 500 assets, all rejected — rejected-grid UX
	TierS6  Tier = "S6"
	TierS7  Tier = "S7"
	TierS8  Tier = "S8"
)

// Spec describes asset/album/tag counts for a tier.
type Spec struct {
	Assets                int
	Albums                int
	Tags                  int
	UploadBatch           int
	RejectedPct           int  // 0–100 of assets rejected during seed
	ConsolidateFirstAlbum bool // link all active assets to first album (detail-at-scale)
}

// TierSpec returns the preset for a tier name (case-insensitive).
func TierSpec(t Tier) Spec {
	switch Tier(normalizeTier(string(t))) {
	case TierS0:
		return Spec{}
	case TierS1:
		return Spec{Assets: 3, Albums: 1, Tags: 1, UploadBatch: 2, RejectedPct: 33}
	case TierS2:
		return Spec{Assets: 47, Albums: 5, Tags: 10, UploadBatch: 8, RejectedPct: 5}
	case TierS3:
		return Spec{Assets: 48, Albums: 5, Tags: 10, UploadBatch: 48, RejectedPct: 5}
	case TierS4:
		return Spec{Assets: 96, Albums: 10, Tags: 20, UploadBatch: 50, RejectedPct: 8}
	case TierS5:
		return Spec{Assets: 500, Albums: 50, Tags: 50, UploadBatch: 100, RejectedPct: 0}
	case TierS5R:
		return Spec{Assets: 500, Albums: 1, Tags: 10, RejectedPct: 100, ConsolidateFirstAlbum: true}
	case TierS6:
		return Spec{Assets: 501, Albums: 50, Tags: 50, RejectedPct: 0}
	case TierS7:
		return Spec{Assets: 2000, Albums: 100, Tags: 100, RejectedPct: 10}
	case TierS8:
		return Spec{Assets: 10000, Albums: 0, Tags: 0}
	default:
		return TierSpec(TierS1)
	}
}

func normalizeTier(s string) string {
	if len(s) >= 2 && (s[0] == 'S' || s[0] == 's') {
		return "S" + s[1:]
	}
	return s
}

// ParseTier parses PHOTO_TOOL_SCALE_TIER / PHOTO_TOOL_UX_FIXTURE_SCALE values.
func ParseTier(raw string) Tier {
	raw = normalizeTier(raw)
	switch Tier(raw) {
	case TierS0, TierS1, TierS2, TierS3, TierS4, TierS5, TierS5R, TierS6, TierS7, TierS8:
		return Tier(raw)
	default:
		return TierS1
	}
}
