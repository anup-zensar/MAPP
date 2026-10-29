// The sentences of feature 003 put together from data (plan "User-visible messages").
import type { Team, UsageCount } from './api';

const AND_LIST = new Intl.ListFormat('en-GB', { type: 'conjunction' });

export function peopleCountText(shown: number, total: number): string {
  return countText(shown, total, 'person', 'people');
}

export function teamsCountText(shown: number, total: number): string {
  return countText(shown, total, 'team', 'teams');
}

export function membersText(count: number): string {
  if (count === 0) {
    return 'No members';
  }
  return count === 1 ? '1 member' : `${count} members`;
}

export function plansText(count: number): string {
  if (count === 0) {
    return 'Not used';
  }
  return count === 1 ? '1 plan' : `${count} plans`;
}

export function usedByText(count: number): string {
  return count === 1 ? 'Used by 1 plan' : `Used by ${count} plans`;
}

export function usageText(usage: UsageCount[]): string {
  const { owner, approver } = usageLists(usage);
  if (!owner && !approver) {
    return "Doesn't own or approve anything yet.";
  }
  return [owner && `Owner of ${owner}`, approver && `approver in ${approver}`]
    .filter(Boolean)
    .join(' · ');
}

export function isInUse(usage: UsageCount[]): boolean {
  return usage.some((use) => use.count > 0);
}

export function ownsText(usage: UsageCount[]): string {
  return usageLists(usage).owner || 'Nothing yet';
}

export function approvesInText(usage: UsageCount[]): string {
  return usageLists(usage).approver || 'No approval workflows';
}

// Whom the person is line manager of: "2 people"; "" for nobody.
export function reportsText(usage: UsageCount[]): string {
  return usageLists(usage).manager;
}

export function deactivateSentences(person: { name: string; usage: UsageCount[] }): string[] {
  const first = firstName(person.name);
  const { owner, approver, manager } = usageLists(person.usage);
  const sentences = [`${first} won't be able to sign in or be chosen in people lists.`];
  if (owner || approver || manager) {
    const parts = [
      owner && `owns ${owner}`,
      approver && `approves in ${approver}`,
      manager && `is line manager of ${manager}`,
    ].filter(Boolean);
    sentences.push(
      `${first} ${AND_LIST.format(parts)}; these keep the name, shown as '${person.name} (inactive)'.`,
    );
  }
  sentences.push('Their teams are kept.');
  return sentences;
}

export function deleteTeamText(memberships: number): string {
  if (memberships === 0) {
    return 'It goes to the recycle bin. You can restore it.';
  }
  const what = memberships === 1 ? 'its team membership' : `its ${memberships} team memberships`;
  return `It goes to the recycle bin together with ${what}. You can restore it.`;
}

export function teamInUseText(team: Pick<Team, 'name' | 'plans'>): string {
  const count = team.plans.length;
  const plans = team.plans
    .map((plan) => (plan.inRecycleBin ? `${plan.name} (in the recycle bin)` : plan.name))
    .join(', ');
  const usedBy = count === 1 ? 'It is used by 1 plan' : `It is used by ${count} plans`;
  return (
    `${usedBy}: ${plans}. Set it inactive instead: it won't be offered for new plans, and ` +
    `these plans keep it, shown as '${team.name} (inactive)'.`
  );
}

export function firstName(name: string): string {
  return name.trim().split(/\s+/)[0] ?? name;
}

function countText(shown: number, total: number, one: string, many: string): string {
  if (shown < total) {
    return `${shown} of ${total} ${total === 1 ? one : many}`;
  }
  return total === 1 ? `1 ${one}` : `${total} ${many}`;
}

function usageLists(usage: UsageCount[]): { owner: string; approver: string; manager: string } {
  const list = (role: UsageCount['role']) =>
    AND_LIST.format(
      usage.filter((use) => use.role === role).map((use) => `${use.count} ${use.label}`),
    );
  return { owner: list('owner'), approver: list('approver'), manager: list('manager') };
}

export const SELF_DEACTIVATION = "You can't deactivate yourself — ask someone else to do it.";

export const DEACTIVATE_TIP = 'People are never deleted, so the history keeps their name.';

export const DISPLAY_CURRENCY_NOTE =
  "Amounts are shown in your display currency with today's rate in Settings; amounts typed in other currencies stay as they are.";

export function currencyLabel(currency: { code: string; name: string }): string {
  return `${currency.code} — ${currency.name}`;
}
