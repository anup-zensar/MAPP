import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { nth } from '@/testing/dom';
import { detailsOf, workflowOf } from '@/testing/fixtures/approval-workflows';
import {
  renderWorkflowsPage,
  workflowList,
  workflowRow,
} from '@/testing/fixtures/approval-workflows-page';
import { reply } from '@/testing/replies';

describe('ApprovalWorkflowsPage (feature 008, US4)', () => {
  it('lists the workflows with their levels, use and status (FR-001, FR-005)', async () => {
    renderWorkflowsPage('/settings/approval-workflows?status=all');

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Approval workflows' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('Approval workflows · MAPP');
    const eu = await screen.findByRole('row', { name: /Lubricants EU approval/ });
    expect(eu).toHaveTextContent('Level 1 · Carlos Mendes → Global brand · Alex Morgan');
    expect(eu).toHaveTextContent('Used by 3 activities (2 waiting for approval)');
    expect(eu).toHaveTextContent('Active');
    const china = workflowRow('China BC approval workflow');
    expect(china).toHaveTextContent('Not used yet');
    expect(china).toHaveTextContent('Inactive');
    expect(screen.getByText('2 of 2 approval workflows')).toBeInTheDocument();
    expect(within(eu).getByRole('link', { name: 'Lubricants EU approval' })).toHaveAttribute(
      'href',
      '/settings/approval-workflows/WF-1001',
    );
  });

  it('shows the active workflows first, searches names and filters by status', async () => {
    const { user } = renderWorkflowsPage();
    await screen.findByRole('row', { name: /Lubricants EU approval/ });
    const search = () => screen.getByRole('searchbox', { name: 'Search approval workflows' });

    expect(screen.getByText('1 of 2 approval workflows')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Active 1' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    expect(screen.queryByRole('row', { name: /China BC/ })).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /^All/ }));
    await user.type(search(), 'china');
    expect(screen.getByText('1 of 2 approval workflows')).toBeInTheDocument();
    expect(screen.queryByRole('row', { name: /Lubricants EU/ })).not.toBeInTheDocument();

    await user.clear(search());
    await user.click(screen.getByRole('button', { name: /^Inactive/ }));
    await user.type(search(), 'lubricants');
    expect(screen.getByText('Nothing matches these filters')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Clear filters' }));
    expect(screen.getByText('2 of 2 approval workflows')).toBeInTheDocument();
  });

  it('shows a warning icon for a workflow with warnings', async () => {
    renderWorkflowsPage('/settings/approval-workflows', [
      reply(
        'GET /api/v1/approval-workflows',
        workflowList([workflowOf({ warnings: ['Level 2 has no active approver'] })]),
      ),
    ]);

    expect(
      await screen.findByRole('button', { name: 'Warnings: Level 2 has no active approver' }),
    ).toBeInTheDocument();
  });

  it('invites to create the first workflow when there is none', async () => {
    const { user } = renderWorkflowsPage('/settings/approval-workflows', [
      reply('GET /api/v1/approval-workflows', workflowList([])),
    ]);

    expect(
      await screen.findByText('No approval workflows yet — create your first approval workflow'),
    ).toBeInTheDocument();
    await user.click(nth(screen.getAllByRole('button', { name: 'New approval workflow' }), 1));
    expect(screen.getByRole('dialog', { name: 'New approval workflow' })).toBeInTheDocument();
  });

  it('opens the details at their address and goes back to the list on close (FR-005)', async () => {
    const { user, router } = renderWorkflowsPage('/settings/approval-workflows/WF-1001', [
      reply(
        'GET /api/v1/approval-workflows/{workflowId}',
        { body: detailsOf() },
        { workflowId: 'WF-1001' },
      ),
    ]);

    const panel = await screen.findByRole('dialog', { name: 'Lubricants EU approval' });
    await user.click(nth(within(panel).getAllByRole('button', { name: 'Close' }), 0));

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/settings/approval-workflows'),
    );
  });

  it('edits from the menu in the side panel', async () => {
    const { user } = renderWorkflowsPage();
    await screen.findByRole('row', { name: /Lubricants EU approval/ });

    await user.click(screen.getByRole('button', { name: 'Actions for Lubricants EU approval' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Edit' }));

    const panel = await screen.findByRole('dialog', { name: 'Edit approval workflow' });
    expect(within(panel).getByRole('textbox', { name: /^Name/ })).toHaveValue(
      'Lubricants EU approval',
    );
  });
});
