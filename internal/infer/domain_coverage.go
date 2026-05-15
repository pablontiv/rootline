package infer

import (
	"github.com/pablontiv/rootline/internal/rules"
)

// DetectMissingDomains is deprecated — domain: field was removed in O14 refactor.
// Returns nil (removed from inference engine).
func DetectMissingDomains(stem *rules.StemFile) []Inference {
	return nil
}
