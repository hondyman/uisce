import React, { createContext, useContext, useState, useCallback, useEffect, ReactNode } from 'react';
import { devWarn } from '../utils/devLogger';

// ============================================================================
// I18N TYPES
// ============================================================================

export type Language = 
  | 'en' // English
  | 'es' // Spanish
  | 'fr' // French
  | 'de' // German
  | 'pt' // Portuguese
  | 'it' // Italian
  | 'nl' // Dutch
  | 'ru' // Russian
  | 'zh' // Chinese (Simplified)
  | 'ja' // Japanese
  | 'ko' // Korean
  | 'ar' // Arabic
  | 'he' // Hebrew
  | 'hi'; // Hindi

export interface LanguageInfo {
  code: Language;
  name: string;
  nativeName: string;
  direction: 'ltr' | 'rtl';
  dateFormat: string;
  numberFormat: Intl.NumberFormatOptions;
}

export const SUPPORTED_LANGUAGES: Record<Language, LanguageInfo> = {
  en: { code: 'en', name: 'English', nativeName: 'English', direction: 'ltr', dateFormat: 'MM/dd/yyyy', numberFormat: { style: 'decimal', useGrouping: true } },
  es: { code: 'es', name: 'Spanish', nativeName: 'Español', direction: 'ltr', dateFormat: 'dd/MM/yyyy', numberFormat: { style: 'decimal', useGrouping: true } },
  fr: { code: 'fr', name: 'French', nativeName: 'Français', direction: 'ltr', dateFormat: 'dd/MM/yyyy', numberFormat: { style: 'decimal', useGrouping: true } },
  de: { code: 'de', name: 'German', nativeName: 'Deutsch', direction: 'ltr', dateFormat: 'dd.MM.yyyy', numberFormat: { style: 'decimal', useGrouping: true } },
  pt: { code: 'pt', name: 'Portuguese', nativeName: 'Português', direction: 'ltr', dateFormat: 'dd/MM/yyyy', numberFormat: { style: 'decimal', useGrouping: true } },
  it: { code: 'it', name: 'Italian', nativeName: 'Italiano', direction: 'ltr', dateFormat: 'dd/MM/yyyy', numberFormat: { style: 'decimal', useGrouping: true } },
  nl: { code: 'nl', name: 'Dutch', nativeName: 'Nederlands', direction: 'ltr', dateFormat: 'dd-MM-yyyy', numberFormat: { style: 'decimal', useGrouping: true } },
  ru: { code: 'ru', name: 'Russian', nativeName: 'Русский', direction: 'ltr', dateFormat: 'dd.MM.yyyy', numberFormat: { style: 'decimal', useGrouping: true } },
  zh: { code: 'zh', name: 'Chinese', nativeName: '中文', direction: 'ltr', dateFormat: 'yyyy/MM/dd', numberFormat: { style: 'decimal', useGrouping: true } },
  ja: { code: 'ja', name: 'Japanese', nativeName: '日本語', direction: 'ltr', dateFormat: 'yyyy/MM/dd', numberFormat: { style: 'decimal', useGrouping: true } },
  ko: { code: 'ko', name: 'Korean', nativeName: '한국어', direction: 'ltr', dateFormat: 'yyyy.MM.dd', numberFormat: { style: 'decimal', useGrouping: true } },
  ar: { code: 'ar', name: 'Arabic', nativeName: 'العربية', direction: 'rtl', dateFormat: 'dd/MM/yyyy', numberFormat: { style: 'decimal', useGrouping: true } },
  he: { code: 'he', name: 'Hebrew', nativeName: 'עברית', direction: 'rtl', dateFormat: 'dd/MM/yyyy', numberFormat: { style: 'decimal', useGrouping: true } },
  hi: { code: 'hi', name: 'Hindi', nativeName: 'हिन्दी', direction: 'ltr', dateFormat: 'dd/MM/yyyy', numberFormat: { style: 'decimal', useGrouping: true } },
};

// Translation dictionary type
export type TranslationKey = keyof typeof defaultTranslations.en;
export type Translations = Record<string, string>;

// ============================================================================
// DEFAULT TRANSLATIONS
// ============================================================================

