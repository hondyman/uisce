import { applyTheme, themes } from '../theme'

export class ThemeService {
  constructor() {
    this.current = "dark"
    this.themes = themes
  }

  setTheme(theme) {
    this.current = theme
    applyTheme(this.themes[theme])
    localStorage.setItem("studio-theme", theme)
  }

  loadTheme() {
    const saved = localStorage.getItem("studio-theme")
    if (saved && this.themes[saved]) {
      this.setTheme(saved)
    }
  }
}