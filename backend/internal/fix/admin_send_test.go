package fix

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/tag"
)

type adminSendCapture struct {
	called bool
	typ    string
}

func (c *adminSendCapture) Send(msg *quickfix.Message, sessionID quickfix.SessionID) error {
	c.called = true
	c.typ, _ = msg.Header.GetString(tag.MsgType)
	return nil
}

func TestAdminSend_RequiresTokenAndDispatches(t *testing.T) {
	adapter := NewPipelineAdapterForTest(nil, nil, nil, 0, &adminSendCapture{})
	admin, err := NewAdminServer("127.0.0.1:0", "secret", adapter)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(admin.server.Handler)
	defer ts.Close()

	body, _ := json.Marshal(map[string]interface{}{
		"msgType": "D",
		"fields":  map[string]string{"11": "CL-1", "55": "AAPL"},
	})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/sessions/FIX.4.4:UISCE->DEMOAGENT/send", bytes.NewReader(body))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/sessions/FIX.4.4:UISCE->DEMOAGENT/send", bytes.NewReader(body))
	req.Header.Set("X-Fix-Admin-Token", "secret")
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	sender, ok := adapter.sender.(*adminSendCapture)
	if !ok || !sender.called {
		t.Fatal("expected adapter sender to be invoked")
	}
}
