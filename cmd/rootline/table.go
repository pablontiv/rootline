package main

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// renderTable writes headers and rows as an aligned table using tabwriter.
func renderTable(w io.Writer, headers []string, rows [][]string) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	_, _ = fmt.Fprintln(tw, strings.Join(headers, "\t"))
	_, _ = fmt.Fprintln(tw, strings.Repeat("-\t", len(headers)))

	for _, row := range rows {
		_, _ = fmt.Fprintln(tw, strings.Join(row, "\t"))
	}

	_ = tw.Flush()
}
