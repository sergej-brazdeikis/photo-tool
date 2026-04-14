package main

import (
	"strings"
	"testing"

	ptapp "photo-tool/internal/app"
)

func TestAppFyneIDConstant(t *testing.T) {
	if ptapp.FyneAppID == "" {
		t.Fatal("FyneAppID must be set — Fyne preferences require app.NewWithID")
	}
	if !strings.Contains(ptapp.FyneAppID, ".") {
		t.Fatalf("FyneAppID should use reverse-DNS form, got %q", ptapp.FyneAppID)
	}
}
