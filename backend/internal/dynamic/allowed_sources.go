package dynamic

var AllowedEnumSources = map[string][]string{
	"oms.account":         {"status", "subtype_code", "base_currency"},
	"oms.trade_order":     {"order_side", "order_status", "subtype_code", "confirmation_status", "liquidity_flag", "base_currency", "quote_currency"},
	"oms.position":        {"subtype_code", "currency"},
	"master.customer":     {"subtype_code", "kyc_status", "relationship_tier"},
	"master.sales_ledger": {"subtype_code", "invoice_status"},
}

var AllowedEnumInterval = map[string]bool{
	"microsecond": true, "millisecond": true, "second": true,
	"minute": true, "hour": true, "day": true,
	"week": true, "month": true, "quarter": true, "year": true,
	"decade": true, "century": true, "millennium": true,
}

var AllowedEnumCalendarType = map[string]bool{
	"gregorian": true, "fiscal": true, "iso_week": true, "custom": true,
}

var AllowedEnumWeekStartDay = map[string]bool{
	"monday": true, "tuesday": true, "wednesday": true, "thursday": true,
	"friday": true, "saturday": true, "sunday": true,
}

var AllowedEnumTimezone = map[string]bool{
	"UTC":                 true,
	"America/New_York":    true,
	"America/Chicago":     true,
	"America/Denver":      true,
	"America/Los_Angeles": true,
	"Europe/London":       true,
	"Europe/Berlin":       true,
	"Europe/Paris":        true,
	"Asia/Tokyo":          true,
	"Asia/Shanghai":       true,
	"Asia/Singapore":      true,
	"Australia/Sydney":    true,
}

var AllowedEnumParsingFunction = map[string]bool{
	"TO_TIMESTAMP": true,
}

var AllowedUnionTypes = map[string]bool{
	"UNION":         true,
	"UNION ALL":     true,
	"INTERSECT":     true,
	"INTERSECT ALL": true,
	"EXCEPT":        true,
	"EXCEPT ALL":    true,
}

var AllowedDateFormatTokens = map[string]bool{
	"YYYY": true, "YY": true,
	"MM": true, "MON": true, "MONTH": true,
	"DD": true, "DY": true, "DAY": true,
	"HH": true, "HH12": true, "HH24": true,
	"MI": true, "SS": true, "MS": true, "US": true,
	"AM": true, "PM": true,
	"TZ": true, "TZH": true, "TZM": true,
}
