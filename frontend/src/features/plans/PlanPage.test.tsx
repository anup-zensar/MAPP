import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { closestTo } from '@/testing/dom';
import { renderPlanPage, summaryCard, withFigures } from '@/testing/fixtures/plan-page';
import { plan } from '@/testing/fixtures/plans';

describe('PlanPage (US3, FR-011 – FR-017)', () => {
  it('shows the header facts and actions (US3#1, FR-011)', async () => {
    renderPlanPage();

    const header = closestTo(
      await screen.findByRole('heading', { level: 1, name: 'FY2026 Lubricants Europe' }),
      'header',
    );
    expect(document.title).toBe('FY2026 Lubricants Europe · MAPP');
    expect(within(header).getByRole('link', { name: 'Plans' })).toHaveAttribute('href', '/plans');
    expect(within(header).getByText('PLAN')).toBeInTheDocument();
    expect(within(header).getByText('Draft')).toBeInTheDocument();
    expect(within(header).getByText('All filled in')).toBeInTheDocument();
    // The meta row of the one page header (design system §2.3).
    const facts = within(header).getByRole('group', { name: 'Plan facts' });
    expect(facts).toHaveTextContent('P-1001');
    expect(facts).toHaveTextContent('Priya Raman');
    expect(facts).toHaveTextContent('01 Jan – 31 Dec 2026');
    expect(facts).toHaveTextContent('AOP plan');
    expect(facts).toHaveTextContent('Lubricants Europe');
    expect(facts).toHaveTextContent('Business unit: Lubricants');
    expect(within(header).getByRole('link', { name: 'Edit' })).toHaveAttribute(
      'href',
      '/plans/P-1001/edit',
    );
    expect(within(header).getByRole('button', { name: 'More' })).toBeInTheDocument();
    // The tabs of the record template, the tree first (design system §2.4 B).
    expect(screen.getAllByRole('tab').map((tab) => tab.textContent)).toEqual([
      'Plan tree',
      'Details',
      'History',
    ]);
    expect(screen.getByRole('tab', { name: 'Plan tree' })).toHaveAttribute('aria-selected', 'true');
  });

  it('keeps the chosen tab in the address (design system §2.4 B)', async () => {
    const { user, router } = renderPlanPage();

    await user.click(await screen.findByRole('tab', { name: 'Details' }));

    expect(router.state.location.search).toBe('?tab=details');
    expect(
      await screen.findByText(
        'Created by Alex Morgan on 01 Mar 2026, last changed by Daniel Okoro on 10 Mar 2026',
      ),
    ).toBeInTheDocument();
    await user.click(screen.getByRole('tab', { name: 'Plan tree' }));
    expect(router.state.location.search).toBe('');
  });

  it('has the sections Home links to: #budget and #completeness (feature 009, FR-012)', async () => {
    renderPlanPage();

    await screen.findByRole('heading', { level: 1, name: 'FY2026 Lubricants Europe' });
    expect(screen.getByRole('region', { name: 'Summary' })).toHaveAttribute('id', 'budget');
    const completeness = document.getElementById('completeness');
    expect(completeness).not.toBeNull();
    expect(completeness?.closest('header')).not.toBeNull();
  });

  it('shows the owner problem in words (FR-021)', async () => {
    renderPlanPage({
      body: plan({
        ownerProblem:
          "Priya Raman isn't in any of the chosen teams — choose another owner or clear the field.",
      }),
    });

    expect(
      await screen.findByText(
        "Priya Raman isn't in any of the chosen teams — choose another owner or clear the field.",
      ),
    ).toBeInTheDocument();
  });

  it('shows a negative remaining budget in red with the amount (US3#2, FR-012)', async () => {
    renderPlanPage(
      withFigures({
        campaignBudgets: { amount: '1300000.00', currency: 'USD' },
        remainingBudget: { amount: '-100000.00', currency: 'USD' },
        overAllocatedBy: { amount: '100000.00', currency: 'USD' },
      }),
    );

    await screen.findByRole('region', { name: 'Summary' });
    // What remains for campaigns is part of the plan budget card (four cards, §2.4 B).
    expect(
      within(summaryCard('Plan budget')).getByText('Over-allocated by $100,000.00'),
    ).toBeInTheDocument();
  });

  it('shows the budget used with its bar, red above 100 % (US3#3)', async () => {
    renderPlanPage(
      withFigures({
        plannedCost: { amount: '960000.00', currency: 'USD' },
        budgetUsedPercent: 80,
      }),
    );

    await screen.findByRole('region', { name: 'Summary' });
    expect(summaryCard('Budget used')).toHaveTextContent('80%');
    expect(within(summaryCard('Budget used')).getByRole('progressbar')).toHaveAttribute(
      'aria-valuenow',
      '80',
    );
  });

  it('says why the budget cards are empty without a plan budget (US3#3)', async () => {
    renderPlanPage(
      withFigures(
        { budget: null, remainingBudget: null, budgetUsedPercent: null },
        { budget: null },
      ),
    );

    await screen.findByRole('region', { name: 'Summary' });
    expect(summaryCard('Plan budget')).toHaveTextContent('—No plan budget set');
    expect(summaryCard('Budget used')).toHaveTextContent('—No plan budget set');
    expect(summaryCard('Planned cost')).toHaveTextContent('Nothing planned yet');
    // "Nothing" is "—" with the reason under it (functional review FR·X-08).
    expect(summaryCard('Activities approved')).toHaveTextContent('—No activities yet');
  });

  it('links the missing field and the items below that need attention (US3#4, FR-017)', async () => {
    const { user, router } = renderPlanPage({
      body: plan({
        businessUnit: [],
        attributes: {},
        completeness: {
          missing: [{ field: 'attributes.SA-BU', label: 'Business unit' }],
          itemsBelow: [
            {
              record: { type: 'campaign', id: 'C-1001', name: 'Retail growth' },
              missing: [
                { field: 'attributes.SA-CLUSTER', label: 'Cluster' },
                { field: 'attributes.SA-REGION', label: 'Region' },
              ],
            },
          ],
        },
      }),
    });

    const header = closestTo(await screen.findByRole('heading', { level: 1 }), 'header');
    expect(within(header).getByText('No business unit yet')).toBeInTheDocument();
    // One completeness chip: its list holds the missing field and the records below (§4.1).
    await user.click(within(header).getByRole('button', { name: '1 field needed' }));
    expect(
      await screen.findByRole('link', { name: "Campaign 'Retail growth' — 2 fields needed" }),
    ).toHaveAttribute('href', '/plans/P-1001/campaigns/C-1001');
    await user.click(screen.getByRole('button', { name: 'Business unit' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1001/edit'));
    expect(router.state.location.search).toBe('?focus=attributes.SA-BU');
  });

  it('shows grey shapes while loading, then an error with Try again', async () => {
    renderPlanPage({ status: 500 });

    expect(screen.getByLabelText('Loading the plan')).toBeInTheDocument();
    expect(await screen.findByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });

  it('says when the plan does not exist', async () => {
    renderPlanPage({
      status: 404,
      body: {
        error: {
          code: 'not_found',
          message: 'This plan no longer exists — someone may have deleted it.',
          fields: [],
        },
      },
    });

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Plan not found' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to Plans' })).toHaveAttribute('href', '/plans');
  });
});
