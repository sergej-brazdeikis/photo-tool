package app

import (
	"database/sql"

	"photo-tool/internal/fixture"
)

// SeedUXJourneyFixture inserts library rows for UX capture (tier from PHOTO_TOOL_UX_FIXTURE_SCALE, default S1).
func SeedUXJourneyFixture(db *sql.DB, root, srcDir string) (uploadA, uploadB string, err error) {
	tier := fixture.ParseTier(UXFixtureScaleTier())
	if fixture.LibrarySeededForTier(root, tier) {
		return fixture.UploadSeedPaths(root, srcDir)
	}
	_, uploadSeeds, err := fixture.SeedLibrary(db, fixture.SeedOptions{
		Tier:   tier,
		Root:   root,
		SrcDir: srcDir,
	})
	if err != nil {
		return "", "", err
	}
	if len(uploadSeeds) >= 2 {
		return uploadSeeds[0], uploadSeeds[1], nil
	}
	if len(uploadSeeds) == 1 {
		return uploadSeeds[0], uploadSeeds[0], nil
	}
	return "", "", nil
}
