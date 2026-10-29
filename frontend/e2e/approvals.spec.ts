import { expect, test, type Locator, type Page } from '@playwright/test';
import {
  callApi,
  escapeRegExp,
  expectNoAxeViolations,
  expectToast,
  signInAs,
  uniqueName,
} from './fixtures';
import { tabTo } from './keyboard';
import { lubricantsActivity, STANDARD_WORKFLOW, type LubricantsActivity } from './lubricants';

// Feature 008's journeys against the real API, on Lubricants activities of the tests' own: Priya Raman
// submits, her line manager Daniel Okoro approves level 1 of the Standard marketing approval (017 seed).
// Journey 4 (approval workflows) is in approval-workflows.spec.ts.

const PRIYA = 'priya.raman@shell.com';
const DANIEL = 'daniel.okoro@shell.com';
const ALEX = 'alex.morgan@shell.com';

function bell(page: Page): Locator {
  return page.getByRole('button', { name: /^Notifications(, \d+ unread)?$/ });
}

async function unreadCount(page: Page): Promise<number> {
  const name = (await bell(page).getAttribute('aria-label')) ?? '';
  return Number(/, (\d+) unread$/.exec(name)?.[1] ?? 0);
}

async function submitThroughApi(page: Page, activity: LubricantsActivity): Promise<void> {
  await callApi(page, 'POST', `/api/v1/activities/${activity.id}/approval/submit`, {
    version: activity.version,
    comment: '',
  });
}

/** Opens Approvals as `email` on `tab`, searches `text` and answers the table. */
async function openApprovals(page: Page, email: string, text: string, tab?: RegExp) {
  await signInAs(page, email);
  await page.goto('/approvals');
  await expect(page.getByRole('heading', { name: 'Approvals', level: 1 })).toBeVisible();
  if (tab) {
    await page.getByRole('tab', { name: tab }).click();
  }
  await page.getByRole('searchbox', { name: 'Search activity name or ID' }).fill(text);
  return page.getByRole('tabpanel').getByRole('table');
}

test('008 US1, US6 (T055 journey 1): Priya submits; Daniel’s bell says an approval is needed and opens it', async ({
  page,
}) => {
  await signInAs(page, PRIYA);
  const activity = await lubricantsActivity(page, 'Bell', '20000.00');

  await page.goto(activity.path);
  await expect(page.getByRole('heading', { name: activity.name, level: 1 })).toBeVisible();
  await page.getByRole('button', { name: 'Submit for approval' }).first().click();
  const submit = page.getByRole('alertdialog', { name: `Submit '${activity.name}' for approval?` });
  await expect(submit.getByText(`Approval workflow: ${STANDARD_WORKFLOW}`)).toBeVisible();
  await expect(submit.getByText('Daniel Okoro')).toBeVisible();
  await submit.getByRole('button', { name: 'Submit for approval' }).click();
  await expectToast(page, `Activity '${activity.name}' submitted for approval`);
  await page.getByRole('tab', { name: 'Approval' }).click();
  await expect(page.locator('#approval').getByText('Waiting', { exact: true })).toBeVisible();

  await signInAs(page, DANIEL);
  await page.goto('/');
  await expect(bell(page)).toHaveAccessibleName(/^Notifications, \d+ unread$/);
  const unread = await unreadCount(page);
  await bell(page).click();
  const notice = page.getByRole('dialog', { name: 'Notifications' }).getByRole('button', {
    name: new RegExp(
      `^Approval needed Priya Raman submitted '${escapeRegExp(activity.name)}' \\(${activity.id}\\) .*Unread$`,
    ),
  });
  await expect(notice).toBeVisible();
  await expectNoAxeViolations(page);

  await notice.click();
  await expect(page).toHaveURL(new RegExp(`${escapeRegExp(activity.path)}#approval$`));
  await expect(page.getByRole('tab', { name: 'Approval' })).toHaveAttribute(
    'aria-selected',
    'true',
  );
  await expect(bell(page)).toHaveAccessibleName(
    unread === 1 ? 'Notifications' : `Notifications, ${unread - 1} unread`,
  );
});

