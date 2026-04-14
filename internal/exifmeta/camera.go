package exifmeta

import (
	"errors"
	"strings"

	"github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
)

// CameraStrings holds optional EXIF Make/Model (trimmed ASCII; empty if absent).
type CameraStrings struct {
	Make  string
	Model string
}

// ReadCamera reads Make and Model when an EXIF blob exists (Story 2.8).
// When EXIF is missing or tags are absent, returns empty strings (callers store NULL).
func ReadCamera(path string) (CameraStrings, error) {
	raw, err := exif.SearchFileAndExtractExif(path)
	if err != nil {
		if errors.Is(err, exif.ErrNoExif) {
			return CameraStrings{}, nil
		}
		return CameraStrings{}, err
	}
	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		return CameraStrings{}, err
	}
	ti := exif.NewTagIndex()
	if err := exif.LoadStandardTags(ti); err != nil {
		return CameraStrings{}, err
	}
	_, index, err := exif.Collect(im, ti, raw)
	if err != nil {
		return CameraStrings{}, err
	}
	makeStr := firstTagPhrase(index, exifcommon.IfdStandardIfdIdentity.String(), "Make")
	modelStr := firstTagPhrase(index, exifcommon.IfdStandardIfdIdentity.String(), "Model")
	return CameraStrings{Make: strings.TrimSpace(makeStr), Model: strings.TrimSpace(modelStr)}, nil
}

func firstTagPhrase(index exif.IfdIndex, ifdKey, tagName string) string {
	ifd := index.Lookup[ifdKey]
	if ifd == nil {
		return ""
	}
	results, err := ifd.FindTagWithName(tagName)
	if err != nil || len(results) == 0 {
		return ""
	}
	phrase, err := results[0].FormatFirst()
	if err != nil {
		return ""
	}
	return phrase
}
