/**
 * Dark Mode Styling Helpers
 * 
 * Utility functions to generate consistent dark mode CSS classes for common UI patterns.
 * These helpers ensure consistent styling across the platform when implementing dark mode.
 * 
 * Usage:
 * ```tsx
 * const cardClasses = getCardClasses();
 * const textClasses = getTextClasses('primary');
 * const badgeClasses = getBadgeClasses('error');
 * ```
 */

/**
 * Get card/surface container classes with dark mode support
 */
export const getCardClasses = (additional?: string) => {
  const base = 'rounded-lg border bg-white dark:bg-surface-dark p-5 border-slate-200 dark:border-border-dark';
  return additional ? `${base} ${additional}` : base;
};

/**
 * Get background color classes for different themes
 */
export const getBackgroundClasses = () => {
  return 'bg-background-light dark:bg-background-dark';
};

/**
 * Get page background with adaptive styling
 */
export const getPageBackgroundClasses = () => {
  return 'min-h-screen bg-background-light dark:bg-background-dark';
};

/**
 * Get text color classes for different hierarchy levels
 */
export const getTextClasses = (level: 'primary' | 'secondary' | 'muted' = 'primary') => {
  const textMap = {
    primary: 'text-slate-900 dark:text-text-light',
    secondary: 'text-slate-500 dark:text-text-dim',
    muted: 'text-slate-400 dark:text-text-dim/70',
  };
  return textMap[level];
};

/**
 * Get border classes with dark mode support
 */
export const getBorderClasses = (variant: 'default' | 'light' | 'subtle' = 'default') => {
  const borderMap = {
    default: 'border border-slate-200 dark:border-border-dark',
    light: 'border border-slate-100 dark:border-border-dark/50',
    subtle: 'border border-slate-50 dark:border-border-dark/30',
  };
  return borderMap[variant];
};

/**
 * Get input field classes with dark mode support
 */
export const getInputClasses = () => {
  return `
    w-full rounded-lg border px-4 py-2
    border-slate-300 bg-white text-slate-800 placeholder-slate-400
    dark:border-border-dark dark:bg-surface-dark dark:text-text-light dark:placeholder-text-dim
    focus:border-primary dark:focus:border-primary
    transition-colors
  `.trim();
};

/**
 * Get button classes for primary, secondary, and ghost buttons
 */
export const getButtonClasses = (variant: 'primary' | 'secondary' | 'ghost' = 'primary') => {
  const buttonMap = {
    primary: `
      inline-flex items-center justify-center rounded-lg px-4 py-2
      bg-primary text-white font-medium
      hover:bg-primary/90 dark:hover:bg-primary/80
      transition-colors
    `,
    secondary: `
      inline-flex items-center justify-center rounded-lg px-4 py-2
      bg-slate-200 text-slate-900
      dark:bg-surface-dark dark:text-text-light
      hover:bg-slate-300 dark:hover:bg-slate-700
      transition-colors
    `,
    ghost: `
      inline-flex items-center justify-center rounded-lg px-4 py-2
      text-slate-700 dark:text-text-dim
      hover:bg-slate-100 dark:hover:bg-slate-800/50
      transition-colors
    `,
  };
  return buttonMap[variant].trim();
};

/**
 * Get badge/chip classes for different severity levels
 */
export const getBadgeClasses = (severity: 'error' | 'warning' | 'info' | 'success' = 'info') => {
  const badgeMap = {
    error: 'inline-flex items-center gap-1 px-2 py-1 rounded text-xs font-semibold uppercase bg-red-100 text-red-700 dark:bg-red-900/50 dark:text-red-300',
    warning: 'inline-flex items-center gap-1 px-2 py-1 rounded text-xs font-semibold uppercase bg-amber-100 text-amber-700 dark:bg-amber-900/50 dark:text-amber-300',
    info: 'inline-flex items-center gap-1 px-2 py-1 rounded text-xs font-semibold uppercase bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300',
    success: 'inline-flex items-center gap-1 px-2 py-1 rounded text-xs font-semibold uppercase bg-emerald-100 text-emerald-700 dark:bg-emerald-900/50 dark:text-emerald-300',
  };
  return badgeMap[severity];
};

