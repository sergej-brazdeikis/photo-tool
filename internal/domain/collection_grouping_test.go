package domain

import "testing"

func TestParseCollectionGrouping(t *testing.T) {
	g, err := ParseCollectionGrouping("stars")
	if err != nil || g != CollectionGroupStars {
		t.Fatalf("%v %v", g, err)
	}
	g, err = ParseCollectionGrouping("DAY")
	if err != nil || g != CollectionGroupDay {
		t.Fatalf("%v %v", g, err)
	}
	g, err = ParseCollectionGrouping("camera")
	if err != nil || g != CollectionGroupCamera {
		t.Fatalf("%v %v", g, err)
	}
	if _, err := ParseCollectionGrouping("nope"); err == nil {
		t.Fatal("expected error")
	}
}
