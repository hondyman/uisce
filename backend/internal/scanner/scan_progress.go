package scanner

import (
	"fmt"
	"strings"
	"time"

	"github.com/hondyman/uisce/backend/models"
)

// Progress reporting. Each report's Percent is 0-100 within the extraction phase; the caller maps it into the
// overall scan. Reports are throttled so a scan of thousands of tables does not flood the stream, except forced
// ones (a new step starting), which always go out.
const progressMinInterval = 250 * time.Millisecond

// SetProgressFunc registers a callback that receives progress while ExtractMetadata runs. Optional.
func (s *AnsiScanner) SetProgressFunc(fn func(models.ScanProgress)) { s.progress = fn }

func (s *AnsiScanner) report(force bool, pct float64, item, msg string, completed, total int) {
	if s.progress == nil {
		return
	}
	if !force && time.Since(s.lastReport) < progressMinInterval {
		return
	}
	s.lastReport = time.Now()
	s.progress(models.ScanProgress{Phase: "scanning", Percent: pct, CurrentItem: item, Completed: completed, Total: total, Message: msg})
}

// countTables is how many tables the scan is about to read, so table progress can show "n of m". Zero (unknown)
// on any error; progress then shows a running count instead.
func (s *AnsiScanner) countTables(schemas []string) int {
	if len(schemas) == 0 {
		return 0
	}
	placeholders := make([]string, len(schemas))
	args := make([]interface{}, len(schemas))
	for i, name := range schemas {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = name
	}
	var n int
	q := fmt.Sprintf(`SELECT count(*) FROM information_schema.tables WHERE table_type = 'BASE TABLE' AND table_schema IN (%s)`, strings.Join(placeholders, ", "))
	if err := s.sourceDB.QueryRow(q, args...).Scan(&n); err != nil {
		return 0
	}
	return n
}

// tableProgress reports one more table read. Tables take 0-50% of the extraction.
func (s *AnsiScanner) tableProgress(schema, table string) {
	s.tablesDone++
	pct := 25.0
	msg := fmt.Sprintf("Reading tables: %s.%s (%d read)", schema, table, s.tablesDone)
	if s.tablesTotal > 0 {
		pct = 50 * float64(s.tablesDone) / float64(s.tablesTotal)
		if pct > 50 {
			pct = 50
		}
		msg = fmt.Sprintf("Reading tables: %s.%s (%d of %d)", schema, table, s.tablesDone, s.tablesTotal)
	}
	s.report(false, pct, schema+"."+table, msg, s.tablesDone, s.tablesTotal)
}

// tablesPercent is the table-reading share of the extraction so far (0-50).
func (s *AnsiScanner) tablesPercent() float64 {
	if s.tablesTotal <= 0 {
		return 0
	}
	p := 50 * float64(s.tablesDone) / float64(s.tablesTotal)
	if p > 50 {
		p = 50
	}
	return p
}
