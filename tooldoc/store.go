package tooldoc

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/toolfoundation/model"
)

// Error values for consistent error handling.
var (
	// ErrNotFound is returned when a tool ID is not found in either the
	// index or the documentation store.
	ErrNotFound = errors.New("tool not found")

	// ErrInvalidDetail is returned when an invalid DetailLevel is provided.
	ErrInvalidDetail = errors.New("invalid detail level")

	// ErrNoTool is returned when schema/full detail is requested but the
	// Tool object is not available from the index or ToolResolver. This can
	// happen when documentation exists but the tool hasn't been registered
	// with index and no resolver is configured.
	ErrNoTool = errors.New("tool required for schema/full level")

	// ErrArgsTooLarge is returned when an example's Args exceeds depth or size caps.
	// The error message includes which example and what limits were exceeded.
	ErrArgsTooLarge = errors.New("args exceeds caps")
)

// Store defines the interface for tool documentation storage.
//
// Contract:
//   - Concurrency: implementations must be safe for concurrent use.
//   - Errors: return ErrNotFound when neither docs nor tool exist; ErrInvalidDetail for
//     invalid detail levels; ErrNoTool when schema/full detail requires a Tool.
//     Callers should use errors.Is.
//   - Ownership: returned ToolDoc and ToolExample slices are caller-owned and may be
//     treated as read-only snapshots.
//   - Determinism: identical inputs over unchanged data must yield stable results.
//   - Nil/zero: empty IDs are treated as not found; maxExamples <= 0 returns zero examples.
type Store interface {
	// DescribeTool returns documentation for a tool at the specified detail level.
	//
	// For DetailSummary: Returns summary only. Works if either docs or tool exist.
	// For DetailSchema/DetailFull: Requires the tool to be available from the
	// index or ToolResolver (returns ErrNoTool if missing, even when docs exist).
	//
	// Returns ErrNotFound if neither tool nor docs exist for the given ID.
	// Returns ErrNoTool if schema/full is requested but no tool can be resolved.
	// Returns ErrInvalidDetail if level is not a valid DetailLevel constant.
	DescribeTool(id string, level DetailLevel) (ToolDoc, error)

	// ListExamples returns up to maxExamples examples for a tool.
	// The effective limit is min(maxExamples, StoreOptions.MaxExamples) when both are set.
	// Returns ErrNotFound if the tool is not registered in docs or index.
	ListExamples(id string, maxExamples int) ([]ToolExample, error)
}

// StoreOptions configures the behavior of a Store implementation.
type StoreOptions struct {
	// Index is the index used for tool lookup.
	// If nil, tools must be provided via explicit registration.
	Index index.Index

	// ToolResolver is an optional injection path for resolving a tool by ID
	// when Index is nil or does not contain the tool.
	ToolResolver func(id string) (*model.Tool, error)

	// MaxExamples is the default maximum number of examples to return.
	// Zero means no limit (use ListExamples max parameter).
	MaxExamples int
}

// docRecord holds registered documentation for a tool.
type docRecord struct {
	summary      string
	notes        string
	examples     []ToolExample
	externalRefs []string
}

// InMemoryStore is an in-memory implementation of Store.
type InMemoryStore struct {
	mu           sync.RWMutex
	index        index.Index
	toolResolver func(id string) (*model.Tool, error)
	docs         map[string]*docRecord
	maxExamples  int
}

// NewInMemoryStore creates a new in-memory documentation store.
func NewInMemoryStore(opts StoreOptions) *InMemoryStore {
	return &InMemoryStore{
		index:        opts.Index,
		toolResolver: opts.ToolResolver,
		docs:         make(map[string]*docRecord),
		maxExamples:  opts.MaxExamples,
	}
}

