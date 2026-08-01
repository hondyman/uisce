package fix_test

import (
	"testing"

	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/tag"
	"github.com/hondyman/uisce/backend/internal/fix"
)

func TestTranslateNewOrderSingle_AllTags(t *testing.T) {
	msg := quickfix.NewMessage()
	msg.Header.SetString(tag.MsgType, "D")
	msg.Body.SetString(tag.ClOrdID, "ORD-991")
	msg.Body.SetString(tag.Symbol, "AAPL")
	msg.Body.SetString(tag.OrderQty, "1000")
	msg.Body.SetString(tag.Price, "185.50")
	msg.Body.SetString(tag.Account, "ACC-101")

	trade := fix.TranslateNewOrderSingle(msg)

	if trade.ExternalOrderID != "ORD-991" {
		t.Errorf("expected ExternalOrderID ORD-991, got %s", trade.ExternalOrderID)
	}
	if trade.Symbol != "AAPL" {
		t.Errorf("expected Symbol AAPL, got %s", trade.Symbol)
	}
	if trade.Quantity != 1000 {
		t.Errorf("expected Quantity 1000, got %f", trade.Quantity)
	}
	if trade.Price != 185.50 {
		t.Errorf("expected Price 185.50, got %f", trade.Price)
	}
	if trade.Account != "ACC-101" {
		t.Errorf("expected Account ACC-101, got %s", trade.Account)
	}
}

func TestBuildExecutionReport_Approved(t *testing.T) {
	trade := fix.ExternalTradeItem{
		ExternalOrderID: "ORD-991",
		Symbol:          "AAPL",
		Quantity:        1000,
		Price:           185.50,
	}
	result := fix.EvaluateResult{
		Approved: true,
	}

	report := fix.BuildExecutionReport(trade, result)

	if report == nil {
		t.Fatal("expected non-nil execution report")
	}

	execType, err := report.Body.GetString(tag.ExecType)
	if err != nil {
		t.Fatalf("failed to get ExecType: %v", err)
	}
	if execType != "0" {
		t.Errorf("expected ExecType 0 (New), got %s", execType)
	}

	ordStatus, err := report.Body.GetString(tag.OrdStatus)
	if err != nil {
		t.Fatalf("failed to get OrdStatus: %v", err)
	}
	if ordStatus != "0" {
		t.Errorf("expected OrdStatus 0 (New), got %s", ordStatus)
	}
}

func TestBuildExecutionReport_Rejected(t *testing.T) {
	trade := fix.ExternalTradeItem{
		ExternalOrderID: "ORD-991",
		Symbol:          "AAPL",
		Quantity:        1000,
		Price:           185.50,
	}
	result := fix.EvaluateResult{
		Approved: false,
	}

	report := fix.BuildExecutionReport(trade, result)

	if report == nil {
		t.Fatal("expected non-nil execution report")
	}

	execType, err := report.Body.GetString(tag.ExecType)
	if err != nil {
		t.Fatalf("failed to get ExecType: %v", err)
	}
	if execType != "3" {
		t.Errorf("expected ExecType 3 (Rejected), got %s", execType)
	}
}