const defaultTranslations = {
  en: {
    // Common
    'common.loading': 'Loading...',
    'common.error': 'Error',
    'common.success': 'Success',
    'common.save': 'Save',
    'common.cancel': 'Cancel',
    'common.delete': 'Delete',
    'common.edit': 'Edit',
    'common.create': 'Create',
    'common.search': 'Search',
    'common.filter': 'Filter',
    'common.export': 'Export',
    'common.import': 'Import',
    'common.refresh': 'Refresh',
    'common.close': 'Close',
    'common.confirm': 'Confirm',
    'common.actions': 'Actions',
    'common.status': 'Status',
    'common.name': 'Name',
    'common.description': 'Description',
    'common.type': 'Type',
    'common.createdAt': 'Created At',
    'common.updatedAt': 'Updated At',
    'common.createdBy': 'Created By',
    'common.noResults': 'No results found',
    'common.selectAll': 'Select All',
    'common.clearAll': 'Clear All',
    
    // Reports
    'reports.title': 'Reports',
    'reports.library': 'Report Library',
    'reports.designer': 'Report Designer',
    'reports.viewer': 'Report Viewer',
    'reports.scheduler': 'Report Scheduler',
    'reports.history': 'Report History',
    'reports.create': 'Create Report',
    'reports.edit': 'Edit Report',
    'reports.delete': 'Delete Report',
    'reports.run': 'Run Report',
    'reports.schedule': 'Schedule Report',
    'reports.parameters': 'Parameters',
    'reports.outputFormat': 'Output Format',
    'reports.favorites': 'Favorites',
    'reports.recent': 'Recent',
    'reports.shared': 'Shared With Me',
    'reports.myReports': 'My Reports',
    
    // Scheduler
    'scheduler.title': 'Schedule',
    'scheduler.create': 'Create Schedule',
    'scheduler.edit': 'Edit Schedule',
    'scheduler.cronExpression': 'Cron Expression',
    'scheduler.frequency': 'Frequency',
    'scheduler.startDate': 'Start Date',
    'scheduler.endDate': 'End Date',
    'scheduler.timezone': 'Timezone',
    'scheduler.nextRun': 'Next Run',
    'scheduler.lastRun': 'Last Run',
    'scheduler.active': 'Active',
    'scheduler.paused': 'Paused',
    'scheduler.delivery': 'Delivery',
    'scheduler.email': 'Email',
    'scheduler.webhook': 'Webhook',
    'scheduler.storage': 'Storage',
    
    // Designer
    'designer.addSection': 'Add Section',
    'designer.addElement': 'Add Element',
    'designer.header': 'Header',
    'designer.body': 'Body',
    'designer.footer': 'Footer',
    'designer.chart': 'Chart',
    'designer.table': 'Table',
    'designer.text': 'Text',
    'designer.image': 'Image',
    'designer.spacer': 'Spacer',
    'designer.pageBreak': 'Page Break',
    'designer.properties': 'Properties',
    'designer.dataBinding': 'Data Binding',
    'designer.styling': 'Styling',
    'designer.preview': 'Preview',
    
    // Collaboration
    'collab.comments': 'Comments',
    'collab.addComment': 'Add Comment',
    'collab.resolve': 'Resolve',
    'collab.reply': 'Reply',
    'collab.share': 'Share',
    'collab.shareWith': 'Share with...',
    'collab.copyLink': 'Copy Link',
    'collab.permissions': 'Permissions',
    'collab.viewer': 'Viewer',
    'collab.editor': 'Editor',
    'collab.admin': 'Admin',
    'collab.collaborators': 'Collaborators',
    'collab.activeNow': 'Active Now',
    
    // Validation
    'validation.required': 'This field is required',
    'validation.minLength': 'Minimum {min} characters required',
    'validation.maxLength': 'Maximum {max} characters allowed',
    'validation.invalidEmail': 'Invalid email address',
    'validation.invalidUrl': 'Invalid URL',
    'validation.invalidCron': 'Invalid cron expression',
    
    // Errors
    'error.generic': 'Something went wrong. Please try again.',
    'error.notFound': 'Resource not found',
    'error.unauthorized': 'You are not authorized to perform this action',
    'error.forbidden': 'Access denied',
    'error.serverError': 'Server error. Please try again later.',
    'error.networkError': 'Network error. Please check your connection.',
    'error.timeout': 'Request timed out. Please try again.',
    'error.quotaExceeded': 'Quota exceeded. Please upgrade your plan.',
    'error.rateLimited': 'Too many requests. Please wait and try again.',
    
    // Success messages
    'success.saved': 'Changes saved successfully',
    'success.created': 'Created successfully',
    'success.deleted': 'Deleted successfully',
    'success.exported': 'Exported successfully',
    'success.scheduled': 'Schedule created successfully',
    'success.shared': 'Shared successfully',
  },
  es: {
    'common.loading': 'Cargando...',
    'common.error': 'Error',
    'common.success': 'Éxito',
    'common.save': 'Guardar',
    'common.cancel': 'Cancelar',
    'common.delete': 'Eliminar',
    'common.edit': 'Editar',
    'common.create': 'Crear',
    'common.search': 'Buscar',
    'common.filter': 'Filtrar',
    'common.export': 'Exportar',
    'common.import': 'Importar',
    'common.refresh': 'Actualizar',
    'common.close': 'Cerrar',
    'common.confirm': 'Confirmar',
    'common.actions': 'Acciones',
    'common.status': 'Estado',
    'common.name': 'Nombre',
    'common.description': 'Descripción',
    'common.type': 'Tipo',
    'common.createdAt': 'Creado',
    'common.updatedAt': 'Actualizado',
    'common.createdBy': 'Creado por',
    'common.noResults': 'No se encontraron resultados',
    'common.selectAll': 'Seleccionar todo',
    'common.clearAll': 'Limpiar todo',
    'reports.title': 'Informes',
    'reports.library': 'Biblioteca de Informes',
    'reports.designer': 'Diseñador de Informes',
    'reports.viewer': 'Visor de Informes',
    'reports.scheduler': 'Programador de Informes',
    'reports.history': 'Historial de Informes',
    'reports.create': 'Crear Informe',
    'reports.edit': 'Editar Informe',
    'reports.delete': 'Eliminar Informe',
    'reports.run': 'Ejecutar Informe',
    'reports.schedule': 'Programar Informe',
    'reports.parameters': 'Parámetros',
    'reports.outputFormat': 'Formato de Salida',
    'reports.favorites': 'Favoritos',
    'reports.recent': 'Recientes',
    'reports.shared': 'Compartidos Conmigo',
    'reports.myReports': 'Mis Informes',
    'error.generic': 'Algo salió mal. Por favor, inténtelo de nuevo.',
    'error.notFound': 'Recurso no encontrado',
    'error.unauthorized': 'No está autorizado para realizar esta acción',
    'success.saved': 'Cambios guardados exitosamente',
    'success.created': 'Creado exitosamente',
    'success.deleted': 'Eliminado exitosamente',
  },
  fr: {
    'common.loading': 'Chargement...',
    'common.error': 'Erreur',
    'common.success': 'Succès',
    'common.save': 'Sauvegarder',
    'common.cancel': 'Annuler',
    'common.delete': 'Supprimer',
    'common.edit': 'Modifier',
    'common.create': 'Créer',
    'common.search': 'Rechercher',
    'common.filter': 'Filtrer',
    'common.export': 'Exporter',
    'common.import': 'Importer',
    'common.refresh': 'Actualiser',
    'common.close': 'Fermer',
    'common.confirm': 'Confirmer',
    'common.actions': 'Actions',
    'common.status': 'Statut',
    'common.name': 'Nom',
    'common.description': 'Description',
    'common.type': 'Type',
    'common.createdAt': 'Créé le',
    'common.updatedAt': 'Mis à jour le',
    'common.createdBy': 'Créé par',
    'common.noResults': 'Aucun résultat trouvé',
    'common.selectAll': 'Tout sélectionner',
    'common.clearAll': 'Tout effacer',
    'reports.title': 'Rapports',
    'reports.library': 'Bibliothèque de Rapports',
    'reports.designer': 'Concepteur de Rapports',
    'reports.viewer': 'Visionneuse de Rapports',
    'reports.scheduler': 'Planificateur de Rapports',
    'reports.history': 'Historique des Rapports',
    'reports.create': 'Créer un Rapport',
    'reports.edit': 'Modifier le Rapport',
    'reports.delete': 'Supprimer le Rapport',
    'reports.run': 'Exécuter le Rapport',
    'reports.schedule': 'Planifier le Rapport',
    'reports.parameters': 'Paramètres',
    'reports.outputFormat': 'Format de Sortie',
    'reports.favorites': 'Favoris',
    'reports.recent': 'Récents',
    'reports.shared': 'Partagés avec Moi',
    'reports.myReports': 'Mes Rapports',
    'error.generic': "Une erreur s'est produite. Veuillez réessayer.",
    'error.notFound': 'Ressource non trouvée',
    'error.unauthorized': "Vous n'êtes pas autorisé à effectuer cette action",
    'success.saved': 'Modifications enregistrées avec succès',
    'success.created': 'Créé avec succès',
    'success.deleted': 'Supprimé avec succès',
  },
  de: {
    'common.loading': 'Laden...',
    'common.error': 'Fehler',
    'common.success': 'Erfolg',
    'common.save': 'Speichern',
    'common.cancel': 'Abbrechen',
    'common.delete': 'Löschen',
    'common.edit': 'Bearbeiten',
    'common.create': 'Erstellen',
    'common.search': 'Suchen',
    'common.filter': 'Filtern',
    'common.export': 'Exportieren',
    'common.import': 'Importieren',
    'common.refresh': 'Aktualisieren',
    'common.close': 'Schließen',
    'common.confirm': 'Bestätigen',
    'common.actions': 'Aktionen',
    'common.status': 'Status',
    'common.name': 'Name',
    'common.description': 'Beschreibung',
    'common.type': 'Typ',
    'common.createdAt': 'Erstellt am',
    'common.updatedAt': 'Aktualisiert am',
    'common.createdBy': 'Erstellt von',
    'common.noResults': 'Keine Ergebnisse gefunden',
    'common.selectAll': 'Alle auswählen',
    'common.clearAll': 'Alle löschen',
    'reports.title': 'Berichte',
    'reports.library': 'Berichtsbibliothek',
    'reports.designer': 'Berichtsdesigner',
    'reports.viewer': 'Berichtsansicht',
    'reports.scheduler': 'Berichtsplaner',
    'reports.history': 'Berichtsverlauf',
    'reports.create': 'Bericht erstellen',
    'reports.edit': 'Bericht bearbeiten',
    'reports.delete': 'Bericht löschen',
    'reports.run': 'Bericht ausführen',
    'reports.schedule': 'Bericht planen',
    'reports.parameters': 'Parameter',
    'reports.outputFormat': 'Ausgabeformat',
    'reports.favorites': 'Favoriten',
    'reports.recent': 'Zuletzt verwendet',
    'reports.shared': 'Mit mir geteilt',
    'reports.myReports': 'Meine Berichte',
    'error.generic': 'Etwas ist schief gelaufen. Bitte versuchen Sie es erneut.',
    'error.notFound': 'Ressource nicht gefunden',
    'error.unauthorized': 'Sie sind nicht berechtigt, diese Aktion durchzuführen',
    'success.saved': 'Änderungen erfolgreich gespeichert',
    'success.created': 'Erfolgreich erstellt',
    'success.deleted': 'Erfolgreich gelöscht',
  },
  zh: {
    'common.loading': '加载中...',
    'common.error': '错误',
    'common.success': '成功',
    'common.save': '保存',
    'common.cancel': '取消',
    'common.delete': '删除',
    'common.edit': '编辑',
    'common.create': '创建',
    'common.search': '搜索',
    'common.filter': '筛选',
    'common.export': '导出',
    'common.import': '导入',
    'common.refresh': '刷新',
    'common.close': '关闭',
    'common.confirm': '确认',
    'common.actions': '操作',
    'common.status': '状态',
    'common.name': '名称',
    'common.description': '描述',
    'common.type': '类型',
    'common.createdAt': '创建时间',
    'common.updatedAt': '更新时间',
    'common.createdBy': '创建者',
    'common.noResults': '未找到结果',
    'common.selectAll': '全选',
    'common.clearAll': '清除全部',
    'reports.title': '报告',
    'reports.library': '报告库',
    'reports.designer': '报告设计器',
    'reports.viewer': '报告查看器',
    'reports.scheduler': '报告调度器',
    'reports.history': '报告历史',
    'reports.create': '创建报告',
    'reports.edit': '编辑报告',
    'reports.delete': '删除报告',
    'reports.run': '运行报告',
    'reports.schedule': '计划报告',
    'reports.parameters': '参数',
    'reports.outputFormat': '输出格式',
    'reports.favorites': '收藏夹',
    'reports.recent': '最近',
    'reports.shared': '与我共享',
    'reports.myReports': '我的报告',
    'error.generic': '出了点问题。请重试。',
    'error.notFound': '资源未找到',
    'error.unauthorized': '您无权执行此操作',
    'success.saved': '更改已成功保存',
    'success.created': '创建成功',
    'success.deleted': '删除成功',
  },
  ja: {
    'common.loading': '読み込み中...',
    'common.error': 'エラー',
    'common.success': '成功',
    'common.save': '保存',
    'common.cancel': 'キャンセル',
    'common.delete': '削除',
    'common.edit': '編集',
    'common.create': '作成',
    'common.search': '検索',
    'common.filter': 'フィルター',
    'common.export': 'エクスポート',
    'common.import': 'インポート',
    'common.refresh': '更新',
    'common.close': '閉じる',
    'common.confirm': '確認',
    'common.actions': 'アクション',
    'common.status': 'ステータス',
    'common.name': '名前',
    'common.description': '説明',
    'common.type': 'タイプ',
    'common.createdAt': '作成日時',
    'common.updatedAt': '更新日時',
    'common.createdBy': '作成者',
    'common.noResults': '結果が見つかりません',
    'common.selectAll': 'すべて選択',
    'common.clearAll': 'すべてクリア',
    'reports.title': 'レポート',
    'reports.library': 'レポートライブラリ',
    'reports.designer': 'レポートデザイナー',
    'reports.viewer': 'レポートビューア',
    'reports.scheduler': 'レポートスケジューラ',
    'reports.history': 'レポート履歴',
    'reports.create': 'レポートを作成',
    'reports.edit': 'レポートを編集',
    'reports.delete': 'レポートを削除',
    'reports.run': 'レポートを実行',
    'reports.schedule': 'レポートをスケジュール',
    'reports.parameters': 'パラメータ',
    'reports.outputFormat': '出力形式',
    'reports.favorites': 'お気に入り',
    'reports.recent': '最近',
    'reports.shared': '共有されたもの',
    'reports.myReports': 'マイレポート',
    'error.generic': '問題が発生しました。もう一度お試しください。',
    'error.notFound': 'リソースが見つかりません',
    'error.unauthorized': 'この操作を実行する権限がありません',
    'success.saved': '変更が正常に保存されました',
    'success.created': '正常に作成されました',
    'success.deleted': '正常に削除されました',
  },
  ar: {
    'common.loading': 'جاري التحميل...',
    'common.error': 'خطأ',
    'common.success': 'نجاح',
    'common.save': 'حفظ',
    'common.cancel': 'إلغاء',
    'common.delete': 'حذف',
    'common.edit': 'تعديل',
    'common.create': 'إنشاء',
    'common.search': 'بحث',
    'common.filter': 'تصفية',
    'common.export': 'تصدير',
    'common.import': 'استيراد',
    'common.refresh': 'تحديث',
    'common.close': 'إغلاق',
    'common.confirm': 'تأكيد',
    'common.actions': 'إجراءات',
    'common.status': 'الحالة',
    'common.name': 'الاسم',
    'common.description': 'الوصف',
    'common.type': 'النوع',
    'common.createdAt': 'تاريخ الإنشاء',
    'common.updatedAt': 'تاريخ التحديث',
    'common.createdBy': 'أنشأها',
    'common.noResults': 'لم يتم العثور على نتائج',
    'common.selectAll': 'تحديد الكل',
    'common.clearAll': 'مسح الكل',
    'reports.title': 'التقارير',
    'reports.library': 'مكتبة التقارير',
    'reports.designer': 'مصمم التقارير',
    'reports.viewer': 'عارض التقارير',
    'reports.scheduler': 'جدولة التقارير',
    'reports.history': 'سجل التقارير',
    'reports.create': 'إنشاء تقرير',
    'reports.edit': 'تعديل التقرير',
    'reports.delete': 'حذف التقرير',
    'reports.run': 'تشغيل التقرير',
    'reports.schedule': 'جدولة التقرير',
    'reports.parameters': 'المعلمات',
    'reports.outputFormat': 'تنسيق الإخراج',
    'reports.favorites': 'المفضلة',
    'reports.recent': 'الأخيرة',
    'reports.shared': 'مشترك معي',
    'reports.myReports': 'تقاريري',
    'error.generic': 'حدث خطأ ما. يرجى المحاولة مرة أخرى.',
    'error.notFound': 'المورد غير موجود',
    'error.unauthorized': 'ليس لديك صلاحية للقيام بهذا الإجراء',
    'success.saved': 'تم حفظ التغييرات بنجاح',
    'success.created': 'تم الإنشاء بنجاح',
    'success.deleted': 'تم الحذف بنجاح',
  },
  // Partial translations for remaining languages - would be completed in production
  pt: { 'common.loading': 'Carregando...', 'common.error': 'Erro', 'common.save': 'Salvar' },
  it: { 'common.loading': 'Caricamento...', 'common.error': 'Errore', 'common.save': 'Salva' },
  nl: { 'common.loading': 'Laden...', 'common.error': 'Fout', 'common.save': 'Opslaan' },
  ru: { 'common.loading': 'Загрузка...', 'common.error': 'Ошибка', 'common.save': 'Сохранить' },
  ko: { 'common.loading': '로딩 중...', 'common.error': '오류', 'common.save': '저장' },
  he: { 'common.loading': 'טוען...', 'common.error': 'שגיאה', 'common.save': 'שמור' },
  hi: { 'common.loading': 'लोड हो रहा है...', 'common.error': 'त्रुटि', 'common.save': 'सहेजें' },
};