// RegisterDoc registers documentation for a tool.
// The entry is validated and truncated to fit within caps.
// If the tool has no existing doc record, one is created.
// Args in examples are deep-copied to prevent external mutation.
//
// Returns ErrArgsTooLarge if any example's Args exceeds MaxArgsDepth or MaxArgsKeys.
func (s *InMemoryStore) RegisterDoc(id string, entry DocEntry) error {
	entry = entry.ValidateAndTruncate()

	// Deep copy examples with their Args and validate caps
	examples := make([]ToolExample, len(entry.Examples))
	for i, ex := range entry.Examples {
		// Deep copy first (normalizes types to map[string]any)
		argsCopy := deepCopyArgs(ex.Args)

		// Validate caps on normalized copy
		stats, valid := ValidateArgs(argsCopy)
		if !valid {
			return fmt.Errorf("%w: example %d (%s) has depth=%d (max %d), keys=%d (max %d)",
				ErrArgsTooLarge, i, ex.Title, stats.Depth, MaxArgsDepth, stats.Keys, MaxArgsKeys)
		}

		examples[i] = ToolExample{
			ID:          ex.ID,
			Title:       ex.Title,
			Description: ex.Description,
			Args:        argsCopy,
			ResultHint:  ex.ResultHint,
		}
	}

	// Copy external refs
	externalRefs := make([]string, len(entry.ExternalRefs))
	copy(externalRefs, entry.ExternalRefs)

	s.mu.Lock()
	defer s.mu.Unlock()

	record, exists := s.docs[id]
	if !exists {
		record = &docRecord{}
		s.docs[id] = record
	}

	record.summary = entry.Summary
	record.notes = entry.Notes
	record.examples = examples
	record.externalRefs = externalRefs

	return nil
}

