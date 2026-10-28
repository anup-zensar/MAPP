import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ROLES } from '@/testing/fixtures/access';
import { CHINA_WORKFLOW, workflowOf } from '@/testing/fixtures/approval-workflows';
import { mockApi } from '@/testing/mock-api';
import { FIELD_HELP_PAGE } from '@/testing/fixtures/field-help';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { SettingsPage } from '@/features/settings';
import { reply } from '@/testing/replies';
import type { components } from '@/api/generated/schema';
import { loadSettingsSections } from './settings-sections';

// The real list of cards: 002's five and Field help (016), then People and Teams (003), Approval
// workflows (008) and Roles and access (014), whose features count their own records.

const OVERVIEW = {
  models: 3,
  activityTypes: 12,
  shellAttributes: 41,
  currencies: 4,
  financeBudgetItems: 1_250,
};

type Person = components['schemas']['Person'];
type Team = components['schemas']['Team'];

const META = {
  version: 1,
  createdAt: '2026-01-05T09:00:00Z',
  createdBy: { id: 'U-1001', name: 'Alex Morgan' },
  updatedAt: '2026-01-05T09:00:00Z',
  updatedBy: { id: 'U-1001', name: 'Alex Morgan' },
};

/** People that only matter by their status here. */
function peopleOf(active: boolean[]): Person[] {
  return active.map((isActive, index) => ({
    ...META,
    id: `U-${2000 + index}`,
    name: `Person ${index}`,
    email: `person.${index}@shell.com`,
    jobTitle: 'Marketer',
    initials: 'PE',
    active: isActive,
    teams: [],
  }));
}

/** Teams that only matter by their status here. */
function teamsOf(active: boolean[]): Team[] {
  return active.map((isActive, index) => ({
    ...META,
    id: `T-${2000 + index}`,
    name: `Team ${index}`,
    description: '',
    colour: 'blue',
    active: isActive,
    members: [],
    plans: [],
  }));
}

function card(title: string): HTMLElement {
  const list = screen.getByRole('list', { name: 'Sections' });
  const item = within(list)
    .getAllByRole('listitem')
    .find((element) => within(element).queryByRole('heading', { name: title }));
  if (!item) {
    throw new Error(`No card "${title}"`);
  }
  return item;
}

describe('SettingsPage, the cards of other features (FR-001)', () => {
  it('shows the People, Teams and Approval workflows cards with their texts and active counts, like the others', async () => {
    mockApi([
      reply('GET /api/v1/settings/overview', { body: OVERVIEW }),
      reply('GET /api/v1/people', {
        body: { items: peopleOf([true, true, true, true, true, true, false]), total: 7 },
      }),
      reply('GET /api/v1/teams', { body: { items: teamsOf([true, true, true]), total: 3 } }),
      reply('GET /api/v1/approval-workflows', {
        body: {
          items: [workflowOf(), workflowOf({ id: 'WF-1003' }), CHINA_WORKFLOW], // CHINA_WORKFLOW is inactive
          total: 3,
        },
      }),
      reply('GET /api/v1/roles', { body: { items: ROLES, total: ROLES.length } }),
      reply('GET /api/v1/field-help', { body: FIELD_HELP_PAGE }),
    ]);
    renderWithProviders(<SettingsPage sections={await loadSettingsSections()} />, {
      route: '/settings',
      user: testUser(),
    });

    expect(
      within(screen.getByRole('list', { name: 'Sections' }))
        .getAllByRole('heading')
        .map((heading) => heading.textContent),
    ).toEqual([
      'Models',
      'Activity types',
      'Shell attributes',
      'Currencies',
      'Finance budget items',
      'Field help',
      'People',
      'Teams',
      'Approval workflows',
      'Purchase requests',
      'Roles and access',
    ]);
    const people = card('People');
    expect(within(people).getByText('Everyone who can sign in to MAPP.')).toBeInTheDocument();
    expect(await within(people).findByText('6 active')).toBeInTheDocument();
    const teams = card('Teams');
    expect(within(teams).getByText('The teams plans belong to.')).toBeInTheDocument();
    expect(await within(teams).findByText('3 active')).toBeInTheDocument();
    const workflows = card('Approval workflows');
    expect(
      within(workflows).getByText('Who approves activities, level by level.'),
    ).toBeInTheDocument();
    const count = await within(workflows).findByText('2 active');
    // The same look as 002's own counts.
    expect(count.className).toBe((await within(card('Models')).findByText('3 active')).className);
    expect(within(people).getByRole('link', { name: 'People' })).toHaveAttribute(
      'href',
      '/settings/people',
    );
    // 032 FR-004: the Purchase requests card follows Approval workflows.
    expect(
      within(card('Purchase requests')).getByRole('link', { name: 'Purchase requests' }),
    ).toHaveAttribute('href', '/settings/purchase-requests');
    const access = card('Roles and access');
    expect(
      within(access).getByText('Who can see and change what, by role and market.'),
    ).toBeInTheDocument();
    expect(await within(access).findByText('5 roles')).toBeInTheDocument();
    expect(within(access).getByRole('link', { name: 'Roles and access' })).toHaveAttribute(
      'href',
      '/settings/access',
    );
    // Field help (016) counts the texts administrators changed.
    const help = card('Field help');
    expect(
      within(help).getByText('What each built-in field means, shown as ⓘ tips.'),
    ).toBeInTheDocument();
    expect(await within(help).findByText('2 edited')).toBeInTheDocument();
    expect(within(help).getByRole('link', { name: 'Field help' })).toHaveAttribute(
      'href',
      '/settings/field-help',
    );
  });
});
