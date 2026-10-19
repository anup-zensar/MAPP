/**
 * The levels of an approval workflow as its form edits them, and back to what the API takes.
 */
import type { ApprovalWorkflow, WorkflowLevelInput } from './workflowsApi';

/** One level as the form edits it; `key` keeps React's rows apart while levels move. */
export interface LevelDraft {
  key: string;
  name: string;
  approverIds: string[];
}

let lastKey = 0;

/** A key no other level of this page has. */
export function newLevelKey(): string {
  lastKey += 1;
  return `level-${lastKey}`;
}

/** An empty level. */
export function emptyLevel(): LevelDraft {
  return { key: newLevelKey(), name: '', approverIds: [] };
}

/** The stored levels as drafts; one empty level for a new workflow. */
export function toDrafts(workflow?: ApprovalWorkflow): LevelDraft[] {
  if (!workflow) {
    return [emptyLevel()];
  }
  return workflow.levels.map((level) => ({
    key: newLevelKey(),
    name: level.name,
    approverIds: level.approvers.map((person) => person.id),
  }));
}

/** What the API takes: the levels in order, names trimmed. */
export function toInputs(levels: LevelDraft[]): WorkflowLevelInput[] {
  return levels.map((level) => ({ name: level.name.trim(), approverIds: level.approverIds }));
}

/**
 * The input names (and ids) of level `number`'s fields, as the API keys their problems
 * ("levels.<n>.approverIds", n counted from 1 as in the messages; data-model §1).
 */
export function levelFieldName(number: number, field: 'name' | 'approverIds'): string {
  return `levels.${number}.${field}`;
}
