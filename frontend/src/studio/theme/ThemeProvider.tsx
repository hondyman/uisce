import React, { useEffect } from 'react'

export function ThemeProvider({ theme, children }) {
  useEffect(() => {
    document.documentElement.setAttribute("data-theme", theme)
  }, [theme])

  return children
}