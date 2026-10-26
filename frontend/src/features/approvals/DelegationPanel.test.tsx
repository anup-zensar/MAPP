import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { HttpHandler } from 'msw';
import { PEOPLE } from '@/testing/fixtures/approval-workflows';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import {
  ALEX,
  CARLOS,
  DANIEL,
  delegationOf,
  NO_DELEGATIONS,
  PRIYA,
} from '@/testing/fixtures/approvals';
import { looseReply } from '@/testing/replies';
import { DelegationPanel } from './DelegationPanel';

const CREATE = 'POST /api/v1/approval-delegations';

function renderPanel(answers: HttpHandler[]) {
  const api = mockApi([looseReply('GET /api/v1/people', PEOPLE), ...answers]);
  const onClose = vi.fn();
  const rendered = renderWithProviders(<DelegationPanel onClose={onClose} />, {
    user: testUser(),
  });
  return { ...rendered, api, onClose };
}

const listOf = (items: ReturnType<typeof delegationOf>[], canManageOthers = false) => ({
  body: { items, total: items.length, canManageOthers },
});

describe('DelegationPanel (feature 017, US4)', () => {
  it('groups the delegations, with Edit and Cancel only where they may change (FR-024)', async () => {
    renderPanel([
      looseReply(
        'GET /api/v1/approval-delegations',
        listOf([
          delegationOf({ id: 'DLG-1002', state: 'current', startDate: '2026-10-19' }),
          delegationOf(),
          delegationOf({
            id: 'DLG-1000',
            person: PRIYA,
            delegate: ALEX,
            state: 'cancelled',
            canChange: false,
            note: '',
          }),
        ]),
      ),
    ]);

    const panel = await screen.findByRole('dialog', { name: 'Approval delegation' });
    const current = await within(panel).findByRole('region', { name: 'Current' });
    expect(current).toHaveTextContent('Daniel Okoro → Carlos Mendes');
    expect(current).toHaveTextContent('Current');
    const planned = within(panel).getByRole('region', { name: 'Planned' });
    expect(planned).toHaveTextContent('03 Nov – 14 Nov 2026');
    expect(planned).toHaveTextContent('Annual leave');
    expect(
      within(planned).getByRole('button', { name: /^Edit delegation Daniel Okoro → Carlos/ }),
    ).toBeInTheDocument();
    const ended = within(panel).getByRole('region', { name: 'Ended and cancelled' });
    expect(ended).toHaveTextContent('Priya Raman → Alex Morgan');
    expect(within(ended).queryByRole('button')).not.toBeInTheDocument();
    // Only someone who may manage everyone's delegations sees the switch.
    expect(within(panel).queryByRole('checkbox', { name: 'Everyone' })).not.toBeInTheDocument();
  });

  it('says what to do when there is no delegation yet', async () => {
    renderPanel([looseReply('GET /api/v1/approval-delegations', NO_DELEGATIONS)]);

    expect(
      await screen.findByText(
        "You haven't delegated your approvals. Add a delegation when you will be away.",
      ),
    ).toBeInTheDocument();
  });

  it('adds a delegation and says so (FR-022)', async () => {
    const { user, api } = renderPanel([
      looseReply('GET /api/v1/approval-delegations', NO_DELEGATIONS),
      looseReply('POST /api/v1/approval-delegations', {
        status: 201,
        body: delegationOf({ person: PRIYA, delegate: DANIEL }),
      }),
    ]);
    await screen.findByText(/You haven't delegated/);

    await user.click(screen.getByRole('button', { name: 'Add delegation' }));
    const form = await screen.findByRole('dialog', { name: 'Add delegation' });
    expect(within(form).queryByRole('combobox', { name: /Delegating person/ })).toBeNull();
    await user.click(await within(form).findByRole('combobox', { name: /^Delegate/ }));
    await user.click(screen.getByRole('option', { name: /Daniel Okoro/ }));
    await user.type(within(form).getByLabelText('Start date'), '2026-11-03');
    await user.type(within(form).getByLabelText('End date'), '2026-11-14');
    await user.type(within(form).getByRole('textbox', { name: /Note/ }), 'Annual leave');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('Delegation to Daniel Okoro saved')).toBeInTheDocument();
    expect(api.bodiesOf(CREATE)).toEqual([
      {
        delegateId: 'U-1003',
        startDate: '2026-11-03',
        endDate: '2026-11-14',
        note: 'Annual leave',
      },
    ]);
    expect(await screen.findByRole('dialog', { name: 'Approval delegation' })).toBeInTheDocument();
  });

  it('shows every problem of the API at its field (FR-022)', async () => {
    const { user } = renderPanel([
      looseReply('GET /api/v1/approval-delegations', NO_DELEGATIONS),
      looseReply('POST /api/v1/approval-delegations', {
        status: 400,
        body: {
          error: {
            code: 'validation_failed',
            message: 'Fix the problems.',
            fields: [
              { field: 'delegateId', message: 'Choose who approves for you.' },
              { field: 'startDate', message: 'Enter the first day.' },
              { field: 'endDate', message: 'Enter the last day.' },
            ],
          },
        },
      }),
    ]);
    await screen.findByText(/You haven't delegated/);

    await user.click(screen.getByRole('button', { name: 'Add delegation' }));
    const form = await screen.findByRole('dialog', { name: 'Add delegation' });
    await within(form).findByRole('combobox', { name: /^Delegate/ });
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(
      await screen.findByRole('link', { name: 'Choose who approves for you.' }),
    ).toHaveAttribute('href', '#delegateId');
    expect(screen.getByRole('link', { name: 'Enter the first day.' })).toHaveAttribute(
      'href',
      '#delegation-startDate',
    );
    expect(within(form).getByLabelText('End date')).toHaveAccessibleDescription(
      expect.stringContaining('Enter the last day.'),
    );
  });

  it('lets someone who manages delegations see everyone and add one for another person (FR-022, FR-024)', async () => {
    const { user, api } = renderPanel([
      looseReply('GET /api/v1/approval-delegations', (request) =>
        new URL(request.url).searchParams.get('scope') === 'all'
          ? listOf([delegationOf({ person: CARLOS, delegate: ALEX })], true)
          : listOf([], true),
      ),
    ]);

    const everyone = await screen.findByRole('checkbox', { name: 'Everyone' });
    await user.click(everyone);
    expect(await screen.findByText('Carlos Mendes → Alex Morgan')).toBeInTheDocument();
    expect(api.requests.some((r) => r.search === '?scope=all')).toBe(true);

    await user.click(screen.getByRole('button', { name: 'Add delegation' }));
    expect(await screen.findByRole('combobox', { name: /Delegating person/ })).toBeInTheDocument();
  });

  it('cancels a delegation after asking, and says so (FR-024)', async () => {
    const { user, api } = renderPanel([
      looseReply('GET /api/v1/approval-delegations', listOf([delegationOf()])),
      looseReply(
        'POST /api/v1/approval-delegations/{delegationId}/cancel',
        {
          body: delegationOf({ state: 'cancelled', canChange: false }),
        },
        { delegationId: 'DLG-1001' },
      ),
    ]);

    await user.click(
      await screen.findByRole('button', { name: /^Cancel delegation Daniel Okoro → Carlos/ }),
    );
    const dialog = await screen.findByRole('alertdialog', {
      name: 'Cancel the delegation to Carlos Mendes?',
    });
    expect(dialog).toHaveTextContent(
      'Carlos Mendes can no longer approve for Daniel Okoro from now on. Decisions already made stay recorded.',
    );
    expect(within(dialog).getByRole('button', { name: 'Keep delegation' })).toBeInTheDocument();
    await user.click(within(dialog).getByRole('button', { name: 'Cancel delegation' }));

    expect(await screen.findByText('Delegation to Carlos Mendes cancelled')).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/approval-delegations/DLG-1001/cancel')).toEqual([
      { version: 1 },
    ]);
  });

  it('changes a delegation, sending only what changed', async () => {
    const { user, api } = renderPanel([
      looseReply('GET /api/v1/approval-delegations', listOf([delegationOf()])),
      looseReply(
        'PATCH /api/v1/approval-delegations/{delegationId}',
        { body: delegationOf() },
        { delegationId: 'DLG-1001' },
      ),
    ]);

    await user.click(
      await screen.findByRole('button', { name: /^Edit delegation Daniel Okoro → Carlos/ }),
    );
    const form = await screen.findByRole('dialog', { name: 'Edit delegation' });
    const note = within(form).getByRole('textbox', { name: /Note/ });
    await user.clear(note);
    await user.type(note, 'Conference');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('Delegation to Carlos Mendes saved')).toBeInTheDocument();
    expect(api.bodiesOf('PATCH /api/v1/approval-delegations/DLG-1001')).toEqual([
      { version: 1, note: 'Conference' },
    ]);
  });
});
