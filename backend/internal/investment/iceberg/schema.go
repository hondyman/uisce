package iceberg

// TaxLot represents the immutable record of a tax lot in the Iceberg table.
// It uses struct tags to define the Parquet schema mapping.
type TaxLot struct {
	LotID        string  `parquet:"name=lot_id, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN"`
	AccountID    string  `parquet:"name=account_id, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN"`
	SecurityID   string  `parquet:"name=security_id, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN"`
	Quantity     float64 `parquet:"name=quantity, type=DOUBLE"`
	CostBasis    float64 `parquet:"name=cost_basis, type=DOUBLE"`
	AcquiredDate int64   `parquet:"name=acquired_date, type=INT64, convertedtype=TIMESTAMP_MICROS"`
	TaxStatus    string  `parquet:"name=tax_status, type=BYTE_ARRAY, convertedtype=UTF8"` // e.g., "ShortTerm", "LongTerm"
	IsWashSale   bool    `parquet:"name=is_wash_sale, type=BOOLEAN"`
	OriginalLotID string `parquet:"name=original_lot_id, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN, repetitiontype=OPTIONAL"` // For splits/mergers
	CreatedAt    int64   `parquet:"name=created_at, type=INT64, convertedtype=TIMESTAMP_MICROS"`
}
