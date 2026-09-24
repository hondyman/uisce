package fix

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/quickfix/config"
	"github.com/quickfixgo/tag"
)

const (
	DemoSenderCompID = "UISCE"
	DemoTargetCompID = "DEMOAGENT"
)

// DemoSessionID is the acceptor-side session the demo agent logs on to.
func DemoSessionID() string {
	return "FIX.4.4:" + DemoSenderCompID + "->" + DemoTargetCompID
}

// ApplyDemoAcceptorSession adds a FIX 4.4 session block for the sample
// initiator. Call when FIX_DEMO_AGENT is enabled so the acceptor has a
// session to log the demo agent onto.
func ApplyDemoAcceptorSession(settings *quickfix.Settings, acceptPort string) error {
	if settings == nil {
		return fmt.Errorf("settings is nil")
	}
	g := settings.GlobalSettings()
	g.Set("ConnectionType", "acceptor")
	g.Set(config.BeginString, "FIX.4.4")
	g.Set(config.HeartBtInt, "30")
	g.Set(config.ResetOnLogon, "Y")
	g.Set(config.ResetOnLogout, "Y")
	g.Set(config.ResetOnDisconnect, "Y")
	g.Set(config.StartTime, "00:00:00")
	g.Set(config.EndTime, "00:00:00")
	g.Set(config.SocketAcceptHost, "0.0.0.0")
	if acceptPort != "" {
		g.Set(config.SocketAcceptPort, acceptPort)
	}

	session := quickfix.NewSessionSettings()
	session.Set(config.BeginString, "FIX.4.4")
	session.Set(config.SenderCompID, DemoSenderCompID)
	session.Set(config.TargetCompID, DemoTargetCompID)
	session.Set(config.HeartBtInt, "30")
	session.Set(config.ResetOnLogon, "Y")
	session.Set(config.ResetOnLogout, "Y")
	session.Set(config.ResetOnDisconnect, "Y")
	session.Set(config.StartTime, "00:00:00")
	session.Set(config.EndTime, "00:00:00")
	if _, err := settings.AddSession(session); err != nil {
		return fmt.Errorf("add demo session: %w", err)
	}
	return nil
}

// DemoAgent is an in-process quickfix initiator that pretends to be a
// broker. It auto-acks NewOrderSingle with a fill ExecutionReport and
// handles cancel / cancel-replace. Dev/demo only — never production.
type DemoAgent struct {
	initiator *quickfix.Initiator
}

type demoApp struct{}

func (demoApp) OnCreate(sessionID quickfix.SessionID) {
	log.Printf("[FIX demo] session created %s", sessionID)
}
func (demoApp) OnLogon(sessionID quickfix.SessionID) {
	log.Printf("[FIX demo] logon %s", sessionID)
}
func (demoApp) OnLogout(sessionID quickfix.SessionID) {
	log.Printf("[FIX demo] logout %s", sessionID)
}
func (demoApp) ToAdmin(*quickfix.Message, quickfix.SessionID) {}
func (demoApp) ToApp(*quickfix.Message, quickfix.SessionID) error {
	return nil
}
func (demoApp) FromAdmin(*quickfix.Message, quickfix.SessionID) quickfix.MessageRejectError {
	return nil
}

func (demoApp) FromApp(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	msgType, _ := msg.Header.GetString(tag.MsgType)
	switch msgType {
	case "D":
		return sendDemoFill(msg, sessionID)
	case "F":
		return sendDemoCancelAck(msg, sessionID)
	case "G":
		return sendDemoReplaceAck(msg, sessionID)
	default:
		return nil
	}
}

func sendDemoFill(nos *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	clOrdID, _ := nos.Body.GetString(tag.ClOrdID)
	symbol, _ := nos.Body.GetString(tag.Symbol)
	side, _ := nos.Body.GetString(tag.Side)
	qty, _ := nos.Body.GetString(tag.OrderQty)
	px, _ := nos.Body.GetString(tag.Price)
	if px == "" {
		px = "1"
	}
	er := quickfix.NewMessage()
	er.Header.SetString(tag.BeginString, "FIX.4.4")
	er.Header.SetString(tag.MsgType, "8")
	er.Body.SetString(tag.ClOrdID, clOrdID)
	er.Body.SetString(tag.OrderID, clOrdID)
	er.Body.SetString(tag.ExecID, "DEMO-"+clOrdID)
	er.Body.SetString(tag.ExecType, "2")  // Fill
	er.Body.SetString(tag.OrdStatus, "2") // Filled
	er.Body.SetString(tag.Symbol, symbol)
	er.Body.SetString(tag.Side, side)
	er.Body.SetString(tag.OrderQty, qty)
	er.Body.SetString(tag.LastQty, qty)
	er.Body.SetString(tag.LastPx, px)
	er.Body.SetString(tag.CumQty, qty)
	er.Body.SetString(tag.AvgPx, px)
	er.Body.SetString(tag.LeavesQty, "0")
	if err := quickfix.SendToTarget(er, sessionID); err != nil {
		log.Printf("[FIX demo] fill send failed: %v", err)
	} else {
		log.Printf("[FIX demo] filled ClOrdID=%s qty=%s px=%s", clOrdID, qty, px)
	}
	return nil
}

