package infer

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
)

// DetectSectionPatterns analyzes heading structure across records in a directory.
// Every record contributes to the denominator. A record contributes once to each
// final-heading family, but selector resolution retains all section occurrences.
func DetectSectionPatterns(records []*extract.Record, threshold float64) ([]Inference, error) {
	if math.IsNaN(threshold) || math.IsInf(threshold, 0) || threshold < 0 || threshold > 1 {
		return nil, fmt.Errorf("section threshold must be finite and in [0,1], got %v", threshold)
	}
	if len(records) == 0 {
		return nil, nil
	}

	total := len(records)
	families := make(map[extract.HeadingKey]*sectionFamily)
	for recordOrdinal, rec := range records {
		if rec == nil || rec.Body == "" {
			continue
		}
		sections := sectionsForInference(rec)
		paths := sectionPathsForInference(sections)
		for sectionOrdinal, sec := range sections {
			if sec.Level <= 0 || sec.Heading == "" {
				continue
			}
			key := extract.HeadingKey{Level: sec.Level, Text: sec.Heading}
			family := families[key]
			if family == nil {
				family = &sectionFamily{key: key, contributors: make(map[int]bool)}
				families[key] = family
			}
			family.contributors[recordOrdinal] = true
			family.occurrences = append(family.occurrences, sectionOccurrence{
				recordPath:     rec.Path,
				recordOrdinal:  recordOrdinal,
				sectionOrdinal: sectionOrdinal,
				path:           paths[sectionOrdinal],
			})
		}
	}

	candidates := make([]sectionCandidate, 0, len(families))
	for _, family := range families {
		count := len(family.contributors)
		freq := float64(count) / float64(total)
		if freq < threshold {
			continue
		}
		exact := exactSectionHeading(family.key)
		typeName := "optional_section"
		if count == total {
			typeName = "required_section"
		}
		candidates = append(candidates, sectionCandidate{
			family:   family,
			field:    sectionFieldName(family.key.Text),
			exact:    exact,
			count:    count,
			freq:     freq,
			typeName: typeName,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].field != candidates[j].field {
			return candidates[i].field < candidates[j].field
		}
		return candidates[i].exact < candidates[j].exact
	})

	for i := range candidates {
		source, err := resolveSectionFamily(candidates[i].family)
		if err != nil {
			return nil, err
		}
		candidates[i].source = source
	}
	if err := rejectSectionFieldCollisions(candidates); err != nil {
		return nil, err
	}

	inferences := make([]Inference, 0, len(candidates))
	for _, candidate := range candidates {
		freqStr := fmt.Sprintf("%.2f", candidate.freq)
		inferences = append(inferences, Inference{
			Type:            candidate.typeName,
			Field:           candidate.field,
			Value:           freqStr,
			Message:         fmt.Sprintf("section %q appears in %d/%d records (%.0f%%) — %s", candidate.exact, candidate.count, total, candidate.freq*100, strings.TrimSuffix(candidate.typeName, "_section")),
			SourceDirective: candidate.source,
		})
	}

	return inferences, nil
}

type sectionFamily struct {
	key          extract.HeadingKey
	contributors map[int]bool
	occurrences  []sectionOccurrence
}

type sectionOccurrence struct {
	recordPath     string
	recordOrdinal  int
	sectionOrdinal int
	path           extract.SectionPath
	fullSource     string
}

type sectionCandidate struct {
	family   *sectionFamily
	field    string
	exact    string
	source   string
	count    int
	freq     float64
	typeName string
}

type selectorCandidate struct {
	source   string
	selector extract.SectionSelector
}

type stableOccurrenceGroup struct {
	identity  string
	selectors []selectorCandidate
}

func sectionsForInference(rec *extract.Record) []extract.Section {
	if len(rec.BodySections) > 0 {
		return rec.BodySections
	}
	if rec.AST != nil {
		return extract.ExtractSections(rec.AST, []byte(rec.Body))
	}
	return extract.ExtractSectionsFromText(rec.Body)
}

