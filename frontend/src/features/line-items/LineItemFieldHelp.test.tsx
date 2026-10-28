import { configure, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { fieldHelpReply, helpOf } from '@/testing/fixtures/field-help-texts';
import { formAnswers } from '@/testing/fixtures/line-item-forms';
import { findNameField, renderCreatePage } from '@/testing/fixtures/line-item-pages';
import { details, launchFilmSection } from '@/testing/fixtures/line-items';
import { testUser } from '@/testing/fixtures/users';
import { LineItemDetailsCard } from './LineItemDetailsCard';
import { LineItemFigures } from './LineItemFigures';
import { LineItemFinanceCard } from './LineItemFinanceCard';
import { SpreadGrid } from './SpreadGrid';
import { UnassignedBalanceRow } from './UnassignedBalanceRow';

// The whole form renders: on a busy computer the first render can take a while.
configure({ asyncUtilTimeout: 10_000 });

const FORM: [string, string][] = [
  ['Line item name', 'lineItem.name'],
  ['Owner', 'lineItem.owner'],
  ['Description', 'lineItem.description'],
  ['Currency', 'lineItem.currency'],
  ['Planned cost', 'lineItem.plannedCost'],
  ['Spread over time', 'lineItem.spread'],
  ['Contracted amount', 'lineItem.contractedAmount'],
  ['Finance budget year', 'lineItem.financeBudgetYear'],
  ['Finance budget item', 'lineItem.financeBudgetItem'],
  ['Request SWBS code from GSAP', 'lineItem.requestSwbs'],
  ['SWBS code', 'lineItem.swbsCode'],
  ['Activity planned cost', 'activity.plannedCost'],
  ['Unassigned now', 'activity.unassignedPlannedCost'],
];

const PAGE: [string, string][] = [
  ['Line item name', 'lineItem.name'],
  ['Owner', 'lineItem.owner'],
  ['Currency', 'lineItem.currency'],
  ['Description', 'lineItem.description'],
  ['Contracted amount', 'lineItem.contractedAmount'],
  ['Finance budget year', 'lineItem.financeBudgetYear'],
  ['Finance budget item', 'lineItem.financeBudgetItem'],
  ['Request SWBS code from GSAP', 'lineItem.requestSwbs'],
  ["Activity's unassigned planned cost", 'activity.unassignedPlannedCost'],
];

const keysOf = (rows: [string, string][]) => rows.map(([, key]) => key);

async function expectOneTipEach(rows: [string, string][]) {
  for (const [label] of rows) {
    const tips = await screen.findAllByRole('button', { name: `About ${label}` });
    expect(tips, label).toHaveLength(1);
  }
}

describe('016 FR-047: the line-item screens show the Field help of every labelled field', () => {
  it('gives each field of the line-item form one ⓘ, and none for an emptied text', async () => {
    renderCreatePage([fieldHelpReply({ ...helpOf(keysOf(FORM)), 'lineItem.description': '' })]);
    await findNameField();

    await expectOneTipEach(FORM.filter(([label]) => label !== 'Description'));
    expect(screen.queryByRole('button', { name: 'About Description' })).toBeNull();
  });

  it('gives each value of the line-item page one ⓘ', async () => {
    mockApi([
      ...formAnswers(),
      fieldHelpReply(helpOf([...keysOf(PAGE), 'lineItem.plannedCost', 'lineItem.spread'])),
    ]);
    const { lineItem, activity, balance } = details();
    renderWithProviders(
      <>
        <LineItemFigures lineItem={lineItem} balance={balance} />
        <LineItemDetailsCard lineItem={lineItem} />
        <LineItemFinanceCard lineItem={lineItem} activity={activity} />
      </>,
      { user: testUser() },
    );

    await expectOneTipEach(PAGE);
    expect(await screen.findAllByRole('button', { name: 'About Planned cost' })).toHaveLength(2);
  });

  it('explains the unassigned planned cost row and the spread grid of the Line items section', async () => {
    mockApi([fieldHelpReply(helpOf(['activity.unassignedPlannedCost', 'lineItem.spread']))]);
    const section = launchFilmSection();
    renderWithProviders(
      <>
        <UnassignedBalanceRow balance={section.balance} />
        <SpreadGrid grid={section.grid} noDates={false} />
      </>,
      { user: testUser() },
    );

    await expectOneTipEach([
      ['Unassigned planned cost', 'activity.unassignedPlannedCost'],
      ['Spread over time', 'lineItem.spread'],
    ]);
  });
});
