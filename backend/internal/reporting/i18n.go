package reporting

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// ============================================================================
// MULTI-LANGUAGE INTERNATIONALIZATION (i18n) SUPPORT
// ============================================================================

// SupportedLocale represents a supported language/locale
type SupportedLocale string

const (
	LocaleEnUS SupportedLocale = "en-US"
	LocaleEnGB SupportedLocale = "en-GB"
	LocaleEsES SupportedLocale = "es-ES"
	LocaleEsMX SupportedLocale = "es-MX"
	LocaleFrFR SupportedLocale = "fr-FR"
	LocaleFrCA SupportedLocale = "fr-CA"
	LocaleDeDE SupportedLocale = "de-DE"
	LocaleItIT SupportedLocale = "it-IT"
	LocalePtBR SupportedLocale = "pt-BR"
	LocalePtPT SupportedLocale = "pt-PT"
	LocaleJaJP SupportedLocale = "ja-JP"
	LocaleZhCN SupportedLocale = "zh-CN"
	LocaleZhTW SupportedLocale = "zh-TW"
	LocaleKoKR SupportedLocale = "ko-KR"
	LocaleArSA SupportedLocale = "ar-SA"
	LocaleHiIN SupportedLocale = "hi-IN"
	LocaleRuRU SupportedLocale = "ru-RU"
	LocaleNlNL SupportedLocale = "nl-NL"
	LocalePlPL SupportedLocale = "pl-PL"
	LocaleTrTR SupportedLocale = "tr-TR"
	LocaleThTH SupportedLocale = "th-TH"
	LocaleViVN SupportedLocale = "vi-VN"
	LocaleIdID SupportedLocale = "id-ID"
	LocaleMsMS SupportedLocale = "ms-MY"
)

// LocaleInfo contains metadata about a locale
type LocaleInfo struct {
	Code        SupportedLocale `json:"code"`
	Name        string          `json:"name"`
	NativeName  string          `json:"native_name"`
	Direction   string          `json:"direction"` // "ltr" or "rtl"
	DateFormat  string          `json:"date_format"`
	NumberGroup string          `json:"number_group"`
	NumberDec   string          `json:"number_decimal"`
	Currency    string          `json:"currency"`
}

// LocaleRegistry contains all supported locales
var LocaleRegistry = map[SupportedLocale]LocaleInfo{
	LocaleEnUS: {LocaleEnUS, "English (US)", "English", "ltr", "MM/DD/YYYY", ",", ".", "USD"},
	LocaleEnGB: {LocaleEnGB, "English (UK)", "English", "ltr", "DD/MM/YYYY", ",", ".", "GBP"},
	LocaleEsES: {LocaleEsES, "Spanish (Spain)", "Español", "ltr", "DD/MM/YYYY", ".", ",", "EUR"},
	LocaleEsMX: {LocaleEsMX, "Spanish (Mexico)", "Español", "ltr", "DD/MM/YYYY", ",", ".", "MXN"},
	LocaleFrFR: {LocaleFrFR, "French (France)", "Français", "ltr", "DD/MM/YYYY", " ", ",", "EUR"},
	LocaleFrCA: {LocaleFrCA, "French (Canada)", "Français", "ltr", "YYYY-MM-DD", " ", ",", "CAD"},
	LocaleDeDE: {LocaleDeDE, "German", "Deutsch", "ltr", "DD.MM.YYYY", ".", ",", "EUR"},
	LocaleItIT: {LocaleItIT, "Italian", "Italiano", "ltr", "DD/MM/YYYY", ".", ",", "EUR"},
	LocalePtBR: {LocalePtBR, "Portuguese (Brazil)", "Português", "ltr", "DD/MM/YYYY", ".", ",", "BRL"},
	LocalePtPT: {LocalePtPT, "Portuguese (Portugal)", "Português", "ltr", "DD/MM/YYYY", " ", ",", "EUR"},
	LocaleJaJP: {LocaleJaJP, "Japanese", "日本語", "ltr", "YYYY/MM/DD", ",", ".", "JPY"},
	LocaleZhCN: {LocaleZhCN, "Chinese (Simplified)", "简体中文", "ltr", "YYYY-MM-DD", ",", ".", "CNY"},
	LocaleZhTW: {LocaleZhTW, "Chinese (Traditional)", "繁體中文", "ltr", "YYYY/MM/DD", ",", ".", "TWD"},
	LocaleKoKR: {LocaleKoKR, "Korean", "한국어", "ltr", "YYYY.MM.DD", ",", ".", "KRW"},
	LocaleArSA: {LocaleArSA, "Arabic", "العربية", "rtl", "DD/MM/YYYY", "٬", "٫", "SAR"},
	LocaleHiIN: {LocaleHiIN, "Hindi", "हिन्दी", "ltr", "DD/MM/YYYY", ",", ".", "INR"},
	LocaleRuRU: {LocaleRuRU, "Russian", "Русский", "ltr", "DD.MM.YYYY", " ", ",", "RUB"},
	LocaleNlNL: {LocaleNlNL, "Dutch", "Nederlands", "ltr", "DD-MM-YYYY", ".", ",", "EUR"},
	LocalePlPL: {LocalePlPL, "Polish", "Polski", "ltr", "DD.MM.YYYY", " ", ",", "PLN"},
	LocaleTrTR: {LocaleTrTR, "Turkish", "Türkçe", "ltr", "DD.MM.YYYY", ".", ",", "TRY"},
	LocaleThTH: {LocaleThTH, "Thai", "ไทย", "ltr", "DD/MM/YYYY", ",", ".", "THB"},
	LocaleViVN: {LocaleViVN, "Vietnamese", "Tiếng Việt", "ltr", "DD/MM/YYYY", ".", ",", "VND"},
	LocaleIdID: {LocaleIdID, "Indonesian", "Bahasa Indonesia", "ltr", "DD/MM/YYYY", ".", ",", "IDR"},
	LocaleMsMS: {LocaleMsMS, "Malay", "Bahasa Melayu", "ltr", "DD/MM/YYYY", ",", ".", "MYR"},
}