// RegisterExamples adds or replaces examples for a tool.
// Examples are validated and truncated to fit within caps.
// Args are deep-copied to prevent external mutation.
//
// Returns ErrArgsTooLarge if any example's Args exceeds MaxArgsDepth or MaxArgsKeys.
func (s *InMemoryStore) RegisterExamples(id string, examples []ToolExample) error {
	limit := len(examples)
	if s.maxExamples > 0 && limit > s.maxExamples {
		limit = s.maxExamples
	}

	truncated := make([]ToolExample, limit)
	for i := 0; i < limit; i++ {
		ex := examples[i]
		// Deep copy first (normalizes types to map[string]any)
		argsCopy := deepCopyArgs(ex.Args)

		// Validate caps on normalized copy
		stats, valid := ValidateArgs(argsCopy)
		if !valid {
			return fmt.Errorf("%w: example %d (%s) has depth=%d (max %d), keys=%d (max %d)",
				ErrArgsTooLarge, i, ex.Title, stats.Depth, MaxArgsDepth, stats.Keys, MaxArgsKeys)
		}

		truncated[i] = ToolExample{
			ID:          ex.ID,
			Title:       ex.Title,
			Description: truncateString(ex.Description, MaxDescriptionLen),
			Args:        argsCopy,
			ResultHint:  truncateString(ex.ResultHint, MaxResultHintLen),
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, exists := s.docs[id]
	if !exists {
		record = &docRecord{}
		s.docs[id] = record
	}

	record.examples = truncated

	return nil
}

// DescribeTool returns documentation for a tool at the specified detail level.
// For schema/full levels, Tool must be available from the index.
func (s *InMemoryStore) DescribeTool(id string, level DetailLevel) (ToolDoc, error) {
	// Validate detail level
	switch level {
	case DetailSummary, DetailSchema, DetailFull:
		// valid
	default:
		return ToolDoc{}, fmt.Errorf("%w: %s", ErrInvalidDetail, level)
	}

	// Copy doc record fields under lock to prevent races
	var summary, notes string
	var examples []ToolExample
	var externalRefs []string
	var hasDoc bool

	s.mu.RLock()
	if docRec := s.docs[id]; docRec != nil {
		hasDoc = true
		summary = docRec.summary
		notes = docRec.notes
		// Deep copy examples for return
		examples = copyExamples(docRec.examples)
		// Copy external refs
		externalRefs = make([]string, len(docRec.externalRefs))
		copy(externalRefs, docRec.externalRefs)
	}
	maxExamples := s.maxExamples
	s.mu.RUnlock()

	// Try to get tool from index - needed for summary fallback and schema/full levels
	var tool *model.Tool
	var resolverErr error
	if s.index != nil {
		t, _, err := s.index.GetTool(id)
		if err == nil {
			tool = &t
		}
	}
	if tool == nil && s.toolResolver != nil {
		t, err := s.toolResolver(id)
		if err != nil {
			resolverErr = err
		} else if t != nil {
			tool = t
		}
	}

	// For schema/full, Tool is REQUIRED per MCP contract
	if level == DetailSchema || level == DetailFull {
		if tool == nil {
			// Propagate resolver error if that's why we don't have a tool
			if resolverErr != nil {
				return ToolDoc{}, resolverErr
			}
			if !hasDoc {
				return ToolDoc{}, fmt.Errorf("%w: %s", ErrNotFound, id)
			}
			return ToolDoc{}, fmt.Errorf("%w: %s", ErrNoTool, id)
		}
	}

	// Build the summary - prefer doc summary, fallback to tool description
	if summary == "" && tool != nil && tool.Description != "" {
		summary = truncateString(tool.Description, MaxSummaryLen)
	}

	var inputModes []string
	var outputModes []string
	var securitySummary string
	var annotations map[string]any
	if tool != nil {
		inputModes = stringSliceFromAny(tool.Meta["inputModes"])
		outputModes = stringSliceFromAny(tool.Meta["outputModes"])
		securitySummary = securitySummaryFromMeta(tool.Meta)
		annotations = annotationsFromTool(tool.Annotations)
	}

	// For summary level, we're done
	if level == DetailSummary {
		if summary == "" && !hasDoc && tool == nil {
			// Propagate resolver errors instead of masking them as not found
			if resolverErr != nil {
				return ToolDoc{}, resolverErr
			}
			return ToolDoc{}, fmt.Errorf("%w: %s", ErrNotFound, id)
		}
		return ToolDoc{
			Summary:         summary,
			InputModes:      inputModes,
			OutputModes:     outputModes,
			SecuritySummary: securitySummary,
			Annotations:     annotations,
		}, nil
	}

	// Build schema info from tool's InputSchema
	var schemaInfo *SchemaInfo
	if tool != nil {
		schemaInfo = deriveSchemaInfo(tool.InputSchema)
	}

	// Build result based on level
	result := ToolDoc{
		Tool:            tool,
		Summary:         summary,
		InputModes:      inputModes,
		OutputModes:     outputModes,
		SecuritySummary: securitySummary,
		Annotations:     annotations,
		SchemaInfo:      schemaInfo,
	}

	if level == DetailFull {
		result.Notes = notes
		result.ExternalRefs = externalRefs
		// Apply MaxExamples cap
		if maxExamples > 0 && len(examples) > maxExamples {
			examples = examples[:maxExamples]
		}
		result.Examples = examples
	}

	return result, nil
}

// ListExamples returns up to maxExamples for a tool.
// The effective limit is min(maxExamples, MaxExamples) when both are set.
func (s *InMemoryStore) ListExamples(id string, maxExamples int) ([]ToolExample, error) {
	// Copy examples under lock to prevent races
	var examples []ToolExample
	var hasDoc bool

	s.mu.RLock()
	if docRec := s.docs[id]; docRec != nil {
		hasDoc = true
		examples = copyExamples(docRec.examples)
	}
	defaultMax := s.maxExamples
	s.mu.RUnlock()

	// Check if tool exists in index or via resolver
	toolExists := false
	if s.index != nil {
		_, _, err := s.index.GetTool(id)
		toolExists = err == nil
	}
	if !toolExists && s.toolResolver != nil {
		t, err := s.toolResolver(id)
		if err != nil {
			// Propagate resolver errors (not ErrNotFound style)
			return nil, err
		}
		toolExists = t != nil
	}

	if !hasDoc && !toolExists {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, id)
	}

	if len(examples) == 0 {
		return []ToolExample{}, nil
	}

	// Compute effective limit: min(maxExamples, defaultMax) when both > 0
	effectiveMax := maxExamples
	if defaultMax > 0 {
		if effectiveMax <= 0 || defaultMax < effectiveMax {
			effectiveMax = defaultMax
		}
	}

	// Apply limit
	if effectiveMax > 0 && len(examples) > effectiveMax {
		examples = examples[:effectiveMax]
	}

	return examples, nil
}

// deepCopyArgs performs a deep copy of Args map.
// This ensures isolation between stored and returned values, preventing
// races and mutation side effects.
//
// For JSON-compatible types (maps, slices, primitives), values are fully
// deep-copied. For unknown types, values are shallow-copied (reference kept)
// to avoid silently dropping data. MCP tool arguments should only contain
// JSON-compatible types.
func deepCopyArgs(args map[string]any) map[string]any {
	if args == nil {
		return nil
	}
	result := make(map[string]any, len(args))
	for k, v := range args {
		result[k] = deepCopyValue(v)
	}
	return result
}

// deepCopyValue recursively copies a value, normalizing to MCP-native shapes.
// Typed maps become map[string]any, typed slices become []any.
// This ensures consistent type assertions in downstream code.
// Unknown types are shallow-copied to preserve data.
func deepCopyValue(v any) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	// Untyped JSON containers
	case map[string]any:
		return deepCopyArgs(val)
	case []any:
		return deepCopySlice(val)

	// Typed maps → normalize to map[string]any
	case map[string]string:
		result := make(map[string]any, len(val))
		for k, v := range val {
			result[k] = v
		}
		return result
	case map[string]int:
		result := make(map[string]any, len(val))
		for k, v := range val {
			result[k] = v
		}
		return result
	case map[string]float64:
		result := make(map[string]any, len(val))
		for k, v := range val {
			result[k] = v
		}
		return result
	case map[string]bool:
		result := make(map[string]any, len(val))
		for k, v := range val {
			result[k] = v
		}
		return result

	// Typed slices → normalize to []any
	case []string:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = v
		}
		return result
	case []int:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = v
		}
		return result
	case []float64:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = v
		}
		return result
	case []bool:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = v
		}
		return result

	// Primitives are immutable, safe to return directly
	case string, bool, float64, float32,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return val
	case json.Number:
		return val

	default:
		// Unknown type: shallow copy to avoid losing data
		// This preserves the reference but won't hide bugs
		return val
	}
}