// ============================================================================
// I18N CONTEXT
// ============================================================================

interface I18nContextValue {
  language: Language;
  languageInfo: LanguageInfo;
  setLanguage: (lang: Language) => void;
  t: (key: string, params?: Record<string, string | number>) => string;
  formatNumber: (value: number, options?: Intl.NumberFormatOptions) => string;
  formatCurrency: (value: number, currency: string) => string;
  formatDate: (date: Date | string, options?: Intl.DateTimeFormatOptions) => string;
  formatRelativeTime: (date: Date | string) => string;
  direction: 'ltr' | 'rtl';
  isRTL: boolean;
}

const I18nContext = createContext<I18nContextValue | undefined>(undefined);

// ============================================================================
// I18N PROVIDER
// ============================================================================

interface I18nProviderProps {
  children: ReactNode;
  defaultLanguage?: Language;
  loadTranslations?: (lang: Language) => Promise<Translations>;
}

export const I18nProvider: React.FC<I18nProviderProps> = ({
  children,
  defaultLanguage = 'en',
  loadTranslations,
}) => {
  const [language, setLanguageState] = useState<Language>(() => {
    // Try to get from localStorage
    const stored = localStorage.getItem('app_language');
    if (stored && stored in SUPPORTED_LANGUAGES) {
      return stored as Language;
    }
    // Try browser preference
    const browserLang = navigator.language.split('-')[0] as Language;
    if (browserLang in SUPPORTED_LANGUAGES) {
      return browserLang;
    }
    return defaultLanguage;
  });

  const [customTranslations, setCustomTranslations] = useState<Partial<Record<Language, Translations>>>({});

  const languageInfo = SUPPORTED_LANGUAGES[language];
  const isRTL = languageInfo.direction === 'rtl';

  // Load custom translations when language changes
  useEffect(() => {
    if (loadTranslations && !customTranslations[language]) {
      loadTranslations(language).then((translations) => {
        setCustomTranslations((prev) => ({ ...prev, [language]: translations }));
      }).catch(console.error);
    }
  }, [language, loadTranslations, customTranslations]);

  // Update document direction for RTL
  useEffect(() => {
    document.documentElement.dir = languageInfo.direction;
    document.documentElement.lang = language;
  }, [language, languageInfo.direction]);

  const setLanguage = useCallback((lang: Language) => {
    setLanguageState(lang);
    localStorage.setItem('app_language', lang);
  }, []);

  // Translation function with interpolation
  const t = useCallback((key: string, params?: Record<string, string | number>): string => {
    // Check custom translations first
    const customDict = customTranslations[language];
    let translation = customDict?.[key];

    // Fall back to default translations
    if (!translation) {
      const defaultDict = defaultTranslations[language] as Record<string, string> | undefined;
      translation = defaultDict?.[key];
    }

    // Fall back to English
    if (!translation) {
      const englishDict = defaultTranslations.en as Record<string, string>;
      translation = englishDict[key];
    }

    // Return key if no translation found
    if (!translation) {
      devWarn(`Missing translation for key: ${key}`);
      return key;
    }

    // Interpolate parameters
    if (params) {
      Object.entries(params).forEach(([param, value]) => {
        translation = translation!.replace(new RegExp(`{${param}}`, 'g'), String(value));
      });
    }

    return translation;
  }, [language, customTranslations]);

  const formatNumber = useCallback((value: number, options?: Intl.NumberFormatOptions): string => {
    return new Intl.NumberFormat(language, { ...languageInfo.numberFormat, ...options }).format(value);
  }, [language, languageInfo.numberFormat]);

  const formatCurrency = useCallback((value: number, currency: string): string => {
    return new Intl.NumberFormat(language, { style: 'currency', currency }).format(value);
  }, [language]);

  const formatDate = useCallback((date: Date | string, options?: Intl.DateTimeFormatOptions): string => {
    const dateObj = typeof date === 'string' ? new Date(date) : date;
    return new Intl.DateTimeFormat(language, options).format(dateObj);
  }, [language]);

  const formatRelativeTime = useCallback((date: Date | string): string => {
    const dateObj = typeof date === 'string' ? new Date(date) : date;
    const now = new Date();
    const diffMs = now.getTime() - dateObj.getTime();
    const diffSec = Math.floor(diffMs / 1000);
    const diffMin = Math.floor(diffSec / 60);
    const diffHour = Math.floor(diffMin / 60);
    const diffDay = Math.floor(diffHour / 24);

    const rtf = new Intl.RelativeTimeFormat(language, { numeric: 'auto' });

    if (diffDay > 0) return rtf.format(-diffDay, 'day');
    if (diffHour > 0) return rtf.format(-diffHour, 'hour');
    if (diffMin > 0) return rtf.format(-diffMin, 'minute');
    return rtf.format(-diffSec, 'second');
  }, [language]);

  const value: I18nContextValue = {
    language,
    languageInfo,
    setLanguage,
    t,
    formatNumber,
    formatCurrency,
    formatDate,
    formatRelativeTime,
    direction: languageInfo.direction,
    isRTL,
  };

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
};

