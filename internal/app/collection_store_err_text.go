package app

import (
	"errors"
	"strings"

	"photo-tool/internal/store"
)

// collectionStoreErrText turns store/DB failures into short copy for inline album forms
// and loupe dialogs. Validation messages from [store] pass through unchanged.
func collectionStoreErrText(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, store.ErrCollectionNotFound) {
		return "This album is no longer in the library. Refresh and try again."
	}
	s := err.Error()
	if strings.Contains(s, "FOREIGN KEY") {
		return "This photo or album is no longer in the library. Refresh Review and try again."
	}
	return s
}

// libraryErrText maps store/read failures to factual copy with an implied next step (Story 2.12 AC5).
func libraryErrText(err error) string {
	if err == nil {
		return ""
	}
	if s := collectionStoreErrText(err); s != err.Error() {
		return s
	}
	return "Could not read the library. Check that the library folder is available, then try again."
}

// userFacingCollectionWriteErrText maps create/update/link/unlink collection failures shown in album
// dialogs and inline forms (Story 2.12 AC5). It keeps store validation lines like
// "create collection: name is required" but avoids surfacing raw SQLite for other failures.
func userFacingCollectionWriteErrText(err error) string {
	if err == nil {
		return ""
	}
	if mapped := collectionStoreErrText(err); mapped != err.Error() {
		return mapped
	}
	plain := err.Error()
	low := strings.ToLower(plain)
	if strings.Contains(low, "sqlite") || strings.Contains(low, "constraint failed") ||
		strings.Contains(low, "no such table") || strings.Contains(low, "sql logic error") {
		return "Could not update the library. Check that the library folder is available, then try again."
	}
	if strings.HasPrefix(plain, "create collection:") || strings.HasPrefix(plain, "update collection:") {
		return plain
	}
	return userFacingDialogErrText(err)
}

// userFacingDialogErrText maps failures shown in dialog.ShowError to short factual copy with a next step.
// Avoid surfacing raw SQLite/driver strings (Story 2.12 AC5).
func userFacingDialogErrText(err error) string {
	if err == nil {
		return ""
	}
	if mapped := collectionStoreErrText(err); mapped != err.Error() {
		return mapped
	}
	low := strings.ToLower(err.Error())
	if strings.Contains(low, "database is locked") || strings.Contains(low, "sqlite_busy") ||
		(strings.Contains(low, "locked") && strings.Contains(low, "sqlite")) {
		return "The library database is busy. Wait a moment, close other copies of this app if any are open, then try again."
	}
	if strings.Contains(low, "disk i/o") || strings.Contains(low, "sqlite_full") || strings.Contains(low, "no space left") {
		return "The library disk may be full or unreadable. Free disk space, check the library folder, then try again."
	}
	if strings.Contains(low, "delete mkdir quarantine") || strings.Contains(low, "delete quarantine rename") ||
		strings.Contains(low, "delete stat source") || strings.Contains(low, "delete asset update") {
		return "Could not move the photo into library trash (.trash). Check disk space and that the library folder is writable, then try again."
	}
	if strings.Contains(low, "delete asset path") || strings.Contains(low, "asset path escapes") || strings.Contains(low, "asset path resolves to library root") {
		return "Could not resolve the photo's path in your library. Return to Review and refresh; if it continues, the database may need repair."
	}
	if strings.Contains(low, "permission denied") {
		return "Permission denied. Check that the library folder and files are readable and writable, then try again."
	}
	return "Could not update the library. Check that the library folder is available, then try again."
}

// userFacingFileOpenErrText is for file-picker failures on the Upload surface (Story 2.12 AC5).
func userFacingFileOpenErrText(err error) string {
	if err == nil {
		return ""
	}
	return "Could not open the selected file. Check permissions and try again, or pick a different file."
}
