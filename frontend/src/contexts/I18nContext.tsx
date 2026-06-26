import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import i18n from 'i18next';
import { initReactI18next, useTranslation as useI18nTranslation } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

// ============================================================================
// TYPES
// ============================================================================

export interface Locale {
  code: string;
  name: string;
  display_name: string;
  direction: 'ltr' | 'rtl';
  date_format: string;
  time_format: string;
  is_active: boolean;
}

interface I18nContextType {
  locale: string;
  setLocale: (locale: string) => void;
  availableLocales: Locale[];
  t: (key: string, defaultValue?: string, options?: any) => string;
  formatDate: (date: Date | string, format?: string) => string;
  formatNumber: (value: number, decimals?: number) => string;
  formatCurrency: (value: number, currency?: string) => string;
}

// ============================================================================
// CONTEXT
// ============================================================================

const I18nContext = createContext<I18nContextType | undefined>(undefined);

// ============================================================================
// i18next CONFIGURATION
// ============================================================================

// Default available locales (can be fetched from API)
const DEFAULT_LOCALES: Locale[] = [
  {
    code: 'en-US',
    name: 'English (United States)',
    display_name: 'English',
    direction: 'ltr',
    date_format: 'MM/DD/YYYY',
    time_format: 'h:mm A',
    is_active: true,
  },
  {
    code: 'es-MX',
    name: 'Spanish (Mexico)',
    display_name: 'Español',
    direction: 'ltr',
    date_format: 'DD/MM/YYYY',
    time_format: 'HH:mm',
    is_active: true,
  },
  {
    code: 'fr-FR',
    name: 'French (France)',
    display_name: 'Français',
    direction: 'ltr',
    date_format: 'DD/MM/YYYY',
    time_format: 'HH:mm',
    is_active: true,
  },
  {
    code: 'de-DE',
    name: 'German (Germany)',
    display_name: 'Deutsch',
    direction: 'ltr',
    date_format: 'DD.MM.YYYY',
    time_format: 'HH:mm',
    is_active: true,
  },
  {
    code: 'ja-JP',
    name: 'Japanese (Japan)',
    display_name: '日本語',
    direction: 'ltr',
    date_format: 'YYYY/MM/DD',
    time_format: 'HH:mm',
    is_active: true,
  },
  {
    code: 'zh-CN',
    name: 'Chinese (Simplified)',
    display_name: '简体中文',
    direction: 'ltr',
    date_format: 'YYYY/MM/DD',
    time_format: 'HH:mm',
    is_active: true,
  },
];

// Initialize i18next
i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    fallbackLng: 'en-US',
    debug: false,
    interpolation: {
      escapeValue: false,
    },
    resources: {
      'en-US': {
        common: {
          // Navigation
          dashboard: 'Dashboard',
          portfolios: 'Portfolios',
          reports: 'Reports',
          processes: 'Processes',
          settings: 'Settings',
          
          // Actions
          save: 'Save',
          cancel: 'Cancel',
          delete: 'Delete',
          edit: 'Edit',
          add: 'Add',
          create: 'Create',
          run: 'Run',
          approve: 'Approve',
          reject: 'Reject',
          submit: 'Submit',
          refresh: 'Refresh',
          
          // Common Terms
          loading: 'Loading...',
          error: 'Error',
          success: 'Success',
          warning: 'Warning',
          info: 'Information',
          noData: 'No data available',
          search: 'Search',
          filter: 'Filter',
          export: 'Export',
          import: 'Import',
        },
        portfolio: {
          marketValue: 'Market Value',
          costBasis: 'Cost Basis',
          unrealizedGL: 'Unrealized Gain/Loss',
          dayChange: 'Day Change',
          ytdReturn: 'YTD Return',
          holdings: 'Holdings',
          performance: 'Performance',
          allocation: 'Allocation',
        },
        report: {
          generateReport: 'Generate Report',
          reportReady: 'Your report is ready',
          executionHistory: 'Execution History',
          parameters: 'Parameters',
          outputFormat: 'Output Format',
        },
      },
      'es-MX': {
        common: {
          dashboard: 'Panel',
          portfolios: 'Portafolios',
          reports: 'Informes',
          processes: 'Procesos',
          settings: 'Configuración',
          save: 'Guardar',
          cancel: 'Cancelar',
          delete: 'Eliminar',
          edit: 'Editar',
          add: 'Agregar',
          create: 'Crear',
          run: 'Ejecutar',
          approve: 'Aprobar',
          reject: 'Rechazar',
        },
      },
      'fr-FR': {
        common: {
          dashboard: 'Tableau de bord',
          portfolios: 'Portefeuilles',
          reports: 'Rapports',
          processes: 'Processus',
          settings: 'Paramètres',
          save: 'Enregistrer',
          cancel: 'Annuler',
          delete: 'Supprimer',
          edit: 'Modifier',
        },
      },
    },
  });

