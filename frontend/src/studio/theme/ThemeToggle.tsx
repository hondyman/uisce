
export function ThemeToggle({ kernel }) {
  const toggle = () => {
    const current = kernel.services.theme.current
    const next = current === "dark" ? "light" : "dark"
    kernel.services.theme.setTheme(next)
  }

  return (
    <button onClick={toggle} className="theme-toggle">
      Toggle Theme
    </button>
  )
}