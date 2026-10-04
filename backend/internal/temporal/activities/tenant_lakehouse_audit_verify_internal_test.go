package activities

import (
	"fmt"
	"testing"
	"time"

	"github.com/hondyman/uisce/backend/internal/lakehouse/infra"
	"github.com/hondyman/uisce/backend/internal/lakehouse/registry"
	"github.com/stretchr/testify/require"
)

var vt0 = time.Date(2026, 10, 3, 1, 2, 3, 456789000, time.UTC)

// pair builds the same chain of n entries as alpha holds it and as the copy holds it.
func pair(n int) ([]registry.AuditEntry, []infra.AuditRow) {
	var a []registry.AuditEntry
	var c []infra.AuditRow
	prev := genesisHash
	for i := 1; i <= n; i++ {
		h := fmt.Sprintf("h%d", i)
		after := fmt.Sprintf(`{"n":%d,"k":"v"}`, i)
		a = append(a, registry.AuditEntry{ID: int64(i), At: vt0.Add(time.Duration(i) * time.Second), ActorID: "alice", ActorRole: "global_admin",
			Action: "configured", After: []byte(after), PrevHash: prev, Hash: h})
		c = append(c, infra.AuditRow{ID: int64(i), At: vt0.Add(time.Duration(i) * time.Second), ActorID: "alice", ActorRole: "global_admin",
			Action: "configured", AfterJSON: after, PrevHash: prev, Hash: h})
		prev = h
	}
	return a, c
}

func TestCompare_AnIdenticalCopyVerifies(t *testing.T) {
	a, c := pair(5)
	p := compareAuditPage(AuditVerifyCursor{}, a, c, 100)
	require.Nil(t, p.Finding)
	require.True(t, p.Done)
	require.Equal(t, AuditVerifyCursor{AfterID: 5, PrevHash: "h5", Rows: 5}, p.Cursor)
}

// The copy is allowed to lag: it runs every 15 minutes. Rows alpha holds beyond the copy are pending.
func TestCompare_AlphaAheadOfTheCopyIsPendingNotAFinding(t *testing.T) {
	a, c := pair(8)
	p := compareAuditPage(AuditVerifyCursor{}, a, c[:5], 100)
	require.Nil(t, p.Finding)
	require.True(t, p.Done)
	require.EqualValues(t, 5, p.Cursor.Rows)
	require.EqualValues(t, 5, p.Cursor.AfterID)
}

func TestCompare_AnEmptyCopyIsDoneWithNothingVerified(t *testing.T) {
	a, _ := pair(3)
	p := compareAuditPage(AuditVerifyCursor{}, a, nil, 100)
	require.Nil(t, p.Finding)
	require.True(t, p.Done)
	require.Zero(t, p.Cursor.Rows, "nothing was compared, so nothing may be reported as verified")
}

func TestCompare_AHoleBelowTheCopysHighestIdIsAFinding(t *testing.T) {
	a, c := pair(6)
	c = append(c[:2], c[3:]...) // the copy lacks id 3
	p := compareAuditPage(AuditVerifyCursor{}, a, c, 100)
	require.NotNil(t, p.Finding)
	require.Equal(t, FindingMissingInCopy, p.Finding.Kind)
	require.EqualValues(t, 3, p.Finding.ID)
	require.EqualValues(t, 2, p.Cursor.AfterID, "the cursor stops at the last good entry")
}

func TestCompare_AnEntryAlphaDoesNotHaveIsAFinding(t *testing.T) {
	a, c := pair(4)
	a = append(a[:1], a[2:]...) // alpha lacks id 2
	p := compareAuditPage(AuditVerifyCursor{}, a, c, 100)
	require.NotNil(t, p.Finding)
	require.Equal(t, FindingExtraInCopy, p.Finding.Kind)
	require.EqualValues(t, 2, p.Finding.ID)
}

func TestCompare_AnEntryTheCopyHasAndAlphaLacksAtTheEnd(t *testing.T) {
	a, c := pair(4)
	p := compareAuditPage(AuditVerifyCursor{}, a[:3], c, 100)
	require.NotNil(t, p.Finding)
	require.Equal(t, FindingExtraInCopy, p.Finding.Kind)
	require.EqualValues(t, 4, p.Finding.ID)
}

func TestCompare_EveryFieldIsCompared(t *testing.T) {
	mut := map[string]func(*infra.AuditRow){
		"at":         func(r *infra.AuditRow) { r.At = r.At.Add(time.Microsecond) },
		"actor_id":   func(r *infra.AuditRow) { r.ActorID = "mallory" },
		"actor_role": func(r *infra.AuditRow) { r.ActorRole = "viewer" },
		"action":     func(r *infra.AuditRow) { r.Action = "deleted" },
		"before":     func(r *infra.AuditRow) { r.BeforeJSON = `{"x":1}` },
		"after":      func(r *infra.AuditRow) { r.AfterJSON = `{"n":2,"k":"CHANGED"}` },
		"prev_hash":  func(r *infra.AuditRow) { r.PrevHash = "forged" },
		"hash":       func(r *infra.AuditRow) { r.Hash = "forged" },
	}
	for field, m := range mut {
		t.Run(field, func(t *testing.T) {
			a, c := pair(3)
			m(&c[1])
			p := compareAuditPage(AuditVerifyCursor{}, a, c, 100)
			require.NotNil(t, p.Finding)
			require.Equal(t, FindingDiffers, p.Finding.Kind)
			require.EqualValues(t, 2, p.Finding.ID)
			require.Contains(t, p.Finding.Detail, field)
		})
	}
}

