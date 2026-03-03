package main

import (
	"testing"
)

func TestIsExplainIndexFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/tmp/project/README.md", true},
		{"/tmp/project/sub/README.md", true},
		{"/tmp/project/doc1.md", false},
		{"/tmp/project/readme.md", false}, // case-sensitive
		{"/tmp/project/README.txt", false},
	}
	for _, tt := range tests {
		got := isExplainIndexFile(tt.path)
		if got != tt.want {
			t.Errorf("isExplainIndexFile(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestSortedMapKeys(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]any
		want []string
	}{
		{"nil map", nil, nil},
		{"empty map", map[string]any{}, nil},
		{"single key", map[string]any{"a": 1}, []string{"a"}},
		{"sorted order", map[string]any{"b": 2, "a": 1, "c": 3}, []string{"a", "b", "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sortedMapKeys(tt.m)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
