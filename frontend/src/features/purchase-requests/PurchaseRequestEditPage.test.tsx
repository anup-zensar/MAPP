import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { components } from '@/api/generated/schema';
import { expectNoAxeViolations } from '@/testing/axe';
import { formAnswers } from '@/testing/fixtures/purchase-request-forms';
import {
  configuration,
  lineItemOption,
  no,
  purchaseRequest,
  routedPurchaseRequest,
} from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { apiError, pending, reply } from '@/testing/replies';
import type { Answer } from '@/testing/replies';
import { PurchaseRequestEditPage } from './PurchaseRequestEditPage';

type PurchaseRequest = components['schemas']['PurchaseRequest'];

const GET = 'GET /api/v1/purchase-requests/{purchaseRequestId}';
const PUT = 'PUT /api/v1/purchase-requests/{purchaseRequestId}';
const SAVED = 'PUT /api/v1/purchase-requests/PR-1002';

function renderEdit(answer: Answer<typeof GET>, extra: Parameters<typeof mockApi>[0] = []) {
  const api = mockApi([...formAnswers(), reply(GET, answer), ...extra]);
  return {
    api,
    ...renderWithProviders(<PurchaseRequestEditPage />, {
      route: '/purchase-requests/PR-1002/edit',
      path: '/purchase-requests/:purchaseRequestId/edit',
      user: testUser(),
    }),
  };
}

function field(id: string): HTMLElement {
  const element = document.getElementById(id);
  if (!element) {
    throw new Error(`No field ${id}`);
  }
  return element;
}

