package semantic

import (
	"reflect"
	"testing"

	"github.com/jonwraymond/tooldiscovery/index"
)

func TestDocumentFromSearchDoc(t *testing.T) {
	tests := []struct {
		name string
		doc  index.SearchDoc
		want Document
	}{
		{
			name: "full conversion",
			doc: index.SearchDoc{
				ID:      "github:create-issue",
				DocText: "create-issue github create issue bug tracker",
				Summary: index.Summary{
					ID:               "github:create-issue",
					Name:             "create-issue",
					Namespace:        "github",
					ShortDescription: "Create a new issue in a GitHub repository",
					Tags:             []string{"github", "issue", "tracker"},
				},
			},
			want: Document{
				ID:          "github:create-issue",
				Name:        "create-issue",
				Namespace:   "github",
				Description: "Create a new issue in a GitHub repository",
				Tags:        []string{"github", "issue", "tracker"},
				Category:    "",
				Text:        "create-issue github create issue bug tracker",
			},
		},
		{
			name: "empty tags",
			doc: index.SearchDoc{
				ID:      "simple-tool",
				DocText: "simple tool description",
				Summary: index.Summary{
					ID:               "simple-tool",
					Name:             "simple-tool",
					ShortDescription: "A simple tool",
					Tags:             nil,
				},
			},
			want: Document{
				ID:          "simple-tool",
				Name:        "simple-tool",
				Namespace:   "",
				Description: "A simple tool",
				Tags:        nil,
				Category:    "",
				Text:        "simple tool description",
			},
		},
		{
			name: "empty document",
			doc:  index.SearchDoc{},
			want: Document{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DocumentFromSearchDoc(tt.doc)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DocumentFromSearchDoc() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDocumentFromSearchDoc_TagsCopied(t *testing.T) {
	original := index.SearchDoc{
		ID: "test",
		Summary: index.Summary{
			ID:   "test",
			Tags: []string{"a", "b"},
		},
	}

	doc := DocumentFromSearchDoc(original)

	// Modify original tags
	original.Summary.Tags[0] = "modified"

	// Document tags should be unchanged
	if doc.Tags[0] != "a" {
		t.Errorf("Tags not copied: got %v, want [a b]", doc.Tags)
	}
}

func TestDocumentsFromSearchDocs(t *testing.T) {
	tests := []struct {
		name string
		docs []index.SearchDoc
		want []Document
	}{
		{
			name: "multiple docs",
			docs: []index.SearchDoc{
				{
					ID:      "tool1",
					DocText: "tool one",
					Summary: index.Summary{ID: "tool1", Name: "tool1"},
				},
				{
					ID:      "tool2",
					DocText: "tool two",
					Summary: index.Summary{ID: "tool2", Name: "tool2"},
				},
			},
			want: []Document{
				{ID: "tool1", Name: "tool1", Text: "tool one"},
				{ID: "tool2", Name: "tool2", Text: "tool two"},
			},
		},
		{
			name: "nil input",
			docs: nil,
			want: nil,
		},
		{
			name: "empty slice",
			docs: []index.SearchDoc{},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DocumentsFromSearchDocs(tt.docs)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DocumentsFromSearchDocs() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSearchDocFromDocument(t *testing.T) {
	tests := []struct {
		name string
		doc  Document
		want index.SearchDoc
	}{
		{
			name: "full conversion",
			doc: Document{
				ID:          "github:create-issue",
				Name:        "create-issue",
				Namespace:   "github",
				Description: "Create a new issue",
				Tags:        []string{"github", "issue"},
				Category:    "vcs",
				Text:        "create-issue github create a new issue github issue",
			},
			want: index.SearchDoc{
				ID:      "github:create-issue",
				DocText: "create-issue github create a new issue github issue",
				Summary: index.Summary{
					ID:               "github:create-issue",
					Name:             "create-issue",
					Namespace:        "github",
					ShortDescription: "Create a new issue",
					Tags:             []string{"github", "issue"},
				},
			},
		},
		{
			name: "rebuilds text when empty",
			doc: Document{
				ID:          "test",
				Name:        "test-tool",
				Description: "A test tool",
				Tags:        []string{"test"},
				Text:        "", // Empty text should be rebuilt
			},
			want: index.SearchDoc{
				ID:      "test",
				DocText: "test-tool A test tool test", // Rebuilt from fields
				Summary: index.Summary{
					ID:               "test",
					Name:             "test-tool",
					ShortDescription: "A test tool",
					Tags:             []string{"test"},
				},
			},
		},
		{
			name: "truncates long description",
			doc: Document{
				ID:          "test",
				Name:        "test",
				Description: "This is a very long description that exceeds the maximum length allowed for short descriptions which is 120 characters and should be truncated",
				Text:        "some text",
			},
			want: index.SearchDoc{
				ID:      "test",
				DocText: "some text",
				Summary: index.Summary{
					ID:               "test",
					Name:             "test",
					ShortDescription: "This is a very long description that exceeds the maximum length allowed for short descriptions which is 120 characters a",
					Tags:             nil,
				},
			},
		},
		{
			name: "empty document",
			doc:  Document{},
			want: index.SearchDoc{
				Summary: index.Summary{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SearchDocFromDocument(tt.doc)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SearchDocFromDocument() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSearchDocFromDocument_TagsCopied(t *testing.T) {
	original := Document{
		ID:   "test",
		Tags: []string{"a", "b"},
		Text: "text",
	}

	searchDoc := SearchDocFromDocument(original)

	// Modify original tags
	original.Tags[0] = "modified"

	// SearchDoc tags should be unchanged
	if searchDoc.Summary.Tags[0] != "a" {
		t.Errorf("Tags not copied: got %v, want [a b]", searchDoc.Summary.Tags)
	}
}

func TestSearchDocsFromDocuments(t *testing.T) {
	tests := []struct {
		name string
		docs []Document
		want []index.SearchDoc
	}{
		{
			name: "multiple docs",
			docs: []Document{
				{ID: "tool1", Name: "tool1", Text: "text1"},
				{ID: "tool2", Name: "tool2", Text: "text2"},
			},
			want: []index.SearchDoc{
				{
					ID:      "tool1",
					DocText: "text1",
					Summary: index.Summary{ID: "tool1", Name: "tool1"},
				},
				{
					ID:      "tool2",
					DocText: "text2",
					Summary: index.Summary{ID: "tool2", Name: "tool2"},
				},
			},
		},
		{
			name: "nil input",
			docs: nil,
			want: nil,
		},
		{
			name: "empty slice",
			docs: []Document{},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SearchDocsFromDocuments(tt.docs)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SearchDocsFromDocuments() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRoundTrip_SearchDocToDocumentAndBack(t *testing.T) {
	original := index.SearchDoc{
		ID:      "github:create-issue",
		DocText: "create-issue github create a new issue github issue tracker",
		Summary: index.Summary{
			ID:               "github:create-issue",
			Name:             "create-issue",
			Namespace:        "github",
			ShortDescription: "Create a new issue in GitHub",
			Tags:             []string{"github", "issue", "tracker"},
		},
	}

	// Convert to Document and back
	doc := DocumentFromSearchDoc(original)
	roundTripped := SearchDocFromDocument(doc)

	// Should preserve key fields
	if roundTripped.ID != original.ID {
		t.Errorf("ID mismatch: got %s, want %s", roundTripped.ID, original.ID)
	}
	if roundTripped.DocText != original.DocText {
		t.Errorf("DocText mismatch: got %s, want %s", roundTripped.DocText, original.DocText)
	}
	if roundTripped.Summary.Name != original.Summary.Name {
		t.Errorf("Name mismatch: got %s, want %s", roundTripped.Summary.Name, original.Summary.Name)
	}
	if roundTripped.Summary.Namespace != original.Summary.Namespace {
		t.Errorf("Namespace mismatch: got %s, want %s", roundTripped.Summary.Namespace, original.Summary.Namespace)
	}
	if !reflect.DeepEqual(roundTripped.Summary.Tags, original.Summary.Tags) {
		t.Errorf("Tags mismatch: got %v, want %v", roundTripped.Summary.Tags, original.Summary.Tags)
	}
}

func TestRoundTrip_DocumentToSearchDocAndBack(t *testing.T) {
	original := Document{
		ID:          "slack:send-message",
		Name:        "send-message",
		Namespace:   "slack",
		Description: "Send a message to a Slack channel",
		Tags:        []string{"slack", "messaging", "chat"},
		Category:    "communication",
		Text:        "send-message slack send a message to a slack channel slack messaging chat",
	}

	// Convert to SearchDoc and back
	searchDoc := SearchDocFromDocument(original)
	roundTripped := DocumentFromSearchDoc(searchDoc)

	// Should preserve key fields (Category is lost in round-trip)
	if roundTripped.ID != original.ID {
		t.Errorf("ID mismatch: got %s, want %s", roundTripped.ID, original.ID)
	}
	if roundTripped.Name != original.Name {
		t.Errorf("Name mismatch: got %s, want %s", roundTripped.Name, original.Name)
	}
	if roundTripped.Namespace != original.Namespace {
		t.Errorf("Namespace mismatch: got %s, want %s", roundTripped.Namespace, original.Namespace)
	}
	if roundTripped.Description != original.Description {
		t.Errorf("Description mismatch: got %s, want %s", roundTripped.Description, original.Description)
	}
	if !reflect.DeepEqual(roundTripped.Tags, original.Tags) {
		t.Errorf("Tags mismatch: got %v, want %v", roundTripped.Tags, original.Tags)
	}
	if roundTripped.Text != original.Text {
		t.Errorf("Text mismatch: got %s, want %s", roundTripped.Text, original.Text)
	}
	// Category is expected to be lost (not stored in SearchDoc)
	if roundTripped.Category != "" {
		t.Errorf("Category should be empty after round-trip, got %s", roundTripped.Category)
	}
}
