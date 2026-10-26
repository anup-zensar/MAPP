import { screen } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { PersonPicker } from './PersonPicker';
import type { PickerPerson } from './PersonPicker';

const EUROPE = { id: 'T-1001', name: 'Lubricants Europe', colour: 'blue', active: true } as const;
const CHINA = { id: 'T-1002', name: 'Mobility China', colour: 'teal', active: true } as const;

function person(
  id: string,
  name: string,
  email: string,
  jobTitle: string,
  teams: PickerPerson['teams'] = [],
  active = true,
): PickerPerson {
  const initials = name
    .split(' ')
    .map((part) => part[0])
    .join('');
  return { id, name, email, jobTitle, initials, active, teams };
}

const PEOPLE: PickerPerson[] = [
  person('U-1001', 'Alex Morgan', 'alex.morgan@shell.com', 'Head of marketing', [EUROPE]),
  person('U-1002', 'Priya Raman', 'priya.raman@shell.com', 'Marketer', [EUROPE]),
  person('U-1003', 'Daniel Okoro', 'daniel.okoro@shell.com', '', [CHINA]),
  person('U-1004', 'Sara Novak', 'sara.novak@shell.com', 'Finance partner', [CHINA]),
  person('U-1005', 'Old Timer', 'old.timer@shell.com', 'Retired', [EUROPE], false),
];

function Single({ start = '', teamIds }: { start?: string; teamIds?: string[] }) {
  const [value, setValue] = useState(start);
  return (
    <>
      <PersonPicker
        label="Owner"
        name="ownerId"
        marking="Needed to publish"
        people={PEOPLE}
        currentUserId="U-1002"
        teamIds={teamIds}
        value={value}
        onChange={setValue}
      />
      <p>chosen: {value}</p>
    </>
  );
}

function field() {
  return screen.getByRole('combobox', { name: 'Owner' });
}

describe('PersonPicker', () => {
  it('offers active people only, the signed-in person first with "(you)" (FR-017, FR-018)', async () => {
    const { user } = renderWithProviders(<Single />);

    await user.click(field());

    expect(screen.getAllByRole('option').map((option) => option.textContent)).toEqual([
      'PRPriya Raman (you)Marketer',
      'AMAlex MorganHead of marketing',
      'DODaniel Okorodaniel.okoro@shell.com',
      'SNSara NovakFinance partner',
    ]);
    expect(screen.getByRole('status')).toHaveTextContent('4 people');
  });

  it('finds people by job title or e-mail (FR-016)', async () => {
    const { user } = renderWithProviders(<Single />);

    await user.type(field(), 'finance');
    expect(screen.getAllByRole('option').map((option) => option.textContent)).toEqual([
      'SNSara NovakFinance partner',
    ]);
    expect(screen.getByRole('status')).toHaveTextContent('1 person');

    await user.clear(field());
    await user.type(field(), 'okoro@');
    await user.keyboard('{Enter}');
    expect(screen.getByText('chosen: U-1003')).toBeInTheDocument();
  });

  it('says when no one matches', async () => {
    const { user } = renderWithProviders(<Single />);

    await user.type(field(), 'zzz');

    expect(
      screen.getByText(
        "No one matches 'zzz'. Check the spelling, or add the person in Settings › People.",
      ),
    ).toBeInTheDocument();
  });

  it('keeps an inactive person already chosen, labelled and explained (FR-018)', () => {
    renderWithProviders(<Single start="U-1005" />);

    expect(screen.getByText('Old Timer (inactive)')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'About Old Timer' })).toBeInTheDocument();
  });

  it('offers only members of the chosen teams, with their team (FR-019)', async () => {
    const { user } = renderWithProviders(<Single teamIds={['T-1002']} />);

    await user.click(field());

    expect(screen.getAllByRole('option').map((option) => option.textContent)).toEqual([
      'DODaniel Okorodaniel.okoro@shell.com · Mobility China',
      'SNSara NovakFinance partner · Mobility China',
    ]);
  });

  it('is locked until teams are chosen (FR-019)', () => {
    renderWithProviders(<Single teamIds={[]} />);
    expect(field()).toHaveAttribute('aria-disabled', 'true');
    expect(field()).toHaveAccessibleDescription(
      "Needed to publish Choose teams first — the owner must belong to one of the plan's teams.",
    );
  });

  it('says when the chosen person is in none of the teams (FR-019)', () => {
    renderWithProviders(<Single start="U-1001" teamIds={['T-1002']} />);

    expect(screen.getByText("Not a member of the plan's teams")).toBeInTheDocument();
  });
});
