package app

import (
	"errors"
	"strings"
	"testing"

	"photo-tool/internal/store"
)

func TestUserFacingDialogErrText(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil", err: nil, want: ""},
		{
			name: "collection not found",
			err:  store.ErrCollectionNotFound,
			want: "This album is no longer in the library. Refresh and try again.",
		},
		{
			name: "foreign key",
			err:  errors.New("FOREIGN KEY constraint failed"),
			want: "This photo or album is no longer in the library. Refresh Review and try again.",
		},
		{
			name: "sqlite locked",
			err:  errors.New("database is locked (5)"),
			want: "The library database is busy. Wait a moment, close other copies of this app if any are open, then try again.",
		},
		{
			name: "sqlite busy token",
			err:  errors.New("sqlite_busy"),
			want: "The library database is busy. Wait a moment, close other copies of this app if any are open, then try again.",
		},
		{
			name: "delete quarantine",
			err:  errors.New("delete quarantine rename: permission denied"),
			want: "Could not move the photo into library trash (.trash). Check disk space and that the library folder is writable, then try again.",
		},
		{
			name: "permission denied generic",
			err:  errors.New("open /photos/foo: permission denied"),
			want: "Permission denied. Check that the library folder and files are readable and writable, then try again.",
		},
		{
			name: "unknown",
			err:  errors.New("some opaque failure"),
			want: "Could not update the library. Check that the library folder is available, then try again.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := userFacingDialogErrText(tt.err)
			if got != tt.want {
				t.Fatalf("userFacingDialogErrText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUserFacingCollectionWriteErrText(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil", err: nil, want: ""},
		{
			name: "validation passes through",
			err:  errors.New("create collection: name is required"),
			want: "create collection: name is required",
		},
		{
			name: "unique constraint not raw",
			err:  errors.New("UNIQUE constraint failed: collections.name"),
			want: "Could not update the library. Check that the library folder is available, then try again.",
		},
		{
			name: "foreign key uses collection mapping",
			err:  errors.New("FOREIGN KEY constraint failed"),
			want: "This photo or album is no longer in the library. Refresh Review and try again.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := userFacingCollectionWriteErrText(tt.err)
			if got != tt.want {
				t.Fatalf("userFacingCollectionWriteErrText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUserFacingFileOpenErrText(t *testing.T) {
	if got := userFacingFileOpenErrText(nil); got != "" {
		t.Fatalf("nil: got %q", got)
	}
	got := userFacingFileOpenErrText(errors.New("boom"))
	if !strings.Contains(got, "Could not open") {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestLibraryErrText_unknownRead(t *testing.T) {
	got := libraryErrText(errors.New("SELECT: no such table: assets"))
	if got != "Could not read the library. Check that the library folder is available, then try again." {
		t.Fatalf("got %q", got)
	}
}
