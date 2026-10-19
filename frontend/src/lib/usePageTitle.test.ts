import { renderHook } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { usePageTitle } from './usePageTitle';

describe('usePageTitle', () => {
  it('names the page in the browser tab as "<Page> · NorthStar" (FR-017)', () => {
    renderHook(() => usePageTitle('Plans'));

    expect(document.title).toBe('Plans · NorthStar');
  });

  it('follows a page whose name changes (FR-017)', () => {
    const { rerender } = renderHook(({ page }) => usePageTitle(page), {
      initialProps: { page: 'Home' },
    });
    expect(document.title).toBe('Home · NorthStar');

    rerender({ page: 'My profile' });

    expect(document.title).toBe('My profile · NorthStar');
  });
});
