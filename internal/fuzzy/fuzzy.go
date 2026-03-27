// Package fuzzy provides Levenshtein-based fuzzy matching for suggestions.
// It is used across rootline to enrich error messages with "did you mean?" hints.
package fuzzy

import (
	"sort"
	"strings"
)

// Match returns the closest candidate within the adaptive threshold max(2, len(input)/3).
// Case-insensitive. Returns "" if no candidate is close enough.
func Match(input string, candidates []string) string {
	if len(candidates) == 0 {
		return ""
	}

	threshold := max(2, len(input)/3)
	best := ""
	bestDist := threshold + 1

	lower := strings.ToLower(input)
	for _, c := range candidates {
		d := distance(lower, strings.ToLower(c))
		if d < bestDist {
			bestDist = d
			best = c
		}
	}
	return best
}

// MatchN returns up to n closest candidates within the adaptive threshold,
// sorted by ascending distance. Returns nil if no candidate is close enough.
func MatchN(input string, candidates []string, n int) []string {
	if len(candidates) == 0 || n <= 0 {
		return nil
	}

	threshold := max(2, len(input)/3)
	lower := strings.ToLower(input)

	type scored struct {
		candidate string
		dist      int
	}
	var hits []scored

	for _, c := range candidates {
		d := distance(lower, strings.ToLower(c))
		if d <= threshold {
			hits = append(hits, scored{c, d})
		}
	}

	if len(hits) == 0 {
		return nil
	}

	sort.Slice(hits, func(i, j int) bool {
		return hits[i].dist < hits[j].dist
	})

	if len(hits) > n {
		hits = hits[:n]
	}

	result := make([]string, len(hits))
	for i, h := range hits {
		result[i] = h.candidate
	}
	return result
}

// Distance returns the Levenshtein edit distance between a and b (case-insensitive).
func Distance(a, b string) int {
	return distance(strings.ToLower(a), strings.ToLower(b))
}

// distance computes Levenshtein edit distance using the Wagner-Fischer algorithm
// with space-optimized two-row approach. Inputs must be pre-lowered.
func distance(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}