// toStringSlice converts various slice types to []string.
// Handles []any, []string, and other common slice types.
// Returns nil if conversion is not possible.
func toStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	switch s := v.(type) {
	case []string:
		// Return a copy to avoid sharing
		result := make([]string, len(s))
		copy(result, s)
		return result
	case []any:
		result := make([]string, 0, len(s))
		for _, item := range s {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	default:
		return nil
	}
}

// deepCopySlice creates a deep copy of a []any slice.
func deepCopySlice(s []any) []any {
	if s == nil {
		return nil
	}
	result := make([]any, len(s))
	for i, v := range s {
		result[i] = deepCopyValue(v)
	}
	return result
}

// copyExamples creates a deep copy of a slice of ToolExamples.
func copyExamples(examples []ToolExample) []ToolExample {
	if examples == nil {
		return nil
	}
	result := make([]ToolExample, len(examples))
	for i, ex := range examples {
		result[i] = ToolExample{
			ID:          ex.ID,
			Title:       ex.Title,
			Description: ex.Description,
			Args:        deepCopyArgs(ex.Args),
			ResultHint:  ex.ResultHint,
		}
	}
	return result
}

// normalizeNumeric converts numeric values to float64 for consistency.
// JSON unmarshaling produces float64, but Go map literals may contain int.
func normalizeNumeric(v any) any {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int8:
		return float64(n)
	case int16:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case uint:
		return float64(n)
	case uint8:
		return float64(n)
	case uint16:
		return float64(n)
	case uint32:
		return float64(n)
	case uint64:
		return float64(n)
	case float32:
		return float64(n)
	default:
		return v
	}
}

// deriveSchemaInfo extracts schema information from an InputSchema.
// Returns nil if derivation is not possible.
// Numeric default values are normalized to float64.
func deriveSchemaInfo(schema any) *SchemaInfo {
	if schema == nil {
		return nil
	}

	// Try to get schema as map
	var schemaMap map[string]any

	switch s := schema.(type) {
	case map[string]any:
		schemaMap = s
	case json.RawMessage:
		if err := json.Unmarshal(s, &schemaMap); err != nil {
			return nil
		}
	case []byte:
		if err := json.Unmarshal(s, &schemaMap); err != nil {
			return nil
		}
	default:
		// Try JSON round-trip
		data, err := json.Marshal(schema)
		if err != nil {
			return nil
		}
		if err := json.Unmarshal(data, &schemaMap); err != nil {
			return nil
		}
	}

	info := &SchemaInfo{}
	hasData := false

	// Extract required fields (handle both []any and []string)
	if req, ok := schemaMap["required"]; ok {
		info.Required = toStringSlice(req)
		if len(info.Required) > 0 {
			hasData = true
		}
	}

	// Extract properties for types and defaults
	if props, ok := schemaMap["properties"]; ok {
		if propsMap, ok := props.(map[string]any); ok {
			info.Types = make(map[string][]string)
			info.Defaults = make(map[string]any)

			for name, prop := range propsMap {
				if propMap, ok := prop.(map[string]any); ok {
					// Extract type (handle string, []any, and []string)
					if t, ok := propMap["type"]; ok {
						if tv, ok := t.(string); ok {
							info.Types[name] = []string{tv}
							hasData = true
						} else if types := toStringSlice(t); len(types) > 0 {
							info.Types[name] = types
							hasData = true
						}
					}

					// Extract default (normalize numeric values to float64)
					if def, ok := propMap["default"]; ok {
						info.Defaults[name] = normalizeNumeric(def)
						hasData = true
					}
				}
			}

			// Clean up empty maps
			if len(info.Types) == 0 {
				info.Types = nil
			}
			if len(info.Defaults) == 0 {
				info.Defaults = nil
			}
		}
	}

	if !hasData {
		return nil
	}

	return info
}
