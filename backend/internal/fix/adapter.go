package fix

import (
	"fmt"
	"log"
	"sync"

	"github.com/quickfixgo/quickfix"
)

type ComplianceEvaluator interface {
	EvaluateTrade(trade ExternalTradeItem) EvaluateResult
}

type EvaluateResult struct {
	Approved        bool
	CanOverride    bool
	HighestSeverity string
	Violations     []Violation
}

type Violation struct {
	RuleID    string
	FieldPath string
	Message   string
}

type ExternalTradeItem struct {
	ExternalOrderID string
	PortfolioID    string
	ISIN          string
	Symbol        string
	Quantity      float64
	Price         float64
	Side          string
	Account       string
}

type Adapter struct {
	app        *fixApplication
	sessionMap map[quickfix.SessionID]struct{}
	sessionMu  sync.RWMutex
	evaluator  ComplianceEvaluator
}

type fixApplication struct {
	onLogon  func(sessionID quickfix.SessionID)
	onLogout func(sessionID quickfix.SessionID)
	onMsg    func(sessionID quickfix.SessionID, msg *quickfix.Message) error
	sessionMu sync.RWMutex
}

func (a *fixApplication) OnCreate(sessionID quickfix.SessionID) {
	log.Printf("[FIX] Session created: %s", sessionID)
}

func (a *fixApplication) OnLogon(sessionID quickfix.SessionID) {
	log.Printf("[FIX] Logon: %s", sessionID)
}

func (a *fixApplication) OnLogout(sessionID quickfix.SessionID) {
	log.Printf("[FIX] Logout: %s", sessionID)
}

func (a *fixApplication) ToAdmin(msg *quickfix.Message, sessionID quickfix.SessionID) {}

func (a *fixApplication) ToApp(msg *quickfix.Message, sessionID quickfix.SessionID) error {
	return nil
}

func (a *fixApplication) FromAdmin(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return nil
}

func (a *fixApplication) FromApp(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	a.sessionMu.RLock()
	handler := a.onMsg
	a.sessionMu.RUnlock()
	if handler != nil {
		if err := handler(sessionID, msg); err != nil {
			return quickfix.NewMessageRejectError(err.Error(), 0, nil)
		}
	}
	return nil
}

func NewAdapter(evaluator ComplianceEvaluator) *Adapter {
	app := &fixApplication{}
	adapter := &Adapter{
		app:        app,
		sessionMap: make(map[quickfix.SessionID]struct{}),
		evaluator:  evaluator,
	}

	app.onLogon = func(sessionID quickfix.SessionID) {
		adapter.sessionMu.Lock()
		adapter.sessionMap[sessionID] = struct{}{}
		adapter.sessionMu.Unlock()
	}

	app.onLogout = func(sessionID quickfix.SessionID) {
		adapter.sessionMu.Lock()
		delete(adapter.sessionMap, sessionID)
		adapter.sessionMu.Unlock()
	}

	app.onMsg = func(sessionID quickfix.SessionID, msg *quickfix.Message) error {
		msgType, _ := msg.Header.GetString(tagMsgType)
		if msgType == "D" {
			trade := TranslateNewOrderSingle(msg)
			result := adapter.evaluator.EvaluateTrade(trade)
			reply := BuildExecutionReport(trade, result)
			if err := quickfix.SendToTarget(reply, sessionID); err != nil {
				log.Printf("[FIX] Failed to send ExecutionReport: %v", err)
			}
		}
		return nil
	}

	return adapter
}

func (a *Adapter) CreateAcceptor(settings *quickfix.Settings) (*quickfix.Acceptor, error) {
	store := quickfix.NewMemoryStoreFactory()
	logFactory := quickfix.NewScreenLogFactory()
	acceptor, err := quickfix.NewAcceptor(a.app, store, settings, logFactory)
	if err != nil {
		return nil, fmt.Errorf("create acceptor: %w", err)
	}
	return acceptor, nil
}
