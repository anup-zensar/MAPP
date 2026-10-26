import { screen, waitFor, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import type { Activity } from './activitiesApi';
import { CopyActivityDialog } from './CopyActivityDialog';
import {
  destinations,
  launchFilm,
  OWN_INITIATIVE,
  RETAIL_EVENTS,
} from '@/testing/fixtures/activities';

/** The dialog as the More menu holds it: closing it takes it off the page. */
function Harness({ activity }: { activity: Activity }) {
  const [open, setOpen] = useState(true);
  return open ? (
    <CopyActivityDialog activity={activity} onClose={() => setOpen(false)} />
  ) : (
    <p>Closed</p>
  );
}

function renderDialog(answers: Record<string, MockHandler>, activity = launchFilm()) {
  const api = mockApi(answers);
  const rendered = renderWithProviders(<Harness activity={activity} />, {
    user: testUser(),
    route: '/plans/P-1001/activities/A-1042',
  });
  return { ...rendered, api };
}

const APPROVED_WITH_LINE_ITEMS = launchFilm({
  status: 'approved',
  displayStatus: 'approved',
  lineItems: { ...launchFilm().lineItems, count: 3 },
});

describe('CopyActivityDialog (feature 006, US5#2, FR-030)', () => {
  it('chooses the activity’s own initiative and says what the copy keeps (US5#2)', async () => {
    renderDialog(
      { 'GET /api/v1/activities/A-1042/destinations': { body: destinations() } },
      APPROVED_WITH_LINE_ITEMS,
    );

    const dialog = screen.getByRole('dialog', { name: "Copy activity 'Launch film'" });
    const copyTo = await within(dialog).findByRole('group', { name: 'Copy to' });
    const own = within(copyTo).getByRole('radio', { name: 'Digital & social (this initiative)' });
    expect(own).toBeChecked();
    expect(own).toHaveFocus();
    expect(within(copyTo).getByRole('radio', { name: 'Retail events' })).toBeDisabled();
    expect(
      within(copyTo).getByRole('radio', { name: 'Retail events' }),
    ).toHaveAccessibleDescription(`01 Jun – 31 Dec 2026 ${RETAIL_EVENTS.copyProblem ?? ''}`);
    expect(dialog).toHaveTextContent(
      'The copy starts as Created, with copies of its 3 line items. Approvals, comments and history are not copied.',
    );
  });

  it('copies into the chosen initiative and opens the copy (US5#2)', async () => {
    const { user, api, router } = renderDialog(
      {
        'GET /api/v1/activities/A-1042/destinations': { body: destinations() },
        'POST /api/v1/activities/A-1042/copy': {
          status: 201,
          body: launchFilm({
            id: 'A-1043',
            name: 'Launch film (copy)',
            initiative: { type: 'initiative', id: 'I-1002', name: 'Print & outdoor' },
          }),
        },
      },
      APPROVED_WITH_LINE_ITEMS,
    );

    await user.click(await screen.findByRole('radio', { name: 'Print & outdoor' }));
    await user.click(screen.getByRole('button', { name: 'Copy activity' }));

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1001/activities/A-1043'),
    );
    expect(api.bodiesOf('POST /api/v1/activities/A-1042/copy')).toEqual([
      { initiativeId: 'I-1002' },
    ]);
    expect(await findMessages()).toHaveTextContent("Activity 'Launch film (copy)' created");
    expect(screen.getByText('Closed')).toBeInTheDocument();
  });

  it('asks for an initiative when its own can’t take the copy', async () => {
    const { user, api } = renderDialog({
      'GET /api/v1/activities/A-1042/destinations': {
        body: destinations([
          { ...OWN_INITIATIVE, copyProblem: RETAIL_EVENTS.copyProblem },
          RETAIL_EVENTS,
        ]),
      },
    });

    await user.click(await screen.findByRole('button', { name: 'Copy activity' }));

    expect(await screen.findByText('Choose another initiative of this plan.')).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/activities/A-1042/copy')).toEqual([]);
  });

  it('shows the API’s refusal in the dialog', async () => {
    const { user } = renderDialog({
      'GET /api/v1/activities/A-1042/destinations': { body: destinations() },
      'POST /api/v1/activities/A-1042/copy': {
        status: 404,
        body: {
          error: {
            code: 'not_found',
            message: 'This activity no longer exists — someone may have deleted it.',
            fields: [],
          },
        },
      },
    });

    await screen.findByRole('group', { name: 'Copy to' });
    await user.click(screen.getByRole('button', { name: 'Copy activity' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'This activity no longer exists — someone may have deleted it.',
    );
  });
});
