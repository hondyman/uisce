package scanner

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hondyman/uisce/backend/models"
)

func TestReport_ThrottlesRoutineEventsButNotForcedOnes(t *testing.T) {
	var got []models.ScanProgress
	s := &AnsiScanner{}
	s.SetProgressFunc(func(p models.ScanProgress) { got = append(got, p) })

	s.report(true, 0, "", "start", 0, 0)
	for i := 0; i < 50; i++ { // a burst well inside the throttle window
		s.report(false, 1, "", "routine", i, 100)
	}
	s.report(true, 50, "", "new step", 0, 0)

	if len(got) != 2 || got[0].Message != "start" || got[1].Message != "new step" {
		t.Fatalf("got %d events %+v; want just the two forced ones", len(got), got)
	}
	if got[0].Phase != "scanning" {
		t.Errorf("phase = %q; want scanning", got[0].Phase)
	}
}

func TestReport_NoCallbackIsANoOp(t *testing.T) {
	(&AnsiScanner{}).report(true, 10, "x", "y", 1, 2) // must not panic
}

func TestTableProgress_ShowsNOfMAndStaysWithinItsShare(t *testing.T) {
	var got []models.ScanProgress
	s := &AnsiScanner{tablesTotal: 4}
	s.SetProgressFunc(func(p models.ScanProgress) { got = append(got, p) })

	s.tableProgress("mdm", "party") // first report always goes out
	if len(got) != 1 {
		t.Fatalf("got %d events", len(got))
	}
	if got[0].Message != "Reading tables: mdm.party (1 of 4)" || got[0].Completed != 1 || got[0].Total != 4 || got[0].Percent != 12.5 {
		t.Errorf("unexpected report: %+v", got[0])
	}
	if got[0].CurrentItem != "mdm.party" {
		t.Errorf("current item = %q", got[0].CurrentItem)
	}

	s.tablesDone = 9 // more tables than the count said (tables created mid-scan): never past the table share
	if p := s.tablesPercent(); p != 50 {
		t.Errorf("percent = %v; want capped at 50", p)
	}
}

func TestTableProgress_UnknownTotalStillReportsARunningCount(t *testing.T) {
	var got []models.ScanProgress
	s := &AnsiScanner{}
	s.SetProgressFunc(func(p models.ScanProgress) { got = append(got, p) })
	s.tableProgress("orm", "security")
	if len(got) != 1 || got[0].Message != "Reading tables: orm.security (1 read)" {
		t.Errorf("got %+v", got)
	}
}

func TestCountTables_ErrorMeansUnknown(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectQuery(`information_schema.tables`).WithArgs("mdm", "orm").WillReturnError(errBoom{})
	s := &AnsiScanner{sourceDB: db}
	if n := s.countTables([]string{"mdm", "orm"}); n != 0 {
		t.Errorf("countTables = %d; want 0 (unknown) on error", n)
	}
	if n := (&AnsiScanner{sourceDB: db}).countTables(nil); n != 0 {
		t.Errorf("no schemas = %d; want 0", n)
	}
}

type errBoom struct{}

func (errBoom) Error() string { return "boom" }
