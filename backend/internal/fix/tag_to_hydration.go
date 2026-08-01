package fix

import (
	"bytes"
	"strconv"

	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/tag"
)

const tagMsgType = 35
const tagClOrdID = 11
const tagSymbol = 55
const tagOrderQty = 38
const tagPrice = 44
const tagSide = 54
const tagAccount = 1
const tagSecurityID = 48
const tagExecType = 150
const tagOrdStatus = 39

func TranslateNewOrderSingle(msg *quickfix.Message) ExternalTradeItem {
	trade := ExternalTradeItem{}

	if val, err := msg.Body.GetString(tagClOrdID); err == nil {
		trade.ExternalOrderID = val
	}
	if val, err := msg.Body.GetString(tagSymbol); err == nil {
		trade.Symbol = val
	}
	if val, err := msg.Body.GetString(tagSecurityID); err == nil {
		trade.ISIN = val
	}
	if val, err := msg.Body.GetString(tagSide); err == nil {
		trade.Side = val
	}
	if val, err := msg.Body.GetString(tagAccount); err == nil {
		trade.Account = val
	}
	if val, err := msg.Body.GetString(tagOrderQty); err == nil {
		if qty, parseErr := strconv.ParseFloat(val, 64); parseErr == nil {
			trade.Quantity = qty
		}
	}
	if val, err := msg.Body.GetString(tagPrice); err == nil {
		if price, parseErr := strconv.ParseFloat(val, 64); parseErr == nil {
			trade.Price = price
		}
	}

	return trade
}

func BuildExecutionReport(trade ExternalTradeItem, result EvaluateResult) *quickfix.Message {
	msg := quickfix.NewMessage()

	msg.Header.SetString(tag.BeginString, "FIX.4.4")
	msg.Header.SetInt(tag.BodyLength, 0)
	msg.Header.SetString(tag.MsgType, "8")

	msg.Body.SetString(tagClOrdID, trade.ExternalOrderID)
	msg.Body.SetString(tagOrderID, trade.ExternalOrderID)

	if result.Approved {
		msg.Body.SetString(tagExecType, "0")
		msg.Body.SetString(tagOrdStatus, "0")
	} else {
		msg.Body.SetString(tagExecType, "3")
		msg.Body.SetString(tagOrdStatus, "3")
	}

	msg.Body.SetString(tagOrderQty, strconv.FormatFloat(trade.Quantity, 'f', -1, 64))
	msg.Body.SetString(tagPrice, strconv.FormatFloat(trade.Price, 'f', -1, 64))
	msg.Body.SetString(tagSymbol, trade.Symbol)
	msg.Body.SetString(tagSide, trade.Side)

	return msg
}

func ParseFIXMessage(raw []byte) (*quickfix.Message, error) {
	msg := quickfix.NewMessage()
	if err := quickfix.ParseMessage(msg, bytes.NewBuffer(raw)); err != nil {
		return nil, err
	}
	return msg, nil
}

func HydrateFromClOrdID(msg *quickfix.Message) string {
	if val, err := msg.Body.GetString(tagClOrdID); err == nil {
		return val
	}
	return ""
}

func SetExecutionReportFields(msg *quickfix.Message, clOrdID, execType, ordStatus string) {
	msg.Body.SetString(tagClOrdID, clOrdID)
	msg.Body.SetString(tagExecType, execType)
	msg.Body.SetString(tagOrdStatus, ordStatus)
}

var tagBeginString = tag.BeginString
var tagBodyLength = tag.BodyLength
var tagMsgSeqNum = tag.MsgSeqNum
var tagSenderCompID = tag.SenderCompID
var tagTargetCompID = tag.TargetCompID
var tagOrderID = tag.OrderID

type tagReadError struct{}

func (e *tagReadError) Error() string { return "tag not found" }
