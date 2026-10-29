import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderPlanPage } from '@/testing/fixtures/plan-page';
import { plan } from '@/testing/fixtures/plans';
import { reply } from '@/testing/replies';

describe('PlanPage actions and history (US5, US6, FR-026 – FR-031)', () => {
  it('offers Publish on a Draft plan and publishes it (US3#1, US5#1)', async () => {
    let published = false;
    const { user } = renderPlanPage(
      () => ({ body: plan({ status: published ? 'published' : 'draft' }) }),
      [
        reply(
          'POST /api/v1/plans/{planId}/publish',
          () => {
            published = true;
            return { body: plan({ status: 'published' }) };
          },
          { planId: 'P-1001' },
        ),
      ],
    );

    await user.click(await screen.findByRole('button', { name: 'Publish' }));

    expect(
      await screen.findByText("Plan 'FY2026 Lubricants Europe' published"),
    ).toBeInTheDocument();
    expect(await screen.findByRole('button', { name: 'Unpublish' })).toBeInTheDocument();
    expect(screen.getByRole('banner')).toHaveTextContent('Published');
  });

  it('goes to the Plans list after a delete, with Undo (US6#4, FR-030)', async () => {
    const { user, router } = renderPlanPage({ body: plan() }, [
      reply(
        'DELETE /api/v1/plans/{planId}',
        {
          body: {
            recycleBinEntry: {
              id: 'RB-1001',
              recordType: 'plan',
              recordTypeLabel: 'plan',
              recordId: 'P-1001',
              recordName: 'FY2026 Lubricants Europe',
              path: [],
              parent: null,
              deletedAt: '2026-03-12T09:00:00Z',
              deletedBy: { id: 'U-1002', name: 'Priya Raman' },
              contents: [],
            },
          },
        },
        { planId: 'P-1001' },
      ),
    ]);

    await user.click(await screen.findByRole('button', { name: 'More' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    await user.click(await screen.findByRole('button', { name: 'Delete plan' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans'));
    expect(
      await screen.findByText("Plan 'FY2026 Lubricants Europe' moved to the recycle bin"),
    ).toBeInTheDocument();
  });

  it('copies the plan from More and opens the copy (US6#1–2, FR-026, FR-027)', async () => {
    const copy = plan({ id: 'P-1004', name: 'FY2026 Lubricants Europe (copy)', version: 1 });
    const { user, router } = renderPlanPage({ body: plan() }, [
      reply(
        'GET /api/v1/plans/{planId}/copy-options',
        {
          body: {
            name: 'FY2026 Lubricants Europe (copy)',
            sameYear: 2026,
            nextYear: { year: 2027, available: true, reason: '' },
            contents: {
              campaigns: 0,
              initiatives: 0,
              activities: 0,
              approvedActivities: 0,
              lineItems: 0,
            },
          },
        },
        { planId: 'P-1001' },
      ),
      reply(
        'POST /api/v1/plans/{planId}/copy',
        { status: 201, body: { plan: copy, notes: [] } },
        { planId: 'P-1001' },
      ),
      reply('GET /api/v1/plans/{planId}', { body: copy }, { planId: 'P-1004' }),
    ]);

    await user.click(await screen.findByRole('button', { name: 'More' }));
    expect((await screen.findAllByRole('menuitem')).map((item) => item.textContent)).toEqual([
      'Copy',
      'Import from Excel',
      'Export to Excel',
      'View history',
      'Delete',
    ]);
    await user.click(screen.getByRole('menuitem', { name: 'Copy' }));
    await screen.findByRole('textbox', { name: /Name of the copy/ });
    await user.click(screen.getByRole('button', { name: 'Copy plan' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1004'));
    expect(
      await screen.findByRole('heading', { level: 1, name: 'FY2026 Lubricants Europe (copy)' }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "Plan 'FY2026 Lubricants Europe (copy)' created as a copy of 'FY2026 Lubricants Europe'",
      ),
    ).toBeInTheDocument();
  });

  it('opens the History tab from More and focuses it (FR-031, 011 FR-011)', async () => {
    const { user, router } = renderPlanPage();

    await user.click(await screen.findByRole('button', { name: 'More' }));
    await user.click(await screen.findByRole('menuitem', { name: 'View history' }));

    await waitFor(() =>
      expect(screen.getByRole('heading', { level: 2, name: 'History' })).toHaveFocus(),
    );
    expect(router.state.location.pathname).toBe('/plans/P-1001');
    expect(router.state.location.search).toBe('?tab=history');
  });

  it('shows the plan’s History in its tab, with Include items inside (011 FR-011 – FR-013)', async () => {
    const { api } = renderPlanPage({ body: plan() }, [], '/plans/P-1001?tab=history');

    const history = await screen.findByRole('region', { name: 'History' });
    expect(within(history).getByText('Every change to this plan, newest first')).toBeVisible();
    expect(within(history).getByRole('checkbox', { name: 'Include items inside' })).toBeVisible();
    expect(within(history).getByRole('link', { name: 'Open in activity log' })).toHaveAttribute(
      'href',
      '/activity-log?record=P-1001',
    );
    expect(await within(history).findByText('No changes recorded yet.')).toBeVisible();
    const asked = api.requests.find((request) => request.path === '/api/v1/activity-log');
    expect(new URLSearchParams(asked?.search).get('recordId')).toBe('P-1001');
  });
});
