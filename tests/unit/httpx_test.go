package unit_test

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"testing"
)

func TestPagination(t *testing.T) {
	p := httpx.Pagination(2, 10, 21)
	if p.CurrentPage != 2 || p.TotalPages != 3 || p.TotalItems != 21 {
		t.Fatalf("unexpected paging: %+v", p)
	}
}
func TestMessageParity(t *testing.T) {
	got := httpx.Message("create", "Role")
	want := "🎉 Role created successfully! Welcome aboard! 🚀"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
func TestUUID(t *testing.T) {
	if httpx.UUID("bad-id") == nil {
		t.Fatal("expected UUID validation error")
	}
	if err := httpx.UUID("00000000-0000-4000-8000-000000000000"); err != nil {
		t.Fatal(err)
	}
}
