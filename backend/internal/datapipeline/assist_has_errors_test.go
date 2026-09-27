package datapipeline

import (
	"errors"
	"testing"
)

func TestHasErrorsIgnoresWarnings(t *testing.T) {
	if HasErrors([]Issue{{Message: "upload a file before running (uri is empty)", Severity: IssueWarning}}) {
		t.Fatal("warning-only issues must not count as errors")
	}
	if !HasErrors([]Issue{{Message: "broken", Severity: IssueError}}) {
		t.Fatal("error severity must count")
	}
	if !HasErrors([]Issue{{Message: "legacy empty severity"}}) {
		t.Fatal("empty severity must count as error for backward compatibility")
	}
	if !HasErrors([]Issue{
		{Message: "upload a file before running (uri is empty)", Severity: IssueWarning},
		{Message: "map node missing", Severity: IssueError},
	}) {
		t.Fatal("mixed warn+error must still be errors")
	}
}

func TestIssueFromErrorEmptyURIIsWarning(t *testing.T) {
	iss := issueFromError(errors.New(`node "src": uri is required`))
	if iss.Severity != IssueWarning {
		t.Fatalf("severity=%q want warning", iss.Severity)
	}
	if iss.NodeID != "src" {
		t.Fatalf("node_id=%q want src", iss.NodeID)
	}
	if iss.Message != "upload a file before running (uri is empty)" {
		t.Fatalf("message=%q", iss.Message)
	}
}
