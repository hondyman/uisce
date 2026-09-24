package tiles

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/tag"
)

// FixDecode parses raw FIX bytes into a tag/value map and emits it as
// record["tags"]. Per HANDOFF_FIX_OVER_PIPELINE.md §9 `fix_decode`.
//
// Required tags (FIX 4.4 spec, all mandatory): 8 BeginString, 9
// BodyLength, 35 MsgType, 49 SenderCompID, 56 TargetCompID, 34
// MsgSeqNum, 52 SendingTime, 10 CheckSum. Validation failures are
// routed via the errors slice (the data-pipeline's error policy
// decides whether to skip / fail / dead-letter).
func FixDecode(ctx context.Context, records []Record) ([]Record, []string, error) {
	out := make([]Record, 0, len(records))
	var errs []string

	for _, rec := range records {
		rawBytes, ok := rec["raw_bytes"].(string)
		if !ok {
			errs = append(errs, "fix_decode: raw_bytes missing or wrong type")
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(rawBytes)
		if err != nil {
			errs = append(errs, fmt.Sprintf("fix_decode: base64: %v", err))
			continue
		}

		tags, perr := parseFIXBytes(raw)
		if perr != nil {
			errs = append(errs, fmt.Sprintf("fix_decode: parse: %v", perr))
			continue
		}

		// Validate required tags. Per spec these MUST be present on
		// every message. Missing tags → record errors.
		missing := validateRequiredTags(tags)
		if len(missing) > 0 {
			errs = append(errs, fmt.Sprintf("fix_decode: missing required tags: %v", missing))
			continue
		}

		// Annotate record with parsed tags + msg_type (extracted from
		// tag 35 for downstream tile convenience).
		newRec := make(Record, len(rec)+2)
		for k, v := range rec {
			newRec[k] = v
		}
		newRec["tags"] = tags
		if mt, ok := tags[strconv.Itoa(int(tag.MsgType))]; ok {
			newRec["msg_type"] = mt
		}
		out = append(out, newRec)
	}

	return out, errs, nil
}

// parseFIXBytes splits a SOH-delimited FIX byte stream into a
// tag-number → value map. SOH is ASCII 0x01.
func parseFIXBytes(raw []byte) (map[string]string, error) {
	out := make(map[string]string)
	for _, field := range bytes.Split(raw, []byte{0x01}) {
		if len(field) == 0 {
			continue
		}
		idx := bytes.IndexByte(field, '=')
		if idx <= 0 {
			continue
		}
		tagNum := string(field[:idx])
		val := string(field[idx+1:])
		out[tagNum] = val
	}
	return out, nil
}

// validateRequiredTags returns the tag numbers that are required by the
// FIX 4.4 spec but missing from the parsed message.
func validateRequiredTags(tags map[string]string) []int {
	required := []int{
		int(tag.BeginString),
		int(tag.BodyLength),
		int(tag.MsgType),
		int(tag.SenderCompID),
		int(tag.TargetCompID),
		int(tag.MsgSeqNum),
		int(tag.SendingTime),
		int(tag.CheckSum),
	}
	var missing []int
	for _, t := range required {
		if _, ok := tags[strconv.Itoa(t)]; !ok {
			missing = append(missing, t)
		}
	}
	return missing
}

// BuildPendingNewExecutionReport constructs the immediate PendingNew
// reply used by the adapter for inbound NewOrderSingle per Amendment 2.
// Kept here (alongside the tile code) so the reply shape is in one
// place — both the adapter and any test fixtures use it.
func BuildPendingNewExecutionReport(clOrdID string) *quickfix.Message {
	if clOrdID == "" {
		return nil
	}
	msg := quickfix.NewMessage()
	msg.Header.SetString(tag.BeginString, "FIX.4.4")
	msg.Header.SetInt(tag.BodyLength, 0)
	msg.Header.SetString(tag.MsgType, "8") // ExecutionReport

	msg.Body.SetString(tag.ClOrdID, clOrdID)
	msg.Body.SetString(tag.OrderID, clOrdID)
	msg.Body.SetString(tag.ExecType, "A")  // PendingNew
	msg.Body.SetString(tag.OrdStatus, "A") // PendingNew
	return msg
}

// ExtractTag pulls a tag value from a record's parsed "tags" map.
// Convenience for downstream tiles that need a typed value.
func ExtractTag(rec Record, tagNum int) (string, bool) {
	tagsAny, ok := rec["tags"].(map[string]string)
	if !ok {
		return "", false
	}
	v, ok := tagsAny[strconv.Itoa(tagNum)]
	return v, ok
}

// QuickfixSessionIDString is the canonical "FIX.4.4:SENDER->TARGET"
// encoding used everywhere (admin API, fix_session_log, etc.).
func QuickfixSessionIDString(beginString, senderComp, targetComp string) string {
	return fmt.Sprintf("%s:%s->%s", beginString, senderComp, targetComp)
}

// ParseQuickfixSessionIDString reverses QuickfixSessionIDString. Returns
// the components if the input matches the expected format.
func ParseQuickfixSessionIDString(s string) (begin, sender, target string, ok bool) {
	// FIX.x.y:COMP1->COMP2
	sep1 := strings.Index(s, ":")
	if sep1 < 0 {
		return
	}
	sep2 := strings.Index(s[sep1+1:], "->")
	if sep2 < 0 {
		return
	}
	sep2 += sep1 + 1
	begin = s[:sep1]
	sender = s[sep1+1 : sep2]
	target = s[sep2+2:]
	ok = true
	return
}
