import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { findMessages, renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { expectNoAxeViolations } from '../shared/axeCheck';
import { validationFailed, versionConflict } from '../testRecords';
import { FieldHelpPage } from './FieldHelpPage';
import { FIELD_HELP_PAGE, PLAN_BUDGET, PLAN_PLANNED_COST } from './fieldHelpTestData';

function renderHelp(answers: Record<string, MockHandler> = {}, route = '/settings/field-help') {
  const api = mockApi({ 'GET /api/field-help': { body: FIELD_HELP_PAGE }, ...answers });
  const rendered = renderWithProviders(<FieldHelpPage />, {
    route,
    path: '/settings/field-help',
    user: testUser(),
  });
  return { ...rendered, api };
}

function row(label: string): HTMLElement {
  return screen.getByRole('rowheader', { name: label }).closest('tr') as HTMLElement;
}

function fieldNames(): string[] {
  return screen.getAllByRole('rowheader').map((cell) => cell.textContent ?? '');
}

async function chooseAction(
  label: string,
  action: string,
  user: ReturnType<typeof renderHelp>['user'],
) {
  await user.click(await screen.findByRole('button', { name: `Actions for ${label}` }));
  await user.click(await screen.findByRole('menuitem', { name: new RegExp(`^${action}`) }));
}

describe('FieldHelpPage: the list (016 FR-045, FR-048)', () => {
  it('lists the plan fields with their help and a Default or Edited chip', async () => {
    renderHelp();

    expect(screen.getByRole('heading', { level: 1, name: 'Field help' })).toBeInTheDocument();
    expect(document.title).toBe('Field help · MAPP');
    await screen.findByRole('table');

    expect(fieldNames()).toEqual(['Plan budget', 'Planned cost', 'BCPIR plan']);
    expect(row('Plan budget')).toHaveTextContent(PLAN_BUDGET.defaultText);
    expect(within(row('Plan budget')).getByText('Default')).toBeInTheDocument();
    expect(within(row('Planned cost')).getByText('Edited')).toBeInTheDocument();
    expect(row('Planned cost')).toHaveTextContent('Priya Raman, 19 Oct 2026');
    expect(row('BCPIR plan')).toHaveTextContent('Empty — no ⓘ is shown');
    expect(screen.getByText('3 of 3 fields')).toBeInTheDocument();
    // Each level is a tab with the number of its fields.
    expect(screen.getByRole('tab', { name: /^Plan/ })).toHaveAttribute('aria-selected', 'true');
    expect(screen.getByRole('tab', { name: /^Campaign/ })).toHaveTextContent('2');
  });

  it('says how many Shell attributes have no help yet, with a link to them (US7-5)', async () => {
    renderHelp();
    await screen.findByRole('table');

    expect(screen.getByText(/3 Shell attributes have no help yet/)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'view them' })).toHaveAttribute(
      'href',
      '/settings/shell-attributes?help=missing',
    );
  });

  it('searches label or text, shows only edited texts, and switches level, in the address', async () => {
    const { user, router } = renderHelp();
    await screen.findByRole('table');

    await user.type(screen.getByRole('searchbox', { name: 'Search field or help text' }), 'spend');
    expect(fieldNames()).toEqual(['Plan budget', 'Planned cost']);
    expect(router.state.location.search).toBe('?q=spend');

    await user.clear(screen.getByRole('searchbox', { name: 'Search field or help text' }));
    await user.click(screen.getByRole('button', { name: /^Edited/ }));
    expect(fieldNames()).toEqual(['Planned cost', 'BCPIR plan']);

    await user.click(screen.getByRole('tab', { name: /^Campaign/ }));
    expect(router.state.location.search).toBe('?show=edited&level=campaign');
    expect(screen.getByText('Nothing matches these filters')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Clear filters' }));
    expect(fieldNames()).toEqual(['Campaign budget', 'Left to plan']);
  });

  it('says when the help texts cannot be loaded', async () => {
    renderHelp({ 'GET /api/field-help': () => Promise.reject(new TypeError('offline')) });

    expect(await screen.findByRole('alert')).toHaveTextContent("can't reach its server");
  });
});