// TranslationKey represents a translation string key
type TranslationKey string

// ReportTranslation stores translations for a report definition
type ReportTranslation struct {
	ID                 uuid.UUID         `db:"id" json:"id"`
	TenantID           uuid.UUID         `db:"tenant_id" json:"tenant_id"`
	ReportDefinitionID uuid.UUID         `db:"report_definition_id" json:"report_definition_id"`
	Locale             SupportedLocale   `db:"locale" json:"locale"`
	Translations       map[string]string `db:"-" json:"translations"`
	TranslationsJSON   []byte            `db:"translations" json:"-"`
	IsComplete         bool              `db:"is_complete" json:"is_complete"`
	LastUpdated        json.RawMessage   `db:"last_updated" json:"last_updated"`
}

// TranslationService manages report translations
type TranslationService struct {
	cache     map[string]*ReportTranslation // tenantID:reportID:locale -> translation
	cacheLock sync.RWMutex
	repo      *Repository
}

// NewTranslationService creates a translation service
func NewTranslationService(repo *Repository) *TranslationService {
	return &TranslationService{
		cache: make(map[string]*ReportTranslation),
		repo:  repo,
	}
}

// TranslationContext holds the current translation context for a request
type TranslationContext struct {
	Locale       SupportedLocale
	Fallback     SupportedLocale
	Translations map[string]string
	LocaleInfo   LocaleInfo
}

// NewTranslationContext creates a new translation context
func NewTranslationContext(locale SupportedLocale) *TranslationContext {
	fallback := LocaleEnUS
	if info, ok := LocaleRegistry[locale]; ok {
		return &TranslationContext{
			Locale:       locale,
			Fallback:     fallback,
			Translations: make(map[string]string),
			LocaleInfo:   info,
		}
	}
	return &TranslationContext{
		Locale:       fallback,
		Fallback:     fallback,
		Translations: make(map[string]string),
		LocaleInfo:   LocaleRegistry[fallback],
	}
}

// T translates a key to the current locale
func (tc *TranslationContext) T(key string, args ...interface{}) string {
	if translation, ok := tc.Translations[key]; ok {
		if len(args) > 0 {
			return fmt.Sprintf(translation, args...)
		}
		return translation
	}
	// Return key if no translation found
	return key
}

