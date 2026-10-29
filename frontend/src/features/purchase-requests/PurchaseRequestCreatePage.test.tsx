import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import { formAnswers } from '@/testing/fixtures/purchase-request-forms';
import { defaults, lineItemOption, purchaseRequest } from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { apiError, pending, reply } from '@/testing/replies';
import type { Answer } from '@/testing/replies';
import { PurchaseRequestCreatePage } from './PurchaseRequestCreatePage';

const DEFAULTS = 'GET /api/v1/purchase-requests/new-defaults';
const CREATE = 'POST /api/v1/purchase-requests';

function renderCreate(
  defaultsAnswer: Answer<typeof DEFAULTS>,
  extra: Parameters<typeof mockApi>[0] = [],
  route = '/purchase-requests/new?lineItemId=L-1006',
) {
  const api = mockApi([...formAnswers(), reply(DEFAULTS, defaultsAnswer), ...extra]);
  return {
    api,
    ...renderWithProviders(<PurchaseRequestCreatePage />, {
      route,
      path: '/purchase-requests/new',
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

async function chooseWorkflows(user: ReturnType<typeof renderCreate>['user']) {
  await user.selectOptions(field('approvalWorkflowId'), 'WF-1101');
  await user.selectOptions(field('modificationWorkflowId'), 'WF-1102');
  await user.type(field('lines[0].deliveryDate'), '2026-11-30');
}

describe('032 FR-003/FR-053/FR-054: new purchase request', () => {
  it('opens pre-filled from the line item: name, owner, the line with its HX key', async () => {
    renderCreate({ body: defaults() });

    expect(await screen.findByLabelText(/^Name/)).toHaveValue('PR for Stand build');
    expect(field('lines[0].amount')).toHaveValue('9,200.00');
    expect(field('lines[0].overallLimit')).toHaveValue('9,200.00');
    expect(screen.getByText('HX1006')).toBeInTheDocument();
    expect(screen.getByText('Generated from the line-item ID')).toBeInTheDocument();
    expect(screen.getByText('SIM-GB01-0001')).toBeInTheDocument();
    expect(await screen.findByRole('button', { name: 'Save' })).toBeInTheDocument();
    await expectNoAxeViolations();
  });

  it('shows a placeholder while the defaults load', async () => {
    renderCreate(() => pending());

    expect(screen.getByRole('heading', { level: 1, name: 'New purchase request' }));
    expect(screen.getAllByRole('status').length).toBeGreaterThan(0);
    await expectNoAxeViolations();
  });

  it('lists the missing fields at once before asking the API', async () => {
    const { api, user } = renderCreate({ body: defaults() });

    await user.clear(await screen.findByLabelText(/^Name/));
    await user.click(screen.getByRole('button', { name: 'Save' }));

    const summary = await screen.findByRole('alert');
    expect(summary).toHaveTextContent('4 problems need your attention');
    expect(summary).toHaveTextContent('Enter a name for the purchase request.');
    expect(summary).toHaveTextContent('Choose the approval workflow.');
    expect(summary).toHaveTextContent('Choose the approval workflow for modification.');
    expect(summary).toHaveTextContent('Enter the delivery date.');
    expect(api.bodiesOf('POST /api/v1/purchase-requests')).toEqual([]);
  });

  it('shows every problem of the API at once, in the summary and at each field', async () => {
    const problems = apiError('validation_failed', 'Some fields need attention.', {
      fields: [
        { field: 'name', message: 'A purchase request with this name exists.' },
        {
          field: 'lines[0].amount',
          message: 'Only 3,200.00 EUR of line item L-1006 is not yet requested.',
        },
      ],
    });
    const { user } = renderCreate({ body: defaults() }, [
      reply(CREATE, { status: 400, body: problems }),
    ]);
    await screen.findByLabelText(/^Name/);
    await chooseWorkflows(user);

    await user.click(screen.getByRole('button', { name: 'Save' }));

    const summary = await screen.findByText('2 problems need your attention');
    const box = summary.closest('[role="alert"]');
    expect(box).toHaveTextContent('Only 3,200.00 EUR of line item L-1006 is not yet requested.');
    expect(field('lines[0].amount')).toHaveAccessibleDescription(
      /Only 3,200.00 EUR of line item L-1006 is not yet requested./,
    );
    expect(field('name')).toHaveAccessibleDescription(/A purchase request with this name exists./);
  });

  it('032 FR-053: a refused Shell attribute of the purchase request shows in the summary and at its field', async () => {
    const problems = apiError('validation_failed', 'Some fields need attention.', {
      fields: [{ field: 'attributes.SA-1101', message: 'Enter a vendor e-mail address.' }],
    });
    const { user } = renderCreate({ body: defaults() }, [
      reply(CREATE, { status: 400, body: problems }),
    ]);
    await screen.findByLabelText(/^Name/);
    await chooseWorkflows(user);

    await user.click(screen.getByRole('button', { name: 'Save' }));

    const summary = await screen.findByText('1 problem needs your attention');
    expect(summary.closest('[role="alert"]')).toHaveTextContent('Enter a vendor e-mail address.');
    expect(field('attributes.SA-1101')).toHaveAccessibleDescription(
      /Enter a vendor e-mail address./,
    );
  });

  it('saves the purchase request and opens it', async () => {
    const { api, router, user } = renderCreate({ body: defaults() }, [
      reply(CREATE, { status: 201, body: purchaseRequest() }),
    ]);
    await screen.findByLabelText(/^Name/);
    await chooseWorkflows(user);

    await user.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/purchase-requests/PR-1002'));
    expect(api.bodiesOf('POST /api/v1/purchase-requests')).toEqual([
      {
        name: 'PR for Stand build',
        description: '',
        ownerId: 'U-1002',
        moaApproverId: null,
        approvalWorkflowId: 'WF-1101',
        modificationWorkflowId: 'WF-1102',
        attributes: {},
        lines: [
          {
            lineItemId: 'L-1006',
            currency: 'EUR',
            amount: '9200.00',
            deliveryDate: '2026-11-30',
            overallLimit: '9200.00',
            description: '',
            attributes: {},
          },
        ],
        then: 'save',
      },
    ]);
  });

  it('032 FR-054: leaving with unsaved changes asks Keep editing or Discard changes (design system §2.5)', async () => {
    const { user, router } = renderCreate({ body: defaults() });
    await user.type(await screen.findByLabelText(/^Name/), ' 2');

    await user.click(screen.getByRole('button', { name: 'Cancel' }));

    const dialog = await screen.findByRole('alertdialog', { name: 'Discard changes?' });
    const answers = within(dialog).getAllByRole('button');
    expect(answers.map((answer) => answer.textContent)).toEqual([
      'Keep editing',
      'Discard changes',
    ]);
    expect(answers[0]).toHaveFocus();
    await user.click(answers[0]);
    expect(router.state.location.pathname).toBe('/purchase-requests/new');
    expect(field('name')).toHaveValue('PR for Stand build 2');

    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    const again = await screen.findByRole('alertdialog', { name: 'Discard changes?' });
    await user.click(within(again).getByRole('button', { name: 'Discard changes' }));
    await waitFor(() => expect(router.state.location.pathname).toBe('/line-items/L-1006'));
  });

  it('shows no access when the person may not raise purchase requests in the plan (403)', async () => {
    renderCreate(
      {
        status: 403,
        body: apiError(
          'forbidden',
          "Raising and changing purchase requests isn't part of your role.",
        ),
      },
      [reply('GET /api/v1/me', { body: testUser() })],
    );

    expect(await screen.findByRole('heading', { name: 'No access' })).toBeInTheDocument();
    expect(
      screen.getByText("Raising and changing purchase requests isn't part of your role."),
    ).toBeInTheDocument();
  });

  it('says so when the line item is not found (404)', async () => {
    renderCreate({ status: 404, body: apiError('not_found', 'Line item not found.') });

    expect(await screen.findByText('Line item not found.')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to Plans' })).toBeInTheDocument();
  });

  it('032 FR-001: without a line item, choosing one comes first', async () => {
    const { router, user } = renderCreate(
      { body: defaults() },
      [
        reply('GET /api/v1/purchase-requests/line-item-options', {
          body: { items: [lineItemOption()], total: 1 },
        }),
      ],
      '/purchase-requests/new',
    );

    expect(await screen.findByRole('heading', { name: 'Choose a line item' })).toBeInTheDocument();
    await expectNoAxeViolations();
    await user.click(await screen.findByRole('button', { name: 'Choose Gift boxes (L-1011)' }));

    await waitFor(() => expect(router.state.location.search).toBe('?lineItemId=L-1011'));
  });
});
