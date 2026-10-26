import { Link } from 'react-router';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { recordHref } from '@/lib/recordLinks';
import { joinWithAnd } from '@/lib/text';
import type { PlanChangeEffects } from '../plansApi';
import styles from './dialogs.module.css';

type GroupChange = PlanChangeEffects['hiddenGroups'][number];

const LEVEL_WORDS: Record<GroupChange['levels'][number], string> = {
  plan: 'plan',
  campaign: 'its campaigns',
  initiative: 'its initiatives',
  activity: 'its activities',
  lineItem: 'its line items',
};

/** "plan", "plan and its activities", "plan, its campaigns and its activities". */
function levelsText(levels: GroupChange['levels']): string {
  return joinWithAnd(levels.map((level) => LEVEL_WORDS[level]));
}

export interface ChangeEffectsDialogProps {
  /** What saving would change; null while the dialog is closed. */
  effects: PlanChangeEffects | null;
  planId: string;
  /** Only the model changed: the button says Change model. */
  modelOnly: boolean;
  onConfirm: () => void;
  onKeepEditing: () => void;
}

/**
 * What a new model or business unit changes, before saving (FR-019, FR-020): the activities whose
 * type the new model doesn't allow, with links, and the Shell attribute groups that will be hidden
 * or shown. **Change model** (or **Save changes**) saves; **Keep editing** goes back to the form.
 */
export function ChangeEffectsDialog({
  effects,
  planId,
  modelOnly,
  onConfirm,
  onKeepEditing,
}: ChangeEffectsDialogProps) {
  const activities = effects?.disallowedActivities ?? [];
  return (
    <ConfirmDialog
      open={effects !== null}
      onOpenChange={(open) => !open && onKeepEditing()}
      title={modelOnly ? 'Change the model?' : 'Save these changes?'}
      confirmLabel={modelOnly ? 'Change model' : 'Save changes'}
      cancelLabel="Keep editing"
      onConfirm={onConfirm}
    >
      {activities.length > 0 && (
        <>
          <p>
            {activities.length === 1
              ? "1 activity has a type the new model doesn't allow:"
              : `${activities.length} activities have a type the new model doesn't allow:`}
          </p>
          <ul className={styles.list}>
            {activities.map((activity) => (
              <li key={activity.id}>
                <Link to={recordHref('activity', activity.id, planId)}>{activity.name}</Link>
              </li>
            ))}
          </ul>
          <p>
            They keep their type but need an allowed type before they can be submitted. Goals will
            be recalculated with the new model.
          </p>
        </>
      )}
      {effects?.hiddenGroups.map((group) => (
        <p key={`hidden-${group.unit}`}>
          {group.unit} attributes will be hidden on the {levelsText(group.levels)}. Their values are
          kept and come back if you add {group.unit} again.
        </p>
      ))}
      {effects?.shownGroups.map((group) => (
        <p key={`shown-${group.unit}`}>
          {group.unit} attributes will be shown on the {levelsText(group.levels)}.
        </p>
      ))}
    </ConfirmDialog>
  );
}