// LocalizedReportDefinition extends ReportDefinition with locale-aware fields
type LocalizedReportDefinition struct {
	*ReportDefinition
	LocalizedDisplayName string               `json:"localized_display_name"`
	LocalizedDescription string               `json:"localized_description"`
	LocalizedCategory    string               `json:"localized_category"`
	LocalizedParameters  []LocalizedParameter `json:"localized_parameters,omitempty"`
	Locale               SupportedLocale      `json:"locale"`
}

// LocalizedParameter is a parameter with translated label/description
type LocalizedParameter struct {
	Parameter
	LocalizedLabel       string            `json:"localized_label"`
	LocalizedDescription string            `json:"localized_description"`
	LocalizedOptions     []LocalizedOption `json:"localized_options,omitempty"`
}

// LocalizedOption is a select option with translated label
type LocalizedOption struct {
	Value          string `json:"value"`
	LocalizedLabel string `json:"localized_label"`
}

// GetLocalizedDefinition returns a report definition with translations applied
func (ts *TranslationService) GetLocalizedDefinition(def *ReportDefinition, locale SupportedLocale) *LocalizedReportDefinition {
	cacheKey := fmt.Sprintf("%s:%s:%s", def.TenantID, def.ID, locale)

	ts.cacheLock.RLock()
	trans := ts.cache[cacheKey]
	ts.cacheLock.RUnlock()

	localized := &LocalizedReportDefinition{
		ReportDefinition:     def,
		LocalizedDisplayName: def.DisplayName,
		LocalizedDescription: def.Description,
		LocalizedCategory:    def.Category,
		Locale:               locale,
	}

	if trans != nil {
		if v, ok := trans.Translations["display_name"]; ok {
			localized.LocalizedDisplayName = v
		}
		if v, ok := trans.Translations["description"]; ok {
			localized.LocalizedDescription = v
		}
		if v, ok := trans.Translations["category"]; ok {
			localized.LocalizedCategory = v
		}

		// Localize parameters
		for _, param := range def.ParametersSchema {
			lp := LocalizedParameter{
				Parameter:            param,
				LocalizedLabel:       param.Label,
				LocalizedDescription: param.Description,
			}

			paramKey := fmt.Sprintf("param.%s.label", param.Name)
			if v, ok := trans.Translations[paramKey]; ok {
				lp.LocalizedLabel = v
			}

			descKey := fmt.Sprintf("param.%s.description", param.Name)
			if v, ok := trans.Translations[descKey]; ok {
				lp.LocalizedDescription = v
			}

			// Localize options
			for _, opt := range param.Options {
				lo := LocalizedOption{
					Value:          opt.Value,
					LocalizedLabel: opt.Label,
				}
				optKey := fmt.Sprintf("param.%s.option.%s", param.Name, opt.Value)
				if v, ok := trans.Translations[optKey]; ok {
					lo.LocalizedLabel = v
				}
				lp.LocalizedOptions = append(lp.LocalizedOptions, lo)
			}

			localized.LocalizedParameters = append(localized.LocalizedParameters, lp)
		}
	}

	return localized
}

// ============================================================================
// NUMBER & DATE FORMATTING
// ============================================================================

// NumberFormatter formats numbers according to locale
type NumberFormatter struct {
	locale LocaleInfo
}

// NewNumberFormatter creates a number formatter for a locale
func NewNumberFormatter(locale SupportedLocale) *NumberFormatter {
	info := LocaleRegistry[locale]
	if info.Code == "" {
		info = LocaleRegistry[LocaleEnUS]
	}
	return &NumberFormatter{locale: info}
}

// FormatNumber formats a number with locale-specific grouping
func (nf *NumberFormatter) FormatNumber(n float64, decimals int) string {
	// Split integer and decimal parts
	format := fmt.Sprintf("%%.%df", decimals)
	str := fmt.Sprintf(format, n)

	parts := strings.Split(str, ".")
	intPart := parts[0]

	// Handle negative numbers
	negative := false
	if strings.HasPrefix(intPart, "-") {
		negative = true
		intPart = intPart[1:]
	}

	// Add grouping separators
	var result strings.Builder
	for i, digit := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			result.WriteString(nf.locale.NumberGroup)
		}
		result.WriteRune(digit)
	}

	// Add decimal part
	if len(parts) > 1 && decimals > 0 {
		result.WriteString(nf.locale.NumberDec)
		result.WriteString(parts[1])
	}

	if negative {
		return "-" + result.String()
	}
	return result.String()
}

