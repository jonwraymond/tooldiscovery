package tooldoc

import "github.com/jonwraymond/toolfoundation/model"

// DetailLevel specifies the amount of detail to return for tool documentation.
type DetailLevel string

const (
	// DetailSummary returns only a short description (1-2 lines).
	// Tool is nil, Notes/Examples are empty. Works without tool in index.
	DetailSummary DetailLevel = "summary"

	// DetailSchema returns the full model.Tool with InputSchema/OutputSchema.
	// SchemaInfo is populated when derivable. Notes are empty at this level.
	// Requires tool to be resolved via index or ToolResolver
	// (returns ErrNoTool otherwise).
	DetailSchema DetailLevel = "schema"

	// DetailFull returns everything: Tool, SchemaInfo, Notes with usage guidance,
	// examples (capped by MaxExamples), and ExternalRefs.
	// Requires tool to be resolved via index or ToolResolver
	// (returns ErrNoTool otherwise).
	DetailFull DetailLevel = "full"
)

// Truncation caps enforced at registration time.
const (
	MaxDescriptionLen = 300  // Maximum length of ToolExample.Description
	MaxResultHintLen  = 200  // Maximum length of ToolExample.ResultHint
	MaxSummaryLen     = 200  // Maximum length of ToolDoc.Summary
	MaxNotesLen       = 2000 // Maximum length of ToolDoc.Notes
)

// Args caps to prevent context pollution when examples are included in LLM context.
const (
	MaxArgsDepth = 5  // Maximum nesting depth for Args maps/slices
	MaxArgsKeys  = 50 // Maximum total size: map keys + slice items across all levels
)

// ToolExample represents a usage example for a tool.
type ToolExample struct {
	// ID is an optional unique identifier for the example.
	ID string `json:"id,omitempty"`

	// Title is a short label for the example.
	Title string `json:"title"`

	// Description is 1-2 sentences explaining what the example demonstrates.
	// Maximum length: MaxDescriptionLen (300 chars).
	Description string `json:"description"`

	// Args is the input payload for the tool call.
	// Should contain JSON-compatible types (strings, numbers, bools,
	// nil, maps, slices) to align with MCP tool argument requirements.
	// Args are deep-copied and normalized to MCP-native shapes on
	// registration and retrieval: typed slices become []any, typed
	// maps become map[string]any. This ensures consistent type
	// assertions in downstream code.
	//
	// Args are validated at registration: maximum depth is MaxArgsDepth (5),
	// maximum total size (map keys + slice items) is MaxArgsKeys (50).
	// Examples with Args exceeding these caps are rejected by
	// RegisterDoc/RegisterExamples.
	Args map[string]any `json:"args"`

	// ResultHint describes the expected shape/semantics of the result.
	// Maximum length: MaxResultHintLen (200 chars).
	ResultHint string `json:"resultHint,omitempty"`
}

// SchemaInfo contains derived information about a tool's input schema.
// This is best-effort only; fields may be nil if derivation is not possible.
type SchemaInfo struct {
	// Required lists the names of required input parameters.
	Required []string `json:"required,omitempty"`

	// Defaults maps parameter names to their default values.
	Defaults map[string]any `json:"defaults,omitempty"`

	// Types maps parameter names to their allowed types.
	// For example: {"limit": ["integer"], "query": ["string"]}
	Types map[string][]string `json:"types,omitempty"`
}

// ToolDoc represents documentation for a tool at varying levels of detail.
type ToolDoc struct {
	// Tool is the full model.Tool definition.
	// Required for schema/full levels; nil for summary.
	Tool *model.Tool `json:"tool,omitempty"`

	// Summary is a short description (1-2 lines).
	// Maximum length: MaxSummaryLen (200 chars).
	Summary string `json:"summary"`

	// SchemaInfo contains derived schema information.
	// Optional; populated at schema/full levels when derivable.
	SchemaInfo *SchemaInfo `json:"schemaInfo,omitempty"`

	// Notes contains human-authored usage guidance, constraints,
	// pagination/auth hints, and error semantics.
	// Full level only. Maximum length: MaxNotesLen (2000 chars).
	Notes string `json:"notes,omitempty"`

	// Examples contains a small set of usage examples (1-3).
	// Optional; typically populated at full level.
	Examples []ToolExample `json:"examples,omitempty"`

	// ExternalRefs contains URLs or resource IDs for additional documentation.
	// Full level only.
	ExternalRefs []string `json:"externalRefs,omitempty"`
}

// DocEntry is the input structure for registering documentation for a tool.
// It contains the custom documentation that augments the tool's metadata.
type DocEntry struct {
	// Summary overrides or supplements the tool's Description.
	// If empty, the tool's Description is used.
	Summary string

	// Notes contains usage guidance, constraints, etc.
	Notes string

	// Examples for this tool.
	Examples []ToolExample

	// ExternalRefs contains URLs or resource IDs.
	ExternalRefs []string
}

// truncateString truncates s to maxLen characters.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// ArgsStats holds metrics computed from an Args map.
type ArgsStats struct {
	Depth int // Maximum nesting depth encountered
	Keys  int // Total size: map keys + slice items across all levels
}

// ValidateArgs checks if Args respects depth and size caps.
// Returns stats and true if valid, or stats and false if caps exceeded.
func ValidateArgs(args map[string]any) (ArgsStats, bool) {
	stats := ArgsStats{}
	if args == nil {
		return stats, true
	}
	stats.Keys, stats.Depth = countArgsMetrics(args, 1)
	return stats, stats.Depth <= MaxArgsDepth && stats.Keys <= MaxArgsKeys
}

// countArgsMetrics recursively counts keys and tracks depth.
// Returns (totalKeys, maxDepth).
func countArgsMetrics(m map[string]any, currentDepth int) (int, int) {
	keys := len(m)
	maxDepth := currentDepth

	for _, v := range m {
		childKeys, childDepth := countValueMetrics(v, currentDepth+1)
		keys += childKeys
		if childDepth > maxDepth {
			maxDepth = childDepth
		}
	}

	return keys, maxDepth
}

// countValueMetrics recursively analyzes a value for size and depth.
// Size counts both map keys and slice items to prevent context pollution.
func countValueMetrics(v any, currentDepth int) (int, int) {
	switch val := v.(type) {
	case map[string]any:
		return countArgsMetrics(val, currentDepth)
	case []any:
		// Count slice items toward size (each item = 1)
		size := len(val)
		maxDepth := currentDepth
		for _, item := range val {
			childSize, childDepth := countValueMetrics(item, currentDepth+1)
			size += childSize
			if childDepth > maxDepth {
				maxDepth = childDepth
			}
		}
		return size, maxDepth
	default:
		return 0, currentDepth - 1 // Primitives don't add depth
	}
}

// ValidateAndTruncate validates and truncates a DocEntry's fields to fit within caps.
// It returns a new DocEntry with truncated values.
func (e DocEntry) ValidateAndTruncate() DocEntry {
	result := DocEntry{
		Summary:      truncateString(e.Summary, MaxSummaryLen),
		Notes:        truncateString(e.Notes, MaxNotesLen),
		ExternalRefs: e.ExternalRefs,
	}

	// Truncate examples
	result.Examples = make([]ToolExample, len(e.Examples))
	for i, ex := range e.Examples {
		result.Examples[i] = ToolExample{
			ID:          ex.ID,
			Title:       ex.Title,
			Description: truncateString(ex.Description, MaxDescriptionLen),
			Args:        ex.Args,
			ResultHint:  truncateString(ex.ResultHint, MaxResultHintLen),
		}
	}

	return result
}
