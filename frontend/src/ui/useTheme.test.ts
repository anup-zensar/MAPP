import { act, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { readStoredPreference } from './theme';
import type { ThemePreference } from './theme';
import { useTheme } from './useTheme';

/** Pretends the computer is set to light or dark mode; switchTo() changes it as a person would. */
function fakeComputerSetting(initial: 'light' | 'dark') {
  let dark = initial === 'dark';
  const listeners = new Set<() => void>();
  vi.stubGlobal('matchMedia', (query: string) => ({
    get matches() {
      return query === '(prefers-color-scheme: dark)' && dark;
    },
    media: query,
    addEventListener: (_type: string, listener: () => void) => listeners.add(listener),
    removeEventListener: (_type: string, listener: () => void) => listeners.delete(listener),
  }));
  return {
    switchTo(theme: 'light' | 'dark') {
      dark = theme === 'dark';
      act(() => listeners.forEach((listener) => listener()));
    },
  };
}

function shownTheme() {
  return document.documentElement.dataset.theme;
}

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
  vi.restoreAllMocks();
});

/** A preference typed as any of them, so the test can change it. */
function preference(value: ThemePreference): ThemePreference {
  return value;
}

describe('useTheme', () => {
  it('shows the chosen theme on the whole app at once (FR-014)', () => {
    const { result, rerender } = renderHook(({ preference }) => useTheme(preference), {
      initialProps: { preference: preference('dark') },
    });
    expect(shownTheme()).toBe('dark');
    expect(result.current).toBe('dark');

    rerender({ preference: 'light' });

    expect(shownTheme()).toBe('light');
    expect(result.current).toBe('light');
  });

  it('follows the computer’s setting when the preference is "system" (FR-014)', () => {
    fakeComputerSetting('dark');

    const { result } = renderHook(() => useTheme('system'));

    expect(shownTheme()).toBe('dark');
    expect(result.current).toBe('dark');
  });

  it('keeps following the computer while the page is open (FR-014)', () => {
    const computer = fakeComputerSetting('light');
    const { result } = renderHook(() => useTheme('system'));
    expect(shownTheme()).toBe('light');

    computer.switchTo('dark');

    expect(shownTheme()).toBe('dark');
    expect(result.current).toBe('dark');
  });

  it('ignores the computer’s setting once a theme was chosen (FR-014)', () => {
    const computer = fakeComputerSetting('dark');
    renderHook(() => useTheme('light'));

    computer.switchTo('light');
    computer.switchTo('dark');

    expect(shownTheme()).toBe('light');
  });

  it('remembers the preference in this browser, never the colour it resolved to (FR-014)', () => {
    fakeComputerSetting('dark');
    const { rerender } = renderHook(({ preference }) => useTheme(preference), {
      initialProps: { preference: preference('system') },
    });
    expect(localStorage.getItem('mapp.theme')).toBe('system');

    rerender({ preference: 'dark' });

    expect(localStorage.getItem('mapp.theme')).toBe('dark');
  });

  it('still shows the theme when the browser blocks storage (FR-014)', () => {
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new DOMException('Storage is blocked', 'SecurityError');
    });

    renderHook(() => useTheme('dark'));

    expect(shownTheme()).toBe('dark');
  });
});

describe('readStoredPreference', () => {
  it('reads the preference this browser remembers (FR-014)', () => {
    localStorage.setItem('mapp.theme', 'dark');

    expect(readStoredPreference()).toBe('dark');
  });

  it('starts with "system", so the first visit follows the computer (FR-014)', () => {
    expect(readStoredPreference()).toBe('system');

    localStorage.setItem('mapp.theme', 'purple');
    expect(readStoredPreference()).toBe('system');
  });

  it('falls back to "system" when the browser blocks storage (FR-014)', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new DOMException('Storage is blocked', 'SecurityError');
    });

    expect(readStoredPreference()).toBe('system');
  });
});
