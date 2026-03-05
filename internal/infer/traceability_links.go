package infer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
)

// traceabilityRe matches patterns like "**Contribuye a**: target" or "Contribuye a: target".
var traceabilityRe = regexp.MustCompile(`(?:\*\*)?(?i)(Contribuye a|Cubre|Satisface)(?:\*\*)?:\s*(.+)`)

// wikiLinkInlineRe matches [[target]] inside a string.
var wikiLinkInlineRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// DetectTraceabilityLinks extracts traceability claims from record bodies.
// Claims like "Contribuye a: X", "Cubre: Y", "Satisface: Z" are detected.
// Targets containing wiki-links produce verified_traceability inferences;
// free-text targets produce unverified_traceability.
func DetectTraceabilityLinks(records []*extract.Record) []Inference {
	var inferences []Inference

	for _, rec := range records {
		if rec.Body == "" {
			continue
		}

		for _, line := range strings.Split(rec.Body, "\n") {
			matches := traceabilityRe.FindAllStringSubmatch(line, -1)
			for _, m := range matches {
				verb := normalizeVerb(m[1])
				targetStr := strings.TrimSpace(m[2])

				// Split comma-separated targets.
				targets := splitTargets(targetStr)

				for _, target := range targets {
					target = strings.TrimSpace(target)
					if target == "" {
						continue
					}

					// Check if target contains a wiki-link.
					if wikiLinkInlineRe.MatchString(target) {
						wikiMatches := wikiLinkInlineRe.FindAllStringSubmatch(target, -1)
						for _, wm := range wikiMatches {
							inferences = append(inferences, Inference{
								Type:    "verified_traceability",
								Source:  rec.Path,
								Field:   verb,
								Value:   wm[1],
								Message: fmt.Sprintf("verified traceability: %s → %s in %s", verb, wm[1], rec.Path),
							})
						}
					} else {
						inferences = append(inferences, Inference{
							Type:    "unverified_traceability",
							Source:  rec.Path,
							Field:   verb,
							Value:   target,
							Message: fmt.Sprintf("unverified traceability: %s → %q in %s (requires_agent: true)", verb, target, rec.Path),
						})
					}
				}
			}
		}
	}

	return inferences
}

func normalizeVerb(verb string) string {
	switch strings.ToLower(verb) {
	case "contribuye a":
		return "contribuye_a"
	case "cubre":
		return "cubre"
	case "satisface":
		return "satisface"
	default:
		return strings.ToLower(strings.ReplaceAll(verb, " ", "_"))
	}
}

func splitTargets(s string) []string {
	// Split on comma, but only if not inside brackets.
	var parts []string
	depth := 0
	start := 0
	for i, ch := range s {
		switch ch {
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}
