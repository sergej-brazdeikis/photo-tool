package fixture

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"photo-tool/internal/config"
	"photo-tool/internal/store"
)

// SeedOptions configures library seeding.
type SeedOptions struct {
	Tier   Tier
	Root   string
	SrcDir string // optional; upload seed JPEGs written here

	// Optional distribution overrides (zero = tier preset / built-in defaults).
	RatedPct  int         // 1–99: only this percent of active assets get a rating; 0 = all rated
	DaySpread int         // spread capture days across N days (default 28 in seed loop)
	TagLabels []string    // if non-empty, use these tag labels instead of ScaleTag%02d
	Cameras   [][2]string // if non-empty, use these make/model pairs for camera metadata
}

// SeedLibrary populates root with tier-scaled assets, collections, tags, and optional upload seeds.
func SeedLibrary(db *sql.DB, opts SeedOptions) (Manifest, []string, error) {
	start := time.Now()
	spec := TierSpec(opts.Tier)
	if opts.Root == "" {
		return Manifest{}, nil, fmt.Errorf("fixture: empty library root")
	}
	if err := config.EnsureLibraryLayout(opts.Root); err != nil {
		return Manifest{}, nil, err
	}
	if spec.Assets == 0 {
		m := Manifest{Tier: string(opts.Tier), Assets: 0, LibraryRoot: opts.Root}
		if opts.SrcDir != "" {
			if err := os.MkdirAll(opts.SrcDir, 0o755); err != nil {
				return Manifest{}, nil, err
			}
		}
		_ = WriteManifest(opts.Root, m)
		return m, nil, nil
	}
	if opts.Tier == TierS1 {
		return seedJourneyBaseline(db, opts, start)
	}
	return seedBulk(db, opts, spec, start)
}

func seedJourneyBaseline(db *sql.DB, opts SeedOptions, start time.Time) (Manifest, []string, error) {
	now := time.Now().Unix()
	root := opts.Root
	aidFree, err := store.InsertAssetWithCamera(db, "ux-cap-free", "2026/04/15/ux-cap-free.jpg", now, now, "", "")
	if err != nil {
		return Manifest{}, nil, err
	}
	aidInAlbum, err := store.InsertAssetWithCamera(db, "ux-cap-in-album", "2026/04/15/ux-cap-in.jpg", now, now, "", "")
	if err != nil {
		return Manifest{}, nil, err
	}
	aidRejected, err := store.InsertAssetWithCamera(db, "ux-cap-rejected", "2026/04/15/ux-cap-rej.jpg", now, now, "", "")
	if err != nil {
		return Manifest{}, nil, err
	}
	cid, err := store.CreateCollection(db, "UXCapAlb", "")
	if err != nil {
		return Manifest{}, nil, err
	}
	if err := store.LinkAssetsToCollection(db, cid, []int64{aidInAlbum}); err != nil {
		return Manifest{}, nil, err
	}
	if _, err := store.RejectAsset(db, aidRejected, now+1); err != nil {
		return Manifest{}, nil, err
	}
	tid, err := store.FindOrCreateTagByLabel(db, "UXCapTag")
	if err != nil {
		return Manifest{}, nil, err
	}
	if err := store.LinkTagToAssets(db, tid, []int64{aidFree}); err != nil {
		return Manifest{}, nil, err
	}
	_ = aidFree
	for _, rel := range []string{"2026/04/15/ux-cap-free.jpg", "2026/04/15/ux-cap-in.jpg", "2026/04/15/ux-cap-rej.jpg"} {
		if err := WriteTinyJPEG(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			return Manifest{}, nil, err
		}
	}
	var uploadSeeds []string
	if opts.SrcDir != "" {
		if err := os.MkdirAll(opts.SrcDir, 0o755); err != nil {
			return Manifest{}, nil, err
		}
		for _, name := range []string{"ux_journey_new_a.jpg", "ux_journey_new_b.jpg"} {
			p := filepath.Join(opts.SrcDir, name)
			if err := WriteTinyJPEG(p); err != nil {
				return Manifest{}, nil, err
			}
			uploadSeeds = append(uploadSeeds, p)
		}
	}
	m := Manifest{
		Tier:         string(opts.Tier),
		Assets:       3,
		Albums:       1,
		Tags:         1,
		Rejected:     1,
		UploadSeeds:  uploadSeeds,
		LibraryRoot:  root,
		GenerationMS: time.Since(start).Milliseconds(),
	}
	_ = WriteManifest(root, m)
	return m, uploadSeeds, nil
}

