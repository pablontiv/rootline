package fix

import (
	"bytes"
	"reflect"
	"sort"

	"gopkg.in/yaml.v3"
)

// frontmatterIndent is the nesting indentation used when re-emitting frontmatter.
// yaml.v3 defaults to 4; two spaces matches the map-based writer this replaced
// and the convention already present in the repository's documents.
const frontmatterIndent = 2

// renderPreservedFrontmatter re-emits fmText with the values from fm applied,
// keeping the original key order, comments and scalar quoting style.
//
// It reports false when fmText is not a YAML mapping it can round-trip, which
// leaves the caller on its map-based fallback. RewriteFrontmatter cannot return
// an error — its signature is shared by six call sites, one of them under
// cmd/rootline — so an unparseable block degrades rather than failing the write.
func renderPreservedFrontmatter(fmText string, fm map[string]any) (string, bool) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(fmText), &doc); err != nil {
		return "", false
	}

	mapping := documentMapping(&doc)
	if mapping == nil {
		return "", false
	}

	seen := reconcileMapping(mapping, fm)
	appendNewKeys(mapping, fm, seen)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(frontmatterIndent)
	if err := enc.Encode(mapping); err != nil {
		return "", false
	}
	if err := enc.Close(); err != nil {
		return "", false
	}
	return buf.String(), true
}

// documentMapping unwraps a parsed document down to its top-level mapping node,
// synthesising an empty mapping for empty frontmatter. It returns nil when the
// block holds something other than a mapping, which this rewrite cannot edit
// key-by-key.
func documentMapping(doc *yaml.Node) *yaml.Node {
	if doc.Kind == 0 || len(doc.Content) == 0 {
		return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil
	}
	return root
}

// reconcileMapping updates the values of keys present in fm, drops keys absent
// from it, and reports which keys it consumed. A key whose value is unchanged is
// left completely untouched so its comment and quoting style survive verbatim.
func reconcileMapping(mapping *yaml.Node, fm map[string]any) map[string]bool {
	seen := make(map[string]bool, len(fm))
	kept := make([]*yaml.Node, 0, len(mapping.Content))

	for i := 0; i+1 < len(mapping.Content); i += 2 {
		keyNode, valNode := mapping.Content[i], mapping.Content[i+1]
		newVal, ok := fm[keyNode.Value]
		if !ok {
			continue // key dropped from the map — remove the pair
		}
		seen[keyNode.Value] = true

		if !valueNodeMatches(valNode, newVal) {
			replaceValueNode(valNode, newVal)
		}
		kept = append(kept, keyNode, valNode)
	}

	mapping.Content = kept
	return seen
}

// valueNodeMatches reports whether the node already decodes to want, in which
// case it must not be rewritten.
func valueNodeMatches(valNode *yaml.Node, want any) bool {
	var current any
	if err := valNode.Decode(&current); err != nil {
		return false
	}
	return reflect.DeepEqual(current, want)
}

// replaceValueNode overwrites a value node in place, carrying its comments over
// so that a changed value does not silently discard the note attached to it.
func replaceValueNode(valNode *yaml.Node, value any) {
	head, line, foot := valNode.HeadComment, valNode.LineComment, valNode.FootComment

	var encoded yaml.Node
	if err := encoded.Encode(value); err != nil {
		return
	}
	*valNode = encoded

	valNode.HeadComment, valNode.LineComment, valNode.FootComment = head, line, foot
}

// appendNewKeys adds keys that fm introduced, after the existing ones. They are
// sorted so repeated runs over the same document produce the same output.
func appendNewKeys(mapping *yaml.Node, fm map[string]any, seen map[string]bool) {
	fresh := make([]string, 0, len(fm))
	for k := range fm {
		if !seen[k] {
			fresh = append(fresh, k)
		}
	}
	sort.Strings(fresh)

	for _, k := range fresh {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: k}
		valNode := &yaml.Node{}
		if err := valNode.Encode(fm[k]); err != nil {
			continue
		}
		mapping.Content = append(mapping.Content, keyNode, valNode)
	}
}
