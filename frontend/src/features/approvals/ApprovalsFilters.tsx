import { SlidersHorizontal } from 'lucide-react';
import { Popover } from 'radix-ui';
import { useId } from 'react';
import { Button, ChoiceField } from '@/components';
import { NO_FILTERS, filtersInUse } from './approvalFilters';
import type { ApprovalFilterOptions, ApprovalFilterValues } from './approvalFilters';
import styles from './ApprovalsFilters.module.css';

export interface ApprovalsFiltersProps {
  filters: ApprovalFilterValues;
  /** The choices, from the tab's rows. */
  options: ApprovalFilterOptions;
  onChange: (filters: ApprovalFilterValues) => void;
}

/**
 * **Filters** (FR-031): a popover with Plan, Approval workflow and Owner — the choices the tab's
 * rows have — and the submitted dates. The button says how many are in use: "Filters (2)". The
 * chips under the toolbar remove them one by one.
 */
export function ApprovalsFilters({ filters, options, onChange }: ApprovalsFiltersProps) {
  const id = useId();
  const inUse = filtersInUse(filters);
  const set = (change: Partial<ApprovalFilterValues>) => onChange({ ...filters, ...change });

  return (
    <Popover.Root>
      <Popover.Trigger asChild>
        <Button icon={<SlidersHorizontal />}>{inUse > 0 ? `Filters (${inUse})` : 'Filters'}</Button>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content
          className={styles.popover}
          align="start"
          sideOffset={8}
          aria-label="Filters"
        >
          <div className={styles.fields}>
            <ChoiceField
              label="Plan"
              name={`${id}-plan`}
              options={options.plans}
              value={filters.planId}
              onChange={(planId) => set({ planId })}
            />
            <ChoiceField
              label="Approval workflow"
              name={`${id}-workflow`}
              options={options.workflows}
              value={filters.workflowId}
              onChange={(workflowId) => set({ workflowId })}
            />
            <ChoiceField
              label="Owner"
              name={`${id}-owner`}
              options={options.owners}
              value={filters.ownerId}
              onChange={(ownerId) => set({ ownerId })}
            />
            <div className={styles.dates}>
              <DateInput
                id={`${id}-from`}
                label="Submitted from"
                value={filters.from}
                max={filters.to || undefined}
                onChange={(from) => set({ from })}
              />
              <DateInput
                id={`${id}-to`}
                label="Submitted to"
                value={filters.to}
                min={filters.from || undefined}
                onChange={(to) => set({ to })}
              />
            </div>
          </div>
          {inUse > 0 && (
            <div className={styles.footer}>
              <Button variant="ghost" size="sm" onClick={() => onChange(NO_FILTERS)}>
                Clear filters
              </Button>
            </div>
          )}
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  );
}

interface DateInputProps {
  id: string;
  label: string;
  value: string;
  min?: string;
  max?: string;
  onChange: (value: string) => void;
}

function DateInput({ id, label, value, min, max, onChange }: DateInputProps) {
  return (
    <div className={styles.date}>
      <label htmlFor={id} className={styles.label}>
        {label}
      </label>
      <input
        id={id}
        type="date"
        className={styles.input}
        value={value}
        min={min}
        max={max}
        onChange={(event) => onChange(event.target.value)}
      />
    </div>
  );
}
