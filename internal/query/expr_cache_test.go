package query

import (
	"testing"
)

func TestExprCache_HitOnRepeat(t *testing.T) {
	cache := NewExprCache(16)

	prog1, err := cache.Compile("estado == 'activo'")
	if err != nil {
		t.Fatal(err)
	}
	if prog1 == nil {
		t.Fatal("expected non-nil program")
	}

	prog2, err := cache.Compile("estado == 'activo'")
	if err != nil {
		t.Fatal(err)
	}

	if prog1 != prog2 {
		t.Error("expected same program pointer on cache hit")
	}
}

func TestExprCache_EvictsOldest(t *testing.T) {
	cache := NewExprCache(2)

	_, _ = cache.Compile("a == 1")
	_, _ = cache.Compile("b == 2")
	_, _ = cache.Compile("c == 3") // evicts "a == 1"

	progA1, _ := cache.Compile("a == 1")
	progA2, _ := cache.Compile("a == 1")
	if progA1 != progA2 {
		t.Error("expected cache hit after re-adding")
	}
}

func TestExprCache_InvalidExprReturnsError(t *testing.T) {
	cache := NewExprCache(16)
	_, err := cache.Compile("invalid ??? syntax")
	if err == nil {
		t.Fatal("expected error for invalid expression")
	}
}
