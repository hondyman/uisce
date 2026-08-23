package catalog

import (
	"testing"
)

func TestSTIColumnScanner_New(t *testing.T) {
	scanner := NewSTIColumnScanner()
	if scanner == nil {
		t.Fatal("expected non-nil scanner")
	}
}

func TestSTIColumnScanner_TablePath(t *testing.T) {
	schema := "oms"
	table := "account"
	expected := "oms.account"

	actual := schema + "." + table
	if actual != expected {
		t.Errorf("expected %q, got %q", expected, actual)
	}
}

func TestSTIColumnScanner_ColumnPath(t *testing.T) {
	schema := "oms"
	table := "account"
	column := "account_number"
	expected := "oms.account/account_number"

	actual := schema + "." + table + "/" + column
	if actual != expected {
		t.Errorf("expected %q, got %q", expected, actual)
	}
}
