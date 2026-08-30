package semanticmatch

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// abbreviationDict maps common financial/DB abbreviations to expansions.
// Applied SYMMETRICALLY to columns and terms, so expansion never hurts.
var abbreviationDict = map[string]string{
	// generic identifier shorthand
	"acct": "account", "accts": "account", "amt": "amount", "amts": "amount",
	"num": "number", "nbr": "number", "no": "number", "nm": "name",
	"cd": "code", "ind": "indicator", "typ": "type", "seq": "sequence",
	"desc": "description", "min": "minimum", "max": "maximum",
	"qty": "quantity", "pct": "percent", "pctg": "percent",
	"dt": "date", "tm": "time", "ts": "timestamp", "yr": "year",
	"mo": "month", "wk": "week", "qtr": "quarter", "prev": "previous",
	"cur": "current", "tot": "total", "avg": "average", "est": "estimated",
	"calc": "calculated", "val": "value", "auth": "authorization",
	"cfg": "configuration", "param": "parameter", "parm": "parameter",
	"req": "request", "msg": "message", "srv": "server", "ref": "reference",
	// finance / domain-specific (from your schemas and BB mnemonics)
	"crrncy": "currency", "curncy": "currency", "cntry": "country",
	"alloc": "allocation", "gen": "generation", "hist": "history",
	"bkr": "broker", "clnt": "client", "cust": "customer",
	"txn": "transaction", "tran": "transaction", "trans": "transaction",
	"sec": "security", "trm": "term", "lt": "long term", "st": "short term",
	"ytd": "year to date", "exp": "expense", "mgmt": "management",
	"uw": "underwriting", "ipo": "initial public offering",
	"rcv": "receivable", "recv": "receivable", "ap": "account payable",
	"ar": "account receivable", "shrs": "share", "shr": "share",
	"gl": "gain loss", "cfwd": "forward", "fwd": "forward",
	"eod": "end of day", "fx": "foreign exchange", "mv": "market value",
	"nav": "net asset value", "npv": "net present value", "pv": "present value",
	"pos": "position", "roa": "return on assets", "roe": "return on equity",
	"vol": "volatility", "imp": "implied", "px": "price",
	"spr": "spread", "sprd": "spread", "mkt": "market", "trd": "trade",
	"tgt": "target", "dflt": "default", "addr": "address", "zip": "postal",
	"ph": "phone", "tel": "phone", "fax": "facsimile", "mgr": "manager",
	"comm": "commission", "ovrd": "override", "div": "dividend",
	"int": "interest", "loc": "local", "denom": "denominator",
	"settle": "settlement", "org": "organization", "mtge": "mortgage",
	"comdty": "commodity", "eqy": "equity", "pfd": "preferred",
	"muni": "municipal", "mmkt": "money market", "idx": "index",
	"dbt": "debit", "crd": "credit", "chg": "change", "cxl": "cancel",
	"rsrv": "reserve", "id": "identifier", "usd": "usd",
}

// Tokenize splits an identifier into lowercase word tokens, handling
// snake_case, camelCase, digit boundaries and punctuation.
func Tokenize(s string) []string {
	var tokens []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			tokens = append(tokens, string(cur))
			cur = cur[:0]
		}
	}
	runes := []rune(strings.TrimSpace(s))
	for i, r := range runes {
		switch {
		case unicode.IsLetter(r):
			if len(cur) > 0 {
				prev := runes[i-1]
				if unicode.IsDigit(prev) {
					flush()
				} else if unicode.IsLower(prev) && unicode.IsUpper(r) {
					// camelCase transition (e.g. settleDate -> settle, date)
					flush()
				} else if unicode.IsUpper(prev) && unicode.IsUpper(r) && i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
					// Acronym to Word transition (e.g. USDAmount -> USD, Amount)
					flush()
				}
			}
			cur = append(cur, unicode.ToLower(r))
		case unicode.IsDigit(r):
			if len(cur) > 0 && !unicode.IsDigit(runes[i-1]) {
				flush()
			}
			cur = append(cur, r)
		default:
			flush()
		}
	}
	flush()
	return tokens
}

// ExpandTokens replaces abbreviations with expansions and light-stems plurals.
func ExpandTokens(toks []string) []string {
	out := make([]string, 0, len(toks)+4)
	for _, t := range toks {
		if full, ok := abbreviationDict[t]; ok && full != "" {
			out = append(out, strings.Fields(full)...)
			continue
		}
		out = append(out, stem(t))
	}
	return out
}

func stem(t string) string {
	if len(t) >= 4 && strings.HasSuffix(t, "s") &&
		!strings.HasSuffix(t, "ss") && !strings.HasSuffix(t, "us") && !strings.HasSuffix(t, "is") {
		return t[:len(t)-1]
	}
	return t
}

