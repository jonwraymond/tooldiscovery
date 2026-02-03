package provider

import (
	"testing"

	"github.com/jonwraymond/toolfoundation/adapter"
)

func TestProviderID(t *testing.T) {
	if got := ProviderID("", "1.0.0"); got != "" {
		t.Errorf("ProviderID empty name = %q, want empty", got)
	}
	if got := ProviderID("agent", ""); got != "agent" {
		t.Errorf("ProviderID without version = %q, want agent", got)
	}
	if got := ProviderID("agent", "1.0.0"); got != "agent:1.0.0" {
		t.Errorf("ProviderID = %q, want agent:1.0.0", got)
	}
}

func TestInMemoryStore_RegisterDescribe(t *testing.T) {
	store := NewInMemoryStore()

	id, err := store.RegisterProvider("", testProvider("Agent", "1.0.0"))
	if err != nil {
		t.Fatalf("RegisterProvider error = %v", err)
	}
	if id != "Agent:1.0.0" {
		t.Errorf("resolved id = %q, want Agent:1.0.0", id)
	}

	got, err := store.DescribeProvider(id)
	if err != nil {
		t.Fatalf("DescribeProvider error = %v", err)
	}
	if got.Name != "Agent" {
		t.Errorf("Name = %q, want Agent", got.Name)
	}
}

func TestInMemoryStore_ListProviders(t *testing.T) {
	store := NewInMemoryStore()

	_, _ = store.RegisterProvider("", testProvider("Beta", "1.0.0"))
	_, _ = store.RegisterProvider("", testProvider("Alpha", "2.0.0"))

	list, err := store.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders error = %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("ListProviders length = %d, want 2", len(list))
	}
	if list[0].Name != "Alpha" {
		t.Errorf("sorted first provider = %q, want Alpha", list[0].Name)
	}
}

func TestInMemoryStore_Describe_NotFound(t *testing.T) {
	store := NewInMemoryStore()

	if _, err := store.DescribeProvider("missing"); err == nil {
		t.Error("expected error for missing provider")
	}
}

func testProvider(name, version string) adapter.CanonicalProvider {
	return adapter.CanonicalProvider{
		Name:        name,
		Description: "desc",
		Version:     version,
	}
}
