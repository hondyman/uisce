package temporal

import "testing"

// TestCancelResponseMsgTypes_V1 is a regression canary for the cancel-response
// matcher. MT548 must NOT be added to this list without also implementing
// cancel-status parsing inside hasCancelResponse — MT548 is the standard
// per-settlement status message, so matching any MT548 would falsely resolve
// CANCEL_PENDING rows to terminal CANCELLED (the "lying CANCELLED" bug).
//
// If you are adding MT548 here, you must first:
//   1. Parse the MT548 body in hasCancelResponse and require a CANC event, and
//   2. Delete this test (its purpose is fulfilled).
func TestCancelResponseMsgTypes_V1(t *testing.T) {
	if len(CancelResponseMsgTypes) != 1 || CancelResponseMsgTypes[0] != "camt.029" {
		t.Fatalf(
			"CancelResponseMsgTypes = %v — growing this list requires cancel-status "+
				"parsing in hasCancelResponse first (see its doc). MT548 without "+
				"parsing resolves live settlements to CANCELLED on any status message.",
			CancelResponseMsgTypes,
		)
	}
}