// NormalizeKey produces a canonical key for exact-match lookups
// (lowercase alphanumerics only: "cash_acct_no" -> "cashacctno").
func NormalizeKey(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

var (
	rangeSepRe      = regexp.MustCompile(`\.{2,}|…+`)
	trailingDigitRe = regexp.MustCompile(`^(.*?)(\d+)$`)
)

// ParseAliases splits multi-column entries like "broker_cd / bkr_cd" and
// expands grouped ranges like "est_settle_cash0…est_settle_cash6" or
// "theme_1…theme_3" into individual column names.
func ParseAliases(s string) []string {
	rep := strings.NewReplacer("/", "\n", ",", "\n", ";", "\n")
	var out []string
	for _, p := range strings.Split(rep.Replace(s), "\n") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if exp := expandRange(p); len(exp) > 1 {
			out = append(out, exp...)
		} else {
			out = append(out, p)
		}
	}
	return out
}

func expandRange(s string) []string {
	loc := rangeSepRe.FindStringIndex(s)
	if loc == nil {
		return nil
	}
	left, right := strings.TrimSpace(s[:loc[0]]), strings.TrimSpace(s[loc[1]:])
	lm := trailingDigitRe.FindStringSubmatch(left)
	rm := trailingDigitRe.FindStringSubmatch(right)
	if lm == nil || rm == nil || lm[1] != rm[1] {
		return nil
	}
	lo, err1 := strconv.Atoi(lm[2])
	hi, err2 := strconv.Atoi(rm[2])
	if err1 != nil || err2 != nil || hi < lo || hi-lo > 100 {
		return nil
	}
	out := make([]string, 0, hi-lo+1)
	for i := lo; i <= hi; i++ {
		out = append(out, fmt.Sprintf("%s%d", lm[1], i))
	}
	return out
}

// ---- Type families ----

type TypeFamily string

const (
	FamilyNumeric  TypeFamily = "numeric"
	FamilyInteger  TypeFamily = "integer"
	FamilyString   TypeFamily = "string"
	FamilyBool     TypeFamily = "boolean"
	FamilyDateTime TypeFamily = "datetime"
	FamilyUnknown  TypeFamily = "unknown"
)

// ParseTypeFamily normalizes either a DB DDL type ("NUMERIC(18,4) NOT NULL",
// "CHAR(1) DEFAULT 'N'") or a Bloomberg FieldType ("Real", "Price") into a family.
func ParseTypeFamily(raw string) TypeFamily {
	if raw == "" {
		return FamilyUnknown
	}
	f := strings.FieldsFunc(raw, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r))
	})
	if len(f) == 0 {
		return FamilyUnknown
	}
	switch strings.ToUpper(f[0]) {
	case "NUMERIC", "DECIMAL", "DEC", "FLOAT", "FLOAT4", "FLOAT8", "DOUBLE", "REAL", "MONEY", "PRICE":
		return FamilyNumeric
	case "INT", "INTEGER", "BIGINT", "SMALLINT", "INT2", "INT4", "INT8":
		return FamilyInteger
	case "VARCHAR", "CHAR", "CHARACTER", "TEXT", "CITEXT", "NCHAR", "NVARCHAR", "BPCHAR", "STRING":
		return FamilyString
	case "TIMESTAMP", "TIMESTAMPTZ", "DATE", "TIME", "DATETIME":
		return FamilyDateTime
	case "BOOL", "BOOLEAN":
		return FamilyBool
	}
	return FamilyUnknown
}

// TypeCompatibility scores how plausible it is that a column of family a
// stores a term of family b.
func TypeCompatibility(a, b TypeFamily) float64 {
	if a == FamilyUnknown || b == FamilyUnknown {
		return 0.5
	}
	if a == b {
		return 1.0
	}
	pair := func(x, y TypeFamily) bool { return (a == x && b == y) || (a == y && b == x) }
	switch {
	case pair(FamilyNumeric, FamilyInteger):
		return 0.9
	case pair(FamilyBool, FamilyString): // CHAR(1) Y/N flag ↔ Bloomberg Boolean
		return 0.6
	case pair(FamilyDateTime, FamilyString):
		return 0.4
	case pair(FamilyNumeric, FamilyString), pair(FamilyInteger, FamilyString):
		return 0.3
	default:
		return 0.2
	}
}

func (c ColumnMeta) TypeFamily() TypeFamily { return ParseTypeFamily(c.DataType) }
func (t *SemanticTerm) Family() TypeFamily  { return ParseTypeFamily(t.RawType) }