describe('FieldHelpPage: edit and reset (016 FR-046)', () => {
  it('edits a text in a side panel and saves it with its version (US7-2)', async () => {
    const newText = 'What the plan may spend in its year.';
    const { user, api, router } = renderHelp({
      'PATCH /api/field-help/plan.budget': {
        body: { ...PLAN_BUDGET, text: newText, isDefault: false, version: 1 },
      },
    });
    await screen.findByRole('table');

    await user.click(screen.getByRole('button', { name: 'Plan budget' }));
    const panel = await screen.findByRole('dialog', { name: 'Edit field help' });
    expect(router.state.location.search).toBe('?key=plan.budget');
    expect(panel).toHaveTextContent('Plan › Plan budget');
    expect(panel).toHaveTextContent('This field uses the default text.');
    const text = within(panel).getByRole('textbox', { name: 'Help text' });
    await user.clear(text);
    await user.type(text, newText);
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(api.bodiesOf('PATCH /api/field-help/plan.budget')).toEqual([
      { text: newText, version: 0 },
    ]);
    expect(
      await within(await findMessages()).findByText("Help of 'Plan budget' saved"),
    ).toBeInTheDocument();
  });

  it('refuses a text over 300 characters; an empty text is allowed (US7-4)', async () => {
    const { user, api } = renderHelp({
      'PATCH /api/field-help/plan.budget': {
        body: { ...PLAN_BUDGET, text: '', isDefault: false, version: 1 },
      },
    });
    await screen.findByRole('table');

    await user.click(screen.getByRole('button', { name: 'Plan budget' }));
    const panel = await screen.findByRole('dialog', { name: 'Edit field help' });
    const text = within(panel).getByRole('textbox', { name: 'Help text' });
    await user.clear(text);
    await user.click(text);
    await user.paste('x'.repeat(301));
    expect(panel).toHaveTextContent('1 character too many');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await within(panel).findAllByText('Use 300 characters or fewer.')).not.toHaveLength(0);
    expect(api.bodiesOf('PATCH /api/field-help/plan.budget')).toEqual([]);

    await user.clear(text);
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    await waitFor(() =>
      expect(api.bodiesOf('PATCH /api/field-help/plan.budget')).toEqual([{ text: '', version: 0 }]),
    );
  });

  it('names who saved first and keeps the typed text (001 FR-029)', async () => {
    const { user } = renderHelp({
      'PATCH /api/field-help/plan.plannedCost': { status: 409, body: versionConflict() },
    });
    await screen.findByRole('table');

    await user.click(screen.getByRole('button', { name: 'Planned cost' }));
    const panel = await screen.findByRole('dialog', { name: 'Edit field help' });
    const text = within(panel).getByRole('textbox', { name: 'Help text' });
    await user.type(text, ' More.');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(
      await within(panel).findByText(/Not saved — Daniel Okoro changed this help text/),
    ).toBeInTheDocument();
    expect(text).toHaveValue(`${PLAN_PLANNED_COST.text} More.`);
  });

  it('shows the problem the API finds under the text', async () => {
    const { user } = renderHelp({
      'PATCH /api/field-help/plan.budget': {
        status: 400,
        body: validationFailed([{ field: 'text', message: 'Use plain text without links.' }]),
      },
    });
    await screen.findByRole('table');

    await user.click(screen.getByRole('button', { name: 'Plan budget' }));
    const panel = await screen.findByRole('dialog', { name: 'Edit field help' });
    await user.type(within(panel).getByRole('textbox', { name: 'Help text' }), ' See https://x');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await within(panel).findAllByText('Use plain text without links.')).not.toHaveLength(0);
  });

  it('resets an edited text to the default after asking (US7-3)', async () => {
    const { user, api } = renderHelp({
      'PATCH /api/field-help/plan.plannedCost': {
        body: { ...PLAN_PLANNED_COST, text: PLAN_PLANNED_COST.defaultText, isDefault: true },
      },
    });
    await screen.findByRole('table');

    await chooseAction('Planned cost', 'Reset to default', user);
    const question = await screen.findByRole('alertdialog', {
      name: "Reset the help of 'Planned cost'?",
    });
    await user.click(within(question).getByRole('button', { name: 'Reset to default' }));

    await waitFor(() =>
      expect(api.bodiesOf('PATCH /api/field-help/plan.plannedCost')).toEqual([
        { text: null, version: 2 },
      ]),
    );
    expect(
      await within(await findMessages()).findByText("Help of 'Planned cost' reset to the default"),
    ).toBeInTheDocument();
  });

  it('offers Reset to default from the panel of an edited text', async () => {
    const { user } = renderHelp({}, '/settings/field-help?key=plan.plannedCost');

    const panel = await screen.findByRole('dialog', { name: 'Edit field help' });
    expect(panel).toHaveTextContent(PLAN_PLANNED_COST.defaultText);
    expect(panel).toHaveTextContent('Edited by Priya Raman');
    await user.click(within(panel).getByRole('button', { name: 'Reset to default' }));

    expect(
      await screen.findByRole('alertdialog', { name: "Reset the help of 'Planned cost'?" }),
    ).toBeInTheDocument();
  });

  it('says a default text has nothing to reset, without asking', async () => {
    const { user, api } = renderHelp();
    await screen.findByRole('table');

    await chooseAction('Plan budget', 'Reset to default', user);

    expect(
      await within(await findMessages()).findByText('This field already uses the default text.'),
    ).toBeInTheDocument();
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(api.requests.some((request) => request.method === 'PATCH')).toBe(false);
  });
});

describe('FieldHelpPage: no axe violations (WCAG 2.2 AA)', () => {
  it.each([
    ['the list', '/settings/field-help', 'table'],
    ['the edit panel', '/settings/field-help?key=plan.plannedCost', 'dialog'],
  ])('%s', async (_state, route, role) => {
    renderHelp({}, route);
    await screen.findByRole(role);

    await expectNoAxeViolations();
  });
});
