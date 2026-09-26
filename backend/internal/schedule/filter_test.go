package schedule

import (
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hondyman/uisce/backend/internal/msgcat"
)

func TestLikePatternMatchesLiterally(t *testing.T) {
	cases := map[string]string{
		"":           "",
		"   ":        "",
		"London":     "%London%",
		" close ":    "%close%",
		"100%":       `%100\%%`,
		"ff_product": `%ff\_product%`,
		`back\slash`: `%back\\slash%`,
		"9200-8":     "%9200-8%",
	}
	for in, want := range cases {
		if got := likePattern(in); got != want {
			t.Errorf("likePattern(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRunFilterFrom(t *testing.T) {
	r := httptest.NewRequest("GET", "/runs?q=london&status=failed&kind=data_pipeline&from=2026-09-01&to=2026-09-25&limit=20", nil)
	f, err := runFilterFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	if f.Query != "london" || f.Status != "failed" || f.Kind != "data_pipeline" || f.Limit != 20 {
		t.Errorf("filter = %+v", f)
	}
	if want := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC); !f.From.Equal(want) {
		t.Errorf("From = %v, want %v", f.From, want)
	}
	// "to" is inclusive: runs on the 25th are in, so the bound is the 26th.
	if want := time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC); !f.To.Equal(want) {
		t.Errorf("To = %v, want %v", f.To, want)
	}

	f, err = runFilterFrom(httptest.NewRequest("GET", "/runs", nil))
	if err != nil || !f.From.IsZero() || !f.To.IsZero() || f.Query != "" {
		t.Errorf("empty filter = %+v, %v", f, err)
	}
}

func TestRunFilterFromRejectsBadDate(t *testing.T) {
	for _, q := range []string{"from=yesterday", "to=2026-13-40"} {
		_, err := runFilterFrom(httptest.NewRequest("GET", "/runs?"+q, nil))
		var me *msgcat.Error
		if !errors.As(err, &me) || me.Set != SetSchedule || me.Nbr != 17 {
			t.Errorf("%s: err = %v, want catalog 9200-17", q, err)
		}
	}
}
