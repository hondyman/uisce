package migrations

import (
	"testing"
)

func TestHasTransactionControl(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
		desc     string
	}{
		{
			name:     "real COMMIT at file scope triggers rejection",
			content:  "BEGIN;\nSELECT 1;\nCOMMIT;",
			expected: true,
			desc:     "real transaction bookends",
		},
		{
			name:     "real ROLLBACK at file scope triggers rejection",
			content:  "BEGIN;\nSELECT 1;\nROLLBACK;",
			expected: true,
			desc:     "ROLLBACK also triggers",
		},
		{
			name:     "RAISE inside DO dollar-quote does NOT trigger",
			content:  "DO $$\nDECLARE\nBEGIN\nRAISE NOTICE 'test';\nEND $$;",
			expected: false,
			desc:     "RAISE NOTICE inside DO $$ (like 20260914_014) is runner-safe",
		},
		{
			name:     "post-commit in comment does NOT trigger",
			content:  "-- this is a post-commit hook\nSELECT 1;",
			expected: false,
			desc:     "commit word inside comment must not match \\bCOMMIT\\b",
		},
		{
			name:     "post-commit on same line as -- comment does NOT trigger",
			content:  "-- Seed a sample validation trigger for the Order BO. This is a post-commit\nSELECT 1;",
			expected: false,
			desc:     "the exact case from 20260914_014: post-commit in comment before --",
		},
		{
			name:     "multi-line comment block then real COMMIT triggers",
			content:  "-- Multi-line\n-- comment\n-- block\nBEGIN;\nSELECT 1;\nCOMMIT;",
			expected: true,
			desc:     "comments followed by real COMMIT must trigger",
		},
		{
			name:     "pre-commit in comment does NOT trigger",
			content:  "-- Why pre-commit and not post-commit?\nSELECT 1;",
			expected: false,
			desc:     "pre-commit word in comment is not a transaction statement",
		},
		{
			name:     "empty content does NOT trigger",
			content:  "",
			expected: false,
			desc:     "empty file is safe",
		},
		{
			name:     "only comments does NOT trigger",
			content:  "-- single line comment\n-- another line\n-- post-commit warning",
			expected: false,
			desc:     "comment-only file is safe",
		},
		{
			name:     "commit inside string literal does NOT trigger",
			content:  "SELECT '-- post-commit' AS example;",
			expected: false,
			desc:     "commit inside single-quoted string is not a statement",
		},
		{
			name:     "COMMIT in double-quoted identifier does NOT trigger",
			content:  `SELECT "COMMIT" AS col;`,
			expected: false,
			desc:     "double-quoted identifiers are not transaction statements",
		},
		{
			name:     "mixed case COMMIT triggers",
			content:  "begin;\nselect 1;\ncommit;",
			expected: true,
			desc:     "case-insensitive matching works",
		},
		{
			name:     "mixed case ROLLBACK triggers",
			content:  "begin;\nselect 1;\nrollback;",
			expected: true,
			desc:     "ROLLBACK case-insensitive matching works",
		},
		{
			name:     "real BEGIN at file scope does NOT trigger alone",
			content:  "BEGIN;\nSELECT 1;",
			expected: false,
			desc:     "BEGIN without COMMIT/ROLLBACK is safe per runner check",
		},
		{
			name:     "rollback in comment does NOT trigger",
			content:  "-- TODO: remove rollback handling after testing\nSELECT 1;",
			expected: false,
			desc:     "rollback word in comment is not a statement",
		},
		{
			name:     "COMMIT with trailing space triggers",
			content:  "BEGIN;\nSELECT 1;\nCOMMIT ;",
			expected: true,
			desc:     "COMMIT with trailing space still matches \\bCOMMIT\\b",
		},
		{
			name:     "COMMIT without semicolon triggers",
			content:  "BEGIN;\nSELECT 1;\nCOMMIT",
			expected: true,
			desc:     "\\bCOMMIT\\b matches at end-of-line word boundary, not just before semicolon",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasTransactionControl(tt.content)
			if got != tt.expected {
				t.Errorf("hasTransactionControl() = %v, want %v for case: %s\n  desc: %s\n  content:\n%s",
					got, tt.expected, tt.name, tt.desc, tt.content)
			}
		})
	}
}
