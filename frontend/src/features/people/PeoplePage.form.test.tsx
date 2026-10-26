import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { findMessages } from '@/testing/render';
import {
  ALEX_DETAILS,
  chooseAction,
  EUROPE,
  person,
  renderPeople,
  validation,
} from '@/testing/fixtures/people';
import { reply } from '@/testing/replies';

describe('PeoplePage (feature 003, US1): New person and Edit person', () => {
  it('shows every problem at once, then creates the person with View (US1-1, US1-2)', async () => {
    let posts = 0;
    const created = { ...person('U-1007', 'Jane Smith', 'Marketer', [EUROPE]), usage: [] };
    const { api, user, router } = renderPeople([
      reply('POST /api/v1/people', () => {
        posts += 1;
        if (posts === 1) {
          return validation([
            { field: 'name', message: 'Enter a name.' },
            { field: 'email', message: 'Enter a valid e-mail address.' },
          ]);
        }
        if (posts === 2) {
          return validation([
            {
              field: 'email',
              message:
                'jane.doe@shell.com already belongs to Jane Doe (inactive) — reactivate them instead.',
            },
          ]);
        }
        return { status: 201, body: created };
      }),
      reply('GET /api/v1/people/{personId}', { body: created }, { personId: 'U-1007' }),
    ]);

    await user.click(await screen.findByRole('button', { name: 'New person' }));
    const panel = await screen.findByRole('dialog', { name: 'New person' });
    expect(
      within(panel).getByText(
        'Their sign-in name. They can sign in straight away with the demo password.',
      ),
    ).toBeInTheDocument();
    await user.type(within(panel).getByRole('textbox', { name: /E-mail/ }), 'jane');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await within(panel).findAllByText('Enter a name.')).toHaveLength(2);
    expect(within(panel).getAllByText('Enter a valid e-mail address.')).toHaveLength(2);

    await user.type(within(panel).getByRole('textbox', { name: /Name/ }), 'Jane Smith');
    await user.type(within(panel).getByRole('textbox', { name: /E-mail/ }), '.doe@shell.com');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));
    expect(
      await within(panel).findAllByText(
        'jane.doe@shell.com already belongs to Jane Doe (inactive) — reactivate them instead.',
      ),
    ).toHaveLength(2);
    expect(within(panel).queryByText('Enter a name.')).not.toBeInTheDocument();

    const email = within(panel).getByRole('textbox', { name: /E-mail/ });
    await user.clear(email);
    await user.type(email, 'jane.smith@shell.com');
    await user.type(within(panel).getByRole('textbox', { name: /Job title/ }), 'Marketer');
    await user.type(await within(panel).findByRole('combobox', { name: 'Teams' }), 'europe');
    await user.keyboard('{Enter}');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await screen.findByText("Person 'Jane Smith' created")).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/people')[2]).toEqual({
      name: 'Jane Smith',
      email: 'jane.smith@shell.com',
      jobTitle: 'Marketer',
      teamIds: ['T-1001'],
    });
    const messages = await findMessages();
    await user.click(within(messages).getByRole('button', { name: 'View' }));
    await waitFor(() => expect(router.state.location.pathname).toBe('/settings/people/U-1007'));
  });

  it('edits a person and says who saved first after a conflict (US1-4)', async () => {
    let patches = 0;
    const { api, user } = renderPeople([
      reply(
        'PATCH /api/v1/people/{personId}',
        () => {
          patches += 1;
          return patches === 1
            ? {
                status: 409,
                body: {
                  error: {
                    code: 'version_conflict',
                    message: 'Not saved — Daniel Okoro changed this person.',
                    fields: [],
                    conflict: { changedBy: 'Daniel Okoro', changedAt: '2026-03-10T14:32:00Z' },
                  },
                },
              }
            : { body: { ...ALEX_DETAILS, jobTitle: 'Marketing director', version: 3 } };
        },
        { personId: 'U-1001' },
      ),
      reply(
        'GET /api/v1/people/{personId}',
        { body: { ...ALEX_DETAILS, version: 2 } },
        { personId: 'U-1001' },
      ),
    ]);

    await chooseAction(user, 'Alex Morgan', 'Edit');
    const panel = await screen.findByRole('dialog', { name: 'Edit person' });
    expect(
      within(panel).getByText('Their sign-in name — the next sign-in uses the new e-mail.'),
    ).toBeInTheDocument();
    const jobTitle = within(panel).getByRole('textbox', { name: /Job title/ });
    await user.clear(jobTitle);
    await user.type(jobTitle, 'Marketing director');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(
      await within(panel).findByText(
        'Not saved — Daniel Okoro changed this person on 10 Mar 2026 at 14:32',
      ),
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(api.requests.some((request) => request.path === '/api/v1/people/U-1001')).toBe(true),
    );
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await screen.findByText("Person 'Alex Morgan' saved")).toBeInTheDocument();
    // Only the changed field and the version (D-210).
    expect(api.bodiesOf('PATCH /api/v1/people/U-1001')).toEqual([
      { jobTitle: 'Marketing director', version: 1 },
      { jobTitle: 'Marketing director', version: 2 },
    ]);
    expect(api.bodiesOf('POST /api/v1/people/U-1001/impact')).toEqual([]);
  });

  it('checks the impact of removing a team; Keep editing sends nothing, Save changes saves', async () => {
    const warning =
      'Alex Morgan owns 1 plan that uses Lubricants Europe: FY2026 Lubricants Europe. Alex will no longer belong to its teams, so its owner will need attention.';
    const { api, user } = renderPeople([
      reply(
        'POST /api/v1/people/{personId}/impact',
        {
          body: { warnings: [{ kind: 'ownerNeedsAttention', message: warning, count: 1 }] },
        },
        { personId: 'U-1001' },
      ),
      reply(
        'PATCH /api/v1/people/{personId}',
        { body: { ...ALEX_DETAILS, teams: [], version: 2 } },
        { personId: 'U-1001' },
      ),
    ]);

    await chooseAction(user, 'Alex Morgan', 'Edit');
    const panel = await screen.findByRole('dialog', { name: 'Edit person' });
    await user.click(
      await within(panel).findByRole('button', { name: 'Remove Lubricants Europe' }),
    );
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    const dialog = await screen.findByRole('alertdialog', {
      name: 'Some plans will need a new owner',
    });
    expect(dialog).toHaveTextContent(warning);
    await user.click(within(dialog).getByRole('button', { name: 'Keep editing' }));
    expect(api.bodiesOf('PATCH /api/v1/people/U-1001')).toEqual([]);

    await user.click(within(panel).getByRole('button', { name: 'Save' }));
    await user.click(
      within(
        await screen.findByRole('alertdialog', { name: 'Some plans will need a new owner' }),
      ).getByRole('button', { name: 'Save changes' }),
    );

    expect(await screen.findByText("Person 'Alex Morgan' saved")).toBeInTheDocument();
    const body = { teamIds: [], version: 1 };
    expect(api.bodiesOf('POST /api/v1/people/U-1001/impact')).toEqual([body, body]);
    expect(api.bodiesOf('PATCH /api/v1/people/U-1001')).toEqual([body]);
  });

  it('shows the impact check problems at the fields', async () => {
    const { api, user } = renderPeople([
      reply(
        'POST /api/v1/people/{personId}/impact',
        validation([{ field: 'name', message: 'Enter a name.' }]),
        { personId: 'U-1001' },
      ),
    ]);

    await chooseAction(user, 'Alex Morgan', 'Edit');
    const panel = await screen.findByRole('dialog', { name: 'Edit person' });
    await user.clear(within(panel).getByRole('textbox', { name: /Name/ }));
    await user.click(
      await within(panel).findByRole('button', { name: 'Remove Lubricants Europe' }),
    );
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await within(panel).findAllByText('Enter a name.')).toHaveLength(2);
    expect(api.bodiesOf('PATCH /api/v1/people/U-1001')).toEqual([]);
  });
});
