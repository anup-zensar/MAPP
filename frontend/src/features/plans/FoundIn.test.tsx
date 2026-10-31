import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { nth } from '@/testing/dom';
import { renderWithProviders } from '@/testing/render';
import { seedList } from '@/testing/fixtures/plans';
import { testUser } from '@/testing/fixtures/users';
import { FoundIn } from './FoundIn';
import { PlanCard } from './PlanCard';

const VENUE_HIRE = {
  level: 'lineItem',
  id: 'L-1001',
  name: 'Venue hire',
  parentId: 'A-1001',
} as const;

describe('039 US2: the Plans list says where the search matched inside a plan', () => {
  it('scenario 1: the card names the line item that holds the text', () => {
    const europe = { ...nth(seedList().items, 0), foundIn: { match: VENUE_HIRE, more: 0 } };

    renderWithProviders(
      <PlanCard plan={europe} actions={[]} pinned={false} onPinToggle={() => {}} />,
      {
        user: testUser(),
        route: '/plans',
        path: '/plans',
      },
    );

    expect(screen.getByText("Found in line item 'Venue hire' (L-1001)")).toBeVisible();
  });

  it('scenario 2: names the first match and how many more', () => {
    const campaign = {
      level: 'campaign',
      id: 'C-1001',
      name: 'Venues',
      parentId: 'P-1001',
    } as const;

    render(<FoundIn foundIn={{ match: campaign, more: 2 }} />);

    expect(screen.getByText("Found in campaign 'Venues' (C-1001) and 2 more")).toBeVisible();
  });

  it('scenario 3: says nothing for a plan found by its own name or ID', () => {
    const { container } = render(<FoundIn />);

    expect(container).toBeEmptyDOMElement();
  });
});