/**
 * Get section header classes for colored sections (like in Entity Details)
 */
export const getSectionHeaderClasses = (color: 'amber' | 'emerald' | 'violet' | 'blue' | 'red' = 'amber') => {
  const headerMap = {
    amber: {
      container: 'p-4 border-b bg-amber-50 border-amber-200 dark:bg-amber-900/20 dark:border-amber-500/30',
      iconBg: 'h-10 w-10 rounded-full bg-amber-100 dark:bg-amber-500/20',
      iconText: 'text-amber-600 dark:text-amber-400',
      title: 'text-slate-900 text-base font-bold dark:text-text-light',
    },
    emerald: {
      container: 'p-4 border-b bg-emerald-50 border-emerald-200 dark:bg-emerald-900/20 dark:border-emerald-500/30',
      iconBg: 'h-10 w-10 rounded-full bg-emerald-100 dark:bg-emerald-500/20',
      iconText: 'text-emerald-600 dark:text-emerald-400',
      title: 'text-slate-900 text-base font-bold dark:text-text-light',
    },
    violet: {
      container: 'p-4 border-b bg-violet-50 border-violet-200 dark:bg-violet-900/20 dark:border-violet-500/30',
      iconBg: 'h-10 w-10 rounded-full bg-violet-100 dark:bg-violet-500/20',
      iconText: 'text-violet-600 dark:text-violet-400',
      title: 'text-slate-900 text-base font-bold dark:text-text-light',
    },
    blue: {
      container: 'p-4 border-b bg-blue-50 border-blue-200 dark:bg-blue-900/20 dark:border-blue-500/30',
      iconBg: 'h-10 w-10 rounded-full bg-blue-100 dark:bg-blue-500/20',
      iconText: 'text-blue-600 dark:text-blue-400',
      title: 'text-slate-900 text-base font-bold dark:text-text-light',
    },
    red: {
      container: 'p-4 border-b bg-red-50 border-red-200 dark:bg-red-900/20 dark:border-red-500/30',
      iconBg: 'h-10 w-10 rounded-full bg-red-100 dark:bg-red-500/20',
      iconText: 'text-red-600 dark:text-red-400',
      title: 'text-slate-900 text-base font-bold dark:text-text-light',
    },
  };
  return headerMap[color];
};

/**
 * Get hover state classes with dark mode support
 */
export const getHoverClasses = () => {
  return 'hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors cursor-pointer';
};

/**
 * Get focus/active state classes
 */
export const getFocusClasses = () => {
  return 'focus:ring-2 focus:ring-primary focus:ring-offset-2 dark:focus:ring-offset-background-dark focus:outline-none';
};

/**
 * Get divider/separator classes
 */
export const getDividerClasses = () => {
  return 'border-t border-slate-200 dark:border-border-dark';
};

/**
 * Get header/title classes
 */
export const getHeaderClasses = (level: 'h1' | 'h2' | 'h3' = 'h2') => {
  const headerMap = {
    h1: 'text-4xl font-black text-slate-900 dark:text-text-light',
    h2: 'text-2xl font-bold text-slate-900 dark:text-text-light',
    h3: 'text-lg font-semibold text-slate-900 dark:text-text-light',
  };
  return headerMap[level];
};

/**
 * Get checkbox/form control classes
 */
export const getFormControlClasses = () => {
  return 'h-4 w-4 rounded border-slate-300 dark:border-border-dark dark:bg-surface-dark text-primary dark:text-primary focus:ring-primary/50 dark:focus:ring-primary/50';
};

/**
 * Get label classes for form fields
 */
export const getLabelClasses = () => {
  return 'text-sm font-medium text-slate-700 dark:text-text-light';
};

/**
 * Get help text classes
 */
export const getHelpTextClasses = () => {
  return 'text-xs text-slate-500 dark:text-text-dim';
};

/**
 * Get code block classes
 */
