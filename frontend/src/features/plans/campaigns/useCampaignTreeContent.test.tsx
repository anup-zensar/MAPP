import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { campaignForm } from '@/testing/fixtures/attribute-forms';
import { renderTree } from '@/testing/fixtures/campaign-tree';
import {
  campaign,
  initiative,
  PEOPLE,
  planAnswer,
  planCampaigns,
  usd,
} from '@/testing/fixtures/campaigns';
import { reply } from '@/testing/replies';

describe('useCampaignTreeContent (feature 005)', () => {
  it('gives the plan tree its campaign rows with their initiatives and the totals row (FR-019, FR-022)', async () => {
    renderTree([
      reply(
        'GET /api/v1/plans/{planId}/campaigns',
        { body: planCampaigns() },
        { planId: 'P-1001' },
      ),
    ]);

    const tree = await screen.findByRole('treegrid', { name: 'Plan tree' });
    const rows = within(tree).getAllByRole('row');
    expect(rows[1]).toHaveTextContent('Helix launch');
    expect(rows[2]).toHaveTextContent('Digital & social');
    expect(within(tree).getByText('1 campaign · 1 initiative · 0 activities')).toBeVisible();
  });

  it('offers Add campaign in the toolbar and on the plan row (FR-001)', async () => {
    const { user, router } = renderTree([
      reply(
        'GET /api/v1/plans/{planId}/campaigns',
        { body: planCampaigns() },
        { planId: 'P-1001' },
      ),
      reply('GET /api/v1/plans/{planId}', { body: planAnswer() }, { planId: 'P-1001' }),
      reply(
        'GET /api/v1/shell-attribute-forms/{level}',
        { body: campaignForm() },
        { level: 'campaign' },
      ),
      reply('GET /api/v1/people', { body: PEOPLE }),
    ]);

    await user.click(await screen.findByRole('link', { name: 'Add campaign' }));
    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1001/campaigns/new'));

    await router.navigate('/plans/P-1001');
    await user.click(
      await screen.findByRole('button', { name: 'Actions for FY2026 Lubricants Europe' }),
    );
    await user.click(await screen.findByRole('menuitem', { name: 'Add campaign' }));
    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1001/campaigns/new'));
  });

  it('offers Add campaign under an empty tree (FR-001)', async () => {
    renderTree([
      reply(
        'GET /api/v1/plans/{planId}/campaigns',
        { body: planCampaigns([]) },
        { planId: 'P-1001' },
      ),
    ]);

    const empty = await screen.findByTestId('empty');
    expect(within(empty).getByRole('link', { name: 'Add campaign' })).toHaveAttribute(
      'href',
      '/plans/P-1001/campaigns/new',
    );
  });

  it('highlights the record named in the location state (FR-004)', async () => {
    renderTree(
      [
        reply(
          'GET /api/v1/plans/{planId}/campaigns',
          { body: planCampaigns() },
          { planId: 'P-1001' },
        ),
      ],
      { pathname: '/plans/P-1001', state: { highlight: 'I-1001' } },
    );

    const tree = await screen.findByRole('treegrid', { name: 'Plan tree' });
    const highlighted = within(tree)
      .getAllByRole('row')
      .filter((row) => row.className.includes('highlighted'));
    expect(highlighted).toHaveLength(1);
    expect(highlighted[0]).toHaveTextContent('Digital & social');
  });

  it('shows a new campaign and the new totals right after it is created (FR-012)', async () => {
    const created = campaign({
      id: 'C-1005',
      name: 'Fleet growth',
      budget: usd('100000.00'),
      initiatives: [],
      counts: { initiatives: 0, activities: 0, approvedActivities: 0, lineItems: 0 },
    });
    let saved = false;
    const { user } = renderTree([
      reply(
        'GET /api/v1/plans/{planId}/campaigns',
        () => ({
          body: saved
            ? planCampaigns([campaign(), created], { allocatedToCampaigns: usd('500000.00') })
            : planCampaigns(),
        }),
        { planId: 'P-1001' },
      ),
      reply('GET /api/v1/plans/{planId}', { body: planAnswer() }, { planId: 'P-1001' }),
      reply(
        'GET /api/v1/shell-attribute-forms/{level}',
        { body: campaignForm() },
        { level: 'campaign' },
      ),
      reply('GET /api/v1/people', { body: PEOPLE }),
      reply(
        'POST /api/v1/plans/{planId}/campaigns',
        () => {
          saved = true;
          return { status: 201, body: created };
        },
        { planId: 'P-1001' },
      ),
    ]);

    await user.click(await screen.findByRole('link', { name: 'Add campaign' }));
    await user.type(await screen.findByLabelText('Campaign name'), 'Fleet growth');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    const tree = await screen.findByRole('treegrid', { name: 'Plan tree' });
    expect(await within(tree).findByRole('link', { name: 'Fleet growth' })).toBeInTheDocument();
    expect(within(tree).getByText('2 campaigns · 1 initiative · 0 activities')).toBeVisible();
    expect(within(tree).getByText('$500,000.00')).toBeVisible();
    const row = within(tree).getByRole('link', { name: 'Fleet growth' }).closest('tr');
    expect(row?.className).toContain('highlighted');
  });

  it('reports a failure to read the rows with Try again', async () => {
    let calls = 0;
    const { user } = renderTree([
      reply(
        'GET /api/v1/plans/{planId}/campaigns',
        () =>
          ++calls === 1
            ? {
                status: 500,
                body: { error: { code: 'internal_error', message: 'Oops', fields: [] } },
              }
            : { body: planCampaigns([campaign({ initiatives: [initiative()] })]) },
        { planId: 'P-1001' },
      ),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('treegrid', { name: 'Plan tree' })).toBeInTheDocument();
  });
});
