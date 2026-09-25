package datapipeline

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Error messages here are shown verbatim to data analysts in the run preview,
// so they name the column and the offending value in plain language.

// --- validate ---------------------------------------------------------

type validateProc struct {
	cfg  ValidateConfig
	seen map[string]struct{}
}

func newValidateProc(n Node) (Processor, error) {
	var c ValidateConfig
	if err := decodeConfig(n, &c); err != nil {
		return nil, err
	}
	return &validateProc{cfg: c, seen: map[string]struct{}{}}, nil
}

func (p *validateProc) Open(context.Context, *RunContext) error { return nil }
func (p *validateProc) Close(context.Context, error) error      { return nil }

func (p *validateProc) Process(_ context.Context, rows []Row) (Result, error) {
	var res Result
rowLoop:
	for _, r := range rows {
		for _, f := range p.cfg.Required {
			if isBlank(r.Data[f]) {
				res.Rejected = append(res.Rejected, Reject{Row: r, Field: f,
					Reason: fmt.Sprintf("%q is required but is empty", f)})
				continue rowLoop
			}
		}
		if len(p.cfg.Unique) > 0 {
			parts := make([]string, len(p.cfg.Unique))
			for i, f := range p.cfg.Unique {
				parts[i] = fmt.Sprint(r.Data[f])
			}
			k := strings.Join(parts, "\x1f")
			if _, dup := p.seen[k]; dup {
				res.Rejected = append(res.Rejected, Reject{Row: r, Field: strings.Join(p.cfg.Unique, ", "),
					Reason: fmt.Sprintf("duplicate value %q for %s", strings.Join(parts, " / "), strings.Join(p.cfg.Unique, " + "))})
				continue
			}
			p.seen[k] = struct{}{}
		}
		res.Out = append(res.Out, r)
	}
	return res, nil
}

func isBlank(v any) bool {
	if v == nil {
		return true
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s) == ""
	}
	return false
}

// --- map --------------------------------------------------------------

type mapProc struct{ cfg MapConfig }

func newMapProc(n Node) (Processor, error) {
	var c MapConfig
	if err := decodeConfig(n, &c); err != nil {
		return nil, err
	}
	return &mapProc{cfg: c}, nil
}

func (p *mapProc) Open(context.Context, *RunContext) error { return nil }
func (p *mapProc) Close(context.Context, error) error      { return nil }

func (p *mapProc) Process(_ context.Context, rows []Row) (Result, error) {
	var res Result
	mapped := map[string]struct{}{}
	for _, f := range p.cfg.Fields {
		mapped[f.From] = struct{}{}
	}
rowLoop:
	for _, r := range rows {
		out := make(map[string]any, len(p.cfg.Fields))
		if p.cfg.KeepUnmapped {
			for k, v := range r.Data {
				if _, m := mapped[k]; !m {
					out[k] = v
				}
			}
		}
		for _, f := range p.cfg.Fields {
			v, ok := r.Data[f.From]
			if (!ok || isBlank(v)) && f.Default != nil {
				v = *f.Default
			}
			nv, err := applyTransform(f, v)
			if err != nil {
				res.Rejected = append(res.Rejected, Reject{Row: r, Field: f.From, Reason: err.Error()})
				continue rowLoop
			}
			out[f.To] = nv
		}
		res.Out = append(res.Out, Row{Num: r.Num, Data: out})
	}
	return res, nil
}

var dateLayouts = []string{
	"2006-01-02", "2006/01/02", "01/02/2006", "1/2/2006", "02-Jan-2006",
	"2 Jan 2006", "20060102", time.RFC3339, "2006-01-02 15:04:05",
}

func applyTransform(f FieldMap, v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	s, isStr := v.(string)
	switch f.Transform {
	case "":
		return v, nil
	case "trim":
		if isStr {
			return strings.TrimSpace(s), nil
		}
		return v, nil
	case "upper":
		if isStr {
			return strings.ToUpper(strings.TrimSpace(s)), nil
		}
		return v, nil
	case "lower":
		if isStr {
			return strings.ToLower(strings.TrimSpace(s)), nil
		}
		return v, nil
	case "to_number":
		if !isStr {
			return v, nil
		}
		t := strings.NewReplacer(",", "", "$", "", " ", "").Replace(s)
		if t == "" {
			return nil, nil
		}
		n, err := strconv.ParseFloat(t, 64)
		if err != nil {
			return nil, fmt.Errorf("%q in column %q is not a number", s, f.From)
		}
		return n, nil
	case "to_date":
		if t, ok := v.(time.Time); ok {
			return t.Format("2006-01-02"), nil
		}
		if !isStr || strings.TrimSpace(s) == "" {
			return nil, nil
		}
		for _, l := range dateLayouts {
			if t, err := time.Parse(l, strings.TrimSpace(s)); err == nil {
				return t.Format("2006-01-02"), nil
			}
		}
		return nil, fmt.Errorf("%q in column %q is not a recognised date", s, f.From)
	case "lookup":
		key := fmt.Sprint(v)
		if out, ok := f.Lookup[key]; ok {
			return out, nil
		}
		return nil, fmt.Errorf("%q in column %q has no match in the lookup table", key, f.From)
	}
	return nil, fmt.Errorf("unknown transform %q", f.Transform)
}
