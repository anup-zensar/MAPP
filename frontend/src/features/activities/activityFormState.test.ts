import { describe, expect, it } from 'vitest';
import { formatDate } from '@/lib/format';
import {
  fromActivity,
  fromDefaults,
  isDirty,
  reapprovalChanges,
  toCreateBody,
  toUpdateBody,
  validate,
} from './activityFormState';
import type { FormState } from './activityFormState';
import { editForm, launchFilm, newForm } from './testData';

function newState(overrides: Partial<FormState> = {}): FormState {
  return {
    ...fromDefaults(newForm()),
    name: 'Trade show',
    activityTypeId: 'AT-1001',
    ...overrides,
  };
}

function check(state: FormState, mode: 'new' | 'edit' = 'new', original?: FormState) {
  const form = mode === 'new' ? newForm() : editForm();
  return validate(state, { form, mode, original, date: formatDate });
}

describe('activityForm (feature 006)', () => {
  it('starts a new activity with the defaults of the initiative and the only workflow (FR-008, FR-012)', () => {
    expect(fromDefaults(newForm())).toMatchObject({
      name: '',
      ownerId: 'U-1002',
      startDate: '2026-01-01',
      endDate: '2026-06-30',
      approvalWorkflowId: 'WF-1001',
      plannedCost: '',
    });
  });

  it('opens an activity with its values as typed text', () => {
    const state = fromActivity(
      launchFilm({
        goals: {
          inq: 250000,
          setLaterStagesMyself: true,
          mql: 6000,
          deals: null,
          revenue: { amount: '3000000.00', currency: 'USD' },
          calculated: false,
        },
      }),
    );
    expect(state).toMatchObject({
      name: 'Launch film',
      activityTypeId: 'AT-1001',
      plannedCost: '120,000.00',
      goals: { inq: '250,000', setLaterStagesMyself: true, mql: '6,000', revenue: '3,000,000.00' },
    });
    expect(isDirty(state, { ...state })).toBe(false);
    expect(isDirty({ ...state, description: 'New' }, state)).toBe(true);
  });

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
    expect(validate(state, { form, mode: 'new', date: formatDate })).toEqual([
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

  it('builds the request bodies, with null for empty fields and no end date while ongoing', () => {
    const state = newState({
      ownerId: '',
      ongoing: true,
      plannedCost: '25,000.00',
      goals: { inq: '250,000', setLaterStagesMyself: false, mql: '9', revenue: '9' },
    });
    expect(toCreateBody(state)).toEqual({
      name: 'Trade show',
      externalName: '',
      activityTypeId: 'AT-1001',
      ownerId: null,
      startDate: '2026-01-01',
      endDate: null,
      ongoing: true,
      plannedCost: { amount: '25000.00', currency: 'USD' },
      approvalWorkflowId: 'WF-1001',
      description: '',
      attributes: {},
      goals: { inq: 250000, setLaterStagesMyself: false, mql: null, revenue: null },
    });
    expect(toUpdateBody(state, 3, true)).toMatchObject({ version: 3, restartApproval: true });
  });

  it('lists the changes that restart approval, old → new (FR-027)', () => {
    const original = fromActivity(launchFilm());
    const state = {
      ...original,
      plannedCost: '150000',
      endDate: '2026-06-30',
      description: 'Not a reason',
    };
    expect(reapprovalChanges(original, state, formatDate)).toEqual([
      { label: 'Planned cost', old: '120,000.00 USD', new: '150,000.00 USD' },
      { label: 'End date', old: '31 May 2026', new: '30 Jun 2026' },
    ]);
    expect(reapprovalChanges(original, { ...original, plannedCost: '120000' }, formatDate)).toEqual(
      [],
    );
  });
});
