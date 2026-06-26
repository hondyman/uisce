export const themes = {
  light: {
    background: "#ffffff",
    foreground: "#000000",
    accent: "#007acc",
    border: "#e5e5e5",
  },
  dark: {
    background: "#1e1e1e",
    foreground: "#ffffff",
    accent: "#007acc",
    border: "#3e3e3e",
  }
}

export function applyTheme(theme) {
  const root = document.documentElement
  for (const [key, value] of Object.entries(theme)) {
    root.style.setProperty(`--${key}`, value)
  }
}