export const getCodeBlockClasses = () => {
  return 'bg-slate-100 dark:bg-background-dark p-4 rounded-lg overflow-x-auto border border-slate-200 dark:border-border-dark';
};

/**
 * Get code text (inline) classes
 */
export const getCodeTextClasses = () => {
  return 'bg-slate-100 dark:bg-slate-800 text-slate-800 dark:text-slate-200 px-1.5 py-0.5 rounded font-mono text-sm';
};

/**
 * Get alert/notification classes for different types
 */
export const getAlertClasses = (type: 'error' | 'warning' | 'success' | 'info' = 'info') => {
  const alertMap = {
    error: 'bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-500/30 text-red-800 dark:text-red-200 px-4 py-3 rounded-lg',
    warning: 'bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-500/30 text-amber-800 dark:text-amber-200 px-4 py-3 rounded-lg',
    success: 'bg-emerald-50 dark:bg-emerald-900/20 border border-emerald-200 dark:border-emerald-500/30 text-emerald-800 dark:text-emerald-200 px-4 py-3 rounded-lg',
    info: 'bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-500/30 text-blue-800 dark:text-blue-200 px-4 py-3 rounded-lg',
  };
  return alertMap[type];
};

/**
 * Get table classes with dark mode support
 */
export const getTableClasses = () => {
  return {
    container: 'w-full overflow-x-auto border border-slate-200 dark:border-border-dark rounded-lg',
    table: 'w-full',
    thead: 'bg-slate-50 dark:bg-slate-900/50 border-b border-slate-200 dark:border-border-dark',
    th: 'px-4 py-3 text-left text-sm font-semibold text-slate-900 dark:text-text-light',
    tbody: 'divide-y divide-slate-200 dark:divide-border-dark',
    tr: 'hover:bg-slate-50 dark:hover:bg-slate-800/30 transition-colors',
    td: 'px-4 py-3 text-sm text-slate-700 dark:text-text-light',
  };
};

/**
 * Get tab classes for tabbed interfaces
 */
export const getTabClasses = () => {
  return {
    container: 'border-b border-slate-200 dark:border-border-dark flex gap-8',
    tab: 'pb-3 px-1 border-b-[3px] border-b-transparent text-slate-500 dark:text-text-dim hover:text-slate-900 dark:hover:text-text-light transition-colors',
    active: 'border-b-slate-900 dark:border-b-primary text-slate-900 dark:text-text-light',
  };
};

/**
 * Get modal/dialog classes with dark mode
 */
export const getModalClasses = () => {
  return {
    overlay: 'fixed inset-0 bg-black/50 dark:bg-black/70 flex items-center justify-center',
    content: 'bg-white dark:bg-surface-dark rounded-lg shadow-lg border border-slate-200 dark:border-border-dark max-w-lg w-full mx-4',
    header: 'border-b border-slate-200 dark:border-border-dark px-6 py-4',
    body: 'px-6 py-4',
    footer: 'border-t border-slate-200 dark:border-border-dark px-6 py-4 flex gap-2 justify-end',
  };
};

/**
 * Combine multiple class helpers (useful for complex components)
 */
export const combineClasses = (...classes: (string | undefined)[]): string => {
  return classes.filter(Boolean).join(' ');
};

/**
 * Generate responsive classes for different breakpoints
 */
export const getResponsiveClasses = (mobileClass: string, desktopClass: string) => {
  return `${mobileClass} md:${desktopClass}`;
};

export default {
  getCardClasses,
  getBackgroundClasses,
  getPageBackgroundClasses,
  getTextClasses,
  getBorderClasses,
  getInputClasses,
  getButtonClasses,
  getBadgeClasses,
  getSectionHeaderClasses,
  getHoverClasses,
  getFocusClasses,
  getDividerClasses,
  getHeaderClasses,
  getFormControlClasses,
  getLabelClasses,
  getHelpTextClasses,
  getCodeBlockClasses,
  getCodeTextClasses,
  getAlertClasses,
  getTableClasses,
  getTabClasses,
  getModalClasses,
  combineClasses,
  getResponsiveClasses,
};