func seedBulk(db *sql.DB, opts SeedOptions, spec Spec, start time.Time) (Manifest, []string, error) {
	root := opts.Root
	now := time.Now().Unix()
	cameras := [][2]string{{"Canon", "EOS R5"}, {"Nikon", "Z8"}, {"Sony", "A7IV"}}
	if len(opts.Cameras) > 0 {
		cameras = opts.Cameras
	}
	daySpread := 28
	if opts.DaySpread > 0 {
		daySpread = opts.DaySpread
	}
	type seededAsset struct {
		id      int64
		seedIdx int
	}
	var assets []seededAsset
	rejectedN := 0

	for i := 0; i < spec.Assets; i++ {
		hash := fmt.Sprintf("scale-%06d-%s", i, opts.Tier)
		day := (i % daySpread) + 1
		rel := fmt.Sprintf("2026/06/%02d/scale-%06d.jpg", day, i)
		cap := now - int64(i*3600)
		cam := cameras[i%len(cameras)]
		id, err := store.InsertAssetWithCamera(db, hash, rel, cap, now, cam[0], cam[1])
		if err != nil {
			return Manifest{}, nil, fmt.Errorf("insert asset %d: %w", i, err)
		}
		if err := WriteTinyJPEG(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			return Manifest{}, nil, err
		}
		if shouldRateAsset(i, opts.RatedPct) {
			rating := (i % 5) + 1
			if err := store.UpdateAssetRating(db, id, rating); err != nil {
				return Manifest{}, nil, err
			}
		}
		if spec.RejectedPct > 0 && i%max(1, 100/spec.RejectedPct) == 0 {
			if _, err := store.RejectAsset(db, id, now+int64(i)); err != nil {
				return Manifest{}, nil, err
			}
			rejectedN++
			continue
		}
		assets = append(assets, seededAsset{id: id, seedIdx: i})
	}

	albumCount := spec.Albums
	if albumCount <= 0 {
		albumCount = 1
	}
	var albumIDs []int64
	for a := 0; a < albumCount; a++ {
		name := fmt.Sprintf("ScaleAlb%02d", a)
		if opts.Tier == TierS4 || opts.Tier == TierS5 || opts.Tier == TierS5R {
			if a == 0 {
				name = "UXCapAlb" // filter compatibility with journey steps
			}
		}
		cid, err := store.CreateCollection(db, name, "")
		if err != nil {
			return Manifest{}, nil, err
		}
		albumIDs = append(albumIDs, cid)
	}
	if opts.Tier == TierS4 || opts.Tier == TierS5 || opts.Tier == TierS6 || opts.Tier == TierS5R {
		if _, err := store.CreateCollection(db, "UXCapNone", ""); err != nil {
			return Manifest{}, nil, err
		}
	}
	for _, rec := range assets {
		if len(albumIDs) == 0 {
			break
		}
		cid := albumIDs[rec.seedIdx%len(albumIDs)]
		if spec.ConsolidateFirstAlbum {
			cid = albumIDs[0]
		}
		if err := store.LinkAssetsToCollection(db, cid, []int64{rec.id}); err != nil {
			return Manifest{}, nil, err
		}
	}

	tagCount := spec.Tags
	if tagCount <= 0 {
		tagCount = 1
	}
	tagLabels := opts.TagLabels
	if len(tagLabels) == 0 {
		for t := 0; t < tagCount; t++ {
			label := fmt.Sprintf("ScaleTag%02d", t)
			if t == 0 && (opts.Tier == TierS4 || opts.Tier == TierS5 || opts.Tier == TierS5R) {
				label = "UXCapTag"
			}
			tagLabels = append(tagLabels, label)
		}
	}
	for t, label := range tagLabels {
		tid, err := store.FindOrCreateTagByLabel(db, label)
		if err != nil {
			return Manifest{}, nil, err
		}
		var linkIDs []int64
		mod := max(len(tagLabels), 1)
		for _, rec := range assets {
			if rec.seedIdx%mod == t {
				linkIDs = append(linkIDs, rec.id)
			}
		}
		if len(linkIDs) > 0 {
			if err := store.LinkTagToAssets(db, tid, linkIDs); err != nil {
				return Manifest{}, nil, err
			}
		}
	}

	var uploadSeeds []string
	batch := spec.UploadBatch
	if batch > 0 && opts.SrcDir != "" {
		if err := os.MkdirAll(opts.SrcDir, 0o755); err != nil {
			return Manifest{}, nil, err
		}
		for u := 0; u < batch; u++ {
			p := filepath.Join(opts.SrcDir, fmt.Sprintf("upload_seed_%03d.jpg", u))
			if err := WriteTinyJPEG(p); err != nil {
				return Manifest{}, nil, err
			}
			uploadSeeds = append(uploadSeeds, p)
		}
	}

	m := Manifest{
		Tier:         string(opts.Tier),
		Assets:       spec.Assets,
		Albums:       albumCount,
		Tags:         len(tagLabels),
		Rejected:     rejectedN,
		UploadSeeds:  uploadSeeds,
		LibraryRoot:  root,
		GenerationMS: time.Since(start).Milliseconds(),
		DiskBytes:    dirSize(root),
	}
	_ = WriteManifest(root, m)
	return m, uploadSeeds, nil
}

// SeedFilesystemTree writes n JPEG files under root for CLI scan tests (S8). Does not touch DB.
func SeedFilesystemTree(root string, n int) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	mt := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		p := filepath.Join(root, fmt.Sprintf("f%06d.jpg", i))
		if err := WriteTinyJPEG(p); err != nil {
			return err
		}
		if err := os.Chtimes(p, mt, mt); err != nil {
			return err
		}
	}
	return nil
}

func dirSize(root string) int64 {
	var total int64
	_ = filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		total += info.Size()
		return nil
	})
	return total
}

func shouldRateAsset(i, ratedPct int) bool {
	if ratedPct <= 0 || ratedPct >= 100 {
		return true
	}
	return i%100 < ratedPct
}
