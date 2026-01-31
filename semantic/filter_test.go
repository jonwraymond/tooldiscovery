package semantic

import "testing"

func TestFilter_Namespace(t *testing.T) {
	docs := []Document{
		{ID: "a", Namespace: "alpha"},
		{ID: "b", Namespace: "beta"},
		{ID: "c", Namespace: "alpha"},
	}

	filtered := FilterByNamespace(docs, "alpha")
	if len(filtered) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(filtered))
	}
	if filtered[0].ID != "a" || filtered[1].ID != "c" {
		t.Fatalf("unexpected order or IDs: %+v", filtered)
	}
}

func TestFilter_Tags(t *testing.T) {
	docs := []Document{
		{ID: "a", Tags: []string{"read", "safe"}},
		{ID: "b", Tags: []string{"write"}},
		{ID: "c", Tags: []string{"safe"}},
	}

	filtered := FilterByTags(docs, []string{"safe"})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(filtered))
	}
	if filtered[0].ID != "a" || filtered[1].ID != "c" {
		t.Fatalf("unexpected order or IDs: %+v", filtered)
	}
}

func TestFilter_Category(t *testing.T) {
	docs := []Document{
		{ID: "a", Category: "docs"},
		{ID: "b", Category: "ops"},
		{ID: "c", Category: "docs"},
	}

	filtered := FilterByCategory(docs, "docs")
	if len(filtered) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(filtered))
	}
	if filtered[0].ID != "a" || filtered[1].ID != "c" {
		t.Fatalf("unexpected order or IDs: %+v", filtered)
	}
}
