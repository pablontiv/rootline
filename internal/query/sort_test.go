package query

import (
	"testing"
)

func TestParseSortKeys_SingleFieldDefaultAsc(t *testing.T) {
	keys, err := ParseSortKeys("prioridad")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].Field != "prioridad" {
		t.Errorf("field = %q, want prioridad", keys[0].Field)
	}
	if keys[0].Desc {
		t.Error("expected Desc=false for default")
	}
}

func TestParseSortKeys_SingleFieldExplicitAsc(t *testing.T) {
	keys, err := ParseSortKeys("prioridad:asc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 1 || keys[0].Field != "prioridad" || keys[0].Desc {
		t.Errorf("got %+v, want [{prioridad false}]", keys)
	}
}

func TestParseSortKeys_SingleFieldDesc(t *testing.T) {
	keys, err := ParseSortKeys("impact_score:desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 1 || keys[0].Field != "impact_score" || !keys[0].Desc {
		t.Errorf("got %+v, want [{impact_score true}]", keys)
	}
}

func TestParseSortKeys_MultipleKeys(t *testing.T) {
	keys, err := ParseSortKeys("prioridad:asc,impact_score:desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	if keys[0].Field != "prioridad" || keys[0].Desc {
		t.Errorf("key[0] = %+v, want {prioridad false}", keys[0])
	}
	if keys[1].Field != "impact_score" || !keys[1].Desc {
		t.Errorf("key[1] = %+v, want {impact_score true}", keys[1])
	}
}

func TestParseSortKeys_MixedDefaults(t *testing.T) {
	keys, err := ParseSortKeys("a,b:desc,c:asc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
	if keys[0].Field != "a" || keys[0].Desc {
		t.Errorf("key[0] = %+v", keys[0])
	}
	if keys[1].Field != "b" || !keys[1].Desc {
		t.Errorf("key[1] = %+v", keys[1])
	}
	if keys[2].Field != "c" || keys[2].Desc {
		t.Errorf("key[2] = %+v", keys[2])
	}
}

func TestParseSortKeys_EmptyString(t *testing.T) {
	keys, err := ParseSortKeys("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("expected 0 keys for empty string, got %d", len(keys))
	}
}

func TestParseSortKeys_InvalidDirection(t *testing.T) {
	_, err := ParseSortKeys("field:invalid")
	if err == nil {
		t.Fatal("expected error for invalid direction")
	}
}

func TestParseSortKeys_TooManyColons(t *testing.T) {
	_, err := ParseSortKeys("field:asc:extra")
	if err == nil {
		t.Fatal("expected error for too many colons")
	}
}

func TestParseSortKeys_WhitespaceHandling(t *testing.T) {
	keys, err := ParseSortKeys(" prioridad : asc , impact_score : desc ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	if keys[0].Field != "prioridad" || keys[0].Desc {
		t.Errorf("key[0] = %+v", keys[0])
	}
	if keys[1].Field != "impact_score" || !keys[1].Desc {
		t.Errorf("key[1] = %+v", keys[1])
	}
}
