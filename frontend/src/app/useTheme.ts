import { useEffect, useSyncExternalStore } from 'react';
import { applyTheme, computerTheme, storePreference, watchComputerTheme } from './theme';
import type { Theme, ThemePreference } from './theme';

/**
 * Shows the whole app in the given preference and returns the colours shown (FR-014).
 * With "system" it follows the computer's setting, also when that changes while the page is
 * open. The preference is remembered in this browser for the next page load.
 */
export function useTheme(preference: ThemePreference): Theme {
  // useSyncExternalStore re-renders this component whenever the computer's setting changes.
  const computer = useSyncExternalStore(watchComputerTheme, computerTheme);
  const theme = preference === 'system' ? computer : preference;

  useEffect(() => {
    storePreference(preference);
  }, [preference]);

  useEffect(() => {
    applyTheme(theme);
  }, [theme]);

  return theme;
}
