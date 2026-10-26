import { joinWith } from '@/session/access';

/** Where the person can change nothing: a plan and its contents, Settings, a global budget, a record. */
export type ViewOnlyArea = 'plan' | 'settings' | 'globalBudget' | 'record';

/**
 * The sentence of the note (014 plan "User-visible messages" 33–35): "View only — your role (Viewer)
 * lets you see this plan but not change it. Ask an administrator (Alex Morgan) if you need to change
 * it." roleNames and administratorNames are sorted by name.
 */
export function viewOnlyText(
  area: ViewOnlyArea,
  roleNames: string[],
  administratorNames: string[],
): string {
  const admins = joinWith(administratorNames, 'or');
  switch (area) {
    case 'settings':
      return `View only — only administrators change Settings. Ask ${admins || 'an administrator'} if something needs to change.`;
    case 'globalBudget':
      return 'View only — you can see this global budget but not change it.';
    case 'plan':
    case 'record': {
      const what = area === 'plan' ? 'this plan' : 'this record';
      const ask = admins
        ? `Ask an administrator (${admins}) if you need to change it.`
        : 'Ask an administrator if you need to change it.';
      if (roleNames.length === 0) {
        return `View only — you can see ${what} but not change it. ${ask}`;
      }
      const roles =
        roleNames.length === 1
          ? `your role (${roleNames[0]}) lets`
          : `your roles (${roleNames.join(', ')}) let`;
      return `View only — ${roles} you see ${what} but not change it. ${ask}`;
    }
  }
}
