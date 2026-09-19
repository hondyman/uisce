package tiles

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
)

type DecodeConfig struct {
	SwiftVersion           string   `json:"swift_version"` // "MT" | "MX"
	ValidateRequiredFields bool     `json:"validate_required_fields"`
	MTParser               MTParser `json:"-"`
	MXParser               MXParser `json:"-"`
}

func NewDecodeTransform(cfg DecodeConfig, db *sql.DB) TileFunc {
	if cfg.SwiftVersion == "MT" && cfg.MTParser == nil {
		cfg.MTParser = inlineMTParser{}
	}
	if cfg.SwiftVersion == "MX" && cfg.MXParser == nil {
		cfg.MXParser = inlineMXParser{}
	}

	return func(ctx context.Context, records []Record) ([]Record, []string, error) {
		out := make([]Record, 0, len(records))
		var errs []string

		for _, rec := range records {
			var raw []byte
			switch v := rec["raw_bytes"].(type) {
			case []byte:
				raw = v
			case string:
				b, err := base64.StdEncoding.DecodeString(v)
				if err != nil {
					errs = append(errs, fmt.Sprintf("decode: invalid base64: %v", err))
					continue
				}
				raw = b
			default:
				errs = append(errs, "decode: raw_bytes missing or wrong type")
				continue
			}

			var fields map[string]string
			var err error

			if cfg.SwiftVersion == "MT" {
				fields, err = cfg.MTParser.Parse(raw)
			} else {
				fields, err = cfg.MXParser.Parse(raw)
			}

			if err != nil {
				errs = append(errs, fmt.Sprintf("decode: parse error: %v", err))
				continue
			}

			if cfg.ValidateRequiredFields {
				missing := validateMTRequired(fields)
				if len(missing) > 0 {
					errs = append(errs, fmt.Sprintf("decode: missing required fields: %v", missing))
					continue
				}
			}

			newRec := make(Record, len(rec)+1)
			for k, v := range rec {
				newRec[k] = v
			}
			newRec["fields"] = fields
			out = append(out, newRec)
		}
		return out, errs, nil
	}
}

type inlineMTParser struct{}

func (inlineMTParser) Parse(raw []byte) (map[string]string, error) {
	s := string(raw)
	fields := make(map[string]string)
	lines := strings.Split(s, "\n")
	var currentTag string
	var currentVal strings.Builder

	for _, l := range lines {
		l = strings.TrimRight(l, "\r")
		if strings.HasPrefix(l, ":") {
			if currentTag != "" {
				fields[currentTag] = strings.TrimSuffix(currentVal.String(), "-}")
			}
			idx := strings.Index(l[1:], ":")
			if idx > 0 {
				currentTag = l[1 : idx+1]
				currentVal.Reset()
				currentVal.WriteString(l[idx+2:])
			}
		} else if currentTag != "" {
			if currentVal.Len() > 0 {
				currentVal.WriteString("\r\n")
			}
			currentVal.WriteString(l)
		}
	}
	if currentTag != "" {
		val := currentVal.String()
		val = strings.TrimSuffix(val, "-}")
		fields[currentTag] = val
	}
	return fields, nil
}

type inlineMXParser struct{}

func (inlineMXParser) Parse(raw []byte) (map[string]string, error) {
	return map[string]string{}, nil
}

func validateMTRequired(fields map[string]string) []string {
	req := []string{"20", "35B", "36", "19A", "98A"}
	var missing []string
	for _, t := range req {
		if _, ok := fields[t]; !ok {
			missing = append(missing, t)
		}
	}
	return missing
}
