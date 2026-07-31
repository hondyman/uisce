import React, { useEffect, ReactNode } from 'react'

interface ThemeProviderProps {
  theme: any;
  children: ReactNode;
}

export function ThemeProvider({theme, children }: ThemeProviderProps) {
  useEffect(() => {
    document.documentElement.setAttribute("data-theme", theme)
  }, [theme])

  return children
}