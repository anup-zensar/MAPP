import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { detailsOf } from '@/testing/fixtures/approval-workflows';
import { NO_DELEGATIONS } from '@/testing/fixtures/approvals';
import { expectCurrentSection, renderApp } from '@/testing/app';
import { reply } from '@/testing/replies';

const NO_ROWS = { body: { items: [], total: 0 } };

// Feature 008's addresses (build guide §5). Feature 008 keeps this file up to date as it replaces
// the stub pages (their own requests go into renderApp's answers).
describe('Approvals routes (feature 008)', () => {
  it('opens /approvals, with Approvals current in the sidebar', async () => {
    renderApp('/approvals', [
      reply('GET /api/v1/approvals', NO_ROWS),
      reply('GET /api/v1/approval-delegations', NO_DELEGATIONS),
    ]);

    expect(await screen.findByRole('heading', { level: 1, name: 'Approvals' })).toBeInTheDocument();
    expect(document.title).toBe('Approvals · SP&A One');
    expectCurrentSection('Approvals');
  });

  it('opens /settings/approval-workflows, with Settings current in the sidebar', async () => {
    renderApp('/settings/approval-workflows', [reply('GET /api/v1/approval-workflows', NO_ROWS)]);

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Approval workflows' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('Approval workflows · SP&A One');
    expectCurrentSection('Settings');
  });

  it('opens /settings/approval-workflows/WF-1001 with the details panel over the list', async () => {
    renderApp('/settings/approval-workflows/WF-1001', [
      reply('GET /api/v1/approval-workflows', NO_ROWS),
      reply(
        'GET /api/v1/approval-workflows/{workflowId}',
        { body: detailsOf() },
        { workflowId: 'WF-1001' },
      ),
    ]);

    expect(
      await screen.findByRole('dialog', { name: 'Lubricants EU approval' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('Approval workflows · SP&A One');
    // The panel is modal: the page and the sidebar behind it are hidden from screen readers.
    const navigation = screen.getByRole('navigation', { name: 'Main', hidden: true });
    expect(
      within(navigation).getByRole('link', { name: 'Settings', hidden: true }),
    ).toHaveAttribute('aria-current', 'page');
  });
});
