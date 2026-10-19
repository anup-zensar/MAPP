/**
 * Light and dark mode (FR-014, research R16).
 *
 * A person chooses a preference: follow the computer ("system", the default), or always light or
 * dark. Signed in, the preference is saved per person by the API; this browser also remembers the
 * last one, so the script in index.html paints the right colours before the app has loaded.
 */

/** What the person chose. */
export type ThemePreference = 'system' | 'light' | 'dark';

/** The colours actually shown. */
export type Theme = 'light' | 'dark';

/** The localStorage key; index.html reads the same key. */
const STORAGE_KEY = 'northstar.theme';
const DARK_QUERY = '(prefers-color-scheme: dark)';

/** The preference this browser remembers; "system" when there is none or storage is blocked. */
export function readStoredPreference(): ThemePreference {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    return stored === 'light' || stored === 'dark' ? stored : 'system';
  } catch {
    return 'system';
  }
}

/** Remembers the preference — never the colour it resolved to — for the next page load. */
export function storePreference(preference: ThemePreference): void {
  try {
    localStorage.setItem(STORAGE_KEY, preference);
  } catch {
    // Storage can be blocked (private windows, strict settings): the next page load then starts
    // light until the person's saved preference arrives.
  }
}

/** The computer's own setting. */
export function computerTheme(): Theme {
  return window.matchMedia(DARK_QUERY).matches ? 'dark' : 'light';
}

/** Calls onChange whenever the computer switches between light and dark; returns how to stop. */
export function watchComputerTheme(onChange: () => void): () => void {
  const query = window.matchMedia(DARK_QUERY);
  query.addEventListener('change', onChange);
  return () => query.removeEventListener('change', onChange);
}

/** Paints the whole app: tokens.css switches every colour on <html data-theme="dark">. */
export function applyTheme(theme: Theme): void {
  document.documentElement.dataset.theme = theme;
}
