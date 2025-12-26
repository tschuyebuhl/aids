package data

import "testing"

func TestSlugifyBasic(t *testing.T) {
	got := Slugify("Hello, World!")
	if got != "hello-world" {
		t.Fatalf("expected hello-world, got %q", got)
	}
}

func TestSlugifyDiacritics(t *testing.T) {
	got := Slugify("Zażółć gęślą jaźń")
	if got != "zazoc-gesla-jazn" {
		t.Fatalf("expected zazoc-gesla-jazn, got %q", got)
	}
}
