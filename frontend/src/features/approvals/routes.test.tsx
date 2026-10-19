import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectCurrentSection, renderApp } from '@/test/app';

// Feature 008's addresses (build guide §5). Feature 008 keeps this file up to date as it replaces
// the stub pages (their own requests go into renderApp's answers).
describe('Approvals routes (feature 008)', () => {
  it('opens /approvals, with Approvals current in the sidebar', async () => {
    renderApp('/approvals');

    expect(await screen.findByRole('heading', { level: 1, name: 'Approvals' })).toBeInTheDocument();
    expect(document.title).toBe('Approvals · NorthStar');
    expectCurrentSection('Approvals');
  });

  it.each(['/settings/approval-workflows', '/settings/approval-workflows/WF-1001'])(
    'opens %s, with Settings current in the sidebar',
    async (path) => {
      renderApp(path);

      expect(
        await screen.findByRole('heading', { level: 1, name: 'Approval workflows' }),
      ).toBeInTheDocument();
      expect(document.title).toBe('Approval workflows · NorthStar');
      expectCurrentSection('Settings');
    },
  );
});
