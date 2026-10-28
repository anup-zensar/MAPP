import { formatAmount, parseMoneyInput } from '@/format/decimal';
import type {
  ApprovalCondition,
  ApprovalConditionInput,
  ApprovalWorkflow,
  ApprovalWorkflowLevel,
  WorkflowLevelInput,
} from './api';

export type ApproverType = 'people' | 'manager' | 'role' | 'owner' | 'recordField';

export type LevelMode = 'any' | 'all';

export interface ConditionDraft {
  plannedCostAtLeast: string;
  plannedCostBelow: string;
  businessUnitIds: string[];
  activityTypeIds: string[];
}

export interface LevelDraft {
  key: string;
  name: string;
  approverType: ApproverType;
  approverIds: string[];
  roleId: string;
  recordField: string;
  mode: LevelMode;
  onlyWhen: boolean;
  appliesWhen: ConditionDraft;
  autoWhen: boolean;
  autoApproveWhen: ConditionDraft;
  backupApproverId: string;
}

export type LevelField =
  | 'name'
  | 'approverType'
  | 'approverIds'
  | 'roleId'
  | 'recordField'
  | 'mode'
  | 'appliesWhen'
  | 'autoApproveWhen'
  | 'backupApproverId';

let lastKey = 0;

export function newLevelKey(): string {
  lastKey += 1;
  return `level-${lastKey}`;
}

export function emptyCondition(): ConditionDraft {
  return { plannedCostAtLeast: '', plannedCostBelow: '', businessUnitIds: [], activityTypeIds: [] };
}

export function emptyLevel(): LevelDraft {
  return {
    key: newLevelKey(),
    name: '',
    approverType: 'people',
    approverIds: [],
    roleId: '',
    recordField: '',
    mode: 'any',
    onlyWhen: false,
    appliesWhen: emptyCondition(),
    autoWhen: false,
    autoApproveWhen: emptyCondition(),
    backupApproverId: '',
  };
}

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
    recordField: level.recordField?.id ?? '',
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

export function toInputs(levels: LevelDraft[]): WorkflowLevelInput[] {
  return levels.map((level) => {
    const input: WorkflowLevelInput = {
      name: level.name.trim(),
      approverType: level.approverType,
      approverIds: level.approverType === 'people' ? level.approverIds : [],
      mode: singlePerson(level.approverType) ? 'any' : level.mode,
      appliesWhen: level.onlyWhen ? conditionInput(level.appliesWhen) : null,
      autoApproveWhen: level.autoWhen ? conditionInput(level.autoApproveWhen) : null,
      backupApproverId: level.backupApproverId,
    };
    if (level.approverType === 'role') {
      input.roleId = level.roleId;
    }
    if (level.approverType === 'recordField') {
      input.recordField = level.recordField;
    }
    return input;
  });
}

/** A manager, a record owner or a person named on the record is one person, so the level has no mode. */
export function singlePerson(type: ApproverType): boolean {
  return type === 'manager' || type === 'owner' || type === 'recordField';
}

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

export function levelFieldName(number: number, field: LevelField): string {
  return `levels.${number}.${field}`;
}
