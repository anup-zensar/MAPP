import { screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { COMING_SOON_SECTIONS } from '@/app/navigation';
import { expectCurrentSection, renderApp } from '@/test/app';
import { resetAllWorkStore } from './allWorkStore';
import { gridResult, PAGE_ANSWERS } from './testData';

afterEach(() => resetAllWorkStore());

// Feature 010's address (build guide §5).
describe('All work route (feature 010)', () => {
  it('opens /all-work from the sidebar address, with All work current and its tab title (FR-001)', async () => {
    renderApp('/all-work', { ...PAGE_ANSWERS, 'GET /api/all-work': { body: gridResult() } });

    expect(await screen.findByRole('heading', { level: 1, name: 'All work' })).toBeInTheDocument();
    await vi.waitFor(() => expect(document.title).toBe('All work · NorthStar'));
    expectCurrentSection('All work');
    expect(screen.queryByText('Coming soon')).not.toBeInTheDocument();
  });

  it('is no longer a Coming soon section', () => {
    expect(COMING_SOON_SECTIONS.map((section) => section.path)).not.toContain('/all-work');
  });
});