// FormatCurrency formats a currency value
func (nf *NumberFormatter) FormatCurrency(n float64, currency string) string {
	if currency == "" {
		currency = nf.locale.Currency
	}

	formatted := nf.FormatNumber(n, 2)

	// Currency symbol placement varies by locale
	switch nf.locale.Code {
	case LocaleDeDE, LocaleFrFR, LocaleEsES, LocaleItIT, LocalePtPT, LocaleRuRU, LocalePlPL:
		return formatted + " " + currency
	default:
		return currency + formatted
	}
}

// FormatPercent formats a percentage
func (nf *NumberFormatter) FormatPercent(n float64, decimals int) string {
	return nf.FormatNumber(n*100, decimals) + "%"
}

// DateFormatter formats dates according to locale
type DateFormatter struct {
	locale LocaleInfo
}

// NewDateFormatter creates a date formatter for a locale
func NewDateFormatter(locale SupportedLocale) *DateFormatter {
	info := LocaleRegistry[locale]
	if info.Code == "" {
		info = LocaleRegistry[LocaleEnUS]
	}
	return &DateFormatter{locale: info}
}

// GetFormat returns the date format string for the locale
func (df *DateFormatter) GetFormat() string {
	return df.locale.DateFormat
}

// ============================================================================
// RTL (RIGHT-TO-LEFT) SUPPORT
// ============================================================================

// IsRTL returns whether the locale uses right-to-left text direction
func IsRTL(locale SupportedLocale) bool {
	info := LocaleRegistry[locale]
	return info.Direction == "rtl"
}

// RTLTransform transforms CSS for RTL layouts
func RTLTransform(css map[string]string, locale SupportedLocale) map[string]string {
	if !IsRTL(locale) {
		return css
	}

	result := make(map[string]string)
	for key, value := range css {
		// Swap left/right properties
		switch key {
		case "margin-left":
			result["margin-right"] = value
		case "margin-right":
			result["margin-left"] = value
		case "padding-left":
			result["padding-right"] = value
		case "padding-right":
			result["padding-left"] = value
		case "text-align":
			if value == "left" {
				result["text-align"] = "right"
			} else if value == "right" {
				result["text-align"] = "left"
			} else {
				result[key] = value
			}
		case "float":
			if value == "left" {
				result["float"] = "right"
			} else if value == "right" {
				result["float"] = "left"
			} else {
				result[key] = value
			}
		default:
			result[key] = value
		}
	}
	return result
}

// ============================================================================
// TRANSLATION EXTRACTION & MANAGEMENT
// ============================================================================

// ExtractTranslationKeys extracts all translatable strings from a report definition
func ExtractTranslationKeys(def *ReportDefinition) map[string]string {
	keys := make(map[string]string)

	// Core fields
	keys["display_name"] = def.DisplayName
	keys["description"] = def.Description
	keys["category"] = def.Category

	// Parameters
	for _, param := range def.ParametersSchema {
		keys[fmt.Sprintf("param.%s.label", param.Name)] = param.Label
		keys[fmt.Sprintf("param.%s.description", param.Name)] = param.Description

		for _, opt := range param.Options {
			keys[fmt.Sprintf("param.%s.option.%s", param.Name, opt.Value)] = opt.Label
		}
	}

	// Layout sections
	if def.Definition != nil {
		for _, section := range def.Definition.Layout.Body.Sections {
			keys[fmt.Sprintf("section.%s.title", section.ID)] = section.Title

			// Table column headers
			for _, col := range section.Columns {
				keys[fmt.Sprintf("section.%s.column.%s", section.ID, col.Label)] = col.Label
			}
		}
	}

	return keys
}

// TranslationExport represents an export format for translation files
type TranslationExport struct {
	ReportKey    string            `json:"report_key"`
	SourceLocale SupportedLocale   `json:"source_locale"`
	TargetLocale SupportedLocale   `json:"target_locale"`
	Strings      map[string]string `json:"strings"`
}

// ExportForTranslation exports translation keys for external translation
func ExportForTranslation(def *ReportDefinition, targetLocale SupportedLocale) *TranslationExport {
	return &TranslationExport{
		ReportKey:    def.ReportKey,
		SourceLocale: LocaleEnUS,
		TargetLocale: targetLocale,
		Strings:      ExtractTranslationKeys(def),
	}
}
