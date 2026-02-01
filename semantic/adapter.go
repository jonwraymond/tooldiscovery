package semantic

import (
	"github.com/jonwraymond/tooldiscovery/index"
)

// DocumentFromSearchDoc converts an index.SearchDoc to a semantic.Document.
// This enables seamless integration between the index package's search
// infrastructure and the semantic package's embedding-based search.
//
// The conversion maps fields as follows:
//   - ID: Canonical tool ID (preserved)
//   - Name: From Summary.Name
//   - Namespace: From Summary.Namespace
//   - Description: From Summary.ShortDescription
//   - Tags: From Summary.Tags (copied)
//   - Category: Empty (not present in SearchDoc)
//   - Text: From DocText (pre-normalized search text)
func DocumentFromSearchDoc(doc index.SearchDoc) Document {
	var tags []string
	if len(doc.Summary.Tags) > 0 {
		tags = make([]string, len(doc.Summary.Tags))
		copy(tags, doc.Summary.Tags)
	}

	return Document{
		ID:          doc.ID,
		Name:        doc.Summary.Name,
		Namespace:   doc.Summary.Namespace,
		Description: doc.Summary.ShortDescription,
		Tags:        tags,
		Category:    "",
		Text:        doc.DocText,
	}
}

// DocumentsFromSearchDocs converts a slice of index.SearchDoc to semantic.Document.
// Returns nil for nil or empty input.
func DocumentsFromSearchDocs(docs []index.SearchDoc) []Document {
	if len(docs) == 0 {
		return nil
	}

	result := make([]Document, len(docs))
	for i, doc := range docs {
		result[i] = DocumentFromSearchDoc(doc)
	}
	return result
}

// SearchDocFromDocument converts a semantic.Document back to an index.SearchDoc.
// This enables results from semantic search to be used with index package APIs.
//
// The conversion maps fields as follows:
//   - ID: Canonical tool ID (preserved)
//   - DocText: From Text field (or rebuilt if empty)
//   - Summary.ID: Same as ID
//   - Summary.Name: From Name
//   - Summary.Namespace: From Namespace
//   - Summary.ShortDescription: From Description (truncated to 120 chars)
//   - Summary.Tags: From Tags (copied)
func SearchDocFromDocument(doc Document) index.SearchDoc {
	var tags []string
	if len(doc.Tags) > 0 {
		tags = make([]string, len(doc.Tags))
		copy(tags, doc.Tags)
	}

	// Truncate description to match index.MaxShortDescriptionLen
	shortDesc := doc.Description
	if len(shortDesc) > index.MaxShortDescriptionLen {
		shortDesc = shortDesc[:index.MaxShortDescriptionLen]
	}

	// Use provided Text or rebuild from fields
	docText := doc.Text
	if docText == "" {
		// Rebuild similar to how index package does it
		docText = buildDocText(doc)
	}

	return index.SearchDoc{
		ID:      doc.ID,
		DocText: docText,
		Summary: index.Summary{
			ID:               doc.ID,
			Name:             doc.Name,
			Namespace:        doc.Namespace,
			ShortDescription: shortDesc,
			Tags:             tags,
		},
	}
}

// SearchDocsFromDocuments converts a slice of semantic.Document to index.SearchDoc.
// Returns nil for nil or empty input.
func SearchDocsFromDocuments(docs []Document) []index.SearchDoc {
	if len(docs) == 0 {
		return nil
	}

	result := make([]index.SearchDoc, len(docs))
	for i, doc := range docs {
		result[i] = SearchDocFromDocument(doc)
	}
	return result
}

// buildDocText creates lowercased search text from document fields.
func buildDocText(doc Document) string {
	// Use the Normalized method to get consistent text
	normalized := doc.Normalized()
	return normalized.Text
}
