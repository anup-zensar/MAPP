import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { HttpHandler } from 'msw';
import { campaignForm, LUBRICANTS, MOBILITY, planForm } from '@/testing/fixtures/attribute-forms';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { shellAttribute } from '@/testing/fixtures/shell-attributes';
import { reply } from '@/testing/replies';
import { AttributePreview } from './AttributePreview';

/** Region, which depends on Cluster on the campaign form (campaignForm()). */
const REGION = shellAttribute({
  id: 'SA-REGION',
  label: 'Region',
  level: 'campaign',
  type: 'multipleChoice',
  marking: 'neededToPublish',
  parentAttribute: { id: 'SA-CLUSTER', name: 'Cluster', active: true },
});

const FLEET_SIZE = shellAttribute({
  id: 'SA-FLEET',
  label: 'Fleet size',
  level: 'plan',
  type: 'number',
  condition: [{ id: MOBILITY, name: 'Mobility', active: true }],
});

const CHECK = 'POST /api/v1/shell-attribute-forms/campaign/check';

function renderPreview(answers: HttpHandler[], attribute = REGION) {
  const api = mockApi([
    reply(
      'GET /api/v1/shell-attribute-forms/{level}',
      { body: campaignForm() },
      { level: 'campaign' },
    ),
    reply('GET /api/v1/shell-attribute-forms/{level}', { body: planForm() }, { level: 'plan' }),
    ...answers,
  ]);
  return {
    ...renderWithProviders(<AttributePreview attribute={attribute} />, { user: testUser() }),
    api,
  };
}

/** The preview card of the campaign form, once its form has arrived. */
async function previewCard() {
  const card = await screen.findByRole('region', { name: 'Preview — Campaign form' });
  await within(card).findByRole('button', { name: 'Try saving' });
  return card;
}

function businessUnits() {
  return screen.getByRole('group', { name: /Preview as a plan whose business unit includes:/ });
}