func sendDemoCancelAck(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	clOrdID, _ := msg.Body.GetString(tag.ClOrdID)
	orig, _ := msg.Body.GetString(tag.OrigClOrdID)
	er := quickfix.NewMessage()
	er.Header.SetString(tag.BeginString, "FIX.4.4")
	er.Header.SetString(tag.MsgType, "8")
	er.Body.SetString(tag.ClOrdID, clOrdID)
	er.Body.SetString(tag.OrigClOrdID, orig)
	er.Body.SetString(tag.OrderID, orig)
	er.Body.SetString(tag.ExecID, "DEMO-CXL-"+clOrdID)
	er.Body.SetString(tag.ExecType, "4")
	er.Body.SetString(tag.OrdStatus, "4")
	if err := quickfix.SendToTarget(er, sessionID); err != nil {
		log.Printf("[FIX demo] cancel ack failed: %v", err)
	}
	return nil
}

func sendDemoReplaceAck(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	clOrdID, _ := msg.Body.GetString(tag.ClOrdID)
	qty, _ := msg.Body.GetString(tag.OrderQty)
	px, _ := msg.Body.GetString(tag.Price)
	if px == "" {
		px = "1"
	}
	er := quickfix.NewMessage()
	er.Header.SetString(tag.BeginString, "FIX.4.4")
	er.Header.SetString(tag.MsgType, "8")
	er.Body.SetString(tag.ClOrdID, clOrdID)
	er.Body.SetString(tag.OrderID, clOrdID)
	er.Body.SetString(tag.ExecID, "DEMO-RPL-"+clOrdID)
	er.Body.SetString(tag.ExecType, "5")
	er.Body.SetString(tag.OrdStatus, "5")
	er.Body.SetString(tag.LastQty, qty)
	er.Body.SetString(tag.LastPx, px)
	if err := quickfix.SendToTarget(er, sessionID); err != nil {
		log.Printf("[FIX demo] replace ack failed: %v", err)
	}
	return nil
}

// StartDemoAgent launches the sample initiator against the local acceptor.
func StartDemoAgent(ctx context.Context, connectHost, connectPort string) (*DemoAgent, error) {
	if connectHost == "" {
		connectHost = "127.0.0.1"
	}
	if connectPort == "" {
		connectPort = "8980"
	}
	if _, err := strconv.Atoi(connectPort); err != nil {
		return nil, fmt.Errorf("invalid connect port %q", connectPort)
	}

	settings := quickfix.NewSettings()
	g := settings.GlobalSettings()
	g.Set("ConnectionType", "initiator")
	g.Set(config.HeartBtInt, "30")
	g.Set(config.ReconnectInterval, "2")
	g.Set(config.ResetOnLogon, "Y")
	g.Set(config.ResetOnLogout, "Y")
	g.Set(config.ResetOnDisconnect, "Y")
	g.Set(config.StartTime, "00:00:00")
	g.Set(config.EndTime, "00:00:00")
	g.Set(config.SocketConnectHost, connectHost)
	g.Set(config.SocketConnectPort, connectPort)

	session := quickfix.NewSessionSettings()
	session.Set(config.BeginString, "FIX.4.4")
	session.Set(config.SenderCompID, DemoTargetCompID)
	session.Set(config.TargetCompID, DemoSenderCompID)
	session.Set(config.HeartBtInt, "30")
	session.Set(config.SocketConnectHost, connectHost)
	session.Set(config.SocketConnectPort, connectPort)
	session.Set(config.ResetOnLogon, "Y")
	session.Set(config.ResetOnLogout, "Y")
	session.Set(config.ResetOnDisconnect, "Y")
	session.Set(config.StartTime, "00:00:00")
	session.Set(config.EndTime, "00:00:00")
	if _, err := settings.AddSession(session); err != nil {
		return nil, err
	}

	store := quickfix.NewMemoryStoreFactory()
	logFactory := quickfix.NewScreenLogFactory()
	initiator, err := quickfix.NewInitiator(demoApp{}, store, settings, logFactory)
	if err != nil {
		return nil, fmt.Errorf("create demo initiator: %w", err)
	}
	if err := initiator.Start(); err != nil {
		return nil, fmt.Errorf("start demo initiator: %w", err)
	}
	agent := &DemoAgent{initiator: initiator}
	go func() {
		<-ctx.Done()
		agent.Stop()
	}()
	// Give logon a moment; callers don't wait for logon to succeed.
	time.Sleep(50 * time.Millisecond)
	log.Printf("[FIX demo] initiator started -> %s:%s session %s", connectHost, connectPort, DemoSessionID())
	return agent, nil
}

func (a *DemoAgent) Stop() {
	if a != nil && a.initiator != nil {
		a.initiator.Stop()
		log.Println("[FIX demo] initiator stopped")
	}
}
