package query

import (
	"fmt"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func BenchmarkExecuteExpr(b *testing.B) {
	// Create 50 records with varying frontmatter.
	var records []*extract.Record
	for i := 0; i < 50; i++ {
		estado := "Pending"
		if i%3 == 0 {
			estado = "Completado"
		}
		records = append(records, &extract.Record{
			Path: fmt.Sprintf("doc%03d.md", i),
			Frontmatter: map[string]any{
				"estado": estado,
				"tipo":   fmt.Sprintf("type%d", i%5),
				"index":  i,
			},
		})
	}

	q := &Query{}

	b.Run("eq", func(b *testing.B) {
		for b.Loop() {
			_, err := ExecuteExpr(records, `estado == "Completado"`, q)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("contains", func(b *testing.B) {
		for b.Loop() {
			_, err := ExecuteExpr(records, `tipo contains "type1"`, q)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("and_compound", func(b *testing.B) {
		for b.Loop() {
			_, err := ExecuteExpr(records, `estado == "Pending" and tipo contains "type"`, q)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
