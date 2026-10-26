import { screen, waitFor, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/users';
import { MoveActivityDialog } from './MoveActivityDialog';
import {
  destination,
  destinations,
  launchFilm,
  OWN_INITIATIVE,
  RETAIL_EVENTS,
  validationFailed,
} from './testData';

const DATES_DONT_FIT = RETAIL_EVENTS.moveProblem ?? '';

/** The dialog as the More menu holds it: closing it takes it off the page. */
function Harness() {
  const [open, setOpen] = useState(true);
  return open ? (
    <MoveActivityDialog activity={launchFilm()} onClose={() => setOpen(false)} />
  ) : (
    <p>Closed</p>
  );
}

function renderDialog(answers: Record<string, MockHandler>) {
  const api = mockApi(answers);
  const rendered = renderWithProviders(<Harness />, {
    user: testUser(),
    route: '/plans/P-1001/activities/A-1042',
  });
  return { ...rendered, api };
}

describe('MoveActivityDialog (feature 006, US5#1, FR-029)', () => {
  it('offers the plan’s initiatives grouped by campaign, the unavailable ones with their reason', async () => {
    renderDialog({ 'GET /api/v1/activities/A-1042/destinations': { body: destinations() } });

    const dialog = screen.getByRole('dialog', { name: "Move activity 'Launch film'" });
    expect(dialog).toHaveAccessibleDescription(
      "Choose another initiative in 'FY2026 Lubricants Europe'. The activity keeps its status, line items and values.",
    );
    const helix = await within(dialog).findByRole('group', { name: 'Helix launch' });
    const rimula = within(dialog).getByRole('group', { name: 'Rimula fleet' });

    const own = within(helix).getByRole('radio', { name: 'Digital & social (this initiative)' });
    expect(own).toBeDisabled();
    expect(own).toHaveAccessibleDescription(
      "01 Jan – 30 Jun 2026 It's already in this initiative.",
    );
    const print = within(helix).getByRole('radio', { name: 'Print & outdoor' });
    expect(print).toBeEnabled();
    expect(print).toHaveAccessibleDescription('01 Jan – 31 Dec 2026');
    const retail = within(rimula).getByRole('radio', { name: 'Retail events' });
    expect(retail).toBeDisabled();
    expect(retail).toHaveAccessibleDescription(`01 Jun – 31 Dec 2026 ${DATES_DONT_FIT}`);
    expect(within(dialog).getByRole('searchbox', { name: 'Search initiatives' })).toHaveFocus();
  });

  it('finds initiatives by their name or their campaign’s name', async () => {
    const { user } = renderDialog({
      'GET /api/v1/activities/A-1042/destinations': { body: destinations() },
    });

    const search = await screen.findByRole('searchbox', { name: 'Search initiatives' });
    await user.type(search, 'rimula');
    expect(screen.getByRole('radio', { name: 'Retail events' })).toBeInTheDocument();
    expect(screen.queryByRole('radio', { name: 'Print & outdoor' })).not.toBeInTheDocument();

    await user.clear(search);
    await user.type(search, 'print');
    expect(screen.getByRole('radio', { name: 'Print & outdoor' })).toBeInTheDocument();
    expect(screen.queryByRole('group', { name: 'Rimula fleet' })).not.toBeInTheDocument();

    await user.clear(search);
    await user.type(search, 'zzz');
    expect(screen.getByText("No initiative matches 'zzz'.")).toBeInTheDocument();
  });

  it('moves the activity with its version and says so (US5#1)', async () => {
    const { user, api, router } = renderDialog({
      'GET /api/v1/activities/A-1042/destinations': { body: destinations() },
      'POST /api/v1/activities/A-1042/move': {
        body: launchFilm({
          initiative: { type: 'initiative', id: 'I-1002', name: 'Print & outdoor' },
          version: 4,
        }),
      },
    });

    await user.click(await screen.findByRole('button', { name: 'Move activity' }));
    expect(await screen.findByText('Choose another initiative of this plan.')).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/activities/A-1042/move')).toEqual([]);

    await user.click(screen.getByRole('radio', { name: 'Print & outdoor' }));
    expect(screen.queryByText('Choose another initiative of this plan.')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Move activity' }));

    expect(await findMessages()).toHaveTextContent(
      "Activity 'Launch film' moved to 'Print & outdoor'",
    );
    expect(api.bodiesOf('POST /api/v1/activities/A-1042/move')).toEqual([
      { initiativeId: 'I-1002', version: 3 },
    ]);
    expect(await screen.findByText('Closed')).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/plans/P-1001/activities/A-1042');
  });

  it('shows the API’s refusal in the dialog and asks for the initiatives again (US5#1)', async () => {
    const refusal =
      "'Print & outdoor' already has an activity called 'Launch film'. Rename the activity first, or pick another initiative.";
    const { user, api } = renderDialog({
      'GET /api/v1/activities/A-1042/destinations': { body: destinations() },
      'POST /api/v1/activities/A-1042/move': validationFailed([
        { field: 'initiativeId', message: refusal },
      ]),
    });

    await user.click(await screen.findByRole('radio', { name: 'Print & outdoor' }));
    await user.click(screen.getByRole('button', { name: 'Move activity' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(refusal);
    await waitFor(() =>
      expect(
        api.requests.filter((request) => request.path === '/api/v1/activities/A-1042/destinations'),
      ).toHaveLength(2),
    );
    expect(screen.getByRole('dialog')).toBeInTheDocument();
  });

  it('says who changed the activity meanwhile (409)', async () => {
    const { user } = renderDialog({
      'GET /api/v1/activities/A-1042/destinations': { body: destinations() },
      'POST /api/v1/activities/A-1042/move': {
        status: 409,
        body: {
          error: {
            code: 'version_conflict',
            message: 'Not saved — Daniel Okoro changed this activity',
            fields: [],
            conflict: { changedBy: 'Daniel Okoro', changedAt: '2026-03-12T09:00:00Z' },
          },
        },
      },
    });

    await user.click(await screen.findByRole('radio', { name: 'Print & outdoor' }));
    await user.click(screen.getByRole('button', { name: 'Move activity' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      /^Not saved — Daniel Okoro changed this activity (at|on) /,
    );
  });

  it('says when there is no other initiative to move it to', async () => {
    renderDialog({
      'GET /api/v1/activities/A-1042/destinations': { body: destinations([OWN_INITIATIVE]) },
    });

    expect(
      await screen.findByText("There's no other initiative in this plan to move it to."),
    ).toBeInTheDocument();
    expect(screen.queryByRole('radio')).not.toBeInTheDocument();
  });

  it('shows grey lines while loading, then the error with Try again', async () => {
    let calls = 0;
    const { user } = renderDialog({
      'GET /api/v1/activities/A-1042/destinations': () =>
        ++calls === 1 ? { status: 500 } : { body: destinations([destination()]) },
    });

    expect(screen.getByRole('status')).toHaveTextContent('Loading…');
    await user.click(await screen.findByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('radio', { name: 'Print & outdoor' })).toBeInTheDocument();
  });

  it('says when the activity is gone, and Cancel closes', async () => {
    const { user } = renderDialog({
      'GET /api/v1/activities/A-1042/destinations': {
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

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'This activity no longer exists — someone may have deleted it.',
    );
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(await screen.findByText('Closed')).toBeInTheDocument();
  });
});
