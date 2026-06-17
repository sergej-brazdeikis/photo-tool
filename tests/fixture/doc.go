// Package fixture provides scale-test library seeding (see internal/fixture).
package fixture

import intfix "photo-tool/internal/fixture"

// Re-export tier types for tests/extended callers.
type Tier = intfix.Tier

const (
	TierS0 = intfix.TierS0
	TierS1 = intfix.TierS1
	TierS2 = intfix.TierS2
	TierS3 = intfix.TierS3
	TierS4 = intfix.TierS4
	TierS5 = intfix.TierS5
	TierS6 = intfix.TierS6
	TierS7 = intfix.TierS7
	TierS8 = intfix.TierS8
)

var (
	TierSpec    = intfix.TierSpec
	ParseTier   = intfix.ParseTier
	SeedLibrary = intfix.SeedLibrary
)
