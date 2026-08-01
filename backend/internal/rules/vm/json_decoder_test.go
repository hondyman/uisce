package vm

import (
	"encoding/json"
	"testing"
)

func TestDecodeJSON_FastPath(t *testing.T) {
	syms := NewSymbolDict()
	enums := NewEnumDict()

	syms.Intern("customer.tier")
	syms.Intern("customer.balance")
	syms.Intern("customer.is_active")
	enums.Intern("GOLD")
	syms.Freeze()
	enums.Freeze()

	jsonData := []byte(`{"customer":{"tier":"GOLD","balance":15000.5,"is_active":true,"ignored_field":"skip me"}}`)

	rec, err := DecodeJSON(jsonData, syms, enums)
	if err != nil {
		t.Fatal(err)
	}
	defer PutFastRecord(rec)

	// Resolve symbol IDs by their intern order.
	tierID, _ := syms.Resolve("customer.tier")
	balanceID, _ := syms.Resolve("customer.balance")
	activeID, _ := syms.Resolve("customer.is_active")

	if rec.StrVals[tierID] != "GOLD" {
		t.Errorf("tier: expected GOLD, got %s", rec.StrVals[tierID])
	}
	if rec.Present[tierID]&HasStr == 0 {
		t.Errorf("tier: Present HasStr bit not set")
	}
	if rec.Present[tierID]&HasEnum == 0 {
		t.Errorf("tier: Present HasEnum bit not set (GOLD should be interned)")
	}
	if rec.EnumVals[tierID] == 0 {
		t.Errorf("tier: EnumVals should be non-zero (interned id)")
	}

	if rec.FNumVals[balanceID] != 15000.5 {
		t.Errorf("balance: expected 15000.5, got %f", rec.FNumVals[balanceID])
	}
	if rec.Present[balanceID]&HasFNum == 0 {
		t.Errorf("balance: Present HasFNum bit not set")
	}

	if !rec.BoolVals[activeID] {
		t.Errorf("is_active: expected true")
	}
	if rec.Present[activeID]&HasBool == 0 {
		t.Errorf("is_active: Present HasBool bit not set")
	}
}

func TestDecodeJSON_UnknownFieldIgnored(t *testing.T) {
	syms := NewSymbolDict()
	enums := NewEnumDict()
	syms.Intern("known")
	syms.Freeze()
	enums.Freeze()

	rec, err := DecodeJSON([]byte(`{"known":1,"unknown":2,"also_unknown":3}`), syms, enums)
	if err != nil {
		t.Fatal(err)
	}
	defer PutFastRecord(rec)

	knownID, _ := syms.Resolve("known")
	if rec.NumVals[knownID] != 1 {
		t.Errorf("known: expected 1, got %d", rec.NumVals[knownID])
	}
}

func TestDecodeJSON_DeeplyNested(t *testing.T) {
	syms := NewSymbolDict()
	enums := NewEnumDict()
	syms.Intern("a.b.c.d")
	syms.Freeze()
	enums.Freeze()

	rec, err := DecodeJSON([]byte(`{"a":{"b":{"c":{"d":42}}}}}`), syms, enums)
	if err != nil {
		t.Fatal(err)
	}
	defer PutFastRecord(rec)

	id, _ := syms.Resolve("a.b.c.d")
	if rec.NumVals[id] != 42 {
		t.Errorf("a.b.c.d: expected 42, got %d", rec.NumVals[id])
	}
}

func TestDecodeJSON_NullIgnored(t *testing.T) {
	syms := NewSymbolDict()
	enums := NewEnumDict()
	syms.Intern("x")
	syms.Freeze()
	enums.Freeze()

	rec, err := DecodeJSON([]byte(`{"x":null}`), syms, enums)
	if err != nil {
		t.Fatal(err)
	}
	defer PutFastRecord(rec)

	id, _ := syms.Resolve("x")
	if rec.Present[id]&HasNum != 0 || rec.Present[id]&HasBool != 0 || rec.Present[id]&HasStr != 0 {
		t.Errorf("null should leave Present bits unset, got %d", rec.Present[id])
	}
}

