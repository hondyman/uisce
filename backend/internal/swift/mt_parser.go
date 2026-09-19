package swift

import (
	"log"
	"strings"
)

// MTMessage represents a parsed ISO 15022 MT message.
type MTMessage struct {
	Block1      string
	Block2      string
	MsgType     string
	BICSender   string
	BICReceiver string
	Fields      map[string]string
	Raw         []byte
}

// ParseMT parses raw MT bytes into MTMessage.
func ParseMT(raw []byte) (*MTMessage, error) {
	msg := &MTMessage{
		Fields: make(map[string]string),
		Raw:    raw,
	}

	str := string(raw)

	// Block 1: {1:F01BANKBEBB0000000000}
	b1Start := strings.Index(str, "{1:")
	if b1Start != -1 {
		b1End := strings.Index(str[b1Start:], "}")
		if b1End != -1 {
			msg.Block1 = str[b1Start+3 : b1Start+b1End]
			if len(msg.Block1) >= 15 {
				msg.BICSender = msg.Block1[3:15]
			}
		}
	}

	// Block 2: {2:I541BANKUS33XBBN}
	b2Start := strings.Index(str, "{2:")
	if b2Start != -1 {
		b2End := strings.Index(str[b2Start:], "}")
		if b2End != -1 {
			msg.Block2 = str[b2Start+3 : b2Start+b2End]
			if len(msg.Block2) >= 4 {
				msg.MsgType = msg.Block2[1:4] // 541
			}
			if len(msg.Block2) >= 16 {
				msg.BICReceiver = msg.Block2[4:16]
			}
		}
	}

	// Block 4: {4:\r\n:20:TXREF001\r\n:35B:ISIN US0378331005\r\n-}
	b4Start := strings.Index(str, "{4:")
	if b4Start != -1 {
		b4End := strings.LastIndex(str, "-}")
		if b4End != -1 && b4End > b4Start {
			b4Content := str[b4Start+3 : b4End]

			lines := strings.Split(b4Content, "\n")
			var currentTag string
			var currentValue strings.Builder

			for _, line := range lines {
				line = strings.TrimRight(line, "\r")
				if line == "" {
					continue
				}

				if strings.HasPrefix(line, ":") {
					// Flush previous tag; warn on collision (repeated tag in flat map —
					// common in 16R/16S sequences where :20: appears per-sequence).
					// Last-write wins; caller should use sequence-scoped parsing for
					// messages with repeated tags in distinct sequences.
					if currentTag != "" {
						if _, collision := msg.Fields[currentTag]; collision {
							log.Printf("[SWIFT] mt_parser: field %q overwritten by repeated tag (flat-map last-wins); use sequence-scoped parsing for multi-sequence MT messages", currentTag)
						}
						msg.Fields[currentTag] = currentValue.String()
					}

					// Find the closing colon of the tag name, e.g. ":98A:" in ":98A:SETT//20241003"
					tagEnd := strings.Index(line[1:], ":")
					if tagEnd == -1 {
						currentTag = ""
						continue
					}
					tagName := ":" + line[1:tagEnd+2] // e.g. ":98A:"
					rest := line[tagEnd+2:]            // e.g. "SETT//20241003"

					// SWIFT qualified fields: the first token before "/" is the qualifier.
					// ":98A:SETT//20241003" → tag=":98A:SETT" value="//20241003"
					// ":20:TXREF001"        → tag=":20:"      value="TXREF001"
					// ":35B:ISIN US123"     → tag=":35B:"     value="ISIN US123"
					// A qualifier is present when the rest begins with 4 uppercase letters
					// followed by "/" (e.g. "SETT/", "BUYR/", "SELL/", "SAFE/").
					if len(rest) >= 5 && rest[4] == '/' && isUpperAlpha(rest[:4]) {
						qualifier := rest[:4]
						currentTag = tagName + qualifier // e.g. ":98A:SETT"
						currentValue.Reset()
						currentValue.WriteString(rest[4:]) // e.g. "//20241003"
					} else {
						currentTag = tagName
						currentValue.Reset()
						currentValue.WriteString(rest)
					}
				} else {
					if currentTag != "" {
						currentValue.WriteString("\n")
						currentValue.WriteString(line)
					}
				}
			}
			if currentTag != "" {
				msg.Fields[currentTag] = currentValue.String()
			}
		}
	}

	return msg, nil
}

// isUpperAlpha returns true if all bytes in s are ASCII uppercase letters.
func isUpperAlpha(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 'A' || s[i] > 'Z' {
			return false
		}
	}
	return true
}

// ExtractTransactionRef returns the :20: field value.
func (m *MTMessage) ExtractTransactionRef() string {
	if val, ok := m.Fields[":20:"]; ok {
		return val
	}
	return ""
}
