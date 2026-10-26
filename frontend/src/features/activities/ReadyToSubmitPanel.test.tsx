import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { ReadyToSubmitPanel } from './ReadyToSubmitPanel';
import { incompleteLaunchFilm, launchFilm } from './testData';

describe('ReadyToSubmitPanel (feature 006, FR-022 – FR-024)', () => {
  it('lists every problem at once, grouped and in order, each a link to where it is fixed', () => {
    renderWithProviders(
      <ReadyToSubmitPanel activity={incompleteLaunchFilm()} open onOpenChange={() => {}} />,
    );

    const panel = screen.getByRole('region', { name: 'Ready to submit' });
    expect(within(panel).getByText('Submitting needs 5 things')).toBeVisible();
    const groups = within(panel).getAllByRole('heading', { level: 3 });
    expect(groups.map((group) => group.textContent)).toEqual(['This activity', 'Line items']);
    const links = within(panel).getAllByRole('link');
    expect(links.map((link) => [link.textContent, link.getAttribute('href')])).toEqual([
      ['Owner — needed to submit', '/plans/P-1001/activities/A-1042/edit?field=ownerId'],
      [
        'Cost centre — needed to submit',
        '/plans/P-1001/activities/A-1042/edit?field=attributes.SA-1015',
      ],
      [
        'Country / OU shares add up to 90.00 % — they must add up to 100.00 %',
        '/plans/P-1001/activities/A-1042/edit?field=attributes.SA-1013',
      ],
      [
        "Line item 'Film shoot' (L-1001): GL code — needed to submit",
        '/line-items/L-1001/edit?field=attributes.SA-1033',
      ],
      ["Line items exceed the activity's planned cost by 500.00 USD", '#line-items'],
    ]);
  });

  it('says "Ready to submit" when nothing is missing, with the advice that never blocks', () => {
    renderWithProviders(
      <ReadyToSubmitPanel
        activity={launchFilm({
          readiness: {
            ready: true,
            problemCount: 0,
            activity: [],
            lineItems: [],
            advice: [
              {
                code: 'unassignedRemaining',
                field: 'lineItems',
                lineItemId: null,
                lineItemName: null,
                message: '20,000.00 USD not yet assigned to a line item',
              },
            ],
          },
        })}
        open={false}
        onOpenChange={() => {}}
      />,
    );

    const panel = screen.getByRole('region', { name: 'Ready to submit' });
    expect(within(panel).getByRole('heading', { level: 3, name: 'Good to know' })).toBeVisible();
    expect(
      within(panel).getByRole('link', { name: '20,000.00 USD not yet assigned to a line item' }),
    ).toHaveAttribute('href', '#line-items');
    expect(within(panel).queryByText(/Submitting needs/)).not.toBeInTheDocument();
  });
});
