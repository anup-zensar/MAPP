import { screen, waitFor } from '@testing-library/react';
import type { ReactElement } from 'react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import type { MockHandler } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { FIELD_HELP_PAGE, PLAN_BUDGET, PLAN_PLANNED_COST } from '@/testing/fixtures/field-help';
import { FieldHelpTip } from './FieldHelpTip';
import { useFieldHelp } from './useFieldHelp';

/** Shows what help() answers for each key, as a page would pass it to a field's tip. */
function Texts({ keys, fallback }: { keys: string[]; fallback?: string }) {
  const { help, ready } = useFieldHelp();
  return (
    <ul aria-label={ready ? 'Ready' : 'Waiting'}>
      {keys.map((key) => (
        <li key={key}>{`${key}: ${help(key, fallback) ?? '(no tip)'}`}</li>
      ))}
    </ul>
  );
}

function renderWith(answer: MockHandler, ui: ReactElement) {
  const api = mockApi({ 'GET /api/v1/field-help': answer });
  renderWithProviders(ui, { user: testUser() });
  return api;
}

describe('useFieldHelp (016 FR-047)', () => {
  it('answers the text of each key; an emptied text hides the tip', async () => {
    renderWith(
      { body: FIELD_HELP_PAGE },
      <Texts keys={['plan.budget', 'plan.plannedCost', 'plan.bcpir']} />,
    );

    await screen.findByRole('list', { name: 'Ready' });
    expect(screen.getAllByRole('listitem').map((item) => item.textContent)).toEqual([
      `plan.budget: ${PLAN_BUDGET.text}`,
      `plan.plannedCost: ${PLAN_PLANNED_COST.text}`,
      'plan.bcpir: (no tip)',
    ]);
  });

  it('answers the fallback while the texts are on their way', () => {
    renderWith(
      () => new Promise<never>(() => {}),
      <Texts keys={['plan.budget']} fallback="The former fixed tip." />,
    );

    expect(screen.getByRole('list', { name: 'Waiting' })).toHaveTextContent(
      'plan.budget: The former fixed tip.',
    );
  });

  it('answers the fallback when the texts cannot be loaded', async () => {
    const api = renderWith(
      { status: 500, body: { error: { code: 'internal_error', message: 'Boom', fields: [] } } },
      <Texts keys={['plan.budget']} fallback="The former fixed tip." />,
    );

    await waitFor(() => expect(api.requests).toHaveLength(1));
    expect(screen.getByRole('listitem')).toHaveTextContent('plan.budget: The former fixed tip.');
  });

  it('answers the fallback for a key the catalogue does not have', async () => {
    renderWith(
      { body: FIELD_HELP_PAGE },
      <Texts keys={['plan.unknown']} fallback="The former fixed tip." />,
    );

    await screen.findByRole('list', { name: 'Ready' });
    expect(screen.getByRole('listitem')).toHaveTextContent('plan.unknown: The former fixed tip.');
  });
});

describe('FieldHelpTip (016 FR-047)', () => {
  it('shows an ⓘ named after its label, with the Field help text', async () => {
    mockApi({ 'GET /api/v1/field-help': { body: FIELD_HELP_PAGE } });
    const { user } = renderWithProviders(
      <p>
        Planned cost <FieldHelpTip helpKey="plan.plannedCost" label="Planned cost" />
      </p>,
      { user: testUser() },
    );

    const tip = await screen.findByRole('button', { name: 'About Planned cost' });
    await user.hover(tip);

    await waitFor(() =>
      expect(screen.getAllByText(PLAN_PLANNED_COST.text).length).toBeGreaterThan(0),
    );
  });

  it('shows nothing for an emptied text', async () => {
    renderWith(
      { body: FIELD_HELP_PAGE },
      <>
        <Texts keys={['plan.bcpir']} />
        <p>
          BCPIR plan <FieldHelpTip helpKey="plan.bcpir" label="BCPIR plan" />
        </p>
      </>,
    );

    await screen.findByRole('list', { name: 'Ready' });
    expect(screen.queryByRole('button', { name: 'About BCPIR plan' })).not.toBeInTheDocument();
  });
});
