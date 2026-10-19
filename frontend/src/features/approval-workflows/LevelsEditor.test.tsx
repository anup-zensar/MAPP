import { screen, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import type { FieldProblem } from '@/components';
import { mockApi } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { toInputs } from './levelDrafts';
import type { LevelDraft } from './levelDrafts';
import { LevelsEditor } from './LevelsEditor';
import { PEOPLE } from './testData';

function Harness({ start, problems = [] }: { start: LevelDraft[]; problems?: FieldProblem[] }) {
  const [levels, setLevels] = useState(start);
  return (
    <>
      <LevelsEditor levels={levels} onChange={setLevels} problems={problems} />
      <output>{JSON.stringify(toInputs(levels))}</output>
    </>
  );
}

const TWO: LevelDraft[] = [
  { key: 'a', name: 'Regional', approverIds: ['U-1005'] },
  { key: 'b', name: 'Global', approverIds: ['U-1001'] },
];

function renderEditor(start: LevelDraft[], problems?: FieldProblem[]) {
  mockApi({ 'GET /api/people': PEOPLE });
  return renderWithProviders(<Harness start={start} problems={problems} />, { user: testUser() });
}

const levels = () => screen.getAllByRole('heading', { level: 3 }).map((h) => h.textContent);
const nameValues = () =>
  screen
    .getAllByRole('textbox', { name: /Level name/ })
    .map((input) => (input as HTMLInputElement).value);

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

  it('chooses approvers with the person picker', async () => {
    const { user } = renderEditor([{ key: 'a', name: '', approverIds: [] }]);

    await user.click(await screen.findByRole('combobox', { name: /Approvers/ }));
    await user.click(screen.getByRole('option', { name: /Daniel Okoro/ }));

    expect(screen.getByText('[{"name":"","approverIds":["U-1003"]}]')).toBeInTheDocument();
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