describe('032 FR-016/FR-053/FR-054: edit', () => {
  it('opens with the stored values and saves them with the version read', async () => {
    const { api, router, user } = renderEdit({ body: purchaseRequest() }, [
      reply(PUT, { body: purchaseRequest() }),
    ]);
    expect(await screen.findByLabelText(/^Name/)).toHaveValue('PR for Stand build');
    await expectNoAxeViolations();

    await user.type(field('name'), ' (stand)');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/purchase-requests/PR-1002'));
    expect(api.bodiesOf(SAVED)).toEqual([
      expect.objectContaining({
        version: 3,
        name: 'PR for Stand build (stand)',
        moaApproverId: 'U-1003',
        then: 'save',
        lines: [expect.objectContaining({ id: 'PRL-1002', amount: '9200.00' })],
      }),
    ]);
  });

  it('shows two problems at once and saves nothing', async () => {
    const { api, user } = renderEdit({ body: purchaseRequest() });
    await user.clear(await screen.findByLabelText(/^Name/));
    await user.clear(field('lines[0].amount'));

    await user.click(screen.getByRole('button', { name: 'Save' }));

    const summary = await screen.findByText('2 problems need your attention');
    expect(summary.closest('[role="alert"]')).toHaveTextContent(
      'Enter a name for the purchase request.',
    );
    expect(field('lines[0].amount')).toHaveAccessibleDescription(
      /Enter an amount above 0.00, with at most two decimals./,
    );
    expect(api.bodiesOf(SAVED)).toEqual([]);
  });

  it('offers Reload after someone else saved first, keeping the entries', async () => {
    const conflict = apiError('version_conflict', 'Someone else changed it.', {
      conflict: { changedBy: 'Daniel Okoro', changedAt: '2026-10-27T09:45:00Z' },
    });
    let reads = 0;
    const { user } = renderEdit(() => {
      reads += 1;
      return { body: purchaseRequest({ version: 3 + reads - 1 }) };
    }, [reply(PUT, { status: 409, body: conflict })]);
    await user.type(await screen.findByLabelText(/^Name/), ' (mine)');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    const failure = await screen.findByText(
      /Not saved — Daniel Okoro changed this purchase request/,
    );
    expect(failure.closest('[role="alert"]')).toHaveTextContent('Your entries are still here.');
    await waitFor(() => expect(reads).toBe(2));
    await user.click(screen.getByRole('button', { name: 'Reload' }));

    await waitFor(() => expect(reads).toBe(3));
    expect(field('name')).toHaveValue('PR for Stand build (mine)');
  });

  it('lists every gate problem of a refused re-submission at once', async () => {
    const notReady = apiError('not_ready', 'Not ready.', {
      missing: [
        {
          record: { type: 'activity', id: 'A-1003', name: 'Billboard winter wave' },
          fields: [
            {
              field: '',
              label: 'Status',
              reason:
                "Activity 'Billboard winter wave' must be approved before this purchase request can be submitted.",
            },
          ],
        },
        {
          record: { type: 'purchaseRequest', id: 'PR-1002', name: 'PR for Stand build' },
          fields: [{ field: 'moaApproverId', label: 'MoA approver' }],
        },
      ],
    });
    const { user } = renderEdit({ body: purchaseRequest({ status: 'submitted' }) }, [
      reply(PUT, { status: 409, body: notReady }),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Save and re-submit' }));

    const failure = await screen.findByText('Not saved — these things are needed first:');
    const box = failure.closest('[role="alert"]');
    expect(box).toHaveTextContent(
      'must be approved before this purchase request can be submitted.',
    );
    expect(box).toHaveTextContent('PR for Stand build: MoA approver');
  });

  it('asks before leaving with unsaved changes', async () => {
    const { router, user } = renderEdit({ body: purchaseRequest() });
    await user.type(await screen.findByLabelText(/^Name/), ' 2');

    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    const dialog = await screen.findByRole('alertdialog', { name: 'Discard changes?' });
    await user.click(within(dialog).getByRole('button', { name: 'Discard changes' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/purchase-requests/PR-1002'));
  });

  it('focuses the field a completeness link names (?field=)', async () => {
    mockApi([...formAnswers(), reply(GET, { body: purchaseRequest() })]);
    renderWithProviders(<PurchaseRequestEditPage />, {
      route: '/purchase-requests/PR-1002/edit?field=lines[0].overallLimit',
      path: '/purchase-requests/:purchaseRequestId/edit',
      user: testUser(),
    });

    await waitFor(() => expect(field('lines[0].overallLimit')).toHaveFocus());
  });

  it('shows a placeholder while the purchase request loads', async () => {
    renderEdit(() => pending());

    expect(screen.getByRole('heading', { level: 1, name: 'Edit purchase request' }));
    await expectNoAxeViolations();
  });

  it('shows a failure with Try again', async () => {
    renderEdit({ status: 500, body: apiError('internal_error', 'Something went wrong.') });

    expect(await screen.findByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });

  it('says so when the purchase request is not found (404)', async () => {
    renderEdit({ status: 404, body: apiError('not_found', 'Purchase request not found.') });

    expect(await screen.findByText('Purchase request not found.')).toBeInTheDocument();
  });

  it('explains why the purchase request cannot be edited', async () => {
    renderEdit({
      body: purchaseRequest({
        access: {
          ...purchaseRequest().access,
          edit: no("Raising and changing purchase requests isn't part of your role."),
        },
      }),
    });

    expect(
      await screen.findByText("Raising and changing purchase requests isn't part of your role."),
    ).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Save' })).not.toBeInTheDocument();
  });

  it('keeps a Closed purchase request read-only', async () => {
    renderEdit({ body: purchaseRequest({ status: 'closed' }) });

    expect(
      await screen.findByText('Closed purchase requests cannot be changed.'),
    ).toBeInTheDocument();
  });
});

describe('032 FR-023/FR-036/FR-039: save label by status', () => {
  async function saveWith(
    pr: PurchaseRequest,
    label: string,
    extra: Parameters<typeof mockApi>[0] = [],
  ) {
    const rendered = renderEdit({ body: pr }, [reply(PUT, { body: pr }), ...extra]);
    await rendered.user.click(await screen.findByRole('button', { name: label }));
    await waitFor(() => expect(rendered.api.bodiesOf(SAVED)).toHaveLength(1));
    return rendered.api.bodiesOf(SAVED)[0];
  }

  it.each([
    ['created', 'Save', 'save'],
    ['declined', 'Save', 'save'],
    ['submitted', 'Save and re-submit', 'resubmit'],
    ['inReview', 'Save and re-submit', 'resubmit'],
  ] as const)('a %s purchase request: %s', async (status, label, then) => {
    expect(await saveWith(purchaseRequest({ status }), label)).toMatchObject({ then });
  });

  it('an approved purchase request: Submit changes', async () => {
    expect(await saveWith(routedPurchaseRequest(), 'Submit changes')).toMatchObject({
      then: 'submitChanges',
    });
  });

  it('a change in review: Save and re-submit', async () => {
    const pr = routedPurchaseRequest({ status: 'inReview', changePending: true });
    expect(await saveWith(pr, 'Save and re-submit')).toMatchObject({ then: 'resubmit' });
  });

  it('only lines added while lines need no approval: Save, and the PR stays approved', async () => {
    const { api, user } = renderEdit({ body: routedPurchaseRequest() }, [
      reply('GET /api/v1/purchase-request-configuration', {
        body: configuration({ linesOnApprovedPurchaseRequests: 'withoutApproval' }),
      }),
      reply('GET /api/v1/purchase-requests/line-item-options', {
        body: { items: [lineItemOption()], total: 1 },
      }),
      reply(PUT, { body: routedPurchaseRequest() }),
    ]);
    expect(await screen.findByRole('button', { name: 'Submit changes' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Add line' }));
    await user.click(await screen.findByRole('button', { name: 'Add line Gift boxes (L-1011)' }));
    await user.type(field('lines[1].deliveryDate'), '2026-12-15');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(api.bodiesOf(SAVED)).toHaveLength(1));
    expect(api.bodiesOf(SAVED)[0]).toMatchObject({
      then: 'save',
      lines: [{ id: 'PRL-1002' }, { lineItemId: 'L-1011', amount: '10000.00', currency: 'USD' }],
    });
  });

  it('Add line is unavailable with the reason when lines may not be added', async () => {
    const reason =
      'Your organisation does not allow lines to be added to approved purchase requests.';
    renderEdit({
      body: routedPurchaseRequest({
        access: { ...routedPurchaseRequest().access, addLine: no(reason) },
      }),
    });

    const add = await screen.findByRole('button', { name: 'Add line' });
    expect(add).toBeDisabled();
    expect(add).toHaveAccessibleDescription(reason);
  });
});
