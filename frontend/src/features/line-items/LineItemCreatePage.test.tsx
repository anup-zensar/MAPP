import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { nth } from '@/testing/dom';
import { attribute, option } from '@/testing/fixtures/attribute-forms';
import { activity, lineItem, usd } from '@/testing/fixtures/line-items';
import { formOptions, lineItemForm, notFound } from '@/testing/fixtures/line-item-forms';
import {
  CREATE,
  findNameField,
  formButton,
  renderCreatePage,
} from '@/testing/fixtures/line-item-pages';
import { reply } from '@/testing/replies';

describe('LineItemCreatePage (US2)', () => {
  it('opens filled in with the defaults, each labelled with its source (scenario 1, FR-008)', async () => {
    renderCreatePage();

    expect(await findNameField()).toHaveValue('Launch film – line item');
    expect(document.title).toBe('New line item · SP&A One');
    expect(screen.getByRole('heading', { level: 1, name: 'New line item' })).toBeInTheDocument();
    expect(
      within(screen.getByRole('navigation', { name: 'Breadcrumb' })).getByRole('link', {
        name: 'Launch film',
      }),
    ).toBeInTheDocument();
    expect(screen.getByText("From the activity's name")).toBeInTheDocument();
    expect(screen.getByLabelText('Currency')).toHaveValue('USD');
    expect(screen.getByLabelText('Planned cost')).toHaveValue('20,000.00');
    expect(screen.getByText("The activity's unassigned planned cost")).toBeInTheDocument();
    expect(screen.getByLabelText('Evenly')).toBeChecked();
    expect(screen.getByText("The plan's default")).toBeInTheDocument();
    expect(screen.getByLabelText('Finance budget year')).toHaveValue('2026');
    expect(screen.getByText("The plan's year")).toBeInTheDocument();
    expect(
      screen.getByText(
        "Items of '2026 Lubricants Budget', the plan's default finance budget, are listed first.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('Request SWBS code from GSAP')).toBeChecked();
    expect(screen.getByText(/On by default/)).toBeInTheDocument();
    expect(
      screen.getByText(
        'Filled by the simulated GSAP once the activity is approved and the request is sent',
      ),
    ).toBeInTheDocument();

    const summary = screen.getByLabelText('Activity summary');
    expect(summary).toHaveTextContent('Activity planned cost120,000.00 USD');
    expect(summary).toHaveTextContent('Unassigned now20,000.00 USD');
    expect(summary).toHaveTextContent('After saving0.00 USD');

    const breadcrumb = screen.getByLabelText('Breadcrumb');
    expect(within(breadcrumb).getByRole('link', { name: 'Launch film' })).toHaveAttribute(
      'href',
      '/plans/P-1001/activities/A-1001',
    );
    expect(within(breadcrumb).getByText('New line item')).toHaveAttribute('aria-current', 'page');
  });

  it('shows the sections in order, each a card, without a finance section (FR-007, 015 FR-017)', async () => {
    renderCreatePage();
    await findNameField();

    const headings = [...document.querySelectorAll('form h2')].map((h) => h.textContent);
    expect(headings).toEqual([
      'Basics',
      'Cost and currency',
      'Spread over time',
      'Finance coding',
      'Link to finance budget',
      'GSAP request',
    ]);
    expect(screen.queryByText('From finance (coming later)')).not.toBeInTheDocument();
    expect(screen.queryByText('Total actuals')).not.toBeInTheDocument();
    expect(screen.getByText(/of \d+ needed fields? filled$/)).toBeInTheDocument();
  });

  it('makes the GL code depend on the spend type (scenarios 3 and 4, FR-014)', async () => {
    const { user, api } = renderCreatePage([
      reply(
        'POST /api/v1/activities/{activityId}/line-items',
        { status: 201, body: lineItem() },
        { activityId: 'A-1001' },
      ),
    ]);
    await findNameField();

    const glCode = screen.getByLabelText('GL code');
    expect(glCode).toBeDisabled();
    expect(screen.getByText('Choose a spend type first.')).toBeInTheDocument();

    await user.selectOptions(screen.getByLabelText('Spend type'), 'ATL – Media');
    await user.selectOptions(glCode, '7250080 – Advertising digital');
    await user.selectOptions(screen.getByLabelText('Spend type'), 'BTL – Event');
    expect(
      screen.getByText(
        "GL code '7250080 – Advertising digital' will be cleared — it doesn't fit 'BTL – Event'.",
      ),
    ).toBeInTheDocument();

    await user.click(formButton('Save'));
    await waitFor(() => expect(api.bodiesOf(CREATE)).toHaveLength(1));
    expect(api.bodiesOf(CREATE)[0]).toMatchObject({
      attributes: { 'SA-ST': 'SO-ST-BTL', 'SA-GL': null },
    });
  });

  it('fills in the only GL code of a spend type (scenario 3)', async () => {
    const { user } = renderCreatePage();
    await findNameField();

    await user.selectOptions(screen.getByLabelText('Spend type'), 'BTL – Event');
    expect(screen.getByLabelText('GL code')).toHaveValue('SO-GL-EVT');
  });

  /** The line-item form as Shell's lists are seeded: GL code depends on no other attribute (D-325). */
  function shellListsForm() {
    const form = lineItemForm();
    nth(form.sections, 0).attributes = [
      attribute({
        id: 'SA-ST',
        label: 'Spend type',
        marking: 'neededToSubmit',
        options: [option('SO-ST-1', 'ATL - Digital'), option('SO-ST-2', 'BTL - Event')],
      }),
      attribute({
        id: 'SA-GL',
        label: 'GL code',
        marking: 'neededToSubmit',
        options: [
          option('SO-GL-1', '7250080 - Advertising - Digital'),
          option('SO-GL-2', '7250150 - Sponsorship programmes'),
        ],
      }),
    ];
    return { body: form };
  }

  it('034 FR-011: offers every GL code whatever the spend type, and never clears it', async () => {
    const { user, api } = renderCreatePage([
      reply('GET /api/v1/shell-attribute-forms/{level}', shellListsForm(), { level: 'lineItem' }),
      reply(
        'POST /api/v1/activities/{activityId}/line-items',
        { status: 201, body: lineItem() },
        { activityId: 'A-1001' },
      ),
    ]);
    await findNameField();

    const glCode = screen.getByLabelText('GL code');
    expect(glCode).toBeEnabled();
    await user.selectOptions(glCode, '7250150 - Sponsorship programmes');
    await user.selectOptions(screen.getByLabelText('Spend type'), 'ATL - Digital');
    expect(screen.queryByText(/will be cleared/)).not.toBeInTheDocument();

    await user.click(formButton('Save'));
    await waitFor(() => expect(api.bodiesOf(CREATE)).toHaveLength(1));
    expect(api.bodiesOf(CREATE)[0]).toMatchObject({
      attributes: { 'SA-ST': 'SO-ST-1', 'SA-GL': 'SO-GL-2' },
    });
  });

  it('says when the activity was deleted (edge case)', async () => {
    const message =
      'This activity was deleted — restore it from the recycle bin to keep working on its line items.';
    renderCreatePage([
      reply('GET /api/v1/activities/{activityId}/line-items/form-options', notFound(message), {
        activityId: 'A-1001',
      }),
    ]);

    expect(await screen.findByText(message)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to the recycle bin' })).toHaveAttribute(
      'href',
      '/recycle-bin',
    );
  });

  it('shows a load failure with Try again', async () => {
    renderCreatePage([
      reply(
        'GET /api/v1/activities/{activityId}/line-items/form-options',
        {
          status: 500,
          body: {
            error: {
              code: 'internal_error',
              message: 'Something went wrong. Please try again.',
              fields: [],
            },
          },
        },
        { activityId: 'A-1001' },
      ),
    ]);

    expect(await screen.findByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });

  it('starts with an empty planned cost when the activity has none (D-144)', async () => {
    const options = formOptions({
      activity: activity({ plannedCost: null }),
      balance: {
        plannedCost: null,
        assigned: usd('0.00'),
        balance: null,
        state: 'noPlannedCost',
        roundingHint: false,
      },
    });
    options.defaults = {
      ...options.defaults,
      plannedCost: null,
      plannedCostSource: 'noActivityPlannedCost',
    };
    renderCreatePage([
      reply(
        'GET /api/v1/activities/{activityId}/line-items/form-options',
        { body: options },
        { activityId: 'A-1001' },
      ),
    ]);

    await findNameField();
    expect(
      screen.getByText('This activity has no planned cost yet, so there is nothing to pre-fill'),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        'This activity has no planned cost yet — the unassigned planned cost appears once it has one.',
      ),
    ).toBeInTheDocument();
  });

  /** The line-item form with an attribute shown only for the Mobility business unit. */
  function mobilityOnlyForm() {
    const form = lineItemForm();
    nth(form.sections, 0).attributes.push(
      attribute({
        id: 'SA-MOB',
        label: 'Mobility channel',
        conditionOptionIds: ['SO-BU-MOB'],
        options: [option('SO-MOB-1', 'Fleet')],
      }),
    );
    return { body: form };
  }

  it('hides a line-item attribute of another business unit than the plan has (002 FR-023, D-212)', async () => {
    renderCreatePage([
      reply('GET /api/v1/shell-attribute-forms/{level}', mobilityOnlyForm(), { level: 'lineItem' }),
    ]);

    await findNameField();
    expect(screen.queryByLabelText('Mobility channel')).not.toBeInTheDocument();
  });

  it("shows it from the activity's Business unit options alone, without asking for the plan (D-212)", async () => {
    const { api } = renderCreatePage([
      reply('GET /api/v1/shell-attribute-forms/{level}', mobilityOnlyForm(), { level: 'lineItem' }),
      reply(
        'GET /api/v1/activities/{activityId}/line-items/form-options',
        {
          body: formOptions({ activity: activity({ businessUnitOptionIds: ['SO-BU-MOB'] }) }),
        },
        { activityId: 'A-1001' },
      ),
    ]);

    await findNameField();
    expect(screen.getByLabelText('Mobility channel')).toBeInTheDocument();
    expect(api.requests.some((request) => request.path.startsWith('/api/v1/plans'))).toBe(false);
  });
});
