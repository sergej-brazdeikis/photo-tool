// Package exifmeta reads capture-time metadata from images for ingest layout (Story 1.2, FR-01).
//
// MVP: JPEG and other formats where dsoprea/go-exif can locate a TIFF/EXIF blob in the file
// (typically JPEG APP1). TIFF-only files are not explicitly validated here; callers may extend.
//
// Timezone rule (used with paths.CanonicalDayDir): EXIF ASCII date/time tags (DateTimeOriginal,
// DateTime) do not include a time zone. Values are parsed as local wall time (time.Local), then
// converted to UTC for the returned instant. MVP does not read OffsetTimeOriginal, sub-second
// fields, DateTimeDigitized, or CreateDate; cameras that only encode offset-aware or alternate
// tags may fall through to filesystem mtime (see [Result].Source).
//
// Fallback chain (each path is explicit in [Result].Source — never silent):
//  1. Exif sub-IFD — DateTimeOriginal
//  2. IFD0 — DateTimeOriginal
//  3. IFD0 — DateTime
//  4. Filesystem modification time — when there is no EXIF blob, EXIF cannot be parsed, no
//     usable datetime tag / value is present, or a tag value is present but does not parse as
//     "2006:01:02 15:04:05". When EXIF bytes exist but internal decode fails, MVP still prefers
//     mtime for placement (SourceMtimeExifUnusable) rather than surfacing the decode error; see
//     ingest logging stories for stricter observability.
//
// Content hashing is not implemented here; use [photo-tool/internal/filehash.SumHex] (or
// [photo-tool/internal/filehash.ReaderHex] with a single [os.File] if you need one open).
package exifmeta

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
)

const exifDateTimeLayout = "2006:01:02 15:04:05"

// Source describes where [Result.UTC] came from (fallback is explicit, never silent).
type Source string

const (
	SourceExifDateTimeOriginal     Source = "exif:Exif/DateTimeOriginal"
	SourceExifIFD0DateTimeOriginal Source = "exif:IFD0/DateTimeOriginal"
	SourceExifIFD0DateTime         Source = "exif:IFD0/DateTime"
	SourceMtimeNoExif              Source = "filesystem:mtime(no_exif_blob)"
	SourceMtimeExifUnusable        Source = "filesystem:mtime(exif_unusable)"
)

// Result is a capture instant for storage layout (UTC) and its provenance.
type Result struct {
	UTC    time.Time
	Source Source
}

// ReadCapture returns the capture instant for path using the documented EXIF → mtime fallback chain.
// Errors from EXIF extraction (other than [exif.ErrNoExif]) and from Stat on mtime fallback include
// path context and unwrap to the underlying failure ([fmt.Errorf] with %w).
func ReadCapture(path string) (Result, error) {
	raw, err := exif.SearchFileAndExtractExif(path)
	if err != nil {
		if errors.Is(err, exif.ErrNoExif) {
			return mtimeResult(path, SourceMtimeNoExif)
		}
		return Result{}, fmt.Errorf("exifmeta: extract exif %q: %w", path, err)
	}

	res, ok, err := captureFromExifBytes(raw)
	if err != nil {
		return mtimeResult(path, SourceMtimeExifUnusable)
	}
	if !ok {
		return mtimeResult(path, SourceMtimeExifUnusable)
	}
	return res, nil
}

func mtimeResult(path string, src Source) (Result, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return Result{}, fmt.Errorf("exifmeta: stat %q: %w", path, err)
	}
	return Result{UTC: fi.ModTime().UTC(), Source: src}, nil
}

func captureFromExifBytes(raw []byte) (Result, bool, error) {
	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		return Result{}, false, err
	}
	ti := exif.NewTagIndex()
	if err := exif.LoadStandardTags(ti); err != nil {
		return Result{}, false, err
	}
	_, index, err := exif.Collect(im, ti, raw)
	if err != nil {
		return Result{}, false, err
	}
	s, src, ok := firstExifDateTime(index)
	if !ok {
		return Result{}, false, nil
	}
	local, err := time.ParseInLocation(exifDateTimeLayout, s, time.Local)
	if err != nil {
		return Result{}, false, nil
	}
	return Result{UTC: local.UTC(), Source: src}, true, nil
}

func firstExifDateTime(index exif.IfdIndex) (value string, src Source, ok bool) {
	type step struct {
		ifdKey string
		tag    string
		src    Source
	}
	// Order matches package fallback documentation.
	steps := []step{
		{exifcommon.IfdExifStandardIfdIdentity.String(), "DateTimeOriginal", SourceExifDateTimeOriginal},
		{exifcommon.IfdStandardIfdIdentity.String(), "DateTimeOriginal", SourceExifIFD0DateTimeOriginal},
		{exifcommon.IfdStandardIfdIdentity.String(), "DateTime", SourceExifIFD0DateTime},
	}
	for _, st := range steps {
		ifd := index.Lookup[st.ifdKey]
		if ifd == nil {
			continue
		}
		results, err := ifd.FindTagWithName(st.tag)
		if err != nil || len(results) == 0 {
			continue
		}
		phrase, err := results[0].FormatFirst()
		if err != nil || phrase == "" {
			continue
		}
		return phrase, st.src, true
	}
	return "", "", false
}
