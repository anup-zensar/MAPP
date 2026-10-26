import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { CHINA, choose, PUBLISHED, renderActions } from '@/testing/fixtures/plan-actions';
import { plan } from '@/testing/fixtures/plans';
import { reply } from '@/testing/replies';
import type { Reply } from '@/testing/replies';

const COPY_OPTIONS = {
  name: 'FY2025 Lubricants Europe (copy)',
  sameYear: 2025,
  nextYear: { year: 2026, available: true, reason: '' },
  contents: { campaigns: 2, initiatives: 4, activities: 7, approvedActivities: 4, lineItems: 15 },
};

describe('usePlanActions: copy (US6#1–2, FR-026 – FR-028)', () => {
  it('copies into the next year and opens the copy with its messages and notes (US6#2)', async () => {
    const { user, api, router } = renderActions(PUBLISHED, [
      reply(
        'GET /api/v1/plans/{planId}/copy-options',
        { body: COPY_OPTIONS },
        { planId: 'P-1003' },
      ),
      reply(
        'POST /api/v1/plans/{planId}/copy',
        {
          status: 201,
          body: {
            plan: plan({ id: 'P-1004', name: 'FY2026 Lubricants Europe', year: 2026 }),
            notes: ['Default finance budget cleared — it belonged to 2025.'],
          },
        },
        { planId: 'P-1003' },
      ),
    ]);

    await choose(user, PUBLISHED.name, 'Copy');
    const dialog = await screen.findByRole('dialog', {
      name: "Copy plan 'FY2025 Lubricants Europe'",
    });
    const name = await within(dialog).findByRole('textbox', { name: /Name of the copy/ });
    await user.clear(name);
    await user.type(name, 'FY2026 Lubricants Europe');
    await user.click(within(dialog).getByRole('radio', { name: 'Next year (2026)' }));
    await user.click(within(dialog).getByRole('button', { name: 'Copy plan' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1004'));
    expect(
      await screen.findByText(
        "Plan 'FY2026 Lubricants Europe' created as a copy of 'FY2025 Lubricants Europe'",
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByText('Default finance budget cleared — it belonged to 2025.'),
    ).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/plans/P-1003/copy')).toEqual([
      { name: 'FY2026 Lubricants Europe', year: 2026, includeContents: true },
    ]);
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  });

  it('closes the dialog and says so when the plan is already gone', async () => {
    const gone: Reply<'POST /api/v1/plans/{planId}/copy'> = {
      status: 404,
      body: {
        error: {
          code: 'not_found' as const,
          message: 'This plan no longer exists — someone may have deleted it.',
          fields: [],
        },
      },
    };
    const { user, router } = renderActions(PUBLISHED, [
      reply(
        'GET /api/v1/plans/{planId}/copy-options',
        { body: COPY_OPTIONS },
        { planId: 'P-1003' },
      ),
      reply('POST /api/v1/plans/{planId}/copy', gone, { planId: 'P-1003' }),
    ]);

    await choose(user, PUBLISHED.name, 'Copy');
    await user.click(await screen.findByRole('button', { name: 'Copy plan' }));

    expect(
      await screen.findByText('This plan no longer exists — someone may have deleted it.'),
    ).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(router.state.location.pathname).toBe('/plans');
  });

  it('starts from the suggestion again each time it opens', async () => {
    const { user } = renderActions(CHINA, [
      reply(
        'GET /api/v1/plans/{planId}/copy-options',
        {
          body: { ...COPY_OPTIONS, name: 'FY2026 Mobility China (copy)', sameYear: 2026 },
        },
        { planId: 'P-1002' },
      ),
    ]);

    await choose(user, CHINA.name, 'Copy');
    const name = await screen.findByRole('textbox', { name: /Name of the copy/ });
    await user.type(name, ' typed');
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());

    await choose(user, CHINA.name, 'Copy');
    expect(await screen.findByRole('textbox', { name: /Name of the copy/ })).toHaveValue(
      'FY2026 Mobility China (copy)',
    );
  });
});
