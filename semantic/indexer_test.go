package semantic

import (
	"context"
	"testing"
)

func TestIndexer_AddUpdateRemove(t *testing.T) {
	idx := NewInMemoryIndex()
	ctx := context.Background()

	doc := Document{ID: "tool-1", Name: "Search", Description: "find"}
	if err := idx.Add(ctx, doc); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	got, ok := idx.Get(ctx, "tool-1")
	if !ok {
		t.Fatalf("expected doc to exist")
	}
	if got.Name != "Search" {
		t.Fatalf("expected Name=Search, got %q", got.Name)
	}

	updated := Document{ID: "tool-1", Name: "SearchV2", Description: "find"}
	if err := idx.Update(ctx, updated); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, ok = idx.Get(ctx, "tool-1")
	if !ok {
		t.Fatalf("expected doc to exist after update")
	}
	if got.Name != "SearchV2" {
		t.Fatalf("expected Name=SearchV2, got %q", got.Name)
	}

	if err := idx.Remove(ctx, "tool-1"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if _, ok := idx.Get(ctx, "tool-1"); ok {
		t.Fatalf("expected doc to be removed")
	}
}

func TestIndexer_DedupByID(t *testing.T) {
	idx := NewInMemoryIndex()
	ctx := context.Background()

	_ = idx.Add(ctx, Document{ID: "tool-1", Name: "First"})
	_ = idx.Add(ctx, Document{ID: "tool-1", Name: "Second"})

	docs := idx.List(ctx)
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	if docs[0].Name != "Second" {
		t.Fatalf("expected Name=Second, got %q", docs[0].Name)
	}
}

func TestIndexerContract_InvalidID(t *testing.T) {
	idx := NewInMemoryIndex()
	ctx := context.Background()

	if err := idx.Add(ctx, Document{}); err == nil {
		t.Fatalf("expected error for empty ID")
	}
	if err := idx.Remove(ctx, ""); err == nil {
		t.Fatalf("expected error for empty ID on remove")
	}
}
