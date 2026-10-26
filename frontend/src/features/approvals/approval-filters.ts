/**
 * Search and filters of the approvals page (FR-031; Proposed decision 008-P12: they run in the web
 * app on the tab's rows). Choices come from the rows themselves, so no filter offers a value that
 * matches nothing.
 */
import type { ChoiceOption } from '@/ui/ChoiceField/ChoiceField';
import { localDate } from '@/format/intl';
import type { ApprovalRow } from './api';

export interface ApprovalFilterValues {
  planId: string;
  workflowId: string;
  ownerId: string;
  /** Submitted on or after this date ("YYYY-MM-DD"); "" for no limit. */
  from: string;
  /** Submitted on or before this date; "" for no limit. */
  to: string;
}

export type ApprovalFilterKey = keyof ApprovalFilterValues;

export const NO_FILTERS: ApprovalFilterValues = {
  planId: '',
  workflowId: '',
  ownerId: '',
  from: '',
  to: '',
};

export interface ApprovalFilterOptions {
  plans: ChoiceOption[];
  workflows: ChoiceOption[];
  owners: ChoiceOption[];
}

/** The plans, workflows and owners of the rows, each once, sorted by name. */
export function filterOptions(rows: ApprovalRow[]): ApprovalFilterOptions {
  const plans = new Map<string, string>();
  const workflows = new Map<string, string>();
  const owners = new Map<string, string>();
  for (const row of rows) {
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
  return { plans: sorted(plans), workflows: sorted(workflows), owners: sorted(owners) };
}

/** The rows that match the search (name or ID, ignoring letter case) and every filter. */
export function applyFilters(
  rows: ApprovalRow[],
  search: string,
  filters: ApprovalFilterValues,
): ApprovalRow[] {
  const text = search.trim().toLowerCase();
  return rows.filter((row) => {
    const submitted = localDate(row.submittedAt);
    return (
      (text === '' ||
        row.activity.name.toLowerCase().includes(text) ||
        row.activity.id.toLowerCase().includes(text)) &&
      (filters.planId === '' || row.path[0]?.id === filters.planId) &&
      (filters.workflowId === '' || row.workflow.id === filters.workflowId) &&
      (filters.ownerId === '' || row.owner?.id === filters.ownerId) &&
      (filters.from === '' || submitted >= filters.from) &&
      (filters.to === '' || submitted <= filters.to)
    );
  });
}

/** How many filters are in use, for "Filters (2)". */
export function filtersInUse(filters: ApprovalFilterValues): number {
  return Object.values(filters).filter((value) => value !== '').length;
}

/** The filters in use as chips: "Plan: FY2026 Lubricants Europe", "Submitted from: 01 Mar 2026". */
export function filterChips(
  filters: ApprovalFilterValues,
  options: ApprovalFilterOptions,
  formatDate: (date: string) => string,
): { key: ApprovalFilterKey; label: string }[] {
  const nameOf = (list: ChoiceOption[], id: string) =>
    list.find((option) => option.id === id)?.label ?? id;
  const chips: { key: ApprovalFilterKey; label: string }[] = [];
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
