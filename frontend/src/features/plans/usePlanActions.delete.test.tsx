import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { findMessages } from '@/testing/render';
import { choose, PUBLISHED, renderActions } from '@/testing/fixtures/plan-actions';
import { renderPlanPage } from '@/testing/fixtures/plan-page';
import { plan } from '@/testing/fixtures/plans';
import { reply } from '@/testing/replies';

const DELETED = {
  recycleBinEntry: {
    id: 'RB-1001',
    recordType: 'plan',
    recordTypeLabel: 'plan',
    recordId: 'P-1003',
    recordName: 'FY2025 Lubricants Europe',
    path: [],
    parent: null,
    deletedAt: '2026-03-12T09:00:00Z',
    deletedBy: { id: 'U-1002', name: 'Priya Raman' },
    contents: [],
  },
};

describe('usePlanActions: delete with Undo (US6, FR-029, FR-030)', () => {
  it('deletes after the question, offers Undo, and Undo brings the plan back (US6#4)', async () => {
    const { user, api, router } = renderActions(PUBLISHED, [
      reply(
        'GET /api/v1/plans/{planId}',
        {
          body: plan({ id: 'P-1003', name: PUBLISHED.name, status: 'published' }),
        },
        { planId: 'P-1003' },
      ),
      reply('DELETE /api/v1/plans/{planId}', { body: DELETED }, { planId: 'P-1003' }),
      reply(
        'POST /api/v1/recycle-bin/{entryId}/undo',
        {
          body: {
            recordType: 'plan',
            recordTypeLabel: 'plan',
            recordId: 'P-1003',
            recordName: PUBLISHED.name,
            details: '',
          },
        },
        { entryId: 'RB-1001' },
      ),
    ]);

    await choose(user, PUBLISHED.name, 'Delete');
    const dialog = await screen.findByRole('alertdialog', {
      name: "Delete plan 'FY2025 Lubricants Europe'?",
    });
    expect(await within(dialog).findByText('This plan is published.')).toBeInTheDocument();
    await user.click(within(dialog).getByRole('button', { name: 'Delete plan' }));

    expect(await screen.findByText("Plan 'FY2025 Lubricants Europe' deleted")).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/plans');
    await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument());

    const messages = await findMessages();
    await user.click(within(messages).getByRole('button', { name: 'Undo' }));

    expect(await screen.findByText("Plan 'FY2025 Lubricants Europe' restored")).toBeInTheDocument();
    expect(api.requests.map((request) => `${request.method} ${request.path}`)).toContain(
      'POST /api/v1/recycle-bin/RB-1001/undo',
    );
  });

  it('says so when the plan is already gone', async () => {
    const { user } = renderActions(PUBLISHED, [
      reply(
        'GET /api/v1/plans/{planId}',
        {
          body: plan({ id: 'P-1003', name: PUBLISHED.name, status: 'published' }),
        },
        { planId: 'P-1003' },
      ),
      reply(
        'DELETE /api/v1/plans/{planId}',
        {
          status: 404,
          body: {
            error: {
              code: 'not_found' as const,
              message: 'This plan no longer exists — someone may have deleted it.',
              fields: [],
            },
          },
        },
        { planId: 'P-1003' },
      ),
    ]);

    await choose(user, PUBLISHED.name, 'Delete');
    await user.click(await screen.findByRole('button', { name: 'Delete plan' }));

    expect(
      await screen.findByText('This plan no longer exists — someone may have deleted it.'),
    ).toBeInTheDocument();
  });
});

describe('038 FR-013 (L-3): delete from the plan page', () => {
  it('never asks for the deleted plan again', async () => {
    const { user, router, api } = renderPlanPage({ body: plan() }, [
      reply(
        'DELETE /api/v1/plans/{planId}',
        {
          body: {
            recycleBinEntry: { ...DELETED.recycleBinEntry, recordId: 'P-1001' },
          },
        },
        { planId: 'P-1001' },
      ),
    ]);

    await user.click(await screen.findByRole('button', { name: 'More' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    await user.click(await screen.findByRole('button', { name: 'Delete plan' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans'));
    await findMessages();
    const deleted = api.requests.findIndex((request) => request.method === 'DELETE');
    const after = api.requests.slice(deleted + 1).map((r) => `${r.method} ${r.path}`);
    expect(after.filter((request) => request.includes('/P-1001'))).toEqual([]);
  });
});
