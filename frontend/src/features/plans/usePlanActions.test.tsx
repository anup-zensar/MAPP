import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { CHINA, choose, PUBLISHED, renderActions } from '@/testing/fixtures/plan-actions';
import { plan } from '@/testing/fixtures/plans';
import { reply } from '@/testing/replies';
import type { Reply } from '@/testing/replies';

const NOT_READY: Reply<'POST /api/v1/plans/{planId}/publish'> = {
  status: 409,
  body: {
    error: {
      code: 'not_ready' as const,
      message: 'Publish needs 3 fields',
      fields: [],
      missing: [
        {
          record: { type: 'plan', id: 'P-1002', name: 'FY2026 Mobility China' },
          fields: [{ field: 'attributes.SA-BU', label: 'Business unit' }],
        },
        {
          record: { type: 'campaign', id: 'C-1001', name: 'Retail growth' },
          fields: [
            { field: 'attributes.SA-CLUSTER', label: 'Cluster' },
            { field: 'attributes.SA-REGION', label: 'Region' },
          ],
        },
      ],
    },
  },
};

function reasonProblem(message: string): Reply<'POST /api/v1/plans/{planId}/unpublish'> {
  return {
    status: 400,
    body: {
      error: {
        code: 'validation_failed' as const,
        message: 'Some fields need your attention.',
        fields: [{ field: 'reason', message }],
      },
    },
  };
}

describe('usePlanActions: publish and unpublish (US5, FR-024, FR-025)', () => {
  it('offers Publish on a Draft plan and Unpublish on a Published one', async () => {
    const { user } = renderActions(PUBLISHED, []);

    await user.click(screen.getByRole('button', { name: `Actions for ${PUBLISHED.name}` }));
    expect((await screen.findAllByRole('menuitem')).map((item) => item.textContent)).toEqual([
      'View',
      'Edit',
      'Copy',
      'Unpublish',
      'Delete',
    ]);
  });

  it('publishes a complete Draft plan (US5#1)', async () => {
    const { user, api } = renderActions(CHINA, [
      reply(
        'POST /api/v1/plans/{planId}/publish',
        {
          body: plan({ id: 'P-1002', name: 'FY2026 Mobility China', status: 'published' }),
        },
        { planId: 'P-1002' },
      ),
    ]);

    await choose(user, CHINA.name, 'Publish');

    expect(await screen.findByText("Plan 'FY2026 Mobility China' published")).toBeInTheDocument();
    expect(api.requests.filter((r) => r.path === '/api/v1/plans/P-1002/publish')).toHaveLength(1);
  });

  it('lists every missing field, grouped by record, with links (US5#2, FR-024)', async () => {
    const { user, router } = renderActions(CHINA, [
      reply('POST /api/v1/plans/{planId}/publish', NOT_READY, { planId: 'P-1002' }),
    ]);

    await choose(user, CHINA.name, 'Publish');

    const dialog = await screen.findByRole('dialog', { name: 'Publish needs 3 fields' });
    expect(dialog).toHaveTextContent('Fill these in, then publish again:');
    expect(
      within(dialog)
        .getAllByRole('heading', { level: 3 })
        .map((h) => h.textContent),
    ).toEqual(['Plan', "Campaign 'Retail growth'"]);
    expect(within(dialog).getByRole('link', { name: 'Cluster' })).toHaveAttribute(
      'href',
      '/plans/P-1002/campaigns/C-1001/edit?focus=attributes.SA-CLUSTER',
    );
    await user.click(within(dialog).getByRole('link', { name: 'Business unit' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1002/edit'));
    expect(router.state.location.search).toBe('?focus=attributes.SA-BU');
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  });

  it('closes the blocked dialog with Close', async () => {
    const { user } = renderActions(CHINA, [
      reply('POST /api/v1/plans/{planId}/publish', NOT_READY, { planId: 'P-1002' }),
    ]);

    await choose(user, CHINA.name, 'Publish');
    await user.click(await screen.findByRole('button', { name: 'Close' }));

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  });

  it('asks why before unpublishing and shows the API’s problems with the reason (US5#3, FR-025)', async () => {
    let calls = 0;
    const { user, api } = renderActions(PUBLISHED, [
      reply(
        'POST /api/v1/plans/{planId}/unpublish',
        () => {
          calls += 1;
          if (calls === 1) return reasonProblem('Enter a reason.');
          if (calls === 2) return reasonProblem('Use 500 characters or fewer.');
          return { body: plan({ id: 'P-1003', name: PUBLISHED.name, status: 'draft' }) };
        },
        { planId: 'P-1003' },
      ),
    ]);

    await choose(user, PUBLISHED.name, 'Unpublish');
    const dialog = await screen.findByRole('dialog', {
      name: "Unpublish plan 'FY2025 Lubricants Europe'?",
    });
    const reason = within(dialog).getByRole('textbox', {
      name: /Why are you unpublishing this plan\?/,
    });
    expect(reason).toHaveFocus();
    expect(dialog).toHaveTextContent(
      'The plan returns to Draft. Your reason is recorded in the activity log.',
    );

    await user.click(within(dialog).getByRole('button', { name: 'Unpublish plan' }));
    expect(await within(dialog).findByText('Enter a reason.')).toBeInTheDocument();

    await user.click(reason);
    await user.paste('x'.repeat(501));
    await user.click(within(dialog).getByRole('button', { name: 'Unpublish plan' }));
    expect(await within(dialog).findByText('Use 500 characters or fewer.')).toBeInTheDocument();

    await user.clear(reason);
    await user.type(reason, 'Budget is being reworked.');
    await user.click(within(dialog).getByRole('button', { name: 'Unpublish plan' }));

    expect(
      await screen.findByText("Plan 'FY2025 Lubricants Europe' unpublished — it's a Draft again"),
    ).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/plans/P-1003/unpublish')).toEqual([
      { reason: '' },
      { reason: 'x'.repeat(501) },
      { reason: 'Budget is being reworked.' },
    ]);
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  });
});
