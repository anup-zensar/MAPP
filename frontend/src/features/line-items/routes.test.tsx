import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderApp } from '@/test/app';

// Feature 007's addresses (build guide §5). Feature 007 keeps this file up to date as it replaces
// the stub pages (their own requests go into renderApp's answers).
describe('Line item routes (feature 007)', () => {
  it.each([
    ['/activities/A-1001/line-items/new', 'New line item'],
    ['/line-items/L-1001', 'Line item'],
    ['/line-items/L-1001/edit', 'Edit line item'],
  ])('opens %s: "%s"', async (path, title) => {
    renderApp(path);

    expect(await screen.findByRole('heading', { level: 1, name: title })).toBeInTheDocument();
    expect(document.title).toBe(`${title} · NorthStar`);
  });
});
