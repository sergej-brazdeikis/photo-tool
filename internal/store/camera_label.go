package store

import (
	"database/sql"
	"regexp"
	"strings"
)

// UnknownCameraLabel is the stable UI/grouping label when camera metadata is missing (Story 2.8 AC5).
const UnknownCameraLabel = "Unknown camera"

var spaceCollapse = regexp.MustCompile(`\s+`)

// NormalizeCameraField trims and collapses internal whitespace to single ASCII spaces.
func NormalizeCameraField(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	return spaceCollapse.ReplaceAllString(s, " ")
}

// CameraLabelFromParts builds the persisted/display grouping key from nullable parts (Story 2.8 AC5).
// Both empty after normalization → ("", false) so the store can persist NULL camera_label (unknown bucket).
// One field only when the other is empty; both present → single-space join. Whitespace-only fields count as empty.
func CameraLabelFromParts(makeStr, modelStr string) (label string, ok bool) {
	m := NormalizeCameraField(makeStr)
	o := NormalizeCameraField(modelStr)
	switch {
	case m == "" && o == "":
		return "", false
	case m == "":
		return o, true
	case o == "":
		return m, true
	default:
		return m + " " + o, true
	}
}

// CameraLabelForStorage returns a sql.NullString for the camera_label column (NULL when unknown).
func CameraLabelForStorage(makeStr, modelStr string) sql.NullString {
	if lbl, ok := CameraLabelFromParts(makeStr, modelStr); ok {
		return sql.NullString{String: lbl, Valid: true}
	}
	return sql.NullString{}
}

// ScanStringPtr returns a *string for INSERT when the value should be NULL if empty.
func ScanStringPtr(ns sql.NullString) any {
	if !ns.Valid || strings.TrimSpace(ns.String) == "" {
		return nil
	}
	s := ns.String
	return &s
}
