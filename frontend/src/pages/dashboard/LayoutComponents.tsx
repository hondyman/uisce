import React, { ReactNode } from 'react';

interface ConsoleBreadcrumbsProps {
  items: Array<{
    label: string;
    href?: string;
    active?: boolean;
  }>;
}

export const ConsoleBreadcrumbs: React.FC<ConsoleBreadcrumbsProps> = ({ items }) => {
  return (
    <nav className="flex items-center gap-2 text-sm text-slate-500 dark:text-slate-400 mb-6">
      {items.map((item, idx) => (
        <React.Fragment key={idx}>
          {idx > 0 && <span className="text-xs">→</span>}
          {item.href ? (
            <a
              href={item.href}
              className="hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
            >
              {item.label}
            </a>
          ) : (
            <span
              className={
                item.active
                  ? 'text-slate-900 dark:text-slate-100 font-semibold'
                  : ''
              }
            >
              {item.label}
            </span>
          )}
        </React.Fragment>
      ))}
    </nav>
  );
};

interface ConsoleHeaderProps {
  title?: string;
  subtitle?: string;
  rightContent?: ReactNode;
}

export const ConsoleHeader: React.FC<ConsoleHeaderProps> = ({
  title,
  subtitle,
  rightContent,
}) => {
  return (
    <div className="mb-8">
      <div className="flex items-center justify-between gap-4">
        <div>
          {title && (
            <h1 className="text-3xl font-extrabold text-slate-900 dark:text-white mb-1">
              {title}
            </h1>
          )}
          {subtitle && (
            <p className="text-sm text-slate-600 dark:text-slate-400">{subtitle}</p>
          )}
        </div>
        {rightContent && <div className="flex items-center gap-4">{rightContent}</div>}
      </div>
    </div>
  );
};

interface ConsoleLayoutProps {
  children: ReactNode;
  maxWidth?: 'sm' | 'md' | 'lg' | 'xl' | '2xl' | 'full';
}

export const ConsoleLayout: React.FC<ConsoleLayoutProps> = ({
  children,
  maxWidth = '2xl',
}) => {
  const maxWidthClass = {
    sm: 'max-w-sm',
    md: 'max-w-md',
    lg: 'max-w-lg',
    xl: 'max-w-xl',
    '2xl': 'max-w-6xl',
    full: 'max-w-full',
  }[maxWidth];

  return (
    <main className={`mx-auto px-6 py-8 ${maxWidthClass}`}>
      {children}
    </main>
  );
};

interface TopNavProps {
  title: string;
  logo?: ReactNode;
  navItems?: Array<{
    label: string;
    href: string;
    active?: boolean;
  }>;
  rightContent?: ReactNode;
}

export const ConsoleTopNav: React.FC<TopNavProps> = ({
  title,
  logo,
  navItems,
  rightContent,
}) => {
  return (
    <header className="sticky top-0 z-50 w-full bg-white dark:bg-slate-900 border-b border-slate-200 dark:border-slate-800 px-6 py-3">
      <div className="max-w-6xl mx-auto flex items-center justify-between">
        <div className="flex items-center gap-8">
          <div className="flex items-center gap-3">
            {logo && <div className="shrink-0">{logo}</div>}
            <h1 className="text-lg font-bold text-slate-900 dark:text-white tracking-tight">
              {title}
            </h1>
          </div>
          {navItems && navItems.length > 0 && (
            <nav className="hidden md:flex items-center gap-6">
              {navItems.map((item, idx) => (
                <a
                  key={idx}
                  href={item.href}
                  className={`text-sm font-medium transition-colors ${
                    item.active
                      ? 'text-blue-600 dark:text-blue-400 border-b-2 border-blue-600 pb-1'
                      : 'text-slate-600 dark:text-slate-400 hover:text-blue-600 dark:hover:text-blue-400'
                  }`}
                >
                  {item.label}
                </a>
              ))}
            </nav>
          )}
        </div>
        {rightContent && <div className="flex items-center gap-4">{rightContent}</div>}
      </div>
    </header>
  );
};

interface StatusBarProps {
  items: Array<{
    label: string;
    value: string | ReactNode;
    icon?: ReactNode;
  }>;
  rightItems?: Array<{
    label: string;
    value: string | ReactNode;
  }>;
}

export const ConsoleStatusBar: React.FC<StatusBarProps> = ({ items, rightItems }) => {
  return (
    <footer className="fixed bottom-0 left-0 right-0 bg-white dark:bg-slate-900 border-t border-slate-200 dark:border-slate-800 px-6 py-2 flex items-center justify-between text-xs font-medium text-slate-600 dark:text-slate-400">
      <div className="flex items-center gap-6">
        {items.map((item, idx) => (
          <div key={idx} className="flex items-center gap-2">
            {item.icon && <span>{item.icon}</span>}
            <span className="text-slate-600 dark:text-slate-400 truncate">{item.label}</span>
            <span className="text-slate-900 dark:text-white font-semibold truncate">
              {item.value}
            </span>
          </div>
        ))}
      </div>
      {rightItems && (
        <div className="flex items-center gap-4">
          {rightItems.map((item, idx) => (
            <div key={idx} className="flex items-center gap-2">
              <span className="text-slate-600 dark:text-slate-400">{item.label}</span>
              <span className="text-slate-900 dark:text-white font-semibold border-l border-slate-200 dark:border-slate-800 pl-4">
                {item.value}
              </span>
            </div>
          ))}
        </div>
      )}
    </footer>
  );
};

interface ConsoleGridProps {
  children: ReactNode;
  columns?: 1 | 2 | 3 | 4;
  gap?: 'sm' | 'md' | 'lg';
}

export const ConsoleGrid: React.FC<ConsoleGridProps> = ({
  children,
  columns = 2,
  gap = 'md',
}) => {
  const colClass = {
    1: 'grid-cols-1',
    2: 'lg:grid-cols-2',
    3: 'lg:grid-cols-3',
    4: 'lg:grid-cols-4',
  }[columns];

  const gapClass = {
    sm: 'gap-3',
    md: 'gap-6',
    lg: 'gap-8',
  }[gap];

  return <div className={`grid grid-cols-1 ${colClass} ${gapClass}`}>{children}</div>;
};
