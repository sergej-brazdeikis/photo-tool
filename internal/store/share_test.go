package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"photo-tool/internal/config"
)

func TestMintDefaultShareLink_happyPath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().Unix()
	if err := InsertAsset(db, "h-share-1", "2024/share1.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = ?`, "h-share-1").Scan(&id); err != nil {
		t.Fatal(err)
	}
	if err := UpdateAssetRating(db, id, 4); err != nil {
		t.Fatal(err)
	}

	before, err := CountShareLinks(db)
	if err != nil {
		t.Fatal(err)
	}
	raw, linkID, err := MintDefaultShareLink(context.Background(), db, id, now+1)
	if err != nil || linkID <= 0 || raw == "" {
		t.Fatalf("mint: raw=%q id=%d err=%v", raw, linkID, err)
	}
	after, err := CountShareLinks(db)
	if err != nil {
		t.Fatal(err)
	}
	if after != before+1 {
		t.Fatalf("row count: before=%d after=%d", before, after)
	}

	sum := sha256.Sum256([]byte(raw))
	wantHash := hex.EncodeToString(sum[:])
	var gotHash string
	var aid int64
	var payload string
	var created int64
	if err := db.QueryRow(`
SELECT token_hash, asset_id, created_at_unix, payload FROM share_links WHERE id = ?`, linkID).
		Scan(&gotHash, &aid, &created, &payload); err != nil {
		t.Fatal(err)
	}
	if gotHash != wantHash {
		t.Fatalf("token_hash: got %q want %q", gotHash, wantHash)
	}
	if aid != id {
		t.Fatalf("asset_id: got %d want %d", aid, id)
	}
	if created != now+1 {
		t.Fatalf("created_at_unix: got %d want %d", created, now+1)
	}
	if gotHash == raw {
		t.Fatal("hash must not equal raw token string")
	}
	var snap map[string]any
	if err := json.Unmarshal([]byte(payload), &snap); err != nil {
		t.Fatal(err)
	}
	rv, ok := snap["rating"]
	if !ok {
		t.Fatalf("payload missing rating: %s", payload)
	}
	rf, ok := rv.(float64)
	if !ok || int(rf) != 4 {
		t.Fatalf("payload rating: %#v", rv)
	}
}

// AC7(c) / AC1: Dismissing the preview without confirm must not write share_links.
// MintDefaultShareLink is the only writer for single-asset rows; pre-mint eligibility reads must be side-effect free.
func TestShareLinks_countUnchangedWhenMintNotCalled(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().Unix()
	if err := InsertAsset(db, "h-share-no-mint", "2024/nomint.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = ?`, "h-share-no-mint").Scan(&id); err != nil {
		t.Fatal(err)
	}

	before, err := CountShareLinks(db)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := AssetEligibleForDefaultShare(db, id)
	if err != nil || !ok {
		t.Fatalf("eligible: ok=%v err=%v", ok, err)
	}
	msg, err := DefaultShareBlockedUserMessage(db, id)
	if err != nil || msg != "" {
		t.Fatalf("gate: msg=%q err=%v", msg, err)
	}
	after, err := CountShareLinks(db)
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("share_links count changed without mint: before=%d after=%d", before, after)
	}
}

func TestMintDefaultShareLink_rejectedBlocked(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().Unix()
	if err := InsertAsset(db, "h-share-r", "2024/r.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = ?`, "h-share-r").Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := RejectAsset(db, id, now+5); err != nil {
		t.Fatal(err)
	}
	_, _, err = MintDefaultShareLink(context.Background(), db, id, now+10)
	if !errors.Is(err, ErrShareAssetIneligible) {
		t.Fatalf("want ErrShareAssetIneligible, got %v", err)
	}
	n, err := CountShareLinks(db)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("share_links rows: got %d", n)
	}
}

func TestMintDefaultShareLink_softDeletedBlocked(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().Unix()
	if err := InsertAsset(db, "h-share-d", "2024/d.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = ?`, "h-share-d").Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE assets SET deleted_at_unix = ? WHERE id = ?`, now+3, id); err != nil {
		t.Fatal(err)
	}
	_, _, err = MintDefaultShareLink(context.Background(), db, id, now+10)
	if !errors.Is(err, ErrShareAssetIneligible) {
		t.Fatalf("want ErrShareAssetIneligible, got %v", err)
	}
	n, err := CountShareLinks(db)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("share_links rows after soft-delete mint: got %d want 0", n)
	}
}

func TestIsSQLiteUniqueTokenHash(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("UNIQUE constraint failed: share_links.token_hash"), true},
		{errors.New("unique constraint failed: share_links.token_hash"), true},
		{errors.New("UNIQUE constraint failed: share_links.asset_id"), false},
		{errors.New("constraint failed"), false},
	} {
		if got := isSQLiteUniqueTokenHash(tc.err); got != tc.want {
			t.Fatalf("err=%v: got %v want %v", tc.err, got, tc.want)
		}
	}
}

