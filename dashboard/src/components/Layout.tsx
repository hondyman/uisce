import React from 'react'

interface LayoutProps {
  children: React.ReactNode
  sidebar?: React.ReactNode
  header?: React.ReactNode
}

export function Layout({ children, sidebar, header }: LayoutProps) {
  return (
    <div className="flex h-screen bg-slate-100">
      {/* Sidebar */}
      {sidebar && (
        <aside className="w-64 bg-brand text-white shadow-lg">
          {sidebar}
        </aside>
      )}

      {/* Main Content */}
      <div className="flex-1 flex flex-col">
        {/* Header */}
        {header && (
          <header className="bg-white shadow border-b border-slate-200">
            {header}
          </header>
        )}

        {/* Content Area */}
        <main className="flex-1 overflow-auto p-8">
          {children}
        </main>
      </div>
    </div>
  )
}

interface CardProps {
  title?: string
  children: React.ReactNode
  className?: string
  footer?: React.ReactNode
}

export function Card({ title, children, className = '', footer }: CardProps) {
  return (
    <div className={`bg-white rounded-lg shadow-md border border-slate-200 ${className}`}>
      {title && (
        <div className="px-6 py-4 border-b border-slate-200">
          <h3 className="text-lg font-semibold text-slate-900">{title}</h3>
        </div>
      )}

      <div className="p-6">
        {children}
      </div>

      {footer && (
        <div className="px-6 py-4 bg-slate-50 border-t border-slate-200">
          {footer}
        </div>
      )}
    </div>
  )
}

interface MetricCardProps {
  label: string
  value: string | number
  change?: number
  trend?: 'up' | 'down' | 'stable'
  color?: 'success' | 'warning' | 'danger' | 'info'
}

export function MetricCard({ label, value, change, trend, color = 'info' }: MetricCardProps) {
  const colorClasses = {
    success: 'text-emerald-600',
    warning: 'text-amber-600',
    danger: 'text-red-600',
    info: 'text-blue-600'
  }

  const trendIcons = {
    up: '↑',
    down: '↓',
    stable: '→'
  }

  return (
    <Card className="flex flex-col justify-between">
      <p className="text-slate-600 text-sm font-medium">{label}</p>
      <div className="mt-2">
        <p className="text-3xl font-bold text-slate-900">{value}</p>
        {change !== undefined && (
          <p className={`mt-1 text-sm font-medium ${colorClasses[color]}`}>
            {trend && <span>{trendIcons[trend]}</span>} {change > 0 ? '+' : ''}{change}%
          </p>
        )}
      </div>
    </Card>
  )
}

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'danger'
  size?: 'sm' | 'md' | 'lg'
  isLoading?: boolean
}

export function Button({ 
  variant = 'primary', 
  size = 'md',
  isLoading = false,
  disabled,
  children,
  className = '',
  ...props 
}: ButtonProps) {
  const baseClasses = 'font-medium rounded-lg transition duration-200 ease-in-out'
  
  const variantClasses = {
    primary: 'bg-blue-600 hover:bg-blue-700 text-white disabled:bg-blue-300',
    secondary: 'bg-slate-200 hover:bg-slate-300 text-slate-900 disabled:bg-slate-100',
    danger: 'bg-red-600 hover:bg-red-700 text-white disabled:bg-red-300'
  }

  const sizeClasses = {
    sm: 'px-3 py-1.5 text-sm',
    md: 'px-4 py-2 text-base',
    lg: 'px-6 py-3 text-lg'
  }

  return (
    <button
      disabled={disabled || isLoading}
      className={`${baseClasses} ${variantClasses[variant]} ${sizeClasses[size]} ${className}`}
      {...props}
    >
      {isLoading ? 'Loading...' : children}
    </button>
  )
}

export function Badge({ 
  children, 
  variant = 'info',
  className = ''
}: { 
  children: React.ReactNode
  variant?: 'success' | 'warning' | 'danger' | 'info'
  className?: string
}) {
  const variantClasses = {
    success: 'bg-emerald-100 text-emerald-800',
    warning: 'bg-amber-100 text-amber-800',
    danger: 'bg-red-100 text-red-800',
    info: 'bg-blue-100 text-blue-800'
  }

  return (
    <span className={`inline-block px-2.5 py-0.5 rounded-full text-xs font-medium ${variantClasses[variant]} ${className}`}>
      {children}
    </span>
  )
}

export function Spinner() {
  return (
    <div className="flex items-center justify-center">
      <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
    </div>
  )
}

export function ErrorMessage({ message }: { message: string }) {
  return (
    <div className="rounded-lg bg-red-50 border border-red-200 p-4">
      <p className="text-red-800 text-sm font-medium">Error: {message}</p>
    </div>
  )
}

export function EmptyState({ 
  title = 'No data',
  description = 'Nothing to display here'
}: { 
  title?: string
  description?: string
}) {
  return (
    <div className="text-center py-12">
      <p className="text-slate-500 text-lg font-medium">{title}</p>
      <p className="text-slate-400 text-sm mt-1">{description}</p>
    </div>
  )
}