describe('AttributePreview (FR-020, US3)', () => {
  it("shows the level's form as a plan with every business unit, and what it still needs", async () => {
    renderPreview([]);

    const card = await previewCard();
    expect(card).toHaveTextContent('Behaves like the real form. Nothing you type here is saved.');
    expect(await within(card).findByRole('group', { name: /^Cluster/ })).toBeInTheDocument();
    const units = businessUnits();
    expect(within(units).getByRole('checkbox', { name: 'Lubricants' })).toBeChecked();
    expect(within(units).getByRole('checkbox', { name: 'Mobility' })).toBeChecked();
    expect(card).toHaveTextContent("Shown because the plan's business unit includes Lubricants.");
    // Cluster, Region and Brands are needed to publish and still empty.
    expect(within(card).getByRole('button', { name: /3 fields needed/ })).toBeInTheDocument();
    expect(card).toHaveTextContent('Shows what saving a real record would say. Nothing is saved.');
  });

  it('offers only the regions of the chosen cluster and fills in the only one (US3-1)', async () => {
    const { user } = renderPreview([]);

    const card = await previewCard();
    const region = await within(card).findByRole('group', { name: /^Region/ });
    expect(region).toHaveTextContent('Choose a cluster first.');
    await user.click(within(card).getByRole('checkbox', { name: 'Retail' }));

    expect(
      await within(card).findByText(
        "Region set to 'RETAIL – CHINA', the only region for this cluster.",
      ),
    ).toBeInTheDocument();
    const regionNow = within(card).getByRole('group', { name: /^Region/ });
    expect(within(regionNow).getByRole('checkbox', { name: 'RETAIL – CHINA' })).toBeChecked();
    expect(
      within(regionNow).queryByRole('checkbox', { name: 'B2B – EUROPE' }),
    ).not.toBeInTheDocument();
    expect(within(card).getByRole('button', { name: /1 field needed/ })).toBeInTheDocument();
  });

  it("hides attributes the plan's business unit doesn't include, keeping their values (US3-3)", async () => {
    const { user } = renderPreview([]);

    const card = await previewCard();
    await user.click(await within(card).findByRole('checkbox', { name: 'Shell Helix' }));
    await user.click(within(businessUnits()).getByRole('checkbox', { name: 'Lubricants' }));

    expect(within(card).queryByRole('group', { name: /^Brands/ })).not.toBeInTheDocument();
    expect(card).not.toHaveTextContent("Shown because the plan's business unit includes");

    await user.click(within(businessUnits()).getByRole('checkbox', { name: 'Lubricants' }));
    expect(card).toHaveTextContent("Shown because the plan's business unit includes Lubricants.");
    expect(within(card).getByRole('checkbox', { name: 'Shell Helix' })).toBeChecked();
  });

  it('says a plan without a business unit shows more attributes once it is chosen', async () => {
    const { user } = renderPreview([]);

    await previewCard();
    await user.click(within(businessUnits()).getByRole('checkbox', { name: 'Lubricants' }));
    await user.click(within(businessUnits()).getByRole('checkbox', { name: 'Mobility' }));

    expect(
      screen.getByText("More Shell attributes appear once the plan's business unit is chosen."),
    ).toBeInTheDocument();
  });

  it('Try saving: says saving would work, what it would clear and what is still needed', async () => {
    const { user, api } = renderPreview([
      reply(
        'POST /api/v1/shell-attribute-forms/{level}/check',
        {
          body: {
            problems: [],
            missing: [{ field: 'attributes.SA-CLUSTER', label: 'Cluster' }],
            cleared: [
              {
                attributeId: 'SA-REGION',
                label: 'Region',
                optionIds: ['SO-CHINA'],
                optionLabels: ['RETAIL – CHINA'],
              },
            ],
            values: {},
          },
        },
        { level: 'campaign' },
      ),
    ]);

    const card = await previewCard();
    await user.click(await within(card).findByRole('checkbox', { name: 'Retail' }));
    await user.click(within(card).getByRole('button', { name: 'Try saving' }));

    expect(await within(card).findByText('Saving would work.')).toBeInTheDocument();
    expect(card).toHaveTextContent('These values would be cleared:');
    expect(within(card).getByText('Region: RETAIL – CHINA')).toBeInTheDocument();
    expect(within(card).getByRole('button', { name: /1 field needed/ })).toBeInTheDocument();
    expect(api.bodiesOf(CHECK)).toEqual([
      {
        values: { 'SA-CLUSTER': ['SO-RETAIL'], 'SA-REGION': ['SO-CHINA'] },
        businessUnitOptionIds: [LUBRICANTS, MOBILITY],
      },
    ]);

    // Any change makes the answer out of date.
    await user.click(within(card).getByRole('checkbox', { name: 'B2B' }));
    expect(within(card).queryByText('Saving would work.')).not.toBeInTheDocument();
  });

  it('Try saving: shows every problem at once, at the fields', async () => {
    const { user } = renderPreview([
      reply(
        'POST /api/v1/shell-attribute-forms/{level}/check',
        {
          body: {
            problems: [
              {
                field: 'attributes.SA-CLUSTER',
                message: 'Choose one of the options of Cluster.',
              },
            ],
            missing: [],
            cleared: [],
            values: {},
          },
        },
        { level: 'campaign' },
      ),
    ]);

    const card = await previewCard();
    await within(card).findByRole('group', { name: /^Cluster/ });
    await user.click(within(card).getByRole('button', { name: 'Try saving' }));

    expect(await within(card).findByText('1 problem needs your attention')).toBeInTheDocument();
    expect(within(card).getAllByText('Choose one of the options of Cluster.')).toHaveLength(2);
    expect(within(card).queryByText('Saving would work.')).not.toBeInTheDocument();
  });

  it('Try saving: says when the check could not be made', async () => {
    const { user } = renderPreview([
      reply(
        'POST /api/v1/shell-attribute-forms/{level}/check',
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
        { level: 'campaign' },
      ),
    ]);

    const card = await previewCard();
    await within(card).findByRole('group', { name: /^Cluster/ });
    await user.click(within(card).getByRole('button', { name: 'Try saving' }));

    expect(await within(card).findByRole('alert')).toHaveTextContent('Something went wrong.');
  });

  it('previews the plan form with its own Business unit deciding what is shown (FR-026)', async () => {
    const { user, api } = renderPreview(
      [
        reply(
          'POST /api/v1/shell-attribute-forms/{level}/check',
          {
            body: { problems: [], missing: [], cleared: [], values: {} },
          },
          { level: 'plan' },
        ),
      ],
      FLEET_SIZE,
    );

    const card = await screen.findByRole('region', { name: 'Preview — Plan form' });
    await within(card).findByRole('group', { name: /^Business unit/ });
    expect(
      screen.queryByRole('group', { name: /Preview as a plan whose business unit includes:/ }),
    ).not.toBeInTheDocument();
    expect(within(card).queryByRole('textbox', { name: /Fleet size/ })).not.toBeInTheDocument();

    await user.click(within(card).getByRole('checkbox', { name: 'Mobility' }));
    expect(await within(card).findByRole('textbox', { name: /Fleet size/ })).toBeInTheDocument();
    await user.click(within(card).getByRole('button', { name: 'Try saving' }));

    await waitFor(() =>
      expect(api.bodiesOf('POST /api/v1/shell-attribute-forms/plan/check')).toEqual([
        { values: { 'SA-BU': [MOBILITY] } },
      ]),
    );
  });

  it('says when the form cannot be loaded, with Try again', async () => {
    renderPreview([
      reply(
        'GET /api/v1/shell-attribute-forms/{level}',
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
        { level: 'campaign' },
      ),
    ]);

    const card = await screen.findByRole('region', { name: 'Preview — Campaign form' });
    expect(await within(card).findByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });
});