test('008 US3 (T055 journey 2): Daniel approves and declines from his inbox, oldest first; Priya hears of the decline', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, PRIYA);
  const label = uniqueName('Inbox');
  const first = await lubricantsActivity(page, label, '20000.00');
  const second = await lubricantsActivity(page, label, '21000.00');
  await submitThroughApi(page, first);
  await submitThroughApi(page, second);

  await signInAs(page, DANIEL);
  await page.goto('/approvals');
  const waiting = page.getByRole('tab', { name: /^Waiting for me \d+ activit/ });
  const count = /(\d+)/.exec((await waiting.textContent()) ?? '')?.[1];
  // The sidebar badge counts what the Waiting for me tab lists.
  await expect(
    page
      .getByRole('navigation', { name: 'Main' })
      .getByRole('link', { name: `Approvals ${count} waiting for you` }),
  ).toBeVisible();
  const table = await openApprovals(page, DANIEL, label);
  const names = table.getByRole('rowheader').getByRole('button');
  await expect(names).toHaveText([first.name, second.name]);

  await table.getByRole('button', { name: first.name, exact: true }).click();
  await page
    .getByRole('dialog', { name: first.name })
    .getByRole('button', { name: 'Approve', exact: true })
    .click();
  await page
    .getByRole('alertdialog', { name: `Approve '${first.name}'?` })
    .getByRole('button', { name: 'Approve', exact: true })
    .click();
  await expectToast(page, `Activity '${first.name}' approved`);
  // The preview moves on to the next row, so Daniel goes on deciding.
  const reason = 'Please split the cost between two quarters.';
  await page
    .getByRole('dialog', { name: second.name })
    .getByRole('button', { name: 'Decline', exact: true })
    .click();
  const decline = page.getByRole('alertdialog', { name: `Decline '${second.name}'?` });
  await decline.getByRole('textbox', { name: /^Why are you declining it\?/ }).fill(reason);
  await decline.getByRole('button', { name: 'Decline activity' }).click();
  await expectToast(page, `Activity '${second.name}' declined`);
  await expect(page.getByRole('dialog', { name: second.name })).toHaveCount(0);
  await expect(names).toHaveCount(0);

  const decided = await openApprovals(page, DANIEL, label, /^Decided by me/);
  await expect(decided.getByRole('button', { name: first.name, exact: true })).toBeVisible();
  await expect(decided.getByRole('button', { name: second.name, exact: true })).toBeVisible();
  // Daniel is not told of his own decisions.
  await bell(page).click();
  const danielPanel = page.getByRole('dialog', { name: 'Notifications' });
  await expect(danielPanel.getByRole('button').first()).toBeVisible();
  await expect(
    danielPanel.getByRole('button', { name: new RegExp(escapeRegExp(label)) }),
  ).toHaveCount(2);
  await expect(
    danielPanel.getByRole('button', {
      name: new RegExp(`^Activity (approved|declined) .*${escapeRegExp(label)}`),
    }),
  ).toHaveCount(0);

  await signInAs(page, PRIYA);
  await page.goto('/');
  await bell(page).click();
  await expect(
    page.getByRole('dialog', { name: 'Notifications' }).getByRole('button', {
      name: new RegExp(
        `^Activity declined Daniel Okoro declined '${escapeRegExp(second.name)}' \\(${second.id}\\): "${escapeRegExp(reason)}"`,
      ),
    }),
  ).toBeVisible();
});

test('008 US2 (T055 journey 3): an administrator approves on behalf of the approver, and the history says so', async ({
  page,
}) => {
  await signInAs(page, PRIYA);
  const activity = await lubricantsActivity(page, 'Behalf', '20000.00');
  await submitThroughApi(page, activity);

  const table = await openApprovals(page, ALEX, activity.name, /^All open/);
  await table.getByRole('button', { name: activity.name, exact: true }).click();
  await page
    .getByRole('dialog', { name: activity.name })
    .getByRole('button', { name: 'Approve', exact: true })
    .click();
  const approve = page.getByRole('alertdialog', { name: `Approve '${activity.name}'?` });
  await expect(
    approve.getByText(
      'You are not an approver of level 1. Your decision will be recorded as made on behalf of Daniel Okoro.',
    ),
  ).toBeVisible();
  await approve.getByRole('button', { name: 'Approve', exact: true }).click();
  await expectToast(page, `Activity '${activity.name}' approved`);

  await signInAs(page, PRIYA);
  await page.goto(activity.path);
  await expect(page.getByText('Approved', { exact: true }).first()).toBeVisible();
  await page.getByRole('tab', { name: 'Approval' }).click();
  await expect(
    page
      .getByRole('region', { name: 'Approval history' })
      .getByText('Alex Morgan on behalf of Daniel Okoro'),
  ).toBeVisible();
});