func TestDecodeJSON_Malformed(t *testing.T) {
	syms := NewSymbolDict()
	enums := NewEnumDict()
	syms.Intern("x")
	syms.Freeze()
	enums.Freeze()

	if _, err := DecodeJSON([]byte(`not json`), syms, enums); err == nil {
		t.Error("expected error for non-JSON input")
	}
}

func TestDecodeJSON_ParityWithProject(t *testing.T) {
	// The streaming decoder and Project() must agree on every field
	// for the same input. Round-trip 50 records.
	syms := NewSymbolDict()
	enums := NewEnumDict()
	syms.Intern("customer.tier")
	syms.Intern("customer.balance")
	syms.Intern("customer.is_active")
	enums.Intern("GOLD")
	enums.Intern("SILVER")
	syms.Freeze()
	enums.Freeze()

	for i := 0; i < 50; i++ {
		tier := "GOLD"
		if i%2 == 0 {
			tier = "SILVER"
		}
		jsonData := []byte(`{"customer":{"tier":"` + tier + `","balance":` +
			itoa(10000+i) + `,"is_active":` + boolstr(i%3 == 0) + `}}`)
		mapData := map[string]any{
			"customer": map[string]any{
				"tier":      tier,
				"balance":   float64(10000 + i),
				"is_active": i%3 == 0,
			},
		}

		recStream, err := DecodeJSON(jsonData, syms, enums)
		if err != nil {
			t.Fatalf("iter %d: stream decode failed: %v", i, err)
		}
		recProj := Project(mapData, syms, enums)

		tierID, _ := syms.Resolve("customer.tier")
		balanceID, _ := syms.Resolve("customer.balance")
		activeID, _ := syms.Resolve("customer.is_active")

		if recStream.StrVals[tierID] != recProj.StrVals[tierID] {
			t.Errorf("iter %d: tier mismatch stream=%q proj=%q",
				i, recStream.StrVals[tierID], recProj.StrVals[tierID])
		}
		if recStream.NumVals[balanceID] != recProj.NumVals[balanceID] {
			t.Errorf("iter %d: balance mismatch stream=%d proj=%d",
				i, recStream.NumVals[balanceID], recProj.NumVals[balanceID])
		}
		if recStream.BoolVals[activeID] != recProj.BoolVals[activeID] {
			t.Errorf("iter %d: is_active mismatch stream=%v proj=%v",
				i, recStream.BoolVals[activeID], recProj.BoolVals[activeID])
		}
		if recStream.Present[tierID] != recProj.Present[tierID] ||
			recStream.Present[balanceID] != recProj.Present[balanceID] ||
			recStream.Present[activeID] != recProj.Present[activeID] {
			t.Errorf("iter %d: Present bits mismatch", i)
		}

		PutFastRecord(recStream)
		PutFastRecord(recProj)
	}
}

func itoa(n int) string {
	return string(rune('0' + n/10000%10)) + string(rune('0' + n/1000%10)) +
		string(rune('0' + n/100%10)) + string(rune('0' + n/10%10)) + string(rune('0' + n%10))
}

func boolstr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func BenchmarkDecodeJSON(b *testing.B) {
	syms := NewSymbolDict()
	enums := NewEnumDict()
	syms.Intern("customer.tier")
	syms.Intern("customer.balance")
	enums.Intern("GOLD")
	syms.Freeze()
	enums.Freeze()

	jsonData := []byte(`{"customer":{"tier":"GOLD","balance":15000}}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec, _ := DecodeJSON(jsonData, syms, enums)
		PutFastRecord(rec)
	}
}

func BenchmarkDecodeJSON_vs_StdLib(b *testing.B) {
	jsonData := []byte(`{"customer":{"tier":"GOLD","balance":15000}}`)

	b.Run("StdLib_Map", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var m map[string]any
			_ = json.Unmarshal(jsonData, &m)
		}
	})

	b.Run("VM_Decoder", func(b *testing.B) {
		syms := NewSymbolDict()
		enums := NewEnumDict()
		syms.Intern("customer.tier")
		syms.Intern("customer.balance")
		enums.Intern("GOLD")
		syms.Freeze()
		enums.Freeze()

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			rec, _ := DecodeJSON(jsonData, syms, enums)
			PutFastRecord(rec)
		}
	})
}