import { describe, expect, it } from 'vitest';
import {
  deactivateSentences,
  deleteTeamText,
  membersText,
  peopleCountText,
  plansText,
  teamInUseText,
  teamsCountText,
  usageText,
  usedByText,
} from './texts';
import type { UsageCount } from './peopleApi';

const use = (role: UsageCount['role'], count: number, label: string): UsageCount => ({
  role,
  recordType: label,
  label,
  count,
});

describe('feature 003 texts', () => {
  it('counts people and teams, with "of" while some are hidden', () => {
    expect(peopleCountText(1, 1)).toBe('1 person');
    expect(peopleCountText(6, 6)).toBe('6 people');
    expect(peopleCountText(2, 6)).toBe('2 of 6 people');
    expect(teamsCountText(1, 1)).toBe('1 team');
    expect(teamsCountText(3, 3)).toBe('3 teams');
    expect(teamsCountText(0, 3)).toBe('0 of 3 teams');
  });

  it('says how many members and plans a team has', () => {
    expect(membersText(0)).toBe('No members');
    expect(membersText(1)).toBe('1 member');
    expect(membersText(4)).toBe('4 members');
    expect(plansText(0)).toBe('Not used');
    expect(plansText(1)).toBe('1 plan');
    expect(plansText(2)).toBe('2 plans');
    expect(usedByText(1)).toBe('Used by 1 plan');
    expect(usedByText(3)).toBe('Used by 3 plans');
  });

  it('says what a person owns and approves (US1-3)', () => {
    expect(usageText([])).toBe("Doesn't own or approve anything yet.");
    expect(
      usageText([
        use('owner', 2, 'plans'),
        use('owner', 5, 'activities'),
        use('approver', 1, 'approval workflow'),
      ]),
    ).toBe('Owner of 2 plans and 5 activities · approver in 1 approval workflow');
    expect(
      usageText([
        use('owner', 2, 'plans'),
        use('owner', 5, 'activities'),
        use('owner', 1, 'initiative'),
      ]),
    ).toBe('Owner of 2 plans, 5 activities and 1 initiative');
    expect(usageText([use('approver', 3, 'activities')])).toBe('approver in 3 activities');
  });

  it('says what deactivating a person means (US1-5, FR-005)', () => {
    expect(
      deactivateSentences({
        name: 'Priya Raman',
        usage: [
          use('owner', 2, 'plans'),
          use('owner', 5, 'activities'),
          use('approver', 1, 'workflow'),
        ],
      }),
    ).toEqual([
      "Priya won't be able to sign in or be chosen in people lists.",
      "Priya owns 2 plans and 5 activities and approves in 1 workflow; these keep the name, shown as 'Priya Raman (inactive)'.",
      'Their teams are kept.',
    ]);
    expect(
      deactivateSentences({ name: 'Sara Novak', usage: [use('approver', 1, 'workflow')] })[1],
    ).toBe("Sara approves in 1 workflow; these keep the name, shown as 'Sara Novak (inactive)'.");
    expect(deactivateSentences({ name: 'Jane Doe', usage: [] })).toEqual([
      "Jane won't be able to sign in or be chosen in people lists.",
      'Their teams are kept.',
    ]);
  });

  it('says what deleting a team takes with it', () => {
    expect(deleteTeamText(0)).toBe('It goes to the recycle bin. You can restore it.');
    expect(deleteTeamText(1)).toBe(
      'It goes to the recycle bin together with its team membership. You can restore it.',
    );
    expect(deleteTeamText(3)).toBe(
      'It goes to the recycle bin together with its 3 team memberships. You can restore it.',
    );
  });

  it('says why a used team cannot be deleted (US2-6)', () => {
    expect(
      teamInUseText({
        name: 'Lubricants Europe',
        plans: [
          { id: 'P-1001', name: 'FY2026 Lubricants Europe', inRecycleBin: false },
          { id: 'P-1004', name: 'Old plan', inRecycleBin: true },
        ],
      }),
    ).toBe(
      "It is used by 2 plans: FY2026 Lubricants Europe, Old plan (in the recycle bin). Set it inactive instead: it won't be offered for new plans, and these plans keep it, shown as 'Lubricants Europe (inactive)'.",
    );
    expect(
      teamInUseText({
        name: 'Mobility China',
        plans: [{ id: 'P-1002', name: 'FY2026 Mobility China', inRecycleBin: false }],
      }),
    ).toMatch(/^It is used by 1 plan: FY2026 Mobility China\. /);
  });
});
