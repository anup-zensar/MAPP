import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { nth } from '@/testing/dom';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { GettingStartedCard } from './GettingStartedCard';

const NO_TEAM_SENTENCE =
  "You're not in a team yet. Plans belong to teams — add yourself to one on the Teams page.";

function card() {
  return screen.getByRole('region', { name: 'Getting started' });
}

describe('GettingStartedCard', () => {
  it("shows 001's five steps in order, each with its link and explanation (FR-020, 001 FR-018)", () => {
    renderWithProviders(<GettingStartedCard reason="noTeamPlans" />, { user: testUser() });

    expect(within(card()).getByText('The Plan module in five steps')).toBeInTheDocument();
    const steps = within(card()).getAllByRole('listitem');
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

  it('hides the step numbers from screen readers, because the list numbers the steps', () => {
    renderWithProviders(<GettingStartedCard reason="noTeamPlans" />, { user: testUser() });

    const firstStep = nth(within(card()).getAllByRole('listitem'), 0);
    expect(within(firstStep).getByText('1')).toHaveAttribute('aria-hidden', 'true');
    expect(within(card()).getByRole('list').tagName).toBe('OL');
  });

  it('tells a person in no team how to join one, with a link to the Teams page (US5 scenario 1)', () => {
    renderWithProviders(<GettingStartedCard reason="notInTeam" />, { user: testUser() });

    expect(within(card()).getByText(/You're not in a team yet/)).toHaveTextContent(
      NO_TEAM_SENTENCE,
    );
    expect(within(card()).getByRole('link', { name: 'Teams page' })).toHaveAttribute(
      'href',
      '/settings/teams',
    );
  });

  it('offers a Viewer in no team only what the role can do and to ask an administrator (038 FR-051, L-25)', () => {
    const viewer = testUser({
      access: {
        roles: [],
        permissions: [{ key: 'plans.view', all: true, teams: [] }],
        administrators: [
          { id: 'U-1001', name: 'Alex Morgan', initials: 'AM', jobTitle: 'Admin', active: true },
        ],
        noSelfApproval: true,
      },
    });
    renderWithProviders(<GettingStartedCard reason="notInTeam" />, { user: viewer });

    expect(within(card()).getByText(/You're not in a team yet/)).toHaveTextContent(
      "You're not in a team yet. Plans belong to teams — ask an administrator (Alex Morgan) to add you to one.",
    );
    expect(within(card()).queryByRole('link', { name: 'Teams page' })).toBeNull();
    expect(within(card()).queryByRole('link', { name: 'Set up Settings' })).toBeNull();
    expect(within(card()).queryByRole('link', { name: 'Create a plan' })).toBeNull();
    expect(within(card()).queryByRole('link', { name: 'Submit for approval' })).toBeNull();
  });

  it('offers a planner the steps of the role, without Settings (038 FR-051)', () => {
    const keys = ['plans.view', 'plans.create', 'campaigns.edit', 'activities.edit'] as const;
    const planner = testUser({
      access: {
        roles: [],
        permissions: keys.map((key) => ({ key, all: true, teams: [] })),
        administrators: [],
        noSelfApproval: true,
      },
    });
    renderWithProviders(<GettingStartedCard reason="noTeamPlans" />, { user: planner });

    expect(
      within(card())
        .getAllByRole('link')
        .map((link) => link.textContent),
    ).toEqual(['Create a plan', 'Add campaigns and initiatives', 'Add activities and line items']);
    expect(within(card()).getByText('The Plan module in three steps')).toBeInTheDocument();
  });
});
