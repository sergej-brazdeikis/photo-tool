package app

import (
	"bytes"
	"database/sql"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"time"

	"photo-tool/internal/store"
)

// WriteTinyJPEGForUX writes a small non-uniform JPEG for UX journey fixtures.
func WriteTinyJPEGForUX(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	const w, h = 48, 36
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	seed := uint32(2166136261)
	for i := 0; i < len(path); i++ {
		seed ^= uint32(path[i])
		seed *= 16777619
	}
	r0 := 35 + uint8(seed%90)
	g0 := 40 + uint8((seed>>8)%100)
	b0 := 45 + uint8((seed>>16)%95)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := (x * 220) / max(w-1, 1)
			dy := (y * 200) / max(h-1, 1)
			r := min(255, int(r0)+dx)
			g := min(255, int(g0)+dy)
			b := min(255, int(b0)+(x+y)*65/max(w+h-2, 1))
			img.Set(x, y, color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// SeedUXJourneyFixture inserts the standard UX capture library rows and files.
func SeedUXJourneyFixture(db *sql.DB, root, srcDir string) (uploadA, uploadB string, err error) {
	now := time.Now().Unix()
	aidFree, err := store.InsertAssetWithCamera(db, "ux-cap-free", "2026/04/15/ux-cap-free.jpg", now, now, "", "")
	if err != nil {
		return "", "", err
	}
	aidInAlbum, err := store.InsertAssetWithCamera(db, "ux-cap-in-album", "2026/04/15/ux-cap-in.jpg", now, now, "", "")
	if err != nil {
		return "", "", err
	}
	aidRejected, err := store.InsertAssetWithCamera(db, "ux-cap-rejected", "2026/04/15/ux-cap-rej.jpg", now, now, "", "")
	if err != nil {
		return "", "", err
	}
	cid, err := store.CreateCollection(db, "UXCapAlb", "")
	if err != nil {
		return "", "", err
	}
	if err := store.LinkAssetsToCollection(db, cid, []int64{aidInAlbum}); err != nil {
		return "", "", err
	}
	if _, err := store.RejectAsset(db, aidRejected, now+1); err != nil {
		return "", "", err
	}
	tid, err := store.FindOrCreateTagByLabel(db, "UXCapTag")
	if err != nil {
		return "", "", err
	}
	if err := store.LinkTagToAssets(db, tid, []int64{aidFree}); err != nil {
		return "", "", err
	}
	_ = aidFree
	for _, rel := range []string{"2026/04/15/ux-cap-free.jpg", "2026/04/15/ux-cap-in.jpg", "2026/04/15/ux-cap-rej.jpg"} {
		if err := WriteTinyJPEGForUX(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			return "", "", err
		}
	}
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		return "", "", err
	}
	uploadA = filepath.Join(srcDir, "ux_journey_new_a.jpg")
	uploadB = filepath.Join(srcDir, "ux_journey_new_b.jpg")
	if err := WriteTinyJPEGForUX(uploadA); err != nil {
		return "", "", err
	}
	if err := WriteTinyJPEGForUX(uploadB); err != nil {
		return "", "", err
	}
	return uploadA, uploadB, nil
}
