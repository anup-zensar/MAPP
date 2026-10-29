import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  ALEX,
  chooseAction,
  DANIEL,
  JANE,
  PRIYA,
  renderPeople,
  SARA,
  validation,
} from '@/testing/fixtures/people';
import { expectNoAxeViolations } from '@/testing/axe';
import { reply } from '@/testing/replies';

const DANIEL_REF = {
  id: DANIEL.id,
  name: DANIEL.name,
  initials: DANIEL.initials,
  jobTitle: DANIEL.jobTitle,
  active: true,
};

const WITH_MANAGERS = reply('GET /api/v1/people', {
  body: {
    items: [
      { ...ALEX, manager: null, reportCount: 1 },
      { ...DANIEL, manager: null, reportCount: 2 },
      JANE,
      { ...PRIYA, manager: DANIEL_REF, reportCount: 0 },
      { ...SARA, manager: DANIEL_REF, reportCount: 0 },
    ],
    total: 5,
  },
});

const DANIEL_DETAILS = {
  ...DANIEL,
  usage: [{ role: 'manager' as const, recordType: 'person', label: 'people', count: 2 }],
};

describe('017 FR-016: the line manager on Settings › People', () => {
  it('shows a person’s line manager, and whom a person is line manager of', async () => {
    renderPeople(
      [WITH_MANAGERS, reply('GET /api/v1/people/{personId}', { body: DANIEL_DETAILS })],
      '/settings/people/U-1003',
    );

    const panel = await screen.findByRole('dialog', { name: 'Daniel Okoro' });
    expect(await within(panel).findByText('No line manager')).toBeVisible();
    expect(within(panel).getByText('Line manager of')).toBeVisible();
    expect(within(panel).getByText('2 people')).toBeVisible();
    await expectNoAxeViolations();
  });

  it('changes Priya’s line manager, sending only that change', async () => {
    const { api, user } = renderPeople([
      WITH_MANAGERS,
      reply('PATCH /api/v1/people/{personId}', { body: { ...PRIYA, usage: [], version: 2 } }),
    ]);

    await chooseAction(user, 'Priya Raman', 'Edit');
    const panel = await screen.findByRole('dialog', { name: 'Edit person' });
    expect(
      within(panel).getByText(
        "Approves this person's activities when a workflow level uses the line manager.",
      ),
    ).toBeVisible();
    await user.click(within(panel).getByRole('button', { name: 'Remove Daniel Okoro' }));
    await user.type(within(panel).getByRole('combobox', { name: 'Line manager' }), 'Alex');
    await user.click(await screen.findByRole('option', { name: /Alex Morgan/ }));
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await screen.findByText("Person 'Priya Raman' saved")).toBeInTheDocument();
    expect(api.bodiesOf('PATCH /api/v1/people/U-1002')).toEqual([
      { managerId: 'U-1001', version: 1 },
    ]);
  });

  it('shows the API’s refusal at the field, with everything else that is wrong', async () => {
    const message = 'Priya Raman reports to Sara Novak, so Sara Novak can’t report to Priya Raman.';
    const { user } = renderPeople([
      WITH_MANAGERS,
      reply('PATCH /api/v1/people/{personId}', validation([{ field: 'managerId', message }])),
    ]);

    await chooseAction(user, 'Sara Novak', 'Edit');
    const panel = await screen.findByRole('dialog', { name: 'Edit person' });
    await user.click(within(panel).getByRole('button', { name: 'Remove Daniel Okoro' }));
    await user.type(within(panel).getByRole('combobox', { name: 'Line manager' }), 'Priya');
    await user.click(await screen.findByRole('option', { name: /Priya Raman/ }));
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await within(panel).findAllByText(message)).toHaveLength(2);
  });
});
