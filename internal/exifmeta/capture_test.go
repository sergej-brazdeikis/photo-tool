package exifmeta

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
)

func TestReadCapture_exifDateTimeOriginal(t *testing.T) {
	const wantLocal = "2019:05:17 12:34:56"
	exifBytes := mustBuildExifWithDateTimeOriginal(t, wantLocal)
	dir := t.TempDir()
	p := filepath.Join(dir, "with_exif.jpg")
	if err := os.WriteFile(p, exifBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ReadCapture(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != SourceExifDateTimeOriginal {
		t.Fatalf("Source: got %q want %q", got.Source, SourceExifDateTimeOriginal)
	}
	wantUTC, err := time.ParseInLocation(exifDateTimeLayout, wantLocal, time.Local)
	if err != nil {
		t.Fatal(err)
	}
	if !got.UTC.Equal(wantUTC.UTC()) {
		t.Fatalf("UTC: got %v want %v", got.UTC, wantUTC.UTC())
	}
}

func TestReadCapture_exifWithoutDateTimeUsesMtimeUnusable(t *testing.T) {
	exifBytes := mustBuildExifWithoutDateTime(t)
	dir := t.TempDir()
	p := filepath.Join(dir, "exif_no_datetime.jpg")
	if err := os.WriteFile(p, exifBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	mt := time.Date(2022, 7, 4, 18, 0, 0, 0, time.UTC)
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}

	got, err := ReadCapture(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != SourceMtimeExifUnusable {
		t.Fatalf("Source: got %q want %q", got.Source, SourceMtimeExifUnusable)
	}
	if got.UTC.Unix() != mt.Unix() {
		t.Fatalf("UTC unix: got %d want %d", got.UTC.Unix(), mt.Unix())
	}
}

func TestReadCapture_noExifUsesMtime(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "plain.jpg")
	if err := writeMinimalJPEG(p); err != nil {
		t.Fatal(err)
	}
	mt := time.Date(2021, 3, 15, 10, 30, 45, 0, time.UTC)
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}

	got, err := ReadCapture(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != SourceMtimeNoExif {
		t.Fatalf("Source: got %q want %q", got.Source, SourceMtimeNoExif)
	}
	// ModTime resolution is filesystem-dependent; compare truncated to seconds.
	if got.UTC.Unix() != mt.Unix() {
		t.Fatalf("UTC unix: got %d want %d (%v vs %v)", got.UTC.Unix(), mt.Unix(), got.UTC, mt)
	}
}

func TestCaptureFromExifBytes_parseErrorReturnsFalse(t *testing.T) {
	_, ok, err := captureFromExifBytes([]byte{0, 1, 2, 3})
	if err == nil {
		t.Fatal("expected parse error from Collect")
	}
	if ok {
		t.Fatal("expected ok false")
	}
}

// mustBuildExifWithoutDateTime encodes a valid EXIF blob (IFD0 only) with no DateTimeOriginal / DateTime,
// so capture falls through to filesystem mtime with SourceMtimeExifUnusable.
func mustBuildExifWithoutDateTime(t *testing.T) []byte {
	t.Helper()
	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		t.Fatal(err)
	}
	ti := exif.NewTagIndex()
	if err := exif.LoadStandardTags(ti); err != nil {
		t.Fatal(err)
	}
	rootIb := exif.NewIfdBuilder(im, ti, exifcommon.IfdStandardIfdIdentity, exifcommon.EncodeDefaultByteOrder)
	if err := rootIb.SetStandardWithName("Make", "photo-tool-test"); err != nil {
		t.Fatal(err)
	}
	ibe := exif.NewIfdByteEncoder()
	blob, err := ibe.EncodeToExif(rootIb)
	if err != nil {
		t.Fatal(err)
	}
	return blob
}

func mustBuildExifWithDateTimeOriginal(t *testing.T, dateTimeOriginal string) []byte {
	t.Helper()
	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		t.Fatal(err)
	}
	ti := exif.NewTagIndex()
	if err := exif.LoadStandardTags(ti); err != nil {
		t.Fatal(err)
	}
	rootIb := exif.NewIfdBuilder(im, ti, exifcommon.IfdStandardIfdIdentity, exifcommon.EncodeDefaultByteOrder)
	exifIb, err := exif.GetOrCreateIbFromRootIb(rootIb, "IFD/Exif")
	if err != nil {
		t.Fatal(err)
	}
	if err := exifIb.SetStandardWithName("DateTimeOriginal", dateTimeOriginal); err != nil {
		t.Fatal(err)
	}
	ibe := exif.NewIfdByteEncoder()
	blob, err := ibe.EncodeToExif(rootIb)
	if err != nil {
		t.Fatal(err)
	}
	return blob
}

func writeMinimalJPEG(path string) error {
	img := image.NewGray(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.Gray{Y: 0x80})
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, &jpeg.Options{Quality: 80})
}
