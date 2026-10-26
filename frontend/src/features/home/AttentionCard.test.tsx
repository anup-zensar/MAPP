import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import type { MockHandler } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { AttentionCard } from './AttentionCard';
import type { HomeAttentionItem } from './api';
import { LUBRICANTS_PLAN, attentionItemOf, reasonOf, usd } from '@/testing/fixtures/home';

const page = (items: HomeAttentionItem[]) => ({ body: { items, total: items.length } });

function renderCard(answer: MockHandler = page([attentionItemOf()])) {
  mockApi({ 'GET /api/v1/home/attention': answer });
  return renderWithProviders(<AttentionCard />, { user: testUser(), path: '/' });
}

function card() {
  return screen.getByRole('region', { name: 'Needs your attention' });
}

/** The item of the card whose name link is `name`. */
function itemOf(name: string): HTMLElement {
  return within(card()).getByRole('link', { name }).closest('li') as HTMLElement;
}

const GLOBAL_BRAND = { type: 'plan', id: 'P-1003', name: 'FY2027 Global Brand' };

describe('AttentionCard', () => {
  it("shows Priya's declined activity with who declined it, when and why, and a link to its approval (US3 scenario 1)", async () => {
    renderCard();

    const name = await within(card()).findByRole('link', { name: 'Launch film' });
    expect(name).toHaveAttribute('href', '/plans/P-1001/activities/A-1042');
    const item = within(itemOf('Launch film'));
    expect(item.getByText('ACTIVITY')).toBeInTheDocument();
    expect(item.getByText('A-1042')).toBeInTheDocument();
    expect(item.getByText('in FY2026 Lubricants Europe')).toBeInTheDocument();
    expect(item.getByText(/^Declined by/)).toHaveTextContent(
      "Declined by Daniel Okoro on 12 Mar 2026: 'Please cut the agency fee' Open approval",
    );
    expect(
      item.getByRole('link', { name: "Open approval for activity 'Launch film'" }),
    ).toHaveAttribute('href', '/plans/P-1001/activities/A-1042#approval');
    expect(card()).toHaveTextContent('Things you own that have a problem');
  });

  it('says what each reason is and links to where it is fixed (FR-011, US3 scenarios 2 and 3)', async () => {
    renderCard(
      page([
        attentionItemOf({
          id: 'A-1043',
          name: 'Trade show',
          reasons: [reasonOf('negativeBalance', { amount: usd('1200.00') })],
        }),
        attentionItemOf({
          recordType: 'campaign',
          recordTypeLabel: 'campaign',
          id: 'C-1010',
          name: 'Brand refresh',
          plan: GLOBAL_BRAND,
          path: [GLOBAL_BRAND],
          reasons: [reasonOf('overAllocated', { amount: usd('25000.00') })],
        }),
        attentionItemOf({
          recordType: 'initiative',
          recordTypeLabel: 'initiative',
          id: 'I-1002',
          name: 'Retail displays',
          reasons: [reasonOf('overBudget', { amount: usd('5000.00') })],
        }),
        attentionItemOf({
          recordType: 'plan',
          recordTypeLabel: 'plan',
          id: 'P-1001',
          name: 'FY2026 Lubricants Europe',
          plan: LUBRICANTS_PLAN,
          path: [],
          reasons: [
            reasonOf('missingFields', {
              missing: [
                { field: 'businessUnit', label: 'Business unit' },
                { field: 'region', label: 'Region' },
                { field: 'description', label: 'Description' },
              ],
              itemsBelow: 1,
            }),
          ],
        }),
      ]),
    );

    await within(card()).findByRole('link', { name: 'Trade show' });

    expect(within(itemOf('Trade show')).getByText(/^Line items exceed/)).toHaveTextContent(
      'Line items exceed the planned cost by $1,200.00',
    );
    expect(
      screen.getByRole('link', { name: "Open line items for activity 'Trade show'" }),
    ).toHaveAttribute('href', '/plans/P-1001/activities/A-1043#line-items');

    expect(within(itemOf('Brand refresh')).getByText('CAMPAIGN')).toBeInTheDocument();
    expect(within(itemOf('Brand refresh')).getByText(/^Over-allocated/)).toHaveTextContent(
      'Over-allocated by $25,000.00',
    );
    expect(
      screen.getByRole('link', { name: "Open budget for campaign 'Brand refresh'" }),
    ).toHaveAttribute('href', '/plans/P-1003/campaigns/C-1010#budget');

    expect(within(itemOf('Retail displays')).getByText(/^Planned cost is/)).toHaveTextContent(
      'Planned cost is $5,000.00 over budget',
    );
    expect(
      screen.getByRole('link', { name: "Open budget for initiative 'Retail displays'" }),
    ).toHaveAttribute('href', '/plans/P-1001/initiatives/I-1002#budget');

    const plan = within(itemOf('FY2026 Lubricants Europe'));
    expect(plan.getByText(/fields needed/)).toHaveTextContent(
      '3 fields needed · 1 item below needs attention',
    );
    // A plan is not "in" a plan.
    expect(plan.queryByText(/^in /)).not.toBeInTheDocument();
    expect(
      screen.getByRole('link', { name: "Show fields for plan 'FY2026 Lubricants Europe'" }),
    ).toHaveAttribute('href', '/plans/P-1001#completeness');
  });

  it('shows a record with several problems once, with all its reasons (US3 scenario 4, FR-012)', async () => {
    renderCard(
      page([
        attentionItemOf({
          reasons: [
            attentionItemOf().reasons[0],
            reasonOf('negativeBalance', { amount: usd('1200.00') }),
            reasonOf('missingFields', { missing: [{ field: 'region', label: 'Region' }] }),
          ],
        }),
      ]),
    );

    await within(card()).findByRole('link', { name: 'Launch film' });
    expect(within(card()).getAllByRole('link', { name: 'Launch film' })).toHaveLength(1);
    const reasons = within(itemOf('Launch film')).getAllByRole('listitem');
    expect(reasons.map((reason) => reason.textContent)).toEqual([
      "Declined by Daniel Okoro on 12 Mar 2026: 'Please cut the agency fee' Open approval",
      'Line items exceed the planned cost by $1,200.00 Open line items',
      '1 field needed Show fields',
    ]);
  });

  it('shows 5 items, as the card beside it, then all of them on "View all (n)" and fewer again (FR-012, design system §2.4 E)', async () => {
    const items = Array.from({ length: 10 }, (_, index) =>
      attentionItemOf({ id: `A-20${index}`, name: `Activity ${index + 1}` }),
    );
    const { user } = renderCard(page(items));

    await within(card()).findByRole('link', { name: 'Activity 5' });
    expect(within(card()).queryByRole('link', { name: 'Activity 6' })).not.toBeInTheDocument();

    const more = within(card()).getByRole('button', { name: 'View all (10)' });
    expect(more).toHaveAttribute('aria-expanded', 'false');
    await user.click(more);

    expect(within(card()).getByRole('link', { name: 'Activity 10' })).toBeInTheDocument();
    await user.click(within(card()).getByRole('button', { name: 'Show fewer' }));
    expect(within(card()).queryByRole('link', { name: 'Activity 6' })).not.toBeInTheDocument();
  });

  it('says so in one sentence when nothing needs attention (US3 scenario 6)', async () => {
    renderCard(page([]));

    expect(
      await within(card()).findByText('Nothing needs your attention right now.'),
    ).toBeInTheDocument();
  });

  it('shows a placeholder while loading, and Try again when it fails (FR-022)', async () => {
    let calls = 0;
    const { user } = renderCard(() =>
      ++calls === 1 ? { status: 500 } : page([attentionItemOf()]),
    );

    expect(within(card()).getByRole('status')).toHaveTextContent('Loading…');
    expect(await within(card()).findByRole('alert')).toHaveTextContent(
      "This section couldn't load",
    );
    await user.click(within(card()).getByRole('button', { name: 'Try again' }));

    expect(await within(card()).findByRole('link', { name: 'Launch film' })).toBeInTheDocument();
  });
});
