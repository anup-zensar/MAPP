import { screen, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import { mockApi } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { emptyLevel, toInputs } from './levelDrafts';
import type { LevelDraft } from './levelDrafts';
import { LevelsEditor } from './LevelsEditor';
import { PEOPLE, RULE_OPTIONS } from './testData';

function Harness({ start, problems = [] }: { start: LevelDraft[]; problems?: FieldProblem[] }) {
  const [levels, setLevels] = useState(start);
  return (
    <>
      <LevelsEditor levels={levels} onChange={setLevels} problems={problems} />
      <output>{JSON.stringify(toInputs(levels))}</output>
    </>
  );
}

const level = (overrides: Partial<LevelDraft>): LevelDraft => ({ ...emptyLevel(), ...overrides });

const TWO: LevelDraft[] = [
  level({ key: 'a', name: 'Regional', approverIds: ['U-1005'] }),
  level({ key: 'b', name: 'Global', approverIds: ['U-1001'] }),
];

function renderEditor(start: LevelDraft[], problems?: FieldProblem[]) {
  const api = mockApi({
    'GET /api/v1/people': PEOPLE,
    'GET /api/v1/approval-rule-options': RULE_OPTIONS,
  });
  return {
    ...renderWithProviders(<Harness start={start} problems={problems} />, { user: testUser() }),
    api,
  };
}

const levels = () => screen.getAllByRole('heading', { level: 3 }).map((h) => h.textContent);
const nameValues = () =>
  screen
    .getAllByRole('textbox', { name: /Level name/ })
    .map((input) => (input as HTMLInputElement).value);
const sent = () => JSON.parse(document.querySelector('output')?.textContent ?? '[]') as unknown[];

describe('LevelsEditor (feature 008, US4)', () => {
  it('adds, moves and removes levels, each button named for its level (FR-002)', async () => {
    const { user } = renderEditor(TWO);
    await screen.findAllByRole('combobox', { name: /Approvers/ });

    expect(levels()).toEqual(['Level 1', 'Level 2']);
    expect(screen.queryByRole('button', { name: 'Move level 1 up' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Move level 2 down' })).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Move level 2 up' }));
    expect(nameValues()).toEqual(['Global', 'Regional']);

    await user.click(screen.getByRole('button', { name: 'Add level' }));
    expect(levels()).toEqual(['Level 1', 'Level 2', 'Level 3']);

    await user.click(screen.getByRole('button', { name: 'Remove level 1' }));
    expect(levels()).toEqual(['Level 1', 'Level 2']);
    expect(nameValues()).toEqual(['Regional', '']);
  });

  it('hides Remove while there is only one level', async () => {
    renderEditor([TWO[0]]);
    await screen.findByRole('combobox', { name: /Approvers/ });

    expect(screen.queryByRole('button', { name: 'Remove level 1' })).not.toBeInTheDocument();
  });

  it('chooses approvers with the person picker, any one approving by default', async () => {
    const { user, api } = renderEditor([level({ key: 'a' })]);

    await user.click(await screen.findByRole('combobox', { name: /Approvers/ }));
    await user.click(screen.getByRole('option', { name: /Daniel Okoro/ }));

    expect(sent()).toEqual([
      {
        name: '',
        approverType: 'people',
        approverIds: ['U-1003'],
        mode: 'any',
        appliesWhen: null,
        autoApproveWhen: null,
        backupApproverId: '',
      },
    ]);
    // Nothing in a level of named people needs the rules' choices.
    expect(api.requests.some((r) => r.path === '/api/v1/approval-rule-options')).toBe(false);
  });

  it('shows each level problem at its field (FR-003)', async () => {
    renderEditor(TWO, [
      { field: 'levels.2.approverIds', message: 'Level 2: choose at least one approver.' },
      { field: 'levels.1.name', message: 'Level 1: use 60 characters or fewer.' },
    ]);

    const pickers = await screen.findAllByRole('combobox', { name: /Approvers/ });
    expect(pickers[1]).toHaveAccessibleDescription(
      expect.stringContaining('Level 2: choose at least one approver.'),
    );
    const names = screen.getAllByRole('textbox', { name: /Level name/ });
    expect(names[0]).toHaveAccessibleDescription(
      expect.stringContaining('Level 1: use 60 characters or fewer.'),
    );
    expect(
      within(names[0].closest('li') as HTMLElement).getByText("Shown as 'Level 1' when empty"),
    ).toBeInTheDocument();
  });
});

describe('LevelsEditor rules (feature 017, US1 and US2)', () => {
  it('the line manager approves alone: no approvers, no several-approvers choice (FR-001, FR-002)', async () => {
    const { user } = renderEditor([level({ key: 'a', approverIds: ['U-1005'], mode: 'all' })]);
    await screen.findByRole('combobox', { name: /Approvers/ });

    expect(screen.getByRole('group', { name: 'Who approves' })).toBeInTheDocument();
    await user.click(screen.getByRole('radio', { name: "The owner's line manager" }));

    expect(screen.queryByRole('combobox', { name: /Approvers/ })).not.toBeInTheDocument();
    expect(
      screen.queryByRole('group', { name: 'When there are several approvers' }),
    ).not.toBeInTheDocument();
    expect(sent()).toEqual([
      expect.objectContaining({ approverType: 'manager', approverIds: [], mode: 'any' }),
    ]);
  });

  it('offers only the roles that approve, and sends the role (FR-001, FR-025)', async () => {
    const { user } = renderEditor([level({ key: 'a' })]);
    await screen.findByRole('combobox', { name: /Approvers/ });

    await user.click(screen.getByRole('radio', { name: 'People with a role' }));
    const role = await screen.findByRole('combobox', { name: /^Role/ });
    await screen.findByRole('option', { name: 'Approver' });
    await user.selectOptions(role, 'Approver');
    await user.click(screen.getByRole('radio', { name: 'All must approve' }));

    expect(sent()).toEqual([
      expect.objectContaining({
        approverType: 'role',
        approverIds: [],
        roleId: 'RL-1005',
        mode: 'all',
      }),
    ]);
  });

  it('reveals a condition with USD amounts, business units and activity types (FR-003, FR-005)', async () => {
    const { user } = renderEditor([level({ key: 'a', approverIds: ['U-1005'] })]);
    await screen.findByRole('combobox', { name: /Approvers/ });

    await user.click(screen.getByRole('checkbox', { name: 'Only needed when' }));
    const condition = screen.getByRole('group', { name: 'Level 1: only needed when' });
    await user.type(
      within(condition).getByRole('textbox', { name: /Planned cost at least \(USD\)/ }),
      '100k',
    );
    await user.click(await within(condition).findByRole('checkbox', { name: 'Mobility' }));
    await user.click(within(condition).getByRole('checkbox', { name: 'Event' }));
    // Inactive values are offered only while chosen.
    expect(within(condition).queryByRole('checkbox', { name: /Retail/ })).not.toBeInTheDocument();

    expect(sent()).toEqual([
      expect.objectContaining({
        appliesWhen: {
          plannedCostAtLeast: '100000.00',
          plannedCostBelow: null,
          businessUnitIds: ['OP-1002'],
          activityTypeIds: ['AT-1001'],
        },
        autoApproveWhen: null,
      }),
    ]);

    // Unticked, the condition is not sent; ticked again, it comes back.
    await user.click(screen.getByRole('checkbox', { name: 'Only needed when' }));
    expect(sent()).toEqual([expect.objectContaining({ appliesWhen: null })]);
    await user.click(screen.getByRole('checkbox', { name: 'Only needed when' }));
    expect(sent()).toEqual([
      expect.objectContaining({
        appliesWhen: expect.objectContaining({ plannedCostAtLeast: '100000.00' }),
      }),
    ]);
  });

  it('keeps an inactive business unit that is already chosen, marked (inactive) (FR-005)', async () => {
    renderEditor([
      level({
        key: 'a',
        approverIds: ['U-1005'],
        autoWhen: true,
        autoApproveWhen: {
          plannedCostAtLeast: '',
          plannedCostBelow: '10,000.00',
          businessUnitIds: ['OP-1003'],
          activityTypeIds: [],
        },
      }),
    ]);

    const condition = screen.getByRole('group', { name: 'Level 1: approve automatically when' });
    expect(
      await within(condition).findByRole('checkbox', { name: 'Retail (inactive)' }),
    ).toBeChecked();
    expect(sent()).toEqual([
      expect.objectContaining({
        autoApproveWhen: {
          plannedCostAtLeast: null,
          plannedCostBelow: '10000.00',
          businessUnitIds: ['OP-1003'],
          activityTypeIds: [],
        },
      }),
    ]);
  });

  it('chooses a backup approver, with what the backup does (FR-006)', async () => {
    const { user } = renderEditor([level({ key: 'a', approverIds: ['U-1005'] })]);

    const backup = await screen.findByRole('combobox', { name: /Backup approver/ });
    expect(backup).toHaveAccessibleDescription(
      expect.stringContaining(
        'Takes over when an approval is escalated, or when nobody else can approve.',
      ),
    );
    await user.click(backup);
    await user.click(screen.getByRole('option', { name: /Alex Morgan/ }));

    expect(sent()).toEqual([expect.objectContaining({ backupApproverId: 'U-1001' })]);
  });

  it('shows the rules problems at their fields (FR-008)', async () => {
    renderEditor(
      [level({ key: 'a', approverType: 'role', onlyWhen: true })],
      [
        { field: 'levels.1.roleId', message: 'Level 1: choose a role.' },
        {
          field: 'levels.1.appliesWhen',
          message: "Level 1: 'Only needed when' needs at least one condition.",
        },
        {
          field: 'levels.1.backupApproverId',
          message: 'Level 1: choose a backup approver, or turn escalation off.',
        },
      ],
    );

    expect(await screen.findByRole('combobox', { name: /^Role/ })).toHaveAccessibleDescription(
      expect.stringContaining('Level 1: choose a role.'),
    );
    expect(screen.getByRole('checkbox', { name: 'Only needed when' })).toHaveAccessibleDescription(
      "Level 1: 'Only needed when' needs at least one condition.",
    );
    expect(
      await screen.findByRole('combobox', { name: /Backup approver/ }),
    ).toHaveAccessibleDescription(
      expect.stringContaining('Level 1: choose a backup approver, or turn escalation off.'),
    );
  });
});
