package swift

import (
	"bytes"
	"encoding/xml"
	"strings"
	"time"
)

// MXMessage represents a parsed ISO 20022 MX message.
type MXMessage struct {
	MsgType   string
	MsgID     string
	CreDtTm   time.Time
	Fields    map[string]string
	Raw       []byte
}

// DetectMXMsgType returns "pacs.008", "pacs.009", or "camt.056" from XML namespace.
func DetectMXMsgType(raw []byte) string {
	d := xml.NewDecoder(bytes.NewReader(raw))
	for {
		t, err := d.Token()
		if err != nil {
			break
		}
		if se, ok := t.(xml.StartElement); ok {
			for _, attr := range se.Attr {
				if attr.Name.Local == "xmlns" {
					parts := strings.Split(attr.Value, ":")
					if len(parts) > 0 {
						val := parts[len(parts)-1]
						if strings.HasPrefix(val, "pacs.") || strings.HasPrefix(val, "camt.") || strings.HasPrefix(val, "head.") {
							// Return just the base like pacs.008
							subs := strings.Split(val, ".")
							if len(subs) >= 2 {
								return subs[0] + "." + subs[1]
							}
							return val
						}
					}
				}
			}
			val := se.Name.Space
			if val != "" {
				parts := strings.Split(val, ":")
				last := parts[len(parts)-1]
				subs := strings.Split(last, ".")
				if len(subs) >= 2 {
					return subs[0] + "." + subs[1]
				}
			}
		}
	}
	
	if strings.Contains(string(raw), "pacs.008") {
		return "pacs.008"
	} else if strings.Contains(string(raw), "pacs.009") {
		return "pacs.009"
	} else if strings.Contains(string(raw), "camt.056") {
		return "camt.056"
	}
	return "MXUNKNOWN"
}

// ParseMX parses ISO 20022 XML. Detects message type from root element namespace.
func ParseMX(raw []byte) (*MXMessage, error) {
	msg := &MXMessage{
		MsgType: DetectMXMsgType(raw),
		Fields:  make(map[string]string),
		Raw:     raw,
	}

	d := xml.NewDecoder(bytes.NewReader(raw))
	var path []string
	var chars strings.Builder

	for {
		t, err := d.Token()
		if err != nil {
			break
		}
		switch e := t.(type) {
		case xml.StartElement:
			path = append(path, e.Name.Local)
			chars.Reset()
		case xml.EndElement:
			if chars.Len() > 0 {
				key := strings.Join(path, "/")
				val := strings.TrimSpace(chars.String())
				if val != "" {
					msg.Fields[key] = val
				}
			}
			if len(path) > 0 {
				path = path[:len(path)-1]
			}
			chars.Reset()
		case xml.CharData:
			chars.Write(e)
		}
	}

	for k, v := range msg.Fields {
		if strings.HasSuffix(k, "GrpHdr/MsgId") {
			msg.MsgID = v
		}
		if strings.HasSuffix(k, "GrpHdr/CreDtTm") {
			parsed, err := time.Parse(time.RFC3339, v)
			if err == nil {
				msg.CreDtTm = parsed
			} else {
				parsed, err = time.Parse("2006-01-02T15:04:05Z", v)
				if err == nil {
					msg.CreDtTm = parsed
				}
			}
		}
	}

	return msg, nil
}
