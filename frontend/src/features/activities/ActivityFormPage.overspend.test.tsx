import { configure, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import { PEOPLE } from '@/testing/fixtures/activities';
import { activityAttributeForm, newForm } from '@/testing/fixtures/activity-forms';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { reply } from '@/testing/replies';
import { ActivityFormPage } from './ActivityFormPage';

configure({ asyncUtilTimeout: 10_000 });

const LIMIT = {
  budget: { type: 'campaign', id: 'C-1001', name: 'Helix launch' },
  leftToPlan: { amount: '5000.00', currency: 'USD' },
};

function openNewForm() {
  const api = mockApi([
    reply(
      'GET /api/v1/initiatives/{initiativeId}/activity-form',
      { body: newForm({ overspendLimits: [LIMIT] }) },
      { initiativeId: 'I-1001' },
    ),
    reply(
      'GET /api/v1/shell-attribute-forms/{level}',
      { body: activityAttributeForm() },
      { level: 'activity' },
    ),
    reply('GET /api/v1/people', { body: PEOPLE }),
  ]);
  const rendered = renderWithProviders(<ActivityFormPage mode="new" />, {
    route: '/plans/P-1001/initiatives/I-1001/activities/new',
    path: '/plans/:planId/initiatives/:initiativeId/activities/new',
    user: testUser(),
  });
  return { ...rendered, api };
}

describe('ActivityFormPage — overspend limits (037 FR-014)', () => {
  it('says what is left to plan, and refuses on the form a cost over it', async () => {
    const { user, api } = openNewForm();
    await user.type(await screen.findByLabelText('Activity name'), 'Launch video');
    expect(
      screen.getByText(
        "5,000.00 USD left to plan in campaign 'Helix launch', which refuses overspending.",
      ),
    ).toBeVisible();

    const cost = screen.getByLabelText('Planned cost');
    await user.clear(cost);
    await user.type(cost, '6000');
    await user.tab();
    const over =
      "This is 1,000.00 USD over the budget of campaign 'Helix launch', which refuses overspending.";

    expect(screen.getAllByText(over)[0]).toBeVisible();
    await expectNoAxeViolations();
    await user.click(screen.getByRole('button', { name: 'Save' }));
    expect(screen.getAllByText(over).length).toBeGreaterThan(0);
    expect(api.bodiesOf('POST /api/v1/initiatives/I-1001/activities')).toHaveLength(0);
  });
});
