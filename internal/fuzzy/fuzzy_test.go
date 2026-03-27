package fuzzy

import (
	"testing"
)

func TestDistance(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"abc", "abc", 0},
		{"kitten", "sitting", 3},
		{"Pending", "Peding", 1},
		{"estado", "estdo", 1},
		{"Completed", "Completado", 2},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			got := Distance(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("Distance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestMatch(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		candidates []string
		want       string
	}{
		{
			name:       "close typo",
			input:      "Peding",
			candidates: []string{"Pending", "Completed"},
			want:       "Pending",
		},
		{
			name:       "no match within threshold",
			input:      "xyz",
			candidates: []string{"Pending", "Completed"},
			want:       "",
		},
		{
			name:       "empty candidates",
			input:      "Pending",
			candidates: nil,
			want:       "",
		},
		{
			name:       "exact match returns it",
			input:      "Pending",
			candidates: []string{"Pending", "Completed"},
			want:       "Pending",
		},
		{
			name:       "case insensitive",
			input:      "pending",
			candidates: []string{"Pending", "Completed"},
			want:       "Pending",
		},
		{
			name:       "spanish field typo",
			input:      "estdo",
			candidates: []string{"estado", "tipo", "titulo"},
			want:       "estado",
		},
		{
			name:       "long string threshold",
			input:      "Especificado",
			candidates: []string{"Especificada", "Completado"},
			want:       "Especificada",
		},
		{
			name:       "too far even for long string",
			input:      "abcdefghijkl",
			candidates: []string{"xyzxyzxyzxyz"},
			want:       "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Match(tt.input, tt.candidates)
			if got != tt.want {
				t.Errorf("Match(%q, %v) = %q, want %q", tt.input, tt.candidates, got, tt.want)
			}
		})
	}
}

func TestMatchN(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		candidates []string
		n          int
		want       []string
	}{
		{
			name:       "multiple close matches sorted by distance",
			input:      "Pendng",
			candidates: []string{"Pending", "Panding", "Completed"},
			n:          3,
			want:       []string{"Pending", "Panding"},
		},
		{
			name:       "limit to n",
			input:      "estado",
			candidates: []string{"estados", "estado2", "estador", "estilo"},
			n:          2,
			want:       []string{"estados", "estado2"},
		},
		{
			name:       "no matches",
			input:      "xyz",
			candidates: []string{"Pending", "Completed"},
			n:          3,
			want:       nil,
		},
		{
			name:       "empty candidates",
			input:      "test",
			candidates: nil,
			n:          3,
			want:       nil,
		},
		{
			name:       "n zero",
			input:      "test",
			candidates: []string{"test"},
			n:          0,
			want:       nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchN(tt.input, tt.candidates, tt.n)
			if len(got) != len(tt.want) {
				t.Fatalf("MatchN(%q, %v, %d) returned %d results, want %d: %v", tt.input, tt.candidates, tt.n, len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("MatchN result[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestThreshold(t *testing.T) {
	// Short string (len=3): threshold = max(2, 3/3) = 2
	// "abc" vs "abd" (distance=1) should match
	got := Match("abc", []string{"abd"})
	if got != "abd" {
		t.Errorf("short string: Match(\"abc\", [\"abd\"]) = %q, want \"abd\"", got)
	}

	// "abc" vs "axyz" (distance=3) should NOT match (threshold=2)
	got = Match("abc", []string{"axyz"})
	if got != "" {
		t.Errorf("short string over threshold: Match(\"abc\", [\"axyz\"]) = %q, want \"\"", got)
	}

	// Long string (len=12): threshold = max(2, 12/3) = 4
	// "abcdefghijkl" vs "abcdefghijXX" (distance=2) should match
	got = Match("abcdefghijkl", []string{"abcdefghijXX"})
	if got != "abcdefghijXX" {
		t.Errorf("long string: Match(\"abcdefghijkl\", [\"abcdefghijXX\"]) = %q, want \"abcdefghijXX\"", got)
	}
}