// ============================================================================
// PROVIDER
// ============================================================================

interface I18nProviderProps {
  children: ReactNode;
}

export const I18nProvider: React.FC<I18nProviderProps> = ({ children }) => {
  const [locale, setLocaleState] = useState<string>(i18n.language || 'en-US');
  const [availableLocales] = useState<Locale[]>(DEFAULT_LOCALES);
  const { t: i18nT } = useI18nTranslation();

  const setLocale = (newLocale: string) => {
    i18n.changeLanguage(newLocale);
    setLocaleState(newLocale);
    localStorage.setItem('preferred_locale', newLocale);
    
    // Update document direction for RTL languages
    const localeInfo = availableLocales.find((l) => l.code === newLocale);
    if (localeInfo) {
      document.documentElement.dir = localeInfo.direction;
    }
  };

  useEffect(() => {
    // Load preferred locale from localStorage
    const savedLocale = localStorage.getItem('preferred_locale');
    if (savedLocale) {
      setLocale(savedLocale);
    }
  }, []);

  const t = (key: string, defaultValue?: string, options?: any): string => {
    const translation = i18nT(key, options);
    // Convert to string to handle all i18next return types
    const translationStr = String(translation);
    return translationStr !== key ? translationStr : (defaultValue || key);
  };

  const formatDate = (date: Date | string, format?: string): string => {
    const d = typeof date === 'string' ? new Date(date) : date;
    const localeInfo = availableLocales.find((l) => l.code === locale);
    const dateFormat = format || localeInfo?.date_format || 'MM/DD/YYYY';
    
    // Simple date formatting (in production, use date-fns or similar)
    const month = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    const year = d.getFullYear();
    
    return dateFormat
      .replace('MM', month)
      .replace('DD', day)
      .replace('YYYY', String(year));
  };

  const formatNumber = (value: number, decimals: number = 2): string => {
    return value.toLocaleString(locale, {
      minimumFractionDigits: decimals,
      maximumFractionDigits: decimals,
    });
  };

  const formatCurrency = (value: number, currency: string = 'USD'): string => {
    return value.toLocaleString(locale, {
      style: 'currency',
      currency: currency,
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    });
  };

  return (
    <I18nContext.Provider
      value={{
        locale,
        setLocale,
        availableLocales,
        t,
        formatDate,
        formatNumber,
        formatCurrency,
      }}
    >
      {children}
    </I18nContext.Provider>
  );
};

// ============================================================================
// HOOKS
// ============================================================================

export const useI18n = (): I18nContextType => {
  const context = useContext(I18nContext);
  if (!context) {
    throw new Error('useI18n must be used within I18nProvider');
  }
  return context;
};

// Hook for common translations
export const useTranslation = (namespace: string = 'common') => {
  const { t } = useI18n();
  return {
    t: (key: string, defaultValue?: string) => t(`${namespace}.${key}`, defaultValue),
  };
};

// ============================================================================
// LANGUAGE SELECTOR COMPONENT
// ============================================================================

import {
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  ListItemText,
  ListItemIcon,
} from '@mui/material';
import { Language as LanguageIcon } from '@mui/icons-material';

interface LanguageSelectorProps {
  variant?: 'standard' | 'outlined' | 'filled';
  size?: 'small' | 'medium';
  showLabel?: boolean;
}

export const LanguageSelector: React.FC<LanguageSelectorProps> = ({
  variant = 'outlined',
  size = 'small',
  showLabel = true,
}) => {
  const { locale, setLocale, availableLocales } = useI18n();

  return (
    <FormControl variant={variant} size={size} sx={{ minWidth: 150 }}>
      {showLabel && <InputLabel>Language</InputLabel>}
      <Select
        value={locale}
        label={showLabel ? 'Language' : undefined}
        onChange={(e) => setLocale(e.target.value)}
        startAdornment={<LanguageIcon sx={{ mr: 1, color: 'action.active' }} />}
      >
        {availableLocales
          .filter((l) => l.is_active)
          .map((l) => (
            <MenuItem key={l.code} value={l.code}>
              <ListItemText primary={l.display_name} secondary={l.name} />
            </MenuItem>
          ))}
      </Select>
    </FormControl>
  );
};

export default I18nProvider;
