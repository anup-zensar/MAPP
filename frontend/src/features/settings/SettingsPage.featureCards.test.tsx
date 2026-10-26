import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ROLES } from '@/features/access/testData';
import { CHINA_WORKFLOW, workflowOf } from '@/features/approval-workflows/testData';
import { mockApi } from '@/test/mockApi';
import { FIELD_HELP_PAGE } from './field-help/fieldHelpTestData';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { SettingsPage } from './SettingsPage';

// The real list of cards: 002's five and Field help (016), then People and Teams (003), Approval
// workflows (008) and Roles and access (014), whose features count their own records.

const OVERVIEW = {
  models: 3,
  activityTypes: 12,
  shellAttributes: 41,
  currencies: 4,
  financeBudgetItems: 1_250,
};

/** A list answer of records that only matter by their status here. */
function listOf(active: boolean[]) {
  return {
    body: {
      items: active.map((isActive, index) => ({
        id: `R-${index}`,
        name: `R ${index}`,
        active: isActive,
      })),
      total: active.length,
    },
  };
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
    mockApi({
      'GET /api/settings/overview': { body: OVERVIEW },
      'GET /api/people': listOf([true, true, true, true, true, true, false]),
      'GET /api/teams': listOf([true, true, true]),
      'GET /api/approval-workflows': {
        body: {
          items: [workflowOf(), workflowOf({ id: 'WF-1003' }), CHINA_WORKFLOW], // CHINA_WORKFLOW is inactive
          total: 3,
        },
      },
      'GET /api/roles': { body: { items: ROLES, total: ROLES.length } },
      'GET /api/field-help': { body: FIELD_HELP_PAGE },
    });
    renderWithProviders(<SettingsPage />, { route: '/settings', user: testUser() });

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
