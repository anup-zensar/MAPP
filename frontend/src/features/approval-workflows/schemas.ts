import { z } from 'zod';
import { parseWholeInput } from '@/format/decimal';
import type { ApprovalWorkflow, WorkflowCreate, WorkflowUpdate } from './api';
import { toDrafts, toInputs } from './level-drafts';
import type { LevelDraft } from './level-drafts';

/** New workflows start with a reminder after 3 days and escalation after 5 (017 FR-007). */
const NEW_REMINDER_DAYS = 3;
const NEW_ESCALATION_DAYS = 5;

/** Days that are no whole number can't be sent: said at once, in the API's words. */
export const workflowSchema = z
  .object({
    name: z.string(),
    description: z.string(),
    levels: z.array(z.custom<LevelDraft>()),
    sendReminder: z.boolean(),
    reminderDays: z.string(),
    escalate: z.boolean(),
    escalationDays: z.string(),
  })
  .superRefine((values, context) => {
    if (timingValue(values.sendReminder, values.reminderDays) === undefined) {
      context.addIssue({
        code: 'custom',
        path: ['reminderDays'],
        message: 'Remind after: enter a whole number of days from 1 to 60.',
      });
    }
    if (timingValue(values.escalate, values.escalationDays) === undefined) {
      context.addIssue({
        code: 'custom',
        path: ['escalationDays'],
        message: 'Escalate after: enter a whole number of days from 1 to 90.',
      });
    }
  });

export type WorkflowValues = z.infer<typeof workflowSchema>;

export function workflowValuesOf(workflow: ApprovalWorkflow | undefined): WorkflowValues {
  const reminderDays = workflow ? workflow.reminderDays : NEW_REMINDER_DAYS;
  const escalationDays = workflow ? workflow.escalationDays : NEW_ESCALATION_DAYS;
  return {
    name: workflow?.name ?? '',
    description: workflow?.description ?? '',
    levels: toDrafts(workflow),
    sendReminder: Boolean(reminderDays),
    reminderDays: String(reminderDays || NEW_REMINDER_DAYS),
    escalate: Boolean(escalationDays),
    escalationDays: String(escalationDays || NEW_ESCALATION_DAYS),
  };
}

/** Something was changed that the API would see: level fields it ignores do not count. */
export function isWorkflowChanged(values: WorkflowValues, start: WorkflowValues): boolean {
  return (
    values.name !== start.name ||
    values.description !== start.description ||
    levelsChanged(values, start) ||
    values.sendReminder !== start.sendReminder ||
    values.reminderDays !== start.reminderDays ||
    values.escalate !== start.escalate ||
    values.escalationDays !== start.escalationDays
  );
}

export function newWorkflow(values: WorkflowValues): WorkflowCreate {
  return {
    name: values.name,
    description: values.description,
    levels: toInputs(values.levels),
    reminderDays: timingValue(values.sendReminder, values.reminderDays) ?? null,
    escalationDays: timingValue(values.escalate, values.escalationDays) ?? null,
  };
}

export function workflowChanges(
  values: WorkflowValues,
  start: WorkflowValues,
  version: number,
): WorkflowUpdate {
  const changes: WorkflowUpdate = { version };
  if (values.name !== start.name) {
    changes.name = values.name;
  }
  if (values.description !== start.description) {
    changes.description = values.description;
  }
  if (levelsChanged(values, start)) {
    changes.levels = toInputs(values.levels);
  }
  const reminderDays = timingValue(values.sendReminder, values.reminderDays);
  if (reminderDays !== timingValue(start.sendReminder, start.reminderDays)) {
    changes.reminderDays = reminderDays ?? null;
  }
  const escalationDays = timingValue(values.escalate, values.escalationDays);
  if (escalationDays !== timingValue(start.escalate, start.escalationDays)) {
    changes.escalationDays = escalationDays ?? null;
  }
  return changes;
}

function levelsChanged(values: WorkflowValues, start: WorkflowValues): boolean {
  return JSON.stringify(toInputs(values.levels)) !== JSON.stringify(toInputs(start.levels));
}

/** The days for the API: null when unticked; undefined when the typed text is no whole number. */
function timingValue(on: boolean, days: string): number | null | undefined {
  if (!on) {
    return null;
  }
  const whole = parseWholeInput(days);
  return whole === null ? undefined : Number(whole);
}
