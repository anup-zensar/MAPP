import { screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { expectCurrentSection, renderApp } from '@/testing/app';
import { gridResult, PAGE_ANSWERS } from '@/testing/fixtures/all-work';
import { reply } from '@/testing/replies';
import { resetAllWorkStore } from './all-work-store';

afterEach(() => resetAllWorkStore());

// Feature 010's address (build guide §5).
describe('All work route (feature 010)', () => {
  it('opens /all-work from the sidebar address, with All work current and its tab title (FR-001)', async () => {
    renderApp('/all-work', [
      ...PAGE_ANSWERS,
      reply('GET /api/v1/all-work', { body: gridResult() }),
    ]);

    expect(await screen.findByRole('heading', { level: 1, name: 'All work' })).toBeInTheDocument();
    await vi.waitFor(() => expect(document.title).toBe('All work · MAPP'));
    expectCurrentSection('All work');
    expect(screen.queryByText('Coming soon')).not.toBeInTheDocument();
  });
});