func TestCompare_AFindingNeverCarriesAValue(t *testing.T) {
	a, c := pair(2)
	c[1].AfterJSON = `{"ssn":"123-45-6789"}`
	p := compareAuditPage(AuditVerifyCursor{}, a, c, 100)
	require.NotNil(t, p.Finding)
	require.NotContains(t, p.Finding.Detail, "123-45-6789", "audit payloads can hold personal data and the report is logged")
}

func TestCompare_JSONIsComparedAsDataAndTimeAtTheMicrosecond(t *testing.T) {
	a, c := pair(2)
	c[0].AfterJSON = `{ "k": "v", "n": 1 }`           // key order and spacing are not meaning
	a[1].At = a[1].At.Add(400 * time.Nanosecond)      // beyond what the database keeps
	a[1].Before, c[1].BeforeJSON = []byte("null"), "" // two spellings of nothing
	require.Nil(t, compareAuditPage(AuditVerifyCursor{}, a, c, 100).Finding)
}

func TestCompare_JSONThatCannotBeReadIsADifferenceNeverAPass(t *testing.T) {
	a, c := pair(1)
	a[0].After, c[0].AfterJSON = []byte("{nope"), "{nope-either"
	p := compareAuditPage(AuditVerifyCursor{}, a, c, 100)
	require.NotNil(t, p.Finding)
	require.Equal(t, FindingDiffers, p.Finding.Kind)
}

// A copy that reproduces alpha's rows but links them wrongly is caught on the chain, not by equality.
func TestCompare_AChainBreakThatEqualityWouldMiss(t *testing.T) {
	a, c := pair(3)
	a[1].PrevHash, c[1].PrevHash = "not-h1", "not-h1" // alpha and the copy agree, and both are broken
	p := compareAuditPage(AuditVerifyCursor{}, a, c, 100)
	require.NotNil(t, p.Finding)
	require.Equal(t, FindingChainBreak, p.Finding.Kind)
	require.EqualValues(t, 2, p.Finding.ID)
}

func TestCompare_TheFirstEntryMustLinkToGenesis(t *testing.T) {
	a, c := pair(2)
	a[0].PrevHash, c[0].PrevHash = "something", "something"
	p := compareAuditPage(AuditVerifyCursor{}, a, c, 100)
	require.NotNil(t, p.Finding)
	require.Equal(t, FindingChainBreak, p.Finding.Kind)
}

// Paging: both pages are cut at the limit, and the walk must resume exactly where it stopped, with the
// link carried across the boundary.
func TestCompare_PagesCarryTheChainAcrossBoundaries(t *testing.T) {
	a, c := pair(10)
	cur := AuditVerifyCursor{}
	for pages := 0; ; pages++ {
		require.Less(t, pages, 20, "paging must make progress")
		ar, cr := after(a, cur.AfterID, 3), afterRows(c, cur.AfterID, 3)
		p := compareAuditPage(cur, ar, cr, 3)
		require.Nil(t, p.Finding)
		cur = p.Cursor
		if p.Done {
			break
		}
	}
	require.EqualValues(t, 10, cur.Rows)
	require.EqualValues(t, 10, cur.AfterID)
}

func TestCompare_ABreakOnALaterPageIsFound(t *testing.T) {
	a, c := pair(10)
	c[6].Hash, a[6].Hash = "x", "x" // entry 8's prev_hash (h7) no longer equals entry 7's hash
	cur := AuditVerifyCursor{}
	var found *AuditFinding
	for pages := 0; pages < 20 && found == nil; pages++ {
		p := compareAuditPage(cur, after(a, cur.AfterID, 3), afterRows(c, cur.AfterID, 3), 3)
		cur, found = p.Cursor, p.Finding
		if p.Done {
			break
		}
	}
	require.NotNil(t, found)
	require.Equal(t, FindingChainBreak, found.Kind)
	require.EqualValues(t, 8, found.ID)
}

// A full page on one side and a short one on the other must not be misread as a hole.
func TestCompare_ADifferentPageCutOnEachSideIsNotAHole(t *testing.T) {
	a, c := pair(9)
	// alpha's page is cut at 3, the copy's page holds 6: only compare through id 3.
	p := compareAuditPage(AuditVerifyCursor{}, a[:3], c[:6], 3)
	require.Nil(t, p.Finding)
	require.False(t, p.Done)
	require.EqualValues(t, 3, p.Cursor.AfterID)
}

func after(a []registry.AuditEntry, id int64, n int) []registry.AuditEntry {
	var out []registry.AuditEntry
	for _, e := range a {
		if e.ID > id && len(out) < n {
			out = append(out, e)
		}
	}
	return out
}

func afterRows(c []infra.AuditRow, id int64, n int) []infra.AuditRow {
	var out []infra.AuditRow
	for _, e := range c {
		if e.ID > id && len(out) < n {
			out = append(out, e)
		}
	}
	return out
}
