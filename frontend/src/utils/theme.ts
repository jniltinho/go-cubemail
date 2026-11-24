/**
 * @file theme.ts
 * @description Applies a theme to the document. A theme is only a set of role values selected by the root's
 * `data-theme` attribute (see style.css and themes/); this module is the only writer of that attribute.
 */

import { ref } from 'vue'

/** One entry of the theme registry served by `GET /api/v1/themes`. */
export interface ThemeInfo {
  id: string
  label: string
  scheme: 'light' | 'dark'
}

/** localStorage key read by public/theme-boot.js before the first paint. */
const CACHE_KEY = 'gorc_theme'

/** Id of the theme applied to the document; reactive so views can recompute what they derive from roles. */
export const activeTheme = ref(document.documentElement.dataset.theme || '')

/** Returns the registered theme with this id, or the default one (or the first entry) when it is unknown. */
export function resolveTheme(id: string, registry: ThemeInfo[], defaultId: string): ThemeInfo | undefined {
  return registry.find((t) => t.id === id) ?? registry.find((t) => t.id === defaultId) ?? registry[0]
}

/** Applies the theme to the whole document and remembers it in this browser. */
export function applyTheme(theme: ThemeInfo): void {
  const root = document.documentElement
  root.dataset.theme = theme.id
  root.style.colorScheme = theme.scheme
  document.querySelector('meta[name="color-scheme"]')?.setAttribute('content', theme.scheme)
  try {
    localStorage.setItem(CACHE_KEY, JSON.stringify({ id: theme.id, scheme: theme.scheme }))
  } catch {
    // storage blocked: the theme still applies, only the pre-paint cache is lost
  }
  activeTheme.value = theme.id
}

/** Id of the theme this browser remembers, or '' when there is none or storage is blocked. */
export function cachedThemeId(): string {
  try {
    return JSON.parse(localStorage.getItem(CACHE_KEY) || '{}').id || ''
  } catch {
    return ''
  }
}

/** Current value of a role, e.g. `roleValue('--color-paper')`; for content that cannot inherit the page's roles. */
export function roleValue(name: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}
