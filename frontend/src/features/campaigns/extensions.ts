/**
 * The places where feature 006 adds its activities to feature 005's tree rows and initiative
 * panel (005 plan, "Plan tree integration"; build guide §5). 005 calls these names; 006 fills them
 * in its own folder (features/activities), so neither feature edits the other's files.
 *
 * - useRowsBelowInitiatives(planId): Map<initiativeId, TreeNode[]> — the activity rows under each
 *   initiative row (they start collapsed).
 * - initiativeExtraActions(initiative, navigate): ActionsMenuItem[] — **Add activity** in an
 *   initiative row's ⋯ menu and its details panel's More menu.
 * - InitiativeActivitiesSection — the "Activities" list and **Add activity** in the initiative
 *   details panel (props planId, initiativeId).
 */
export {
  activityActionsForInitiative as initiativeExtraActions,
  useActivityRows as useRowsBelowInitiatives,
} from '@/features/activities/activityTreeRows';
export { InitiativeActivities as InitiativeActivitiesSection } from '@/features/activities/InitiativeActivities';
