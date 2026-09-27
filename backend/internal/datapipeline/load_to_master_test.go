package datapipeline

import (
	"testing"

	"github.com/google/uuid"
)

func TestValidateAccountRefsMissingParty(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	row := AccountStagingRow{
		ID:            uuid.New(),
		HolderPartyID: &id,
	}
	defs := map[string]AttributeDef{
		"grantor_id": {FieldCd: "grantor_id"},
	}
	partyExists := map[string]bool{}
	ws := validateAccountRefs(row, map[string]any{
		"grantor_id": "22222222-2222-2222-2222-222222222222",
	}, defs, partyExists)
	if len(ws) < 2 {
		t.Fatalf("expected warnings for holder + grantor, got %d", len(ws))
	}
}

func TestCollectUUIDRefs(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	rows := []AccountStagingRow{{
		CustodianID:      &id,
		CustomAttributes: []byte(`{"trustee_ids":["22222222-2222-2222-2222-222222222222"]}`),
	}}
	ids := collectUUIDRefs(rows, map[string]AttributeDef{"trustee_ids": {FieldCd: "trustee_ids"}}, []string{"custodian_id"})
	if len(ids) != 2 {
		t.Fatalf("got %v", ids)
	}
}
