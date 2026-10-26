import { describe, expect, it } from 'vitest';
import { formatDate } from '@/format/intl';
import { launchFilm } from '@/testing/fixtures/activities';
import { newForm } from '@/testing/fixtures/activity-forms';
import {
  fromActivity,
  fromDefaults,
  reapprovalChanges,
  toCreateBody,
  toUpdateBody,
} from './activity-form-state';
import type { FormState } from './activity-form-state';

function newState(overrides: Partial<FormState> = {}): FormState {
  return {
    ...fromDefaults(newForm()),
    name: 'Trade show',
    activityTypeId: 'AT-1001',
    ...overrides,
  };
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
