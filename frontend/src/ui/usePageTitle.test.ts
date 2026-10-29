import { renderHook } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { usePageTitle } from './usePageTitle';

describe('usePageTitle', () => {
  it('follows a page whose name changes (FR-017)', () => {
    const { rerender } = renderHook(({ page }) => usePageTitle(page), {
      initialProps: { page: 'Home' },
    });
    expect(document.title).toBe('Home · SP&A One');

    rerender({ page: 'My profile' });

    expect(document.title).toBe('My profile · SP&A One');
  });

  it('leaves the tab alone with null, for a page under a side panel that names it', () => {
    document.title = 'New campaign · SP&A One';

    renderHook(() => usePageTitle(null));

    expect(document.title).toBe('New campaign · SP&A One');
  });
});
