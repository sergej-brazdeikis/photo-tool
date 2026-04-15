package exifmeta

import (
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
)

func TestReadCapture_extractExifErrorWraps(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod does not reliably deny read for this probe on Windows")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "unreadable.jpg")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(p, 0o644) })

	_, err := ReadCapture(p)
	if err == nil {
		t.Fatal("expected error for unreadable file")
	}
	if !strings.Contains(err.Error(), "exifmeta: extract exif") {
		t.Fatalf("want extract wrap prefix, got %v", err)
	}
	if errors.Unwrap(err) == nil {
		t.Fatalf("expected fmt.Errorf wrap chain, got %v", err)
	}
}

func TestReadCapture_exifIFD0DateTime(t *testing.T) {
	const wantLocal = "2020:06:01 08:09:10"
	exifBytes := mustBuildExifWithIFD0DateTime(t, wantLocal)
	dir := t.TempDir()
	p := filepath.Join(dir, "ifd0_dt.jpg")
	if err := os.WriteFile(p, exifBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ReadCapture(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != SourceExifIFD0DateTime {
		t.Fatalf("Source: got %q want %q", got.Source, SourceExifIFD0DateTime)
	}
	wantUTC, err := time.ParseInLocation(exifDateTimeLayout, wantLocal, time.Local)
	if err != nil {
		t.Fatal(err)
	}
	if !got.UTC.Equal(wantUTC.UTC()) {
		t.Fatalf("UTC: got %v want %v", got.UTC, wantUTC.UTC())
	}
}

func TestReadCapture_exifIFD0DateTimeOriginal(t *testing.T) {
	const wantLocal = "2018:11:30 01:02:03"
	exifBytes := mustBuildExifWithIFD0DateTimeOriginal(t, wantLocal)
	dir := t.TempDir()
	p := filepath.Join(dir, "ifd0_dto.jpg")
	if err := os.WriteFile(p, exifBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ReadCapture(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != SourceExifIFD0DateTimeOriginal {
		t.Fatalf("Source: got %q want %q", got.Source, SourceExifIFD0DateTimeOriginal)
	}
	wantUTC, err := time.ParseInLocation(exifDateTimeLayout, wantLocal, time.Local)
	if err != nil {
		t.Fatal(err)
	}
	if !got.UTC.Equal(wantUTC.UTC()) {
		t.Fatalf("UTC: got %v want %v", got.UTC, wantUTC.UTC())
	}
}

func TestReadCapture_prefersExifSubifdDateTimeOriginalOverIFD0DateTime(t *testing.T) {
	const exifDTO = "2019:05:17 12:34:56"
	const ifd0DT = "2021:01:01 00:00:00"
	exifBytes := mustBuildExifWithExifDTOAndIFD0DateTime(t, exifDTO, ifd0DT)
	dir := t.TempDir()
	p := filepath.Join(dir, "precedence.jpg")
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
	wantUTC, err := time.ParseInLocation(exifDateTimeLayout, exifDTO, time.Local)
	if err != nil {
		t.Fatal(err)
	}
	if !got.UTC.Equal(wantUTC.UTC()) {
		t.Fatalf("UTC: got %v want %v (IFD0 DateTime must not win)", got.UTC, wantUTC.UTC())
	}
}

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

func TestReadCapture_malformedDateTimeOriginalFallsBackToMtime(t *testing.T) {
	// Present-but-unparseable EXIF datetime must not win over mtime (AC2 non-silent fallback).
	exifBytes := mustBuildExifWithDateTimeOriginal(t, "not-a-valid-exif-datetime")
	dir := t.TempDir()
	p := filepath.Join(dir, "bad_dt.jpg")
	if err := os.WriteFile(p, exifBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	mt := time.Date(2023, 12, 25, 15, 4, 5, 0, time.UTC)
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

func mustBuildExifWithIFD0DateTime(t *testing.T, dateTime string) []byte {
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
	if err := rootIb.SetStandardWithName("DateTime", dateTime); err != nil {
		t.Fatal(err)
	}
	ibe := exif.NewIfdByteEncoder()
	blob, err := ibe.EncodeToExif(rootIb)
	if err != nil {
		t.Fatal(err)
	}
	return blob
}

func mustBuildExifWithIFD0DateTimeOriginal(t *testing.T, dateTimeOriginal string) []byte {
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
	if err := rootIb.SetStandardWithName("DateTimeOriginal", dateTimeOriginal); err != nil {
		t.Fatal(err)
	}
	ibe := exif.NewIfdByteEncoder()
	blob, err := ibe.EncodeToExif(rootIb)
	if err != nil {
		t.Fatal(err)
	}
	return blob
}

func mustBuildExifWithExifDTOAndIFD0DateTime(t *testing.T, exifDateTimeOriginal, ifd0DateTime string) []byte {
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
	if err := rootIb.SetStandardWithName("DateTime", ifd0DateTime); err != nil {
		t.Fatal(err)
	}
	exifIb, err := exif.GetOrCreateIbFromRootIb(rootIb, "IFD/Exif")
	if err != nil {
		t.Fatal(err)
	}
	if err := exifIb.SetStandardWithName("DateTimeOriginal", exifDateTimeOriginal); err != nil {
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
