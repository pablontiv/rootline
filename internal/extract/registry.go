package extract

import (
	"fmt"
	"path/filepath"
)

// Registry maps file extensions and names to extractors.
type Registry struct {
	byName      map[string]Extractor
	byExtension map[string]Extractor
}

// NewRegistry creates a registry with the built-in extractors.
func NewRegistry() *Registry {
	r := &Registry{
		byName:      make(map[string]Extractor),
		byExtension: make(map[string]Extractor),
	}
	r.Register(&MarkdownExtractor{})
	return r
}

// NewASTRegistry creates a registry with AST parsing enabled.
// Use this when body structure (sections, headings) needs to be inspected.
func NewASTRegistry() *Registry {
	r := &Registry{
		byName:      make(map[string]Extractor),
		byExtension: make(map[string]Extractor),
	}
	parseAST := true
	r.Register(&MarkdownExtractor{ParseAST: &parseAST})
	return r
}

// Register adds an extractor to the registry.
// Panics on duplicate name or extension (programmer error, not user error).
func (r *Registry) Register(e Extractor) {
	if _, exists := r.byName[e.Name()]; exists {
		panic(fmt.Sprintf("extractor %q already registered", e.Name()))
	}
	r.byName[e.Name()] = e
	for _, ext := range e.Extensions() {
		if existing, exists := r.byExtension[ext]; exists {
			panic(fmt.Sprintf("extension %q already registered by %q", ext, existing.Name()))
		}
		r.byExtension[ext] = e
	}
}

// ForFile returns the extractor for a given file path.
// If stemExtractor is non-empty, it is used as a name-based override.
// Returns nil if no extractor matches (file is skipped, not an error).
func (r *Registry) ForFile(path string, stemExtractor string) Extractor {
	if stemExtractor != "" {
		return r.byName[stemExtractor]
	}
	ext := filepath.Ext(path)
	return r.byExtension[ext]
}
