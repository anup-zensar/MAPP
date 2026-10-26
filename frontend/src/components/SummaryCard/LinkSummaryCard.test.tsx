import { screen } from '@testing-library/react';
import { ClipboardList } from 'lucide-react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { LinkSummaryCard } from './LinkSummaryCard';

describe('LinkSummaryCard', () => {
  it('is one link named "<label>: <value> — <link label>" that opens its list (FR-007, WCAG 2.5.3)', async () => {
    const { user, router } = renderWithProviders(
      <LinkSummaryCard
        icon={<ClipboardList />}
        tint="var(--blue)"
        label="My teams' plans"
        tip="Plans that have at least one of your teams."
        value="3"
        detail={<span>2 published · 1 draft</span>}
        to="/plans?scope=myTeams"
        linkLabel="open the Plans list"
      />,
      { user: testUser(), path: '/' },
    );

    const link = screen.getByRole('link', { name: "My teams' plans: 3 — open the Plans list" });
    expect(link).toHaveTextContent('3');
    expect(screen.getByText('2 published · 1 draft')).toBeInTheDocument();

    await user.click(link);

    expect(router.state.location.pathname).toBe('/plans');
    expect(router.state.location.search).toBe('?scope=myTeams');
  });

  it('explains its label with an info tip, without opening the list (FR-005)', async () => {
    const { user, router } = renderWithProviders(
      <LinkSummaryCard
        icon={<ClipboardList />}
        tint="var(--blue)"
        label="All plans"
        tip="Every plan in MAPP."
        value="5"
        to="/plans?scope=all"
        linkLabel="open the Plans list"
      />,
      { user: testUser(), path: '/' },
    );

    await user.click(screen.getByRole('button', { name: 'About All plans' }));

    expect(screen.getByRole('dialog', { name: 'About All plans' })).toHaveTextContent(
      'Every plan in MAPP.',
    );
    expect(router.state.location.pathname).toBe('/');
  });

  it('gives the full amount of a shortened value in its name and on hover (FR-018)', async () => {
    const { user } = renderWithProviders(
      <LinkSummaryCard
        icon={<ClipboardList />}
        tint="var(--blue)"
        label="Planned cost vs plan budget"
        tip="Planned cost of the plans in scope compared with their plan budgets."
        value="$1M"
        fullValue="$1,003,000.00"
        to="/plans?scope=myTeams"
        linkLabel="open the Plans list"
      />,
      { user: testUser(), path: '/' },
    );

    const link = screen.getByRole('link', {
      name: 'Planned cost vs plan budget: $1M ($1,003,000.00) — open the Plans list',
    });
    await user.tab(); // the ⓘ comes first
    await user.tab();

    expect(link).toHaveFocus();
    expect(await screen.findByRole('tooltip')).toHaveTextContent('$1,003,000.00');
  });

  it('colours the value only when the colour means something (FR-005)', () => {
    renderWithProviders(
      <LinkSummaryCard
        icon={<ClipboardList />}
        tint="var(--status-review)"
        label="Waiting for my approval"
        tip="Submitted activities waiting at a level where you are an approver."
        value="2"
        tone="review"
        to="/approvals?tab=waiting"
        linkLabel="open Approvals"
      />,
      { user: testUser(), path: '/' },
    );

    const link = screen.getByRole('link', { name: 'Waiting for my approval: 2 — open Approvals' });
    expect(link.parentElement?.className).toMatch(/review/);
  });

  it('scrolls to a part of the page for an address that starts with # (D-174)', () => {
    renderWithProviders(
      <LinkSummaryCard
        icon={<ClipboardList />}
        tint="var(--level-campaign)"
        label="Activities I own"
        tip="Activities you own, in every plan."
        value="6"
        to="#my-activities"
        linkLabel="open your activities"
      />,
      { user: testUser(), path: '/' },
    );

    expect(
      screen.getByRole('link', { name: 'Activities I own: 6 — open your activities' }),
    ).toHaveAttribute('href', '#my-activities');
  });
});
