package tenantnetwork

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

const testTenantID = "11111111-1111-1111-1111-111111111111"

type fakeStore struct {
	calls    int
	tenantID string
	cidrs    []string
	err      error
}

func (f *fakeStore) ReplaceTenantAllowlist(_ context.Context, tenantID string, cidrs []string) error {
	f.calls++
	f.tenantID = tenantID
	f.cidrs = cidrs
	return f.err
}

func TestEmptyEntriesSkipsAndMakesNoStoreCall(t *testing.T) {
	st := &fakeStore{}
	res, err := (&Activities{Store: st}).ApplyIpAllowlist(context.Background(), Input{TenantID: testTenantID})
	if err != nil {
		t.Fatalf("ApplyIpAllowlist: %v", err)
	}
	if res.Applied || st.calls != 0 {
		t.Fatalf("empty input applied=%v calls=%d, want skipped", res.Applied, st.calls)
	}
}

func TestEntriesNormalizeToCanonicalCIDRs(t *testing.T) {
	st := &fakeStore{}
	in := Input{TenantID: testTenantID, Entries: []string{"10.0.0.5", "192.168.1.0/24", "2001:db8::1", "10.0.0.5"}}
	res, err := (&Activities{Store: st}).ApplyIpAllowlist(context.Background(), in)
	if err != nil {
		t.Fatalf("ApplyIpAllowlist: %v", err)
	}
	want := []string{"10.0.0.5/32", "192.168.1.0/24", "2001:db8::1/128"}
	if !reflect.DeepEqual(res.Normalized, want) {
		t.Fatalf("normalized = %v, want %v", res.Normalized, want)
	}
	if st.tenantID != testTenantID || !reflect.DeepEqual(st.cidrs, want) {
		t.Fatalf("store got tenant=%q cidrs=%v", st.tenantID, st.cidrs)
	}
}

func TestIdempotentReplaceGivesSameStoreCall(t *testing.T) {
	st := &fakeStore{}
	a := &Activities{Store: st}
	in := Input{TenantID: testTenantID, Entries: []string{"10.1.0.0/16", "10.0.0.1"}}
	if _, err := a.ApplyIpAllowlist(context.Background(), in); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	first := append([]string(nil), st.cidrs...)
	if _, err := a.ApplyIpAllowlist(context.Background(), in); err != nil {
		t.Fatalf("second apply: %v", err)
	}
	if !reflect.DeepEqual(first, st.cidrs) {
		t.Fatalf("retry changed the write: %v then %v", first, st.cidrs)
	}
}

func TestRejectsBadEntriesBeforeStore(t *testing.T) {
	tests := []struct {
		name  string
		entry string
	}{
		{"wildcard", "*"},
		{"semicolon", "10.0.0.1; DROP"},
		{"quote", `10.0.0.1"`},
		{"newline", "10.0.0.1\n"},
		{"space", " 10.0.0.1"},
		{"empty", ""},
		{"bad octet", "10.0.0.256"},
		{"bad prefix", "10.0.0.0/33"},
		{"bad v6", "2001:::1"},
		{"hostname", "example.com"},
		{"unicode digits", "١٠.0.0.1"},
		{"overlong", strings.Repeat("1", 65)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := &fakeStore{}
			_, err := (&Activities{Store: st}).ApplyIpAllowlist(context.Background(), Input{TenantID: testTenantID, Entries: []string{"10.2.0.0/16", tc.entry}})
			if err == nil {
				t.Fatalf("accepted entry %q", tc.entry)
			}
			if st.calls != 0 {
				t.Fatal("store written for a rejected entry")
			}
		})
	}
}

func TestRejectsTooManyEntries(t *testing.T) {
	entries := make([]string, maxEntries+1)
	for i := range entries {
		entries[i] = "10.0.0.1"
	}
	st := &fakeStore{}
	if _, err := (&Activities{Store: st}).ApplyIpAllowlist(context.Background(), Input{TenantID: testTenantID, Entries: entries}); err == nil {
		t.Fatal("accepted too many entries")
	}
	if st.calls != 0 {
		t.Fatal("store written for too many entries")
	}
}

func TestRejectsInvalidTenantID(t *testing.T) {
	for _, id := range []string{"", "not-a-uuid", "11111111-1111-1111-1111-111111111111; drop"} {
		st := &fakeStore{}
		if _, err := (&Activities{Store: st}).ApplyIpAllowlist(context.Background(), Input{TenantID: id, Entries: []string{"10.0.0.1"}}); err == nil {
			t.Errorf("accepted tenant ID %q", id)
		}
		if st.calls != 0 {
			t.Errorf("store written for tenant ID %q", id)
		}
	}
}

func TestStoreFailureIsGenericAndKeepsNoEntries(t *testing.T) {
	st := &fakeStore{err: errors.New("store echoed 10.9.9.9 for tenant " + testTenantID)}
	_, err := (&Activities{Store: st}).ApplyIpAllowlist(context.Background(), Input{TenantID: testTenantID, Entries: []string{"10.9.9.9"}})
	if err == nil {
		t.Fatal("store failure not reported")
	}
	if strings.Contains(err.Error(), "10.9.9.9") || strings.Contains(err.Error(), testTenantID) {
		t.Fatalf("store failure leaked detail: %v", err)
	}
}

func TestWithoutStoreFailsClosed(t *testing.T) {
	if _, err := (&Activities{}).ApplyIpAllowlist(context.Background(), Input{TenantID: testTenantID, Entries: []string{"10.0.0.1"}}); err == nil {
		t.Fatal("ran without a store")
	}
}
