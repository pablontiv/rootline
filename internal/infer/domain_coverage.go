package infer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/rules"
)

// DetectMissingDomains flags schema fields that lack a domain declaration.
// Fields of type "section" are skipped — they don't carry semantic meaning.
func DetectMissingDomains(stem *rules.StemFile) []Inference {
	if stem == nil || len(stem.Schema) == 0 {
		return nil
	}

	var inferences []Inference

	names := make([]string, 0, len(stem.Schema))
	for name := range stem.Schema {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		sf := stem.Schema[name]
		if sf.Domain != "" {
			continue
		}
		if sf.Type == "section" {
			continue
		}

		msg := fmt.Sprintf("Field %q", name)
		if sf.Type != "" {
			msg += fmt.Sprintf(" (type: %s", sf.Type)
			if len(sf.Values) > 0 {
				msg += fmt.Sprintf(", values: [%s]", strings.Join(sf.Values, ", "))
			}
			msg += ")"
		}
		msg += " has no domain declared"

		inferences = append(inferences, Inference{
			Type:    "missing_domain",
			Source:  sf.Source,
			Field:   name,
			Message: msg,
		})
	}

	return inferences
}