func sectionPathsForInference(sections []extract.Section) []extract.SectionPath {
	paths := make([]extract.SectionPath, len(sections))
	path := make(extract.SectionPath, 0, 6)
	for i, sec := range sections {
		if sec.Level <= 0 {
			continue
		}
		for len(path) > 0 && path[len(path)-1].Level >= sec.Level {
			path = path[:len(path)-1]
		}
		path = append(path, extract.HeadingKey{Level: sec.Level, Text: sec.Heading})
		paths[i] = append(extract.SectionPath(nil), path...)
	}
	return paths
}

func resolveSectionFamily(family *sectionFamily) (string, error) {
	occurrences := append([]sectionOccurrence(nil), family.occurrences...)
	for i := range occurrences {
		source, err := extract.CanonicalSectionSelectorSource(extract.SectionSelector(occurrences[i].path))
		if err != nil {
			return "", err
		}
		occurrences[i].fullSource = source
	}
	sortSectionOccurrences(occurrences)

	if err := rejectDuplicateSectionPaths(family.key, occurrences); err != nil {
		return "", err
	}

	selectorBySource := make(map[string]selectorCandidate)
	for _, occurrence := range occurrences {
		for start := len(occurrence.path) - 1; start >= 0; start-- {
			source, err := extract.CanonicalSectionSelectorSource(extract.SectionSelector(occurrence.path[start:]))
			if err != nil {
				return "", err
			}
			parsed, err := extract.ParseBodySource(source)
			if err != nil {
				return "", err
			}
			selectorBySource[source] = selectorCandidate{source: source, selector: parsed.Selector}
		}
	}

	selectors := make([]selectorCandidate, 0, len(selectorBySource))
	for _, selector := range selectorBySource {
		selectors = append(selectors, selector)
	}
	sort.Slice(selectors, func(i, j int) bool {
		if len(selectors[i].selector) != len(selectors[j].selector) {
			return len(selectors[i].selector) < len(selectors[j].selector)
		}
		return selectors[i].source < selectors[j].source
	})

	contributorOrdinals := make([]int, 0, len(family.contributors))
	for recordOrdinal := range family.contributors {
		contributorOrdinals = append(contributorOrdinals, recordOrdinal)
	}
	sort.Slice(contributorOrdinals, func(i, j int) bool {
		left, right := firstOccurrenceForRecord(occurrences, contributorOrdinals[i]), firstOccurrenceForRecord(occurrences, contributorOrdinals[j])
		if left.recordPath != right.recordPath {
			return left.recordPath < right.recordPath
		}
		return contributorOrdinals[i] < contributorOrdinals[j]
	})

	groupsByIdentity := make(map[string]*stableOccurrenceGroup)
	for _, selector := range selectors {
		selected := make([]sectionOccurrence, 0, len(contributorOrdinals))
		common := true
		for _, recordOrdinal := range contributorOrdinals {
			matches := matchingOccurrences(occurrences, recordOrdinal, selector.selector)
			if len(matches) != 1 {
				common = false
				break
			}
			selected = append(selected, matches[0])
		}
		if !common {
			continue
		}
		identity := occurrenceGroupIdentity(selected)
		group := groupsByIdentity[identity]
		if group == nil {
			group = &stableOccurrenceGroup{identity: identity}
			groupsByIdentity[identity] = group
		}
		group.selectors = append(group.selectors, selector)
	}

	exact := exactSectionHeading(family.key)
	if len(groupsByIdentity) == 0 {
		return "", fmt.Errorf("section family %q has no common selector", exact)
	}

	groups := make([]*stableOccurrenceGroup, 0, len(groupsByIdentity))
	for _, group := range groupsByIdentity {
		sort.Slice(group.selectors, func(i, j int) bool {
			if len(group.selectors[i].selector) != len(group.selectors[j].selector) {
				return len(group.selectors[i].selector) < len(group.selectors[j].selector)
			}
			return group.selectors[i].source < group.selectors[j].source
		})
		groups = append(groups, group)
	}
	sort.Slice(groups, func(i, j int) bool {
		left, right := groups[i].selectors[0], groups[j].selectors[0]
		if left.source != right.source {
			return left.source < right.source
		}
		return groups[i].identity < groups[j].identity
	})
	if len(groups) > 1 {
		sources := make([]string, 0, len(groups))
		for _, group := range groups {
			sources = append(sources, group.selectors[0].source)
		}
		return "", fmt.Errorf("section family %q has multiple stable occurrence groups: %s", exact, strings.Join(sources, ", "))
	}
	return groups[0].selectors[0].source, nil
}

