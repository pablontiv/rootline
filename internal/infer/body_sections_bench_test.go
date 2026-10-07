package infer

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func BenchmarkDetectSectionPatternsHierarchical(b *testing.B) {
	for _, count := range []int{100, 400, 800} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			records := makeHierarchicalSectionRecords(count)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				inferences, err := DetectSectionPatterns(records, 1)
				if err != nil {
					b.Fatal(err)
				}
				if len(inferences) != 1 || inferences[0].SourceDirective != `body.section["### Notes"]` {
					b.Fatalf("unexpected inferences: %+v", inferences)
				}
			}
		})
	}
}

func makeHierarchicalSectionRecords(count int) []*extract.Record {
	roots := []string{"Product", "Operations"}
	parents := []string{"Planning", "Delivery"}
	records := make([]*extract.Record, count)
	for i := range records {
		route := i % 4
		records[i] = &extract.Record{
			Path: fmt.Sprintf("group-%d/record-%04d.md", route, i),
			Body: "section fixture",
			BodySections: []extract.Section{
				{Level: 1, Heading: roots[route/2]},
				{Level: 2, Heading: parents[route%2]},
				{Level: 3, Heading: "Notes"},
			},
		}
	}
	return records
}
