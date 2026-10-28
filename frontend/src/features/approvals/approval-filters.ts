/**
 * Search and filters of the approvals page: they run on the tab's rows (008-P12), and their choices come
 * from the rows, so no filter offers a value that matches nothing.
 */
import type { ChoiceOption } from '@/ui/ChoiceField/ChoiceField';
import { localDate } from '@/format/intl';
import type { ApprovalRow } from './api';

export interface ApprovalFilterValues {
  kind: string;
  planId: string;
  workflowId: string;
  ownerId: string;
  from: string;
  to: string;
}

export type ApprovalFilterKey = keyof ApprovalFilterValues;

export const NO_FILTERS: ApprovalFilterValues = {
  kind: '',
  planId: '',
  workflowId: '',
  ownerId: '',
  from: '',
  to: '',
};

export function isFilterKey(key: string): key is ApprovalFilterKey {
  return Object.hasOwn(NO_FILTERS, key);
}

export interface ApprovalFilterOptions {
  kinds: ChoiceOption[];
  plans: ChoiceOption[];
  workflows: ChoiceOption[];
  owners: ChoiceOption[];
}

export function filterOptions(rows: ApprovalRow[]): ApprovalFilterOptions {
  const kinds = new Map<string, string>();
  const plans = new Map<string, string>();
  const workflows = new Map<string, string>();
  const owners = new Map<string, string>();
  for (const row of rows) {
    kinds.set(row.subject.kind, row.subject.label);
    const plan = row.path[0];
    if (plan) {
      plans.set(plan.id, plan.name);
    }
    const workflowName = row.workflow.active
      ? row.workflow.name
      : `${row.workflow.name} (inactive)`;
    workflows.set(row.workflow.id, workflowName);
    if (row.owner) {
      owners.set(row.owner.id, row.owner.name);
    }
  }
  return {
    kinds: sorted(kinds),
    plans: sorted(plans),
    workflows: sorted(workflows),
    owners: sorted(owners),
  };
}

export function applyFilters(
  rows: ApprovalRow[],
  search: string,
  filters: ApprovalFilterValues,
): ApprovalRow[] {
  const text = search.trim().toLowerCase();
  return rows.filter((row) => matchesSearch(row, text) && matchesFilters(row, filters));
}

function matchesSearch(row: ApprovalRow, text: string): boolean {
  return (
    text === '' ||
    row.subject.name.toLowerCase().includes(text) ||
    row.subject.id.toLowerCase().includes(text)
  );
}

function matchesFilters(row: ApprovalRow, filters: ApprovalFilterValues): boolean {
  const submitted = localDate(row.submittedAt);
  return (
    (filters.kind === '' || row.subject.kind === filters.kind) &&
    (filters.planId === '' || row.path[0]?.id === filters.planId) &&
    (filters.workflowId === '' || row.workflow.id === filters.workflowId) &&
    (filters.ownerId === '' || row.owner?.id === filters.ownerId) &&
    (filters.from === '' || submitted >= filters.from) &&
    (filters.to === '' || submitted <= filters.to)
  );
}

export function filtersInUse(filters: ApprovalFilterValues): number {
  return Object.values(filters).filter((value) => value !== '').length;
}

export function filterChips(
  filters: ApprovalFilterValues,
  options: ApprovalFilterOptions,
  formatDate: (date: string) => string,
): { key: ApprovalFilterKey; label: string }[] {
  const nameOf = (list: ChoiceOption[], id: string) =>
    list.find((option) => option.id === id)?.label ?? id;
  const chips: { key: ApprovalFilterKey; label: string }[] = [];
  if (filters.kind) {
    chips.push({ key: 'kind', label: `Record type: ${nameOf(options.kinds, filters.kind)}` });
  }
  if (filters.planId) {
    chips.push({ key: 'planId', label: `Plan: ${nameOf(options.plans, filters.planId)}` });
  }
  if (filters.workflowId) {
    const name = nameOf(options.workflows, filters.workflowId);
    chips.push({ key: 'workflowId', label: `Approval workflow: ${name}` });
  }
  if (filters.ownerId) {
    chips.push({ key: 'ownerId', label: `Owner: ${nameOf(options.owners, filters.ownerId)}` });
  }
  if (filters.from) {
    chips.push({ key: 'from', label: `Submitted from: ${formatDate(filters.from)}` });
  }
  if (filters.to) {
    chips.push({ key: 'to', label: `Submitted to: ${formatDate(filters.to)}` });
  }
  return chips;
}

function sorted(entries: Map<string, string>): ChoiceOption[] {
  return [...entries]
    .map(([id, label]) => ({ id, label, active: true }))
    .sort((a, b) => a.label.localeCompare(b.label, 'en', { sensitivity: 'base' }));
}
