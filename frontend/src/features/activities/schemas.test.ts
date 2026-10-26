import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import type { z } from 'zod';
import { formatDate } from '@/format/intl';
import { launchFilm } from '@/testing/fixtures/activities';
import { editForm, newForm } from '@/testing/fixtures/activity-forms';
import { fromActivity, fromDefaults } from './activity-form-state';
import type { FormState } from './activity-form-state';
import { activityFormSchema, focusFieldOf, tabOf, tabSearch } from './schemas';

function newState(overrides: Partial<FormState> = {}): FormState {
  return {
    ...fromDefaults(newForm()),
    name: 'Trade show',
    activityTypeId: 'AT-1001',
    ...overrides,
  };
}

/** The schema's problems in the order it finds them, as the form lists them. */
function problemsOf(schema: z.ZodType<FormState>, state: FormState) {
  const result = schema.safeParse(state);
  return result.success
    ? []
    : result.error.issues.map((issue) => ({ field: issue.path.join('.'), message: issue.message }));
}

function check(state: FormState, mode: 'new' | 'edit' = 'new', original?: FormState) {
  const form = mode === 'new' ? newForm() : editForm();
  return problemsOf(activityFormSchema({ form, mode, original, date: formatDate }), state);
}

describe('the activity form’s checks (feature 006)', () => {
  it('needs a name and a type only (FR-003, FR-006)', () => {
    expect(check(newState())).toEqual([]);
    expect(check(newState({ name: '  ', activityTypeId: '' }))).toEqual([
      { field: 'name', message: 'Enter a name.' },
      { field: 'activityTypeId', message: 'Choose an activity type.' },
    ]);
    expect(check(newState({ name: 'x'.repeat(121), externalName: 'y'.repeat(121) }))).toEqual([
      { field: 'name', message: 'Use 120 characters or fewer.' },
      { field: 'externalName', message: 'Use 120 characters or fewer.' },
    ]);
  });

  it('refuses a name already used in the initiative, ignoring case and spaces (FR-006)', () => {
    expect(check(newState({ name: '  launch FILM ' }))).toEqual([
      {
        field: 'name',
        message: "'Digital & social' already has an activity called 'Launch film'.",
      },
    ]);
  });

  it('lists every date problem at once (FR-009)', () => {
    expect(check(newState({ startDate: '2026-05-31', endDate: '2026-03-01' }))).toEqual([
      { field: 'endDate', message: 'End date must be on or after the start date (31 May 2026).' },
    ]);
    const inside = "Dates must be inside the initiative's dates (01 Jan 2026 – 30 Jun 2026).";
    expect(check(newState({ startDate: '2025-12-01', endDate: '2026-07-31' }))).toEqual([
      { field: 'startDate', message: inside },
      { field: 'endDate', message: inside },
    ]);
  });

  it('refuses an end date after 31 Dec ten years after the plan year (FR-009)', () => {
    const form = newForm();
    form.context.initiative.startDate = null;
    form.context.initiative.endDate = null;
    const state = newState({ startDate: '2026-01-01', endDate: '2037-01-01' });
    expect(problemsOf(activityFormSchema({ form, mode: 'new', date: formatDate }), state)).toEqual([
      {
        field: 'endDate',
        message: "The end date can't be after 31 Dec 2036 — 10 years after the plan's year.",
      },
    ]);
  });

  it('checks unchanged dates of an edited activity no more (edge case: initiative dates moved)', () => {
    const original = fromActivity(launchFilm({ startDate: '2025-12-01' }));
    expect(check({ ...original, description: 'New' }, 'edit', original)).toEqual([]);
    expect(check({ ...original, startDate: '2025-12-02' }, 'edit', original)).toHaveLength(1);
  });

  it('checks only the start of an ongoing activity (FR-010)', () => {
    expect(check(newState({ ongoing: true, endDate: '2027-12-31' }))).toEqual([]);
  });

  it('refuses a negative or unreadable planned cost; 0.00 is fine (FR-011)', () => {
    expect(check(newState({ plannedCost: '-5' }))).toEqual([
      { field: 'plannedCost', message: "Planned cost can't be negative." },
    ]);
    expect(check(newState({ plannedCost: 'abc' }))).toEqual([
      { field: 'plannedCost', message: 'Enter an amount, such as 12,500.00.' },
    ]);
    expect(check(newState({ plannedCost: '0' }))).toEqual([]);
  });

  it('needs whole numbers of 0 or more for INQ and typed MQL, and revenue not negative (FR-018)', () => {
    const goals = { inq: '12.5', setLaterStagesMyself: true, mql: '-5', revenue: '-1' };
    expect(check(newState({ goals }))).toEqual([
      { field: 'goals.inq', message: 'Enter a whole number of 0 or more.' },
      { field: 'goals.mql', message: 'Enter a whole number of 0 or more.' },
      { field: 'goals.revenue', message: "Revenue can't be negative." },
    ]);
  });

  it('refuses a description over 2,000 characters', () => {
    expect(check(newState({ description: 'x'.repeat(2001) }))).toEqual([
      { field: 'description', message: 'Use 2,000 characters or fewer.' },
    ]);
  });
});

const TABS = ['overview', 'details', 'approval', 'history'] as const;

describe('the activity page’s address (006 FR-020, FR-025)', () => {
  it('opens a tab for any address, and the tab it wrote', () => {
    fc.assert(
      fc.property(fc.string(), fc.string(), (search, hash) => {
        expect(TABS).toContain(tabOf(search, hash));
      }),
    );
    fc.assert(
      fc.property(fc.constantFrom(...TABS), (tab) => {
        expect(tabOf(tabSearch(tab), '')).toBe(tab);
      }),
    );
  });

  it('opens the tab of a #part before ?tab=', () => {
    expect(tabOf('?tab=details', '#approval')).toBe('approval');
    expect(tabOf('?tab=history', '#line-items')).toBe('overview');
  });

  it('reads ?field= from any address, and the field a Ready to submit link names', () => {
    fc.assert(
      fc.property(fc.string(), (query) => {
        const field = focusFieldOf(new URLSearchParams(query));
        expect(field === null || typeof field === 'string').toBe(true);
      }),
    );
    fc.assert(
      fc.property(fc.string(), (field) => {
        expect(focusFieldOf(new URLSearchParams({ field }))).toBe(field);
      }),
    );
  });
});
