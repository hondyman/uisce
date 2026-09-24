package trading

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendFixOrderActivity_PostsToAdminNotSocket(t *testing.T) {
	var gotToken, gotPath, gotType string
	var fields map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-Fix-Admin-Token")
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		var req struct {
			MsgType string            `json:"msgType"`
			Fields  map[string]string `json:"fields"`
		}
		_ = json.Unmarshal(body, &req)
		gotType = req.MsgType
		fields = req.Fields
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"sent"}`))
	}))
	defer srv.Close()

	err := SendFixOrderActivity(context.Background(), FIXOrderInput{
		Order:      Order{OrderID: "o1", Symbol: "AAPL", Quantity: 10, Side: "BUY", Price: 1.5},
		AdminURL:   srv.URL,
		AdminToken: "dev-fix-admin",
		SessionID:  "FIX.4.4:UISCE->DEMOAGENT",
		ClOrdID:    "CL-1",
		Command:    "NewOrderSingle",
	})
	if err != nil {
		t.Fatalf("SendFixOrderActivity: %v", err)
	}
	if gotToken != "dev-fix-admin" {
		t.Fatalf("expected admin token header, got %q", gotToken)
	}
	if gotPath != "/sessions/FIX.4.4:UISCE->DEMOAGENT/send" {
		t.Fatalf("unexpected path %q", gotPath)
	}
	if gotType != "D" {
		t.Fatalf("expected MsgType D, got %s", gotType)
	}
	if fields["11"] != "CL-1" || fields["55"] != "AAPL" {
		t.Fatalf("unexpected fields %#v", fields)
	}
}

func TestSendFixOrderActivity_RequiresAdminURL(t *testing.T) {
	err := SendFixOrderActivity(context.Background(), FIXOrderInput{ClOrdID: "x"})
	if err == nil {
		t.Fatal("expected error when admin_url is empty")
	}
}
