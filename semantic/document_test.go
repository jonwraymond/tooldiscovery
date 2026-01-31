package semantic

import "testing"

func TestDocument_NormalizeText(t *testing.T) {
	doc := Document{
		ID:          "tool-1",
		Name:        "Search",
		Description: "Find items",
		Tags:        []string{"B", "a"},
	}

	norm := doc.Normalized()
	want := "Search Find items a b"
	if norm.Text != want {
		t.Fatalf("normalized text = %q, want %q", norm.Text, want)
	}
}

func TestDocument_TagsSorted(t *testing.T) {
	doc := Document{
		ID:   "tool-1",
		Tags: []string{"Zeta", "alpha", "Beta"},
	}

	norm := doc.Normalized()
	want := []string{"alpha", "beta", "zeta"}
	if len(norm.Tags) != len(want) {
		t.Fatalf("normalized tags length = %d, want %d", len(norm.Tags), len(want))
	}
	for i := range want {
		if norm.Tags[i] != want[i] {
			t.Fatalf("normalized tags[%d] = %q, want %q", i, norm.Tags[i], want[i])
		}
	}
}
