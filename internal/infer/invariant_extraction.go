package infer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
)

var invariantRe = regexp.MustCompile(`INV(\d+):\s*(.+)`)

// targetSections lists heading names where invariants are declared.
var targetSections = map[string]bool{
	"Invariantes": true,
	"Preserva":    true,
}

// DetectInvariants extracts invariant declarations from record bodies.
// It scans sections named "Invariantes" or "Preserva" for patterns like
// "INV1: description text". Code blocks are excluded by the AST-based
// section extraction.
func DetectInvariants(records []*extract.Record) []Inference {
	var inferences []Inference

	for _, rec := range records {
		if rec.AST == nil || rec.Body == "" {
			continue
		}

		source := []byte(rec.Body)
		sections := extract.ExtractSections(rec.AST, source)

		for _, sec := range sections {
			if !targetSections[sec.Heading] {
				continue
			}

			matches := invariantRe.FindAllStringSubmatch(sec.Content, -1)
			for _, m := range matches {
				id := "INV" + m[1]
				text := strings.TrimSpace(m[2])
				inferences = append(inferences, Inference{
					Type:    "invariant_declaration",
					Source:  rec.Path,
					Field:   id,
					Value:   text,
					Message: fmt.Sprintf("invariant %s declared in section %q of %s: %s", id, sec.Heading, rec.Path, text),
				})
			}
		}
	}

	return inferences
}
