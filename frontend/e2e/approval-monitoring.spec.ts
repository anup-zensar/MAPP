import { expect, test, type Page } from '@playwright/test';
import { expectNoAxeViolations, expectToast, signInAs } from './fixtures';
import { tabTo } from './keyboard';
import { cancelCurrentDelegations } from './planning-changes';

// Feature 017's journeys 5 and 6 (plan, Tests › End-to-end) against the real API: monitoring, reminders and
// escalation on the demo data, then axe and a keyboard-only delegation.

const ALEX = 'alex.morgan@shell.com';
const DANIEL = 'daniel.okoro@shell.com';
const TRADE_ADS = 'Trade magazine ads';

async function openMonitoring(page: Page) {
  await page.goto('/approvals');
  await page.getByRole('tab', { name: 'Monitoring' }).click();
  return page.getByRole('tabpanel', { name: 'Monitoring' });
}

function typedDate(offsetDays: number): string {
  const day = new Date(Date.now() + offsetDays * 86_400_000).toISOString().slice(0, 10);
  const [year, month, date] = day.split('-');
  return `${date}${month}${year}`;
}

// The order of a date field's parts follows the browser's language; fixed so the typed digits land right.
test.use({ locale: 'en-GB' });

test.describe('with the seeded overdue round', () => {
  // The demo round of Trade magazine ads has waited two days, past Lubricants EU approval's escalation
  // after 2; no other journey runs escalations, and 030's journey later approves it as Daniel Okoro.
  test('017 US5, US6 (T047 journey 5): Monitoring shows the overdue round; running escalations hands it to the backup approver', async ({
    page,
  }) => {
    await signInAs(page, ALEX);
    const monitoring = await openMonitoring(page);
    const overdue = monitoring.getByRole('region', { name: 'Overdue' });
    const row = overdue.getByRole('row', { name: new RegExp(`^${TRADE_ADS} A-1004 `) });
    await expect(row).toContainText('Lubricants EU approval');
    await expect(row).toContainText(/waiting \d+ days/);
    await expect(
      monitoring.getByRole('region', { name: 'Waiting by level', exact: true }),
    ).toBeVisible();
    await expectNoAxeViolations(page);

    await monitoring.getByRole('button', { name: 'Run reminders and escalations' }).click();
    await page
      .getByRole('alertdialog', { name: 'Run reminders and escalations now?' })
      .getByRole('button', { name: 'Run now' })
      .click();
    await expect(page.getByRole('region', { name: 'Messages (F8)' })).toContainText(
      /(No reminders|\d+ reminders? sent), 1 escalation/,
    );
    await expect(row).toContainText('Escalated to Alex Morgan');

    // The backup approver may decide now; the decision itself is left to Daniel Okoro (030).
    await page.goto('/approvals');
    await page.getByRole('searchbox', { name: 'Search activity name or ID' }).fill(TRADE_ADS);
    await page.getByRole('button', { name: TRADE_ADS, exact: true }).click();
    await page
      .getByRole('dialog', { name: TRADE_ADS })
      .getByRole('button', { name: 'Approve', exact: true })
      .click();
    const approve = page.getByRole('alertdialog', { name: `Approve '${TRADE_ADS}'?` });
    await expect(approve).toContainText('You approve as backup approver of level 1.');
    await approve.getByRole('button', { name: 'Cancel' }).click();
    await expect(approve).toHaveCount(0);
  });
});

test('017 (T048 journey 6): the workflow form with conditions and errors and the delegation screens pass axe; a delegation is added by keyboard alone', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, ALEX);
  await page.goto('/settings/approval-workflows');
  await page.getByRole('button', { name: 'New approval workflow' }).click();
  const panel = page.getByRole('dialog', { name: 'New approval workflow' });
  await panel.getByRole('checkbox', { name: 'Only needed when' }).check();
  await panel.getByRole('checkbox', { name: 'Approve automatically when' }).check();
  await panel.getByRole('button', { name: 'Save', exact: true }).click();
  const problems = panel.getByRole('alert');
  await expect(
    problems.getByRole('heading', { name: '5 problems need your attention' }),
  ).toBeVisible();
  await expect(
    problems.getByRole('link', {
      name: "Level 1: 'Only needed when' needs at least one condition.",
    }),
  ).toBeVisible();
  await expectNoAxeViolations(page);

  await signInAs(page, DANIEL);
  await page.goto('/approvals');
  await expect(page.getByRole('heading', { name: 'Approvals', level: 1 })).toBeVisible();
  try {
    await tabTo(page, page.getByRole('button', { name: 'Delegate my approvals' }));
    await page.keyboard.press('Enter');
    const delegations = page.getByRole('dialog', { name: 'Approval delegation' });
    await expect(delegations.getByRole('region', { name: 'Planned' })).toBeVisible();
    await expectNoAxeViolations(page);

    await tabTo(page, delegations.getByRole('button', { name: 'Add delegation' }));
    await page.keyboard.press('Enter');
    const form = page.getByRole('dialog', { name: 'Add delegation' });
    await tabTo(page, form.getByRole('combobox', { name: 'Delegate' }));
    await page.keyboard.type('Carlos');
    await expect(page.getByRole('option', { name: /^Carlos Mendes/ })).toBeVisible();
    await page.keyboard.press('ArrowDown');
    await page.keyboard.press('Enter');
    await tabTo(page, form.getByRole('textbox', { name: 'Start date' }));
    await page.keyboard.type(typedDate(0));
    await tabTo(page, form.getByRole('textbox', { name: 'End date' }));
    await page.keyboard.type(typedDate(2));
    await tabTo(page, form.getByRole('textbox', { name: 'Note (optional)' }));
    await page.keyboard.type('Away at a customer workshop');
    await expectNoAxeViolations(page);
    await tabTo(page, form.getByRole('button', { name: 'Save', exact: true }));
    await page.keyboard.press('Enter');
    await expectToast(page, 'Delegation to Carlos Mendes saved');
    await expect(delegations.getByText('Away at a customer workshop')).toBeVisible();
  } finally {
    await cancelCurrentDelegations(page, DANIEL);
  }
});
