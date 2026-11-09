// Package reports builds titled, timestamped, tabular reports from a user's
// career snapshot. Each report has a title, a generated-at date-time stamp,
// named columns, and data rows — viewable in the UI or exported as CSV.
package reports

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Column names one report column.
type Column struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// Report is a rendered tabular report.
type Report struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	GeneratedAt time.Time  `json:"generatedAt"`
	Columns     []Column   `json:"columns"`
	Rows        [][]string `json:"rows"`
}

// MarshalJSON encodes nil Rows as [] instead of null.
func (r *Report) MarshalJSON() ([]byte, error) {
	type plain Report
	out := plain(*r)
	if out.Rows == nil {
		out.Rows = [][]string{}
	}
	return json.Marshal(out)
}

// CSV renders the report with its title and timestamp as leading comment
// rows, then a header row, then data.
func (r *Report) CSV() ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	meta := [][]string{
		{r.Title},
		{"Generated at", r.GeneratedAt.Format(time.RFC3339)},
		{},
	}
	header := make([]string, len(r.Columns))
	for i, c := range r.Columns {
		header[i] = c.Label
	}

	for _, row := range append(meta, header) {
		if err := w.Write(row); err != nil {
			return nil, fmt.Errorf("writing csv: %w", err)
		}
	}
	for _, row := range r.Rows {
		if err := w.Write(escapeFormulas(row)); err != nil {
			return nil, fmt.Errorf("writing csv: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("flushing csv: %w", err)
	}
	return buf.Bytes(), nil
}

// escapeFormulas neutralizes CSV formula injection: user-entered cells that
// start with =, +, -, or @ would otherwise execute as formulas when opened
// in a spreadsheet, so they are prefixed with a single quote.
func escapeFormulas(row []string) []string {
	out := make([]string, len(row))
	for i, cell := range row {
		if strings.HasPrefix(cell, "=") || strings.HasPrefix(cell, "+") ||
			strings.HasPrefix(cell, "-") || strings.HasPrefix(cell, "@") {
			cell = "'" + cell
		}
		out[i] = cell
	}
	return out
}

// Info describes an available report for the report list endpoint.
type Info struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}
