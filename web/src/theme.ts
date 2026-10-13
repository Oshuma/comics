import { useCallback, useEffect, useState } from 'react'

export type Theme = 'light' | 'dark'

// Keep in sync with the inline script in index.html.
const storageKey = 'theme'
const darkQuery = '(prefers-color-scheme: dark)'

// Navbar background (bg-body-tertiary) per theme, for the browser UI color.
const themeColors: Record<Theme, string> = { light: '#f8f9fa', dark: '#2b3035' }

function storedTheme(): Theme | null {
  try {
    const t = localStorage.getItem(storageKey)
    return t === 'light' || t === 'dark' ? t : null
  } catch {
    return null
  }
}

function systemTheme(): Theme {
  return window.matchMedia(darkQuery).matches ? 'dark' : 'light'
}

function applyTheme(theme: Theme) {
  document.documentElement.setAttribute('data-bs-theme', theme)
  document.querySelector('meta[name="theme-color"]')?.setAttribute('content', themeColors[theme])
}

// The current theme: the saved choice, otherwise the system setting.
export function useTheme() {
  const [theme, setTheme] = useState<Theme>(() => storedTheme() ?? systemTheme())

  useEffect(() => applyTheme(theme), [theme])

  // Follow system changes until a theme has been chosen.
  useEffect(() => {
    const media = window.matchMedia(darkQuery)
    const onChange = () => {
      if (!storedTheme()) setTheme(systemTheme())
    }
    media.addEventListener('change', onChange)
    return () => media.removeEventListener('change', onChange)
  }, [])

  const toggle = useCallback(() => {
    setTheme((current) => {
      const next = current === 'dark' ? 'light' : 'dark'
      try {
        localStorage.setItem(storageKey, next)
      } catch {
        // Not persisted; the choice still applies for this visit.
      }
      return next
    })
  }, [])

  return { theme, toggle }
}
