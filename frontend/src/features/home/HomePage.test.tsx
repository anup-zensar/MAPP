import { screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import type { CurrentUser } from '@/app/useSession';
import { renderWithProviders } from '@/test/render';
import { HomePage } from './HomePage';
import { testUser } from '@/test/users';

const PRIYA: CurrentUser = testUser();

function gettingStarted() {
  return screen.getByRole('region', { name: 'Getting started' });
}

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('HomePage', () => {
  it('greets the person by first name (US3 scenario 1, FR-018)', () => {
    renderWithProviders(<HomePage />, { user: PRIYA });

    expect(
      screen.getByRole('heading', { level: 1, name: 'Welcome back, Priya.' }),
    ).toBeInTheDocument();
    expect(screen.getByText('Home')).toBeInTheDocument();
    expect(
      screen.getByText("Here's how planning works in NorthStar — follow the steps in order."),
    ).toBeInTheDocument();
  });

  it('shows the five getting-started steps in order, each with its link and explanation (FR-018)', () => {
    renderWithProviders(<HomePage />, { user: PRIYA });

    const card = gettingStarted();
    expect(within(card).getByText('The Plan module in five steps')).toBeInTheDocument();
    const steps = within(card).getAllByRole('listitem');
    expect(
      steps.map((step) => {
        const link = within(step).getByRole('link');
        return [link.textContent, link.getAttribute('href'), step.textContent];
      }),
    ).toEqual([
      [
        'Set up Settings',
        '/settings',
        '1Set up SettingsAdd the models, activity types, Shell attributes and currencies your plans use.',
      ],
      ['Create a plan', '/plans', '2Create a planGive the plan its year, budget, owner and teams.'],
      [
        'Add campaigns and initiatives',
        '/plans',
        '3Add campaigns and initiativesSplit the plan into campaigns, and each campaign into initiatives with their own budgets.',
      ],
      [
        'Add activities and line items',
        '/plans',
        '4Add activities and line itemsDescribe each piece of work and plan its costs as line items.',
      ],
      [
        'Submit for approval',
        '/approvals',
        '5Submit for approvalSend activities to their approvers and follow their status.',
      ],
    ]);
  });

  it('hides the step numbers from screen readers, because the list numbers the steps (plan "Home")', () => {
    renderWithProviders(<HomePage />, { user: PRIYA });

    const firstStep = within(gettingStarted()).getAllByRole('listitem')[0];
    expect(within(firstStep).getByText('1')).toHaveAttribute('aria-hidden', 'true');
    expect(within(gettingStarted()).getByRole('list').tagName).toBe('OL');
  });

  it('opens the page of a step from its link (US3 independent test)', async () => {
    const { user, router } = renderWithProviders(<HomePage />, { user: PRIYA });

    await user.click(screen.getByRole('link', { name: 'Submit for approval' }));

    expect(router.state.location.pathname).toBe('/approvals');
  });

  it('names the page "Home · NorthStar" in the browser tab (FR-017)', () => {
    renderWithProviders(<HomePage />, { user: PRIYA });

    expect(document.title).toBe('Home · NorthStar');
  });
});
