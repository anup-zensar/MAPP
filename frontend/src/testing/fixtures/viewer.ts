import type { components } from '@/api/generated/schema';
import type { CurrentUser } from '@/session/useSession';
import { testUser } from './users';

type Availability = components['schemas']['ActionAvailability'];

export const ALLOWED: Availability = { available: true, reason: '' };

/** The API's refusal when no role of the person includes the action (014 message 2). */
export function refused(noun: string, role = 'Viewer'): Availability {
  return {
    available: false,
    reason: `${noun} isn't part of your role (${role}). Ask an administrator if you need it.`,
  };
}

export function viewOnlyNote(what: 'this plan' | 'this record'): string {
  return `View only — your role (Viewer) lets you see ${what} but not change it. Ask an administrator (Alex Morgan) if you need to change it.`;
}

export const VIEW_ONLY_NOTE = viewOnlyNote('this record');

/** Sara Novak, a Viewer of Lubricants Europe: she sees its plans and changes nothing. */
export const VIEWER: CurrentUser = testUser({
  id: 'U-1006',
  name: 'Sara Novak',
  firstName: 'Sara',
  email: 'sara.novak@shell.com',
  access: {
    roles: [
      {
        id: 'RA-1007',
        role: { id: 'RL-1005', name: 'Viewer', builtIn: 'viewer' },
        scope: { type: 'myTeams', teams: [], text: 'My teams' },
        scopeEmpty: false,
      },
    ],
    permissions: [
      { key: 'plans.view', all: false, teams: [] },
      { key: 'data.export', all: false, teams: [] },
      { key: 'finance.view', all: false, teams: [] },
    ],
    administrators: [
      {
        id: 'U-1001',
        name: 'Alex Morgan',
        initials: 'AM',
        jobTitle: 'Administrator',
        active: true,
      },
    ],
    noSelfApproval: true,
  },
});

const EDIT_ACTIVITIES = refused('Changing activities and line items');

export const VIEWER_ACTIVITY_ACCESS: components['schemas']['ActivityAccess'] = {
  edit: EDIT_ACTIVITIES,
  delete: refused('Deleting activities and line items'),
  move: EDIT_ACTIVITIES,
  copy: EDIT_ACTIVITIES,
  addLineItem: EDIT_ACTIVITIES,
  editLineItems: EDIT_ACTIVITIES,
  deleteLineItems: refused('Deleting activities and line items'),
  submit: refused('Submitting activities'),
  seeFinance: ALLOWED,
};

const EDIT_CAMPAIGNS = refused('Changing campaigns and initiatives');

export const VIEWER_CONTENT_ACCESS: components['schemas']['PlanContentAccess'] = {
  addCampaign: EDIT_CAMPAIGNS,
  editCampaigns: EDIT_CAMPAIGNS,
  deleteCampaigns: refused('Deleting campaigns and initiatives'),
  addActivity: EDIT_ACTIVITIES,
  editActivities: EDIT_ACTIVITIES,
  deleteActivities: refused('Deleting activities and line items'),
  submitActivities: refused('Submitting activities'),
};

export const VIEWER_PLAN_ACCESS: components['schemas']['PlanAccess'] = {
  edit: refused('Editing plans'),
  editBudget: refused('Editing plans'),
  delete: refused('Deleting plans'),
  publish: refused('Publishing plans'),
  copy: refused('Creating plans'),
  content: VIEWER_CONTENT_ACCESS,
};

export const VIEWER_CONTAINER_ACCESS: components['schemas']['ContainerAccess'] = {
  edit: EDIT_CAMPAIGNS,
  delete: refused('Deleting campaigns and initiatives'),
  move: EDIT_CAMPAIGNS,
  copy: EDIT_CAMPAIGNS,
  addChild: EDIT_CAMPAIGNS,
};

export const VIEWER_LINE_ITEM_ACCESS: components['schemas']['LineItemAccess'] = {
  add: EDIT_ACTIVITIES,
  edit: EDIT_ACTIVITIES,
  delete: refused('Deleting activities and line items'),
  duplicate: EDIT_ACTIVITIES,
  seeFinance: ALLOWED,
};
