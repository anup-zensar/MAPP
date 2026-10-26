import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { findMessages } from '@/testing/render';
import {
  ALEX_DETAILS,
  chooseAction,
  JANE,
  openActions,
  renderPeople,
  SARA,
} from '@/testing/fixtures/people';
import { reply } from '@/testing/replies';

describe('PeoplePage (feature 003, US1): Deactivate and Reactivate', () => {
  it('asks before deactivating, with what the person owns and Cancel focused (US1-5)', async () => {
    const { api, user } = renderPeople([
      reply('GET /api/v1/people/{personId}', { body: ALEX_DETAILS }, { personId: 'U-1001' }),
      reply(
        'POST /api/v1/people/{personId}/status',
        { body: { ...ALEX_DETAILS, active: false, version: 2 } },
        { personId: 'U-1001' },
      ),
    ]);

    await chooseAction(user, 'Alex Morgan', 'Deactivate');
    const dialog = await screen.findByRole('alertdialog', { name: 'Deactivate Alex Morgan?' });
    expect(within(dialog).getByRole('button', { name: 'Cancel' })).toHaveFocus();
    expect(
      await within(dialog).findByText(
        "Alex owns 2 plans and 5 activities and approves in 1 approval workflow; these keep the name, shown as 'Alex Morgan (inactive)'.",
      ),
    ).toBeInTheDocument();
    expect(dialog).toHaveTextContent("Alex won't be able to sign in or be chosen in people lists.");
    expect(dialog).toHaveTextContent('Their teams are kept.');

    await user.click(within(dialog).getByRole('button', { name: 'Deactivate person' }));

    expect(await screen.findByText("Person 'Alex Morgan' deactivated")).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/people/U-1001/status')).toEqual([
      { active: false, version: 1 },
    ]);
  });

  it('sends nothing when the deactivation is cancelled', async () => {
    const { api, user } = renderPeople([
      reply('GET /api/v1/people/{personId}', { body: ALEX_DETAILS }, { personId: 'U-1001' }),
    ]);

    await chooseAction(user, 'Alex Morgan', 'Deactivate');
    const dialog = await screen.findByRole('alertdialog', { name: 'Deactivate Alex Morgan?' });
    await user.click(within(dialog).getByRole('button', { name: 'Cancel' }));

    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/people/U-1001/status')).toEqual([]);
  });

  it("explains that you can't deactivate yourself (US1-6)", async () => {
    const { api, user } = renderPeople();

    const menu = await openActions(user, 'Priya Raman');
    const deactivate = within(menu).getByRole('menuitem', { name: 'Deactivate' });
    expect(deactivate).toHaveAccessibleDescription(
      "You can't deactivate yourself — ask someone else to do it.",
    );
    await user.click(deactivate);

    expect(
      await screen.findByText("You can't deactivate yourself — ask someone else to do it."),
    ).toBeInTheDocument();
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(api.requests.some((request) => request.method === 'POST')).toBe(false);
  });

  it('deactivates someone who owns nothing at once, with Undo (design system §6.7)', async () => {
    const saraDetails = { ...SARA, usage: [] };
    let posts = 0;
    const { api, user } = renderPeople([
      reply('GET /api/v1/people/{personId}', { body: saraDetails }, { personId: 'U-1004' }),
      reply(
        'POST /api/v1/people/{personId}/status',
        () => {
          posts += 1;
          return posts === 1
            ? { body: { ...saraDetails, active: false, version: 2 } }
            : { body: { ...saraDetails, active: true, version: 3 } };
        },
        { personId: 'U-1004' },
      ),
    ]);

    await chooseAction(user, 'Sara Novak', 'Deactivate');

    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    const messages = await findMessages();
    expect(
      await within(messages).findByText("Person 'Sara Novak' deactivated"),
    ).toBeInTheDocument();
    await user.click(within(messages).getByRole('button', { name: 'Undo' }));

    expect(await screen.findByText("Person 'Sara Novak' reactivated")).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/people/U-1004/status')).toEqual([
      { active: false, version: 1 },
      { active: true, version: 2 },
    ]);
  });

  it('reactivates an inactive person (US1-7)', async () => {
    const { api, user } = renderPeople(
      [
        reply(
          'POST /api/v1/people/{personId}/status',
          {
            body: { ...JANE, active: true, version: 2, usage: [] },
          },
          { personId: 'U-1006' },
        ),
      ],
      '/settings/people?status=inactive',
    );

    await chooseAction(user, 'Jane Doe', 'Reactivate');

    expect(await screen.findByText("Person 'Jane Doe' reactivated")).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/people/U-1006/status')).toEqual([
      { active: true, version: 1 },
    ]);
  });

  it('says why a status change was refused', async () => {
    const { user } = renderPeople(
      [
        reply(
          'POST /api/v1/people/{personId}/status',
          {
            status: 404,
            body: {
              error: { code: 'not_found', message: "This person doesn't exist.", fields: [] },
            },
          },
          { personId: 'U-1006' },
        ),
      ],
      '/settings/people?status=inactive',
    );

    await chooseAction(user, 'Jane Doe', 'Reactivate');

    expect(await screen.findByText("This person doesn't exist.")).toBeInTheDocument();
  });
});