// ============================================================================
// HOOKS
// ============================================================================

export const useI18n = (): I18nContextValue => {
  const context = useContext(I18nContext);
  if (!context) {
    throw new Error('useI18n must be used within an I18nProvider');
  }
  return context;
};

// Shorthand hook for translation
export const useTranslation = () => {
  const { t, language } = useI18n();
  return { t, language };
};

// Hook for formatting
export const useFormatting = () => {
  const { formatNumber, formatCurrency, formatDate, formatRelativeTime, language } = useI18n();
  return { formatNumber, formatCurrency, formatDate, formatRelativeTime, language };
};

// ============================================================================
// COMPONENTS
// ============================================================================

interface TransProps {
  i18nKey: string;
  params?: Record<string, string | number>;
  tag?: keyof JSX.IntrinsicElements;
  className?: string;
}

export const Trans: React.FC<TransProps> = ({ i18nKey, params, tag: Tag = 'span', className }) => {
  const { t } = useI18n();
  return <Tag className={className}>{t(i18nKey, params)}</Tag>;
};

// Language selector component
interface LanguageSelectorProps {
  showNativeName?: boolean;
  showFlag?: boolean;
  className?: string;
}

export const LanguageSelector: React.FC<LanguageSelectorProps> = ({
  showNativeName = true,
  className,
}) => {
  const { language, setLanguage } = useI18n();

  return (
    <select
      value={language}
      onChange={(e) => setLanguage(e.target.value as Language)}
      className={className}
      aria-label="Select language"
    >
      {Object.values(SUPPORTED_LANGUAGES).map((lang) => (
        <option key={lang.code} value={lang.code}>
          {showNativeName ? `${lang.nativeName} (${lang.name})` : lang.name}
        </option>
      ))}
    </select>
  );
};

// RTL wrapper component
interface RTLWrapperProps {
  children: ReactNode;
  className?: string;
}

export const RTLWrapper: React.FC<RTLWrapperProps> = ({ children, className }) => {
  const { direction } = useI18n();
  return (
    <div dir={direction} className={className}>
      {children}
    </div>
  );
};