func sortSectionOccurrences(occurrences []sectionOccurrence) {
	sort.Slice(occurrences, func(i, j int) bool {
		if occurrences[i].recordPath != occurrences[j].recordPath {
			return occurrences[i].recordPath < occurrences[j].recordPath
		}
		if occurrences[i].recordOrdinal != occurrences[j].recordOrdinal {
			return occurrences[i].recordOrdinal < occurrences[j].recordOrdinal
		}
		if occurrences[i].sectionOrdinal != occurrences[j].sectionOrdinal {
			return occurrences[i].sectionOrdinal < occurrences[j].sectionOrdinal
		}
		return occurrences[i].fullSource < occurrences[j].fullSource
	})
}

func rejectDuplicateSectionPaths(key extract.HeadingKey, occurrences []sectionOccurrence) error {
	type duplicateKey struct {
		recordOrdinal int
		fullSource    string
	}
	byPath := make(map[duplicateKey][]sectionOccurrence)
	for _, occurrence := range occurrences {
		key := duplicateKey{recordOrdinal: occurrence.recordOrdinal, fullSource: occurrence.fullSource}
		byPath[key] = append(byPath[key], occurrence)
	}

	var duplicates []string
	for _, matches := range byPath {
		if len(matches) < 2 {
			continue
		}
		ordinals := make([]string, 0, len(matches))
		for _, match := range matches {
			ordinals = append(ordinals, strconv.Itoa(match.sectionOrdinal))
		}
		duplicates = append(duplicates, fmt.Sprintf("%q (record %d), %s (section ordinals %s)", matches[0].recordPath, matches[0].recordOrdinal, matches[0].fullSource, strings.Join(ordinals, ", ")))
	}
	if len(duplicates) == 0 {
		return nil
	}
	sort.Strings(duplicates)
	return fmt.Errorf("duplicate body section path for family %q: %s", exactSectionHeading(key), strings.Join(duplicates, "; "))
}

func firstOccurrenceForRecord(occurrences []sectionOccurrence, recordOrdinal int) sectionOccurrence {
	for _, occurrence := range occurrences {
		if occurrence.recordOrdinal == recordOrdinal {
			return occurrence
		}
	}
	return sectionOccurrence{recordOrdinal: recordOrdinal}
}

func matchingOccurrences(occurrences []sectionOccurrence, recordOrdinal int, selector extract.SectionSelector) []sectionOccurrence {
	var matches []sectionOccurrence
	for _, occurrence := range occurrences {
		if occurrence.recordOrdinal == recordOrdinal && extract.SectionSelectorMatches(occurrence.path, selector) {
			matches = append(matches, occurrence)
		}
	}
	return matches
}

func occurrenceGroupIdentity(occurrences []sectionOccurrence) string {
	var identity strings.Builder
	for _, occurrence := range occurrences {
		fmt.Fprintf(&identity, "%s\x00%d\x00%d\x00%s\x00", occurrence.recordPath, occurrence.recordOrdinal, occurrence.sectionOrdinal, occurrence.fullSource)
	}
	return identity.String()
}

func exactSectionHeading(key extract.HeadingKey) string {
	return strings.Repeat("#", key.Level) + " " + key.Text
}

func rejectSectionFieldCollisions(candidates []sectionCandidate) error {
	var collisions []string
	for i := 0; i < len(candidates); {
		j := i + 1
		for j < len(candidates) && candidates[j].field == candidates[i].field {
			j++
		}
		if j-i > 1 {
			headings := make([]string, 0, j-i)
			for _, candidate := range candidates[i:j] {
				headings = append(headings, strconv.Quote(candidate.exact))
			}
			collisions = append(collisions, fmt.Sprintf("%s: %s", candidates[i].field, strings.Join(headings, ", ")))
		}
		i = j
	}
	if len(collisions) > 0 {
		return fmt.Errorf("section field name collision: %s", strings.Join(collisions, "; "))
	}
	return nil
}
