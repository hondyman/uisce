package semanticmatch

import (
	"encoding/csv"
	"io"
	"os"
	"strings"
)

func headerIndex(header []string) map[string]int {
	m := map[string]int{}
	for i, h := range header {
		m[strings.TrimSpace(h)] = i
	}
	return m
}

// LoadBloombergFields loads BB_Fields.csv into the registry.
func LoadBloombergFields(reg *TermRegistry, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	header, err := r.Read()
	if err != nil {
		return err
	}
	idx := headerIndex(header)
	for {
		rec, err := r.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		get := func(name string) string {
			i, ok := idx[name]
			if !ok || i >= len(rec) {
				return ""
			}
			return strings.TrimSpace(rec[i])
		}
		mnem := get("FieldMnemonic")
		if mnem == "" {
			continue
		}
		reg.AddTerm(&SemanticTerm{
			ID:          "bb:" + mnem,
			Source:      "bloomberg",
			Mnemonic:    mnem,
			Name:        get("Description"),
			Description: get("Definition"),
			Domain:      get("Category"),
			RawType:     get("FieldType"),
		}, nil)
	}
}

// LoadGlossary loads Semantic_Definitions.csv: curated terms + table-scoped
// seeds + same-name-rule seeds. Grouped ranges ("est_settle_cash0…6") and
// synonym lists ("broker_cd / bkr_cd") are expanded automatically.
func LoadGlossary(reg *TermRegistry, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	header, err := r.Read()
	if err != nil {
		return err
	}
	idx := headerIndex(header)
	for {
		rec, err := r.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		get := func(name string) string {
			i, ok := idx[name]
			if !ok || i >= len(rec) {
				return ""
			}
			return strings.TrimSpace(rec[i])
		}
		if get("business_term") == "" {
			continue
		}
		var tables []string
		for _, s := range strings.Split(get("source_tables"), ";") {
			if s = strings.TrimSpace(s); s != "" {
				tables = append(tables, s)
			}
		}
		reg.AddTerm(&SemanticTerm{
			ID:          "glossary:" + get("id"),
			Source:      "glossary",
			Name:        get("business_term"),
			Aliases:     ParseAliases(get("column_name")),
			Description: get("definition"),
			Domain:      get("domain"),
			RawType:     get("data_type"),
		}, tables)
	}
}

// LoadColumnsCSV loads scanned column metadata. Expected headers:
// table, column, data_type [, schema, database, description, samples]
// samples is a '|'-separated list of sampled values (optional but valuable).
func LoadColumnsCSV(path string) ([]ColumnMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.LazyQuotes = true
	header, err := r.Read()
	if err != nil {
		return nil, err
	}
	idx := headerIndex(header)
	var cols []ColumnMeta
	for {
		rec, err := r.Read()
		if err == io.EOF {
			return cols, nil
		}
		if err != nil {
			return nil, err
		}
		get := func(name string) string {
			i, ok := idx[name]
			if !ok || i >= len(rec) {
				return ""
			}
			return strings.TrimSpace(rec[i])
		}
		if get("table") == "" || get("column") == "" {
			continue
		}
		dt := get("data_type")
		col := ColumnMeta{
			Schema:      get("schema"),
			Table:       get("table"),
			Column:      get("column"),
			DataType:    dt,
			Description: get("description"),
		}
		col.Nullable = !strings.Contains(strings.ToUpper(dt), "NOT NULL")
		if s := get("samples"); s != "" {
			col.SampleValues = strings.Split(s, "|")
		}
		cols = append(cols, col)
	}
}
