import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectCurrentSection, renderApp } from '@/test/app';

// Feature 011's address for the Recycle bin (build guide §5), under More in the sidebar.
describe('Recycle bin route (feature 011)', () => {
  it('opens /recycle-bin: "Recycle bin", current in the sidebar, no longer Coming soon (FR-014)', async () => {
    renderApp('/recycle-bin', { 'GET /api/v1/recycle-bin': { body: { items: [], total: 0 } } });

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Recycle bin' }),
    ).toBeInTheDocument();
    expect(screen.queryByText('Coming soon')).not.toBeInTheDocument();
    expect(document.title).toBe('Recycle bin · MAPP');
    expectCurrentSection('Recycle bin');
  });
});