func TestResolveDefaultShareLink_roundTrip(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().Unix()
	if err := InsertAsset(db, "h-res-1", "2024/r1.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = ?`, "h-res-1").Scan(&id); err != nil {
		t.Fatal(err)
	}
	raw, linkID, err := MintDefaultShareLink(context.Background(), db, id, now+1)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ResolveDefaultShareLink(context.Background(), db, raw)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ShareLinkID != linkID || got.AssetID != id {
		t.Fatalf("resolve: %#v (linkID=%d asset=%d)", got, linkID, id)
	}
}

func TestResolveDefaultShareLink_emptyToken(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	got, err := ResolveDefaultShareLink(context.Background(), db, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("want nil, got %#v", got)
	}
}

func TestResolveDefaultShareLink_unknownToken(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	got, err := ResolveDefaultShareLink(context.Background(), db, "not-a-real-token-xxxxxxxxxxxxxxxxxxxxxxxxxx")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("want nil, got %#v", got)
	}
}

func TestResolveDefaultShareLink_rejectedAssetMiss(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().Unix()
	if err := InsertAsset(db, "h-res-rj", "2024/rj.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = ?`, "h-res-rj").Scan(&id); err != nil {
		t.Fatal(err)
	}
	raw, _, err := MintDefaultShareLink(context.Background(), db, id, now+1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RejectAsset(db, id, now+5); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveDefaultShareLink(context.Background(), db, raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("rejected asset: want nil, got %#v", got)
	}
}

func TestResolveDefaultShareLink_softDeletedAssetMiss(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().Unix()
	if err := InsertAsset(db, "h-res-del", "2024/rdel.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = ?`, "h-res-del").Scan(&id); err != nil {
		t.Fatal(err)
	}
	raw, _, err := MintDefaultShareLink(context.Background(), db, id, now+1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE assets SET deleted_at_unix = ? WHERE id = ?`, now+9, id); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveDefaultShareLink(context.Background(), db, raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("trashed asset: want nil, got %#v", got)
	}
}

func TestDefaultShareBlockedUserMessage(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	msg, err := DefaultShareBlockedUserMessage(db, 0)
	if err != nil || msg == "" {
		t.Fatalf("id0: msg=%q err=%v", msg, err)
	}

	now := time.Now().Unix()
	if err := InsertAsset(db, "h-gate", "2024/g.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = ?`, "h-gate").Scan(&id); err != nil {
		t.Fatal(err)
	}
	msg, err = DefaultShareBlockedUserMessage(db, id)
	if err != nil || msg != "" {
		t.Fatalf("eligible: msg=%q err=%v", msg, err)
	}

	if _, err := RejectAsset(db, id, now+1); err != nil {
		t.Fatal(err)
	}
	msg, err = DefaultShareBlockedUserMessage(db, id)
	if err != nil || msg == "" {
		t.Fatalf("rejected: msg=%q err=%v", msg, err)
	}
}

func TestPackageMintSentinelErrors_distinct(t *testing.T) {
	t.Parallel()
	if errors.Is(ErrPackageTooManyAssets, ErrShareAssetIneligible) {
		t.Fatal("ErrPackageTooManyAssets must not match ErrShareAssetIneligible")
	}
	if errors.Is(ErrPackageNoEligibleAssets, ErrShareAssetIneligible) {
		t.Fatal("ErrPackageNoEligibleAssets must not match ErrShareAssetIneligible")
	}
	if !errors.Is(ErrPackageTooManyAssets, ErrPackageTooManyAssets) {
		t.Fatal("ErrPackageTooManyAssets errors.Is stability")
	}
	if !errors.Is(ErrPackageNoEligibleAssets, ErrPackageNoEligibleAssets) {
		t.Fatal("ErrPackageNoEligibleAssets errors.Is stability")
	}
}

func TestPackagePrepareEligibleForMint_dropsRejected(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().Unix()
	if err := InsertAsset(db, "p-elig", "2024/e.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	if err := InsertAsset(db, "p-rj", "2024/r.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var idOK, idR int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = 'p-elig'`).Scan(&idOK); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = 'p-rj'`).Scan(&idR); err != nil {
		t.Fatal(err)
	}
	if _, err := RejectAsset(db, idR, now+1); err != nil {
		t.Fatal(err)
	}
	got, err := PackagePrepareEligibleForMint(context.Background(), db, []int64{idOK, idR})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != idOK {
		t.Fatalf("eligible: %#v", got)
	}
}

func TestPackagePrepareEligibleForMint_noEligible(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().Unix()
	if err := InsertAsset(db, "p-only-r", "2024/or.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var idR int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = 'p-only-r'`).Scan(&idR); err != nil {
		t.Fatal(err)
	}
	if _, err := RejectAsset(db, idR, now+1); err != nil {
		t.Fatal(err)
	}
	_, err = PackagePrepareEligibleForMint(context.Background(), db, []int64{idR})
	if !errors.Is(err, ErrPackageNoEligibleAssets) {
		t.Fatalf("want ErrPackageNoEligibleAssets, got %v", err)
	}
}

func TestMintPackageShareLink_happyPathAndDedupeMembers(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().Unix()
	if err := InsertAsset(db, "p-a", "2024/pa.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	if err := InsertAsset(db, "p-b", "2024/pb.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var idA, idB int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = 'p-a'`).Scan(&idA); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = 'p-b'`).Scan(&idB); err != nil {
		t.Fatal(err)
	}

	raw, linkID, err := MintPackageShareLink(context.Background(), db, []int64{idA, idA, idB}, now+1, ShareSnapshotPayload{DisplayTitle: "T"})
	if err != nil || linkID <= 0 || raw == "" {
		t.Fatalf("mint: id=%d raw=%q err=%v", linkID, raw, err)
	}

	var nMem int
	if err := db.QueryRow(`SELECT COUNT(*) FROM share_link_members WHERE share_link_id = ?`, linkID).Scan(&nMem); err != nil {
		t.Fatal(err)
	}
	if nMem != 2 {
		t.Fatalf("member rows: got %d want 2", nMem)
	}

	pkg, err := ResolvePackageShareLink(context.Background(), db, raw)
	if err != nil || pkg == nil {
		t.Fatalf("resolve pkg: %#v err=%v", pkg, err)
	}
	if len(pkg.MemberIDs) != 2 || pkg.MemberIDs[0] != idA || pkg.MemberIDs[1] != idB {
		t.Fatalf("members: %#v", pkg.MemberIDs)
	}
}

func TestMintPackageShareLink_rejectsIneligibleInsideTx(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().Unix()
	if err := InsertAsset(db, "p-x", "2024/px.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	if err := InsertAsset(db, "p-y", "2024/py.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var idX, idY int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = 'p-x'`).Scan(&idX); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = 'p-y'`).Scan(&idY); err != nil {
		t.Fatal(err)
	}
	if _, err := RejectAsset(db, idY, now+2); err != nil {
		t.Fatal(err)
	}

	before, _ := CountShareLinks(db)
	_, _, err = MintPackageShareLink(context.Background(), db, []int64{idX, idY}, now+5, ShareSnapshotPayload{})
	if !errors.Is(err, ErrShareAssetIneligible) {
		t.Fatalf("want ErrShareAssetIneligible, got %v", err)
	}
	after, _ := CountShareLinks(db)
	if after != before {
		t.Fatalf("share_links after failed mint: before=%d after=%d", before, after)
	}
}

func TestMintPackageShareLink_tooManyCap(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ids := make([]int64, packageShareMaxEligibleAssets+1)
	for i := range ids {
		h := fmt.Sprintf("cap-%d", i)
		rel := fmt.Sprintf("2024/cap/%d.jpg", i)
		if err := InsertAsset(db, h, rel, int64(i), int64(i)); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = ?`, h).Scan(&ids[i]); err != nil {
			t.Fatal(err)
		}
	}
	_, _, err = MintPackageShareLink(context.Background(), db, ids, 99, ShareSnapshotPayload{})
	if !errors.Is(err, ErrPackageTooManyAssets) {
		t.Fatalf("want ErrPackageTooManyAssets, got %v", err)
	}
}

func TestPackageShare_linkCountUnchangedWhenMintNotCalled(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().Unix()
	if err := InsertAsset(db, "p-no-mint", "2024/nm.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = 'p-no-mint'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	before, err := CountShareLinks(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = PackagePrepareEligibleForMint(context.Background(), db, []int64{id})
	if err != nil {
		t.Fatal(err)
	}
	after, err := CountShareLinks(db)
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("prepare without mint changed share_links: %d -> %d", before, after)
	}
}

func TestResolvePackageShareLink_stillListsAfterMemberRejected(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().Unix()
	if err := InsertAsset(db, "snap-a", "2024/sa.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	if err := InsertAsset(db, "snap-b", "2024/sb.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var idA, idB int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = 'snap-a'`).Scan(&idA); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = 'snap-b'`).Scan(&idB); err != nil {
		t.Fatal(err)
	}

	raw, _, err := MintPackageShareLink(context.Background(), db, []int64{idA, idB}, now+1, ShareSnapshotPayload{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RejectAsset(db, idB, now+9); err != nil {
		t.Fatal(err)
	}

	pkg, err := ResolvePackageShareLink(context.Background(), db, raw)
	if err != nil || pkg == nil {
		t.Fatalf("resolve: %#v err=%v", pkg, err)
	}
	if len(pkg.MemberIDs) != 2 || pkg.MemberIDs[0] != idA || pkg.MemberIDs[1] != idB {
		t.Fatalf("snapshot members: %#v", pkg.MemberIDs)
	}
}
