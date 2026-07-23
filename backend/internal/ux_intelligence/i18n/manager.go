package i18n

import (
	"context"
)

type Translation struct {
	Key     string `json:"key"`
	Locale  string `json:"locale"`
	Value   string `json:"value"`
	Context string `json:"context,omitempty"`
}

type TranslationMemory struct {
	// Storage for all translations
}

func NewTranslationMemory() *TranslationMemory {
	return &TranslationMemory{}
}

func (tm *TranslationMemory) GetTranslations(ctx context.Context, locale string) ([]Translation, error) {
	// Mock: Return translations for locale
	// Real: Query translation database

	translations := []Translation{
		{Key: "common.submit", Locale: locale, Value: getLocalizedValue("submit", locale)},
		{Key: "common.cancel", Locale: locale, Value: getLocalizedValue("cancel", locale)},
		{Key: "common.save", Locale: locale, Value: getLocalizedValue("save", locale)},
		{Key: "common.delete", Locale: locale, Value: getLocalizedValue("delete", locale)},
	}

	return translations, nil
}

func (tm *TranslationMemory) AddTranslation(ctx context.Context, translation *Translation) error {
	// Mock: Store translation
	// Real: Save to translation database with versioning
	return nil
}

type LocaleManager struct {
	// Manages tenant-specific language packs and fallback rules
}

func NewLocaleManager() *LocaleManager {
	return &LocaleManager{}
}

func (lm *LocaleManager) GetTenantLocales(ctx context.Context, tenantID string) ([]string, error) {
	// Mock: Return supported locales for tenant
	// Real: Query tenant configuration
	return []string{"en", "es", "fr", "ja", "de"}, nil
}

func (lm *LocaleManager) GetFallbackLocale(ctx context.Context, locale string) string {
	// Fallback rules: specific locale -> language -> en
	switch locale {
	case "en-US", "en-GB", "en-CA":
		return "en"
	case "es-ES", "es-MX":
		return "es"
	case "fr-FR", "fr-CA":
		return "fr"
	default:
		return "en"
	}
}

func getLocalizedValue(key, locale string) string {
	values := map[string]map[string]string{
		"submit": {
			"en": "Submit",
			"es": "Enviar",
			"fr": "Soumettre",
			"ja": "送信",
			"de": "Einreichen",
		},
		"cancel": {
			"en": "Cancel",
			"es": "Cancelar",
			"fr": "Annuler",
			"ja": "キャンセル",
			"de": "Abbrechen",
		},
		"save": {
			"en": "Save",
			"es": "Guardar",
			"fr": "Enregistrer",
			"ja": "保存",
			"de": "Speichern",
		},
		"delete": {
			"en": "Delete",
			"es": "Eliminar",
			"fr": "Supprimer",
			"ja": "削除",
			"de": "Löschen",
		},
	}

	if localeMap, ok := values[key]; ok {
		if val, ok := localeMap[locale]; ok {
			return val
		}
	}
	return key
}
