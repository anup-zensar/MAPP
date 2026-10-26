/**
 * The levels of an approval workflow as its form edits them, and back to what the API takes
 * (008 FR-002, 017 FR-001 – FR-006). A level stored before feature 017 has none of its fields and
 * reads with the defaults of 017 data-model §10: named people, any one approves, no conditions,
 * no backup approver.
 */
import { formatAmount, parseMoneyInput } from '@/format/decimal';
import type {
  ApprovalCondition,
  ApprovalConditionInput,
  ApprovalWorkflow,
  ApprovalWorkflowLevel,
  WorkflowLevelInput,
} from './workflowsApi';

/** Who approves a level (017 FR-001). */
export type ApproverType = 'people' | 'manager' | 'role';

/** When a level has several approvers: the first approval decides, or every approver must. */
export type LevelMode = 'any' | 'all';

/** A condition as its fields hold it; amounts as typed ("100,000.00"). */
export interface ConditionDraft {
  plannedCostAtLeast: string;
  plannedCostBelow: string;
  businessUnitIds: string[];
  activityTypeIds: string[];
}

/** One level as the form edits it; `key` keeps React's rows apart while levels move. */
export interface LevelDraft {
  key: string;
  name: string;
  approverType: ApproverType;
  approverIds: string[];
  roleId: string;
  mode: LevelMode;
  /** "Only needed when" is ticked; its condition is kept while unticked, so ticking brings it back. */
  onlyWhen: boolean;
  appliesWhen: ConditionDraft;
  /** "Approve automatically when" is ticked. */
  autoWhen: boolean;
  autoApproveWhen: ConditionDraft;
  backupApproverId: string;
}

/** The fields of a level that the API keys problems by ("levels.<n>.<field>"). */
export type LevelField =
  | 'name'
  | 'approverType'
  | 'approverIds'
  | 'roleId'
  | 'mode'
  | 'appliesWhen'
  | 'autoApproveWhen'
  | 'backupApproverId';

let lastKey = 0;

/** A key no other level of this page has. */
export function newLevelKey(): string {
  lastKey += 1;
  return `level-${lastKey}`;
}

/** A condition with nothing set. */
export function emptyCondition(): ConditionDraft {
  return { plannedCostAtLeast: '', plannedCostBelow: '', businessUnitIds: [], activityTypeIds: [] };
}

/** An empty level: named people, any one approves, no conditions. */
export function emptyLevel(): LevelDraft {
  return {
    key: newLevelKey(),
    name: '',
    approverType: 'people',
    approverIds: [],
    roleId: '',
    mode: 'any',
    onlyWhen: false,
    appliesWhen: emptyCondition(),
    autoWhen: false,
    autoApproveWhen: emptyCondition(),
    backupApproverId: '',
  };
}

/** The stored levels as drafts; one empty level for a new workflow. */
export function toDrafts(workflow?: ApprovalWorkflow): LevelDraft[] {
  if (!workflow) {
    return [emptyLevel()];
  }
  return workflow.levels.map(toDraft);
}

function toDraft(level: ApprovalWorkflowLevel): LevelDraft {
  const approverType = level.approverType ?? 'people';
  return {
    key: newLevelKey(),
    name: level.name,
    approverType,
    approverIds: approverType === 'people' ? level.approvers.map((person) => person.id) : [],
    roleId: level.role?.id ?? '',
    mode: level.mode ?? 'any',
    onlyWhen: Boolean(level.appliesWhen),
    appliesWhen: conditionDraft(level.appliesWhen),
    autoWhen: Boolean(level.autoApproveWhen),
    autoApproveWhen: conditionDraft(level.autoApproveWhen),
    backupApproverId: level.backupApprover?.id ?? '',
  };
}

function conditionDraft(condition: ApprovalCondition | null | undefined): ConditionDraft {
  if (!condition) {
    return emptyCondition();
  }
  return {
    plannedCostAtLeast: condition.plannedCostAtLeast
      ? formatAmount(condition.plannedCostAtLeast.amount)
      : '',
    plannedCostBelow: condition.plannedCostBelow
      ? formatAmount(condition.plannedCostBelow.amount)
      : '',
    businessUnitIds: condition.businessUnits.map((unit) => unit.id),
    activityTypeIds: condition.activityTypes.map((type) => type.id),
  };
}

/**
 * What the API takes: the levels in order, names trimmed, only the fields of the chosen source
 * (the line manager always "any one approves"), and a condition only while its box is ticked.
 */
export function toInputs(levels: LevelDraft[]): WorkflowLevelInput[] {
  return levels.map((level) => {
    const input: WorkflowLevelInput = {
      name: level.name.trim(),
      approverType: level.approverType,
      approverIds: level.approverType === 'people' ? level.approverIds : [],
      mode: level.approverType === 'manager' ? 'any' : level.mode,
      appliesWhen: level.onlyWhen ? conditionInput(level.appliesWhen) : null,
      autoApproveWhen: level.autoWhen ? conditionInput(level.autoApproveWhen) : null,
      backupApproverId: level.backupApproverId,
    };
    if (level.approverType === 'role') {
      input.roleId = level.roleId;
    }
    return input;
  });
}

/**
 * A condition for the API: amounts as plain decimals ("100000.00"), or as typed when they are not
 * amounts, so the API names the problem; an empty amount is no part of the condition.
 */
function conditionInput(condition: ConditionDraft): ApprovalConditionInput {
  return {
    plannedCostAtLeast: amountInput(condition.plannedCostAtLeast),
    plannedCostBelow: amountInput(condition.plannedCostBelow),
    businessUnitIds: condition.businessUnitIds,
    activityTypeIds: condition.activityTypeIds,
  };
}

function amountInput(text: string): string | null {
  const trimmed = text.trim();
  if (trimmed === '') {
    return null;
  }
  return parseMoneyInput(trimmed)?.amount ?? trimmed;
}

/**
 * The input names (and ids) of level `number`'s fields, as the API keys their problems
 * ("levels.<n>.approverIds", n counted from 1 as in the messages; data-model §1).
 */
export function levelFieldName(number: number, field: LevelField): string {
  return `levels.${number}.${field}`;
}
