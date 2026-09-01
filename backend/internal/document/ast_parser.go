package document

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type BoundingBox struct {
	X1 float64 `json:"x1"`
	Y1 float64 `json:"y1"`
	X2 float64 `json:"x2"`
	Y2 float64 `json:"y2"`
}

type ExtractedCell struct {
	PageNumber   int         `json:"page_number"`
	TableIndex   int         `json:"table_index"`
	RowIndex     int         `json:"row_index"`
	ColIndex     int         `json:"col_index"`
	RowHeader    string      `json:"row_header"`
	ColHeader    string      `json:"col_header"`
	RawText      string      `json:"raw_text"`
	NumericValue *float64    `json:"numeric_value,omitempty"`
	Currency     string      `json:"currency"`
	BBox         BoundingBox `json:"bbox"`
	FootnoteRef  string      `json:"footnote_ref,omitempty"`
}

type DocumentASTService struct {
	db *sql.DB
}

func NewDocumentASTService(db *sql.DB) *DocumentASTService {
	return &DocumentASTService{db: db}
}

// IngestStatementGrid persists extracted table cells and links numeric callouts to footnotes
func (s *DocumentASTService) IngestStatementGrid(
	ctx context.Context,
	tenantID uuid.UUID,
	docKey, docType, fileName, objectStoreURI string,
	totalPages int,
	cells []ExtractedCell,
	footnotes map[string]string,
) (uuid.UUID, error) {
	if tenantID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	docID := uuid.New()
	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s:%s:%s", docKey, fileName, objectStoreURI)))
	docChecksum := hex.EncodeToString(hasher.Sum(nil))

	if s.db == nil {
		return docID, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback()

	docInsert := `
		INSERT INTO catalog_doc.document_manifest (
			document_id, tenant_id, document_key, document_type,
			file_name, object_store_uri, sha256_checksum, total_pages
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);`

	if _, err := tx.ExecContext(ctx, docInsert,
		docID, tenantID, docKey, docType, fileName, objectStoreURI, docChecksum, totalPages); err != nil {
		return uuid.Nil, fmt.Errorf("failed creating document manifest: %w", err)
	}

	footnoteRegex := regexp.MustCompile(`(?i)(?:note|clause|\*|\()(\d+|[a-z])\)?`)

	for _, c := range cells {
		cellID := uuid.New()
		bboxJSON, _ := json.Marshal(c.BBox)

		cellInsert := `
			INSERT INTO catalog_doc.statement_table_cells (
				cell_id, document_id, tenant_id, page_number,
				table_index, row_index, col_index, row_header,
				col_header, raw_text, numeric_value, currency, bbox_coordinates
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);`

		if _, err := tx.ExecContext(ctx, cellInsert,
			cellID, docID, tenantID, c.PageNumber, c.TableIndex,
			c.RowIndex, c.ColIndex, c.RowHeader, c.ColHeader,
			c.RawText, c.NumericValue, c.Currency, bboxJSON); err != nil {
			return uuid.Nil, fmt.Errorf("failed inserting statement cell: %w", err)
		}

		fnKey := c.FootnoteRef
		if fnKey == "" && footnoteRegex.MatchString(c.RawText) {
			fnKey = footnoteRegex.FindString(c.RawText)
		}

		if fnKey != "" {
			if fnBody, ok := footnotes[strings.TrimSpace(fnKey)]; ok {
				fnInsert := `
					INSERT INTO catalog_doc.cell_footnote_bindings (
						binding_id, tenant_id, cell_id, footnote_number, footnote_text, valuation_hierarchy
					) VALUES ($1, $2, $3, $4, $5, 'LEVEL_3_UNOBSERVABLE');`

				_, _ = tx.ExecContext(ctx, fnInsert,
					uuid.New(), tenantID, cellID, fnKey, fnBody)
			}
		}
	}

	return docID, tx.Commit()
}

// ParseFinancialNumber extracts float values from accounting notation
func ParseFinancialNumber(input string) *float64 {
	clean := strings.TrimSpace(input)
	if clean == "" || clean == "-" || clean == "—" {
		return nil
	}
	isNegative := false
	if strings.HasPrefix(clean, "(") && strings.HasSuffix(clean, ")") {
		isNegative = true
		clean = strings.TrimPrefix(strings.TrimSuffix(clean, ")"), "(")
	}
	clean = strings.ReplaceAll(clean, ",", "")
	clean = strings.ReplaceAll(clean, "$", "")
	val, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return nil
	}
	if isNegative {
		val = -val
	}
	return &val
}
