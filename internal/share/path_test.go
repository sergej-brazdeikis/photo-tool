package share

import "testing"

func TestShareHTTPPath(t *testing.T) {
	t.Parallel()
	if got := ShareHTTPPath("abcXYZ-_"); got != "/s/abcXYZ-_" {
		t.Fatalf("got %q", got)
	}
}

func TestShareImageHTTPPath(t *testing.T) {
	t.Parallel()
	if got := ShareImageHTTPPath("abcXYZ-_"); got != "/i/abcXYZ-_" {
		t.Fatalf("got %q", got)
	}
}

func TestSharePackageMemberImagePath(t *testing.T) {
	t.Parallel()
	if got := SharePackageMemberImagePath("tok", 3); got != "/i/tok/3" {
		t.Fatalf("got %q", got)
	}
	if got := SharePackageMemberImagePath("tok", -1); got != "/i/tok/0" {
		t.Fatalf("negative pos: got %q", got)
	}
}
