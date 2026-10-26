import { renderHook } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { usePageTitle } from './usePageTitle';

describe('usePageTitle', () => {
  it('follows a page whose name changes (FR-017)', () => {
    const { rerender } = renderHook(({ page }) => usePageTitle(page), {
      initialProps: { page: 'Home' },
    });
    expect(document.title).toBe('Home · MAPP');

    rerender({ page: 'My profile' });

    expect(document.title).toBe('My profile · MAPP');
  });

  it('leaves the tab alone with null, for a page under a side panel that names it', () => {
    document.title = 'New campaign · MAPP';

    renderHook(() => usePageTitle(null));

    expect(document.title).toBe('New campaign · MAPP');
  });
});
