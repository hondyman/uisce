package tiles

import (
	"context"
	"fmt"
	"strings"
)

type InstructionEmitConfig struct {
	MsgType              string `json:"msg_type"`               // "MT541" | "MT543"
	SwiftVersion         string `json:"swift_version"`          // "MT"
	TransactionRefSource string `json:"transaction_ref_source"` // semantic field to use as :20:
}

func NewInstructionEmitTransform(cfg InstructionEmitConfig, loader TagMappingLoader) TileFunc {
	return func(ctx context.Context, records []Record) ([]Record, []string, error) {
		out := make([]Record, 0, len(records))
		var errs []string
		tctx := TenantFromContext(ctx)

		for _, rec := range records {
			semantic, ok := rec["semantic"].(map[string]any)
			if !ok {
				errs = append(errs, "instruction_emit: missing semantic map")
				continue
			}

			mappings, err := loader.LoadMappings(ctx, tctx.TenantID, cfg.SwiftVersion, cfg.MsgType)
			if err != nil {
				errs = append(errs, fmt.Sprintf("instruction_emit: failed to load mappings: %v", err))
				continue
			}

			var builder strings.Builder
			bicSender, _ := rec["bic_sender"].(string)
			bicReceiver, _ := rec["bic_receiver"].(string)
			if bicSender == "" {
				bicSender = "UNKNOWN"
			}
			if bicReceiver == "" {
				bicReceiver = "UNKNOWN"
			}

			mtNum := strings.TrimPrefix(cfg.MsgType, "MT")

			builder.WriteString(fmt.Sprintf("{1:F01%s0000000000}\n", padBIC(bicSender)))
			builder.WriteString(fmt.Sprintf("{2:I%s%sN}\n", mtNum, padBIC(bicReceiver)))
			builder.WriteString("{4:\n")

			// Always enforce transaction ref if provided
			if cfg.TransactionRefSource != "" {
				if ref, ok := semantic[cfg.TransactionRefSource]; ok {
					builder.WriteString(fmt.Sprintf(":20:%v\n", ref))
				}
			}

			for _, m := range mappings {
				// skip if we already wrote 20 and it's 20, but it's simpler to just write all from mappings.
				if cfg.TransactionRefSource != "" && m.FieldTag == "20" {
					continue
				}

				valAny, ok := semantic[m.SemanticField]
				if !ok {
					continue
				}
				valStr := fmt.Sprintf("%v", valAny)
				if m.TransformFn.Valid && m.TransformFn.String != "" {
					valStr = applyReverseTransform(valStr, m.TransformFn.String)
				}
				builder.WriteString(fmt.Sprintf(":%s:%s\n", m.FieldTag, valStr))
			}

			builder.WriteString("-}")

			newRec := make(Record, len(rec)+2)
			for k, v := range rec {
				newRec[k] = v
			}
			newRec["outbound_raw"] = []byte(builder.String())
			newRec["msg_type"] = cfg.MsgType
			out = append(out, newRec)
		}

		return out, errs, nil
	}
}

func padBIC(bic string) string {
	if len(bic) == 8 {
		return bic + "XXXX"
	}
	if len(bic) < 12 {
		return bic + strings.Repeat("X", 12-len(bic))
	}
	return bic[:12]
}

func applyReverseTransform(val, fn string) string {
	return val
}
