import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { expectNoAxeViolations } from '@/testing/axe';
import { testUser } from '@/testing/fixtures/users';
import { ALEX_RUN, DEMO_RUN, EMPTY_RUN, runsPage } from '@/testing/fixtures/swbs-runs';
import { apiError, pending, reply } from '@/testing/replies';
import { SimulatedGsapPanel } from './SimulatedGsapPanel';

const RUNS = 'GET /api/v1/simulated-gsap/swbs-runs';
const SEND = 'POST /api/v1/simulated-gsap/swbs-runs';

function renderPanel(answers: Parameters<typeof mockApi>[0]) {
  const api = mockApi(answers);
  return { api, ...renderWithProviders(<SimulatedGsapPanel />, { user: testUser() }) };
}

describe('030 FR-013: the Simulated GSAP panel', () => {
  it('shows a placeholder while the runs load', async () => {
    renderPanel([reply(RUNS, () => pending())]);

    const panel = screen.getByRole('region', { name: 'Simulated GSAP' });
    expect(within(panel).queryByRole('button')).not.toBeInTheDocument();
    await expectNoAxeViolations();
  });

  it('shows a failure with Try again', async () => {
    renderPanel([
      reply(RUNS, { status: 500, body: apiError('internal_error', 'Something went wrong.') }),
    ]);

    expect(await screen.findByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });

  it('says in place why a person without the permission sees nothing', async () => {
    renderPanel([
      reply(RUNS, {
        status: 403,
        body: apiError('forbidden', "Importing GSAP extracts isn't part of your role."),
      }),
      reply('GET /api/v1/me', { body: testUser() }),
    ]);

    expect(
      await screen.findByText(
        'Running the simulated GSAP needs the permission to import GSAP extracts.',
      ),
    ).toBeInTheDocument();
  });

  it('says when nothing has run yet', async () => {
    renderPanel([reply(RUNS, { body: runsPage({ items: [], total: 0, readyToSend: 0 }) })]);

    expect(await screen.findByText('No runs yet.')).toBeInTheDocument();
  });

  it('lists the runs newest first with source, who, when and the codes', async () => {
    renderPanel([
      reply(RUNS, { body: runsPage({ items: [ALEX_RUN, DEMO_RUN], total: 7, readyToSend: 2 }) }),
    ]);

    expect(await screen.findByText('2 line items are ready to send')).toBeInTheDocument();
    const list = screen.getByRole('list', { name: 'Runs, newest first' });
    const [first, second] = Array.from(list.children);
    expect(first).toHaveTextContent('Simulated GSAP · SWR-1002');
    expect(first).toHaveTextContent('by Alex Morgan');
    expect(first).toHaveTextContent('Stand build (L-1006) · SIM-GB01-0001');
    expect(second).toHaveTextContent('Demo data (simulated GSAP) · SWR-1001');
    expect(screen.getByText('Showing the 2 latest of 7 runs')).toBeInTheDocument();
    await expectNoAxeViolations();
  });

  it('030 FR-009: shows a run that found nothing to send', async () => {
    renderPanel([reply(RUNS, { body: runsPage({ items: [EMPTY_RUN], readyToSend: 0 }) })]);

    expect(await screen.findByText('Nothing was ready to send.')).toBeInTheDocument();
  });

  it('offers no send when nothing is ready, and says why', async () => {
    renderPanel([reply(RUNS, { body: runsPage({ readyToSend: 0 }) })]);

    const send = await screen.findByRole('button', {
      name: 'Send SWBS requests to simulated GSAP',
    });
    expect(send).toBeDisabled();
    expect(send).toHaveAccessibleDescription('No line item is waiting for an SWBS code');
  });

  it('sends after confirmation and shows what came back', async () => {
    const { api, user } = renderPanel([
      reply(RUNS, { body: runsPage() }),
      reply(SEND, { status: 201, body: ALEX_RUN }),
    ]);

    await user.click(
      await screen.findByRole('button', { name: 'Send SWBS requests to simulated GSAP' }),
    );
    const dialog = await screen.findByRole('alertdialog', {
      name: 'Send 1 SWBS request to simulated GSAP?',
    });
    expect(dialog).toHaveTextContent(
      "The simulated GSAP returns one SWBS code per line item at once. Codes can't be changed or removed in SP&A One.",
    );
    await expectNoAxeViolations();
    await user.click(
      within(dialog).getByRole('button', { name: 'Send SWBS requests to simulated GSAP' }),
    );

    const result = await screen.findByRole('status');
    expect(result).toHaveTextContent('Simulated GSAP returned 2 SWBS codes');
    expect(result).toHaveTextContent('Media booking (L-1005) · SIM-NL02-0001');
    expect(api.bodiesOf(SEND)).toEqual([{}]);
  });

  it('shows the refusal when the run is refused', async () => {
    const { user } = renderPanel([
      reply(RUNS, { body: runsPage() }),
      reply(SEND, {
        status: 409,
        body: apiError('not_allowed', 'Not sent — these line items are not ready.'),
      }),
    ]);

    await user.click(
      await screen.findByRole('button', { name: 'Send SWBS requests to simulated GSAP' }),
    );
    const dialog = await screen.findByRole('alertdialog');
    await user.click(
      within(dialog).getByRole('button', { name: 'Send SWBS requests to simulated GSAP' }),
    );

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Not sent — these line items are not ready.',
    );
  });
});
