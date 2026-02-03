package search

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"

	"github.com/jonwraymond/tooldiscovery/index"
)

// computeFingerprint generates a stable hash of the document slice.
// The fingerprint changes when document content changes, enabling
// efficient cache invalidation for the BM25 index.
func computeFingerprint(docs []index.SearchDoc) string {
	h := sha256.New()

	for _, doc := range docs {
		// Write ID
		h.Write([]byte(doc.ID))
		h.Write([]byte{0}) // separator

		// Write DocText
		h.Write([]byte(doc.DocText))
		h.Write([]byte{0})

		// Write Summary fields
		h.Write([]byte(doc.Summary.ID))
		h.Write([]byte{0})
		h.Write([]byte(doc.Summary.Name))
		h.Write([]byte{0})
		h.Write([]byte(doc.Summary.Namespace))
		h.Write([]byte{0})
		h.Write([]byte(doc.Summary.ShortDescription))
		h.Write([]byte{0})
		h.Write([]byte(doc.Summary.Summary))
		h.Write([]byte{0})
		h.Write([]byte(doc.Summary.Category))
		h.Write([]byte{0})
		h.Write([]byte(strings.Join(doc.Summary.InputModes, "\x01")))
		h.Write([]byte{0})
		h.Write([]byte(strings.Join(doc.Summary.OutputModes, "\x01")))
		h.Write([]byte{0})
		h.Write([]byte(doc.Summary.SecuritySummary))
		h.Write([]byte{0})

		// Write Tags (sorted for order-independence, then joined with separator)
		sortedTags := slices.Clone(doc.Summary.Tags)
		slices.Sort(sortedTags)
		h.Write([]byte(strings.Join(sortedTags, "\x01")))
		h.Write([]byte{0})
	}

	return hex.EncodeToString(h.Sum(nil))
}
