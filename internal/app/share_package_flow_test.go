package app

import "testing"

func TestPackageSharePreviewMandatoryBeforeMint(t *testing.T) {
	t.Parallel()
	if !PackageSharePreviewMandatoryBeforeMint() {
		t.Fatal("AC2: manifest preview must stay mandatory before package mint")
	}
}

func TestAudiencePresetShareMetadata_nonSecurity(t *testing.T) {
	t.Parallel()
	dt, al := audiencePresetShareMetadata("Close friends")
	if dt == "" || al == "" {
		t.Fatalf("preset metadata: %q %q", dt, al)
	}
	dt2, al2 := audiencePresetShareMetadata("No preset")
	if dt2 != "" || al2 != "" {
		t.Fatalf("No preset should not set title: %q %q", dt2, al2)
	}
}
