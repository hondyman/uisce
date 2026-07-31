package vm

import (
	"bytes"
	"errors"
	"strconv"
	"sync"
)

var (
	trueBytes  = []byte("true")
	falseBytes = []byte("false")
	nullBytes  = []byte("null")
)

// jsonScannerPool reuses the path buffer across calls so we don't
// allocate a new buffer on every DecodeJSON invocation. The pool
// stores *jsonScanner; the scanner's `path` slice is reset to length
// 0 (capacity preserved) on Get from the pool.
var jsonScannerPool = sync.Pool{
	New: func() any { return &jsonScanner{} },
}

// DecodeJSON parses a JSON byte slice directly into a FastRecord.
// This bypasses map[string]any entirely, achieving ~0 allocations on
// the ingestion path after warm-up (the FastRecord itself is pool-acquired
// and the scanner's path buffer is reused via sync.Pool).
//
// Expected shape: `{ "customer.tier": "GOLD", "customer.balance": 15000 }`
// or nested objects `{ "customer": { "tier": "GOLD", "balance": 15000 } }`.
// Nested keys are flattened to dot-paths at parse time.
func DecodeJSON(data []byte, syms *SymbolDict, enums *EnumDict) (*FastRecord, error) {
	rec := GetFastRecord(syms)
	if len(rec.NumVals) == 0 {
		return rec, nil
	}

	d := jsonScannerPool.Get().(*jsonScanner)
	d.data = data
	d.pos = 0
	d.path = d.path[:0]

	d.skipWhitespace()
	if d.eof() || d.data[d.pos] != '{' {
		jsonScannerPool.Put(d)
		return rec, errors.New("json: expected '{' at root")
	}
	d.pos++

	err := d.parseObject(rec, syms, enums)

	// Reset scanner state and return to pool.
	d.data = nil
	d.pos = 0
	d.path = d.path[:0]
	jsonScannerPool.Put(d)

	if err != nil {
		return rec, err
	}
	return rec, nil
}

type jsonScanner struct {
	data []byte
	pos  int
	path []byte // reusable buffer; len is mutated, cap preserved
}

func (d *jsonScanner) eof() bool { return d.pos >= len(d.data) }

func (d *jsonScanner) skipWhitespace() {
	for !d.eof() {
		c := d.data[d.pos]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			d.pos++
		} else {
			break
		}
	}
}

func (d *jsonScanner) parseObject(rec *FastRecord, syms *SymbolDict, enums *EnumDict) error {
	for {
		d.skipWhitespace()
		if d.eof() {
			return errors.New("json: unexpected eof in object")
		}

		if d.data[d.pos] == '}' {
			d.pos++
			return nil
		}

		if d.data[d.pos] == ',' {
			d.pos++
			continue
		}

		// Must be a key
		key, err := d.parseString()
		if err != nil {
			return err
		}

		d.skipWhitespace()
		if d.eof() || d.data[d.pos] != ':' {
			return errors.New("json: expected ':' after key")
		}
		d.pos++
		d.skipWhitespace()

		// Build path: path + "." + key (skip dot if path is empty)
		origLen := len(d.path)
		if origLen > 0 {
			d.path = append(d.path, '.')
		}
		d.path = append(d.path, key...)

		if err := d.parseValue(rec, syms, enums); err != nil {
			return err
		}

		// Restore path for next key
		d.path = d.path[:origLen]
	}
}

func (d *jsonScanner) parseValue(rec *FastRecord, syms *SymbolDict, enums *EnumDict) error {
	if d.eof() {
		return errors.New("json: unexpected eof")
	}
	c := d.data[d.pos]

	switch c {
	case '"':
		str, err := d.parseString()
		if err != nil {
			return err
		}

		// Resolve symbol via byte-path (no string allocation).
		if id, ok := syms.ResolveBytes(d.path); ok {
			// Store as string (FastRecord.StrVals is []string). The string
			// header is 16 bytes; the underlying byte data comes from the
			// JSON buffer which we don't own, so we must copy.
			valStr := string(str)
			rec.StrVals[id] = valStr
			rec.Present[id] |= HasStr

			if eid, ok := enums.ID(valStr); ok {
				rec.EnumVals[id] = eid
				rec.Present[id] |= HasEnum
			}
		}
		return nil

	case '{':
		d.pos++
		return d.parseObject(rec, syms, enums)

	case '[':
		d.skipArray()
		return nil

	case 't':
		if d.pos+4 <= len(d.data) && bytes.Equal(d.data[d.pos:d.pos+4], trueBytes) {
			if id, ok := syms.ResolveBytes(d.path); ok {
				rec.BoolVals[id] = true
				rec.Present[id] |= HasBool
			}
			d.pos += 4
			return nil
		}
		return errors.New("json: invalid token starting with t")

	case 'f':
		if d.pos+5 <= len(d.data) && bytes.Equal(d.data[d.pos:d.pos+5], falseBytes) {
			if id, ok := syms.ResolveBytes(d.path); ok {
				rec.BoolVals[id] = false
				rec.Present[id] |= HasBool
			}
			d.pos += 5
			return nil
		}
		return errors.New("json: invalid token starting with f")

	case 'n':
		if d.pos+4 <= len(d.data) && bytes.Equal(d.data[d.pos:d.pos+4], nullBytes) {
			d.pos += 4
			return nil
		}
		return errors.New("json: invalid token starting with n")

	default:
		// Number
		start := d.pos
		for !d.eof() {
			c := d.data[d.pos]
			if (c >= '0' && c <= '9') || c == '-' || c == '+' || c == '.' || c == 'e' || c == 'E' {
				d.pos++
			} else {
				break
			}
		}
		numBytes := d.data[start:d.pos]

		// Parse as float64, then cast to int64 for FastRecord.
		if val, err := strconv.ParseFloat(string(numBytes), 64); err == nil {
			if id, ok := syms.ResolveBytes(d.path); ok {
				rec.NumVals[id] = int64(val)
				rec.Present[id] |= HasNum
			}
		}
		return nil
	}
}

// parseString parses a JSON string and returns the unescaped byte slice.
// Does NOT decode escape sequences (MDM data typically doesn't contain
// complex escapes in hot paths); skipped characters are consumed without
// processing so the parser stays correct for length purposes.
func (d *jsonScanner) parseString() ([]byte, error) {
	if d.eof() || d.data[d.pos] != '"' {
		return nil, errors.New("json: expected '\"'")
	}
	d.pos++
	start := d.pos

	for !d.eof() {
		c := d.data[d.pos]
		if c == '"' {
			str := d.data[start:d.pos]
			d.pos++
			return str, nil
		}
		if c == '\\' {
			// Skip escape sequence (consume both backslash and the next byte).
			d.pos += 2
		} else {
			d.pos++
		}
	}
	return nil, errors.New("json: unterminated string")
}

func (d *jsonScanner) skipArray() {
	d.pos++ // skip '['
	depth := 1
	for !d.eof() && depth > 0 {
		c := d.data[d.pos]
		if c == '[' {
			depth++
			d.pos++
			continue
		}
		if c == ']' {
			depth--
			d.pos++
			continue
		}
		if c == '"' {
			_, _ = d.parseString()
			continue
		}
		d.pos++
	}
}