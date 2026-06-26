import type { FC, ReactNode } from 'react';

/**
 * Example Component Showing Dark Mode Best Practices
 * 
 * This component demonstrates how to properly support dark mode using:
 * 1. Tailwind's dark: prefix
 * 2. CSS variables from index.css
 * 3. Proper contrast ratios
 * 4. Responsive design with theme support
 */

interface ExampleCardProps {
  title: string;
  description?: string;
  status?: 'success' | 'warning' | 'error' | 'info';
  children?: ReactNode;
}

export const ExampleThemeCard: FC<ExampleCardProps> = ({
  title,
  description,
  status = 'info',
  children,
}) => {
  // Status-specific styling for dark mode
  const statusStyles = {
    success: {
      border: 'border-green-200 dark:border-green-800',
      bg: 'bg-green-50 dark:bg-green-950/20',
      badge: 'bg-green-100 dark:bg-green-900/40 text-green-700 dark:text-green-300',
      indicator: 'bg-green-500 dark:bg-green-400',
    },
    warning: {
      border: 'border-yellow-200 dark:border-yellow-800',
      bg: 'bg-yellow-50 dark:bg-yellow-950/20',
      badge: 'bg-yellow-100 dark:bg-yellow-900/40 text-yellow-700 dark:text-yellow-300',
      indicator: 'bg-yellow-500 dark:bg-yellow-400',
    },
    error: {
      border: 'border-red-200 dark:border-red-800',
      bg: 'bg-red-50 dark:bg-red-950/20',
      badge: 'bg-red-100 dark:bg-red-900/40 text-red-700 dark:text-red-300',
      indicator: 'bg-red-500 dark:bg-red-400',
    },
    info: {
      border: 'border-blue-200 dark:border-blue-800',
      bg: 'bg-blue-50 dark:bg-blue-950/20',
      badge: 'bg-blue-100 dark:bg-blue-900/40 text-blue-700 dark:text-blue-300',
      indicator: 'bg-blue-500 dark:bg-blue-400',
    },
  };

  const currentStatus = statusStyles[status];

  return (
    <div
      className={`
        rounded-lg border-2 transition-all duration-300
        ${currentStatus.border}
        ${currentStatus.bg}
      `}
    >
      {/* Header */}
      <div className="p-4 border-b border-inherit">
        <div className="flex items-center gap-3">
          {/* Status indicator */}
          <div
            className={`
              w-3 h-3 rounded-full flex-shrink-0
              ${currentStatus.indicator}
            `}
          />

          {/* Title and badge */}
          <div className="flex-1">
            <h3 className="text-lg font-semibold text-foreground">
              {title}
            </h3>
          </div>

          {/* Status badge */}
          <span
            className={`
              px-2.5 py-0.5 rounded-full text-xs font-medium capitalize
              ${currentStatus.badge}
            `}
          >
            {status}
          </span>
        </div>

        {/* Description */}
        {description && (
          <p className="text-sm text-muted-foreground mt-2">
            {description}
          </p>
        )}
      </div>

      {/* Content */}
      {children && (
        <div className="p-4 text-foreground">
          {children}
        </div>
      )}
    </div>
  );
};

/**
 * Example: Dashboard Section with Multiple Cards
 */
export const ExampleDashboardSection: FC = () => {
  return (
    <div className="bg-background text-foreground min-h-screen p-4">
      {/* Header */}
      <div className="max-w-6xl mx-auto mb-8">
        <h1 className="text-3xl font-bold mb-2">Dashboard</h1>
        <p className="text-muted-foreground">
          This example shows how to implement dark mode properly
        </p>
      </div>

      {/* Grid of cards */}
      <div className="max-w-6xl mx-auto grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-8">
        <ExampleThemeCard
          title="Success State"
          description="Everything is working perfectly"
          status="success"
        >
          <p className="text-sm">
            This card shows how to style success states with proper
            dark mode support.
          </p>
        </ExampleThemeCard>

        <ExampleThemeCard
          title="Warning State"
          description="Something needs attention"
          status="warning"
        >
          <p className="text-sm">
            This card demonstrates warning styling that looks good
            in both light and dark modes.
          </p>
        </ExampleThemeCard>

        <ExampleThemeCard
          title="Error State"
          description="An error has occurred"
          status="error"
        >
          <p className="text-sm">
            This shows error state with high contrast to draw attention.
          </p>
        </ExampleThemeCard>
      </div>

      {/* Color reference section */}
      <div className="max-w-6xl mx-auto bg-card border border-border rounded-lg p-6">
        <h2 className="text-2xl font-bold mb-6 text-foreground">
          Color Palette Reference
        </h2>

        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          {/* Color swatch examples */}
          <div>
            <div className="h-20 bg-background border border-border rounded-lg mb-2" />
            <p className="text-xs font-mono text-muted-foreground">
              background
            </p>
          </div>

          <div>
            <div className="h-20 bg-card border border-border rounded-lg mb-2" />
            <p className="text-xs font-mono text-muted-foreground">
              card
            </p>
          </div>

          <div>
            <div className="h-20 bg-primary rounded-lg mb-2" />
            <p className="text-xs font-mono text-muted-foreground">
              primary
            </p>
          </div>

          <div>
            <div className="h-20 bg-secondary rounded-lg mb-2" />
            <p className="text-xs font-mono text-muted-foreground">
              secondary
            </p>
          </div>

          <div>
            <div className="h-20 bg-accent rounded-lg mb-2" />
            <p className="text-xs font-mono text-muted-foreground">
              accent
            </p>
          </div>

          <div>
            <div className="h-20 bg-destructive rounded-lg mb-2" />
            <p className="text-xs font-mono text-muted-foreground">
              destructive
            </p>
          </div>
        </div>
      </div>

      {/* Code example section */}
      <div className="max-w-6xl mx-auto mt-8 bg-card border border-border rounded-lg p-6">
        <h2 className="text-2xl font-bold mb-6 text-foreground">
          How to Use
        </h2>

        <div className="bg-muted p-4 rounded-lg mb-4 overflow-x-auto">
          <code className="text-muted-foreground text-sm font-mono">
            {`// Use Tailwind's dark: prefix
<div className="bg-white dark:bg-slate-900 text-black dark:text-white">
  Content
</div>

// Or use CSS variables
<div className="bg-background text-foreground border border-border">
  Content
</div>`}
          </code>
        </div>

        <p className="text-muted-foreground text-sm">
          The second approach is recommended as it automatically uses the theme
          colors defined in your application.
        </p>
      </div>
    </div>
  );
};

export default ExampleDashboardSection;
