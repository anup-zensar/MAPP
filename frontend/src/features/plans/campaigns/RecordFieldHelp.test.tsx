import { within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { DETAILS, findDetails, findForm, renderPanel } from '@/testing/fixtures/campaign-panel';
import { fieldHelpReply, helpOf } from '@/testing/fixtures/field-help-texts';

const PANEL: [string, string][] = [
  ['ID', 'common.id'],
  ['Owner', 'campaign.owner'],
  ['Dates', 'campaign.dates'],
  ['Budget', 'campaign.budget'],
  ['Planned cost', 'campaign.plannedCost'],
  ['Budget used', 'campaign.budgetUsed'],
  ['Left to plan', 'campaign.leftToPlan'],
  ['Goals', 'campaign.goals'],
  ['Description', 'campaign.description'],
  ['Approval progress', 'campaign.approvalProgress'],
  ['Completeness', 'common.completeness'],
];

const FORM: [string, string][] = [
  ['Campaign name', 'campaign.name'],
  ['Owner', 'campaign.owner'],
  ['Dates', 'campaign.dates'],
  ['Description', 'campaign.description'],
  ['Campaign budget', 'campaign.budget'],
  ['Remaining plan budget', 'campaign.remainingPlanBudget'],
];

const keysOf = (rows: [string, string][]) => rows.map(([, key]) => key);

async function expectOneTipEach(scope: HTMLElement, rows: [string, string][]) {
  for (const [label] of rows) {
    const tips = await within(scope).findAllByRole('button', { name: `About ${label}` });
    expect(tips, label).toHaveLength(1);
  }
}

describe('016 FR-047: campaign and initiative panels show the Field help of their values', () => {
  it('gives each fact, figure and section of the details panel one ⓘ', async () => {
    renderPanel(DETAILS, [fieldHelpReply(helpOf(keysOf(PANEL)))]);

    await expectOneTipEach(await findDetails(), PANEL);
  });

  it('gives each field of the form one ⓘ, and none for an emptied text', async () => {
    renderPanel(`${DETAILS}/edit`, [
      fieldHelpReply({ ...helpOf(keysOf(FORM)), 'campaign.description': '' }),
    ]);
    const form = await findForm();

    await expectOneTipEach(
      form,
      FORM.filter(([label]) => label !== 'Description'),
    );
    expect(within(form).queryByRole('button', { name: 'About Description' })).toBeNull();
  });
});