test('008 US5 (T055 journey 5): Priya recalls a submission with a reason, and a planned-cost change restarts approval', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, PRIYA);
  const activity = await lubricantsActivity(page, 'Recall', '20000.00');
  await submitThroughApi(page, activity);

  await page.goto(activity.path);
  await page.getByRole('tab', { name: 'Approval' }).click();
  const section = page.locator('#approval');
  await section.getByRole('button', { name: 'Recall submission' }).click();
  const recall = page.getByRole('alertdialog', {
    name: `Recall the submission of '${activity.name}'?`,
  });
  const reason = 'The agency changed its quote.';
  await recall.getByRole('textbox', { name: 'Reason (optional)' }).fill(reason);
  await recall.getByRole('button', { name: 'Recall submission' }).click();
  await expect(page.getByText('Created', { exact: true }).first()).toBeVisible();
  const history = section.getByRole('region', { name: 'Approval history' });
  await expect(history.getByText(/^Recalled/).first()).toBeVisible();
  await expect(history.getByText(reason)).toBeVisible();

  // Submitted again, then its planned cost changes on the edit form (006).
  const { version } = await callApi<{ version: number }>(
    page,
    'GET',
    `/api/v1/activities/${activity.id}`,
  );
  await submitThroughApi(page, { ...activity, version });
  await page.goto(`${activity.path}/edit`);
  await page.getByRole('textbox', { name: /^Planned cost/ }).fill('25000');
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  const restart = page.getByRole('alertdialog', { name: 'This activity will need approval again' });
  await expect(restart.getByText('Planned cost: 20,000.00 USD → 25,000.00 USD')).toBeVisible();
  await restart.getByRole('button', { name: 'Save and restart approval' }).click();
  await expectToast(page, `Activity '${activity.name}' saved`);
  await expect(page.getByText('Created', { exact: true }).first()).toBeVisible();
  await page.getByRole('tab', { name: 'Approval' }).click();
  await expect(history.getByText(/^Approval restarted/).first()).toBeVisible();
  await expect(
    history.getByText('Planned cost changed from 20,000.00 USD to 25,000.00 USD'),
  ).toBeVisible();
});

test('008 SC-008 (T055 journey 6): Approvals with the preview and the decline dialog pass axe, and an approval works by keyboard alone', async ({
  page,
}) => {
  await signInAs(page, PRIYA);
  const declined = await lubricantsActivity(page, 'Axe', '20000.00');
  const approved = await lubricantsActivity(page, 'Keys', '20000.00');
  await submitThroughApi(page, declined);
  await submitThroughApi(page, approved);

  const table = await openApprovals(page, DANIEL, declined.name);
  await table.getByRole('button', { name: declined.name, exact: true }).click();
  const preview = page.getByRole('dialog', { name: declined.name });
  await expect(preview.getByText(`${declined.id} · Waiting at level 1 of 3`)).toBeVisible();
  await expectNoAxeViolations(page);
  await preview.getByRole('button', { name: 'Decline', exact: true }).click();
  const decline = page.getByRole('alertdialog', { name: `Decline '${declined.name}'?` });
  await decline.getByRole('button', { name: 'Decline activity' }).click();
  await expect(
    decline.getByText('Add a comment explaining why the activity is declined.'),
  ).toBeVisible();
  await expectNoAxeViolations(page);
  await page.keyboard.press('Escape');
  await expect(decline).toHaveCount(0);
  await page.keyboard.press('Escape');
  await expect(preview).toHaveCount(0);

  const search = page.getByRole('searchbox', { name: 'Search activity name or ID' });
  await tabTo(page, search);
  await page.keyboard.press('ControlOrMeta+a');
  await page.keyboard.type(approved.name);
  await tabTo(page, table.getByRole('button', { name: approved.name, exact: true }));
  await page.keyboard.press('Enter');
  const keyboardPreview = page.getByRole('dialog', { name: approved.name });
  await tabTo(page, keyboardPreview.getByRole('button', { name: 'Approve', exact: true }));
  await page.keyboard.press('Enter');
  const approve = page.getByRole('alertdialog', { name: `Approve '${approved.name}'?` });
  await tabTo(page, approve.getByRole('button', { name: 'Approve', exact: true }));
  await page.keyboard.press('Enter');
  await expectToast(page, `Activity '${approved.name}' approved`);
  await page.keyboard.press('Escape');
  await expect(page.getByRole('dialog', { name: approved.name })).toHaveCount(0);
});
