/**
 * The sentences of feature 003 that are put together from data (plan "User-visible messages"):
 * counts, what a person owns and approves, the deactivate dialog and the delete-team dialog.
 */
import type { Team, UsageCount } from './peopleApi';

/** "2 plans and 5 activities", "2 plans, 5 activities and 1 initiative". */
const AND_LIST = new Intl.ListFormat('en-GB', { type: 'conjunction' });

/** "1 person" · "6 people" · "2 of 6 people" while a search or filter hides some. */
export function peopleCountText(shown: number, total: number): string {
  return countText(shown, total, 'person', 'people');
}

/** "1 team" · "3 teams" · "1 of 3 teams" while a search or filter hides some. */
export function teamsCountText(shown: number, total: number): string {
  return countText(shown, total, 'team', 'teams');
}

/** A team's members in the list: "1 member", "4 members" or "No members". */
export function membersText(count: number): string {
  if (count === 0) {
    return 'No members';
  }
  return count === 1 ? '1 member' : `${count} members`;
}

/** The plans using a team, in the list: "1 plan", "2 plans" or "Not used". */
export function plansText(count: number): string {
  if (count === 0) {
    return 'Not used';
  }
  return count === 1 ? '1 plan' : `${count} plans`;
}

/** The plans using a team, in its details: "Used by 1 plan", "Used by 2 plans". */
export function usedByText(count: number): string {
  return count === 1 ? 'Used by 1 plan' : `Used by ${count} plans`;
}

/**
 * What a person owns and approves, in their details: "Owner of 2 plans and 5 activities ·
 * approver in 1 approval workflow" (one part alone when the other is empty).
 */
export function usageText(usage: UsageCount[]): string {
  const { owner, approver } = usageLists(usage);
  if (!owner && !approver) {
    return "Doesn't own or approve anything yet.";
  }
  return [owner && `Owner of ${owner}`, approver && `approver in ${approver}`]
    .filter(Boolean)
    .join(' · ');
}

/** What the deactivate dialog says about a person, sentence by sentence (FR-005). */
export function deactivateSentences(person: { name: string; usage: UsageCount[] }): string[] {
  const first = firstName(person.name);
  const { owner, approver } = usageLists(person.usage);
  const sentences = [`${first} won't be able to sign in or be chosen in people lists.`];
  if (owner || approver) {
    const parts = [owner && `owns ${owner}`, approver && `approves in ${approver}`].filter(Boolean);
    sentences.push(
      `${first} ${parts.join(' and ')}; these keep the name, shown as '${person.name} (inactive)'.`,
    );
  }
  sentences.push('Their teams are kept.');
  return sentences;
}

/** What goes to the recycle bin with an unused team (delete dialog). */
export function deleteTeamText(memberships: number): string {
  if (memberships === 0) {
    return 'It goes to the recycle bin. You can restore it.';
  }
  const what = memberships === 1 ? 'its team membership' : `its ${memberships} team memberships`;
  return `It goes to the recycle bin together with ${what}. You can restore it.`;
}

/** Why a team that plans use can't be deleted, and what to do instead. */
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

/** "Priya" of "Priya Raman". */
export function firstName(name: string): string {
  return name.trim().split(/\s+/)[0] ?? name;
}

function countText(shown: number, total: number, one: string, many: string): string {
  if (shown < total) {
    return `${shown} of ${total} ${total === 1 ? one : many}`;
  }
  return total === 1 ? `1 ${one}` : `${total} ${many}`;
}

/** The owner and approver lists of a person's usage, "" for a role with nothing. */
function usageLists(usage: UsageCount[]): { owner: string; approver: string } {
  const list = (role: UsageCount['role']) =>
    AND_LIST.format(
      usage.filter((use) => use.role === role).map((use) => `${use.count} ${use.label}`),
    );
  return { owner: list('owner'), approver: list('approver') };
}

/** Said wherever someone tries to deactivate themselves, by the web app and the API (FR-006). */
export const SELF_DEACTIVATION = "You can't deactivate yourself — ask someone else to do it.";

/** Said under Deactivate in the ⋯ menu and the details panel (FR-008). */
export const DEACTIVATE_TIP = 'People are never deleted, so the history keeps their name.';
