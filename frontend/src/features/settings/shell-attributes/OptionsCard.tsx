import { Download, Plus, Upload } from 'lucide-react';
import { useState } from 'react';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { SearchField } from '@/ui/SearchField/SearchField';
import { SegmentedFilter } from '@/ui/SegmentedFilter/SegmentedFilter';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { useToast } from '@/ui/Toast/useToast';
import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import { formatWhole } from '@/format/decimal';
import { SettingDeleteDialog } from '../shared/SettingDeleteDialog';
import { ImportOptionsDialog } from './ImportOptionsDialog';
import { downloadOptionsTemplate } from './importTemplate';
import { OptionPanel } from './OptionPanel';
import {
  CHANGED_BY_ANY_CHANGE,
  useDeleteOption,
  useMoveOption,
  useSetOptionStatus,
  useShellAttributeOptions,
} from './shellAttributesApi';
import type { OptionStatus, ShellAttribute, ShellAttributeOption } from './shellAttributesApi';
import styles from './ShellAttributes.module.css';
import { useFailureToast, useStatusChange } from './useStatusChange';

const STATUS_OPTIONS: { value: OptionStatus; label: string }[] = [
  { value: 'active', label: 'Active' },
  { value: 'inactive', label: 'Inactive' },
  { value: 'all', label: 'All' },
];

/** The side panel that is open: a new option, or the option to rename. */
type OpenPanel = { option?: ShellAttributeOption };

/**
 * The Options card of a choice attribute (feature 002, FR-018, FR-020, FR-031, SC-007): its
 * options in display order with their own "Used by", searched by the API (any part of the label),
 * 100 at a time with **Show more**; Add, Edit, Move, Set inactive, Reactivate and Delete; and long
 * lists imported from a spreadsheet made from the downloadable template.
 */
export function OptionsCard({ attribute }: { attribute: ShellAttribute }) {
  const [q, setQ] = useState('');
  const [status, setStatus] = useState<OptionStatus>('all');
  const [panel, setPanel] = useState<OpenPanel | null>(null);
  const [deleting, setDeleting] = useState<ShellAttributeOption | null>(null);
  const [importing, setImporting] = useState(false);
  const [announcement, setAnnouncement] = useState('');
  const options = useShellAttributeOptions(attribute.id, { q, status });
  const setOptionStatus = useSetOptionStatus();
  const moveOption = useMoveOption();
  const deleteOption = useDeleteOption();
  const changeStatus = useStatusChange('option');
  const showFailure = useFailureToast('option');
  const { show } = useToast();
  const isDependent = attribute.parentAttribute !== null;
  const filtered = q.trim() !== '' || status !== 'all';

  function setActive(option: ShellAttributeOption, active: boolean) {
    return changeStatus({
      name: option.label,
      active,
      run: () => setOptionStatus.mutateAsync({ id: option.id, active, version: option.version }),
    });
  }

  async function move(option: ShellAttributeOption, direction: 'up' | 'down') {
    try {
      await moveOption.mutateAsync({ option, direction });
      setAnnouncement(`Moved '${option.label}' ${direction}.`);
    } catch (error) {
      showFailure(error);
    }
  }

  function clearFilters() {
    setQ('');
    setStatus('all');
  }

  const items = options.data?.pages.flatMap((page) => page.items) ?? [];
  const total = options.data?.pages[0]?.total ?? 0;

  function actionsOf(option: ShellAttributeOption, index: number): ActionsMenuItem[] {
    // Without a filter the list is the whole order, so its ends are known.
    const isFirst = !filtered && index === 0;
    const isLast = !filtered && index === total - 1;
    const deleteLock = option.locks.find((lock) => lock.field === 'delete')?.reason;
    return [
      { label: 'Edit', onSelect: () => setPanel({ option }) },
      ...(isFirst ? [] : [{ label: 'Move up', onSelect: () => move(option, 'up') }]),
      ...(isLast ? [] : [{ label: 'Move down', onSelect: () => move(option, 'down') }]),
      {
        label: option.active ? 'Set inactive' : 'Reactivate',
        onSelect: () => setActive(option, !option.active),
      },
      {
        label: 'Delete',
        tone: 'danger',
        separatorBefore: true,
        // The last option can't be deleted: the reason shows, and nothing is sent (D-211, §6.6).
        hint: deleteLock,
        onSelect: () => (deleteLock ? show({ message: deleteLock }) : setDeleting(option)),
      },
    ];
  }

  return (
    <Card
      title="Options"
      actions={
        <div className={styles.cardButtons}>
          <Button icon={<Plus />} onClick={() => setPanel({})}>
            Add option
          </Button>
          <Button icon={<Upload />} onClick={() => setImporting(true)}>
            Import from a spreadsheet
          </Button>
          <Button icon={<Download />} onClick={() => downloadOptionsTemplate(attribute)}>
            Download template
          </Button>
        </div>
      }
    >
      <div className={styles.optionsToolbar}>
        <SearchField
          label="Search options"
          placeholder="Search options"
          value={q}
          onChange={setQ}
        />
        <SegmentedFilter
          label="Status"
          options={STATUS_OPTIONS}
          value={status}
          onChange={setStatus}
        />
      </div>

      {!options.data ? (
        options.isError ? (
          <ErrorState error={options.error} onRetry={() => options.refetch()} />
        ) : (
          <Skeleton lines={5} height="18px" />
        )
      ) : items.length === 0 ? (
        <EmptyState
          title="Nothing matches these filters"
          action={<Button onClick={clearFilters}>Clear filters</Button>}
        />
      ) : (
        <>
          <div className={styles.tableScroll}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th scope="col">Label</th>
                  {isDependent && <th scope="col">Linked to</th>}
                  <th scope="col">Used by</th>
                  <th scope="col">Status</th>
                  <th scope="col">
                    <span className="visually-hidden">Actions</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {items.map((option, index) => (
                  <tr key={option.id}>
                    <th scope="row">{option.label}</th>
                    {isDependent && (
                      <td>
                        {option.parentOptions.length > 0
                          ? option.parentOptions
                              .map((parent) =>
                                parent.active ? parent.name : `${parent.name} (inactive)`,
                              )
                              .join(', ')
                          : '—'}
                      </td>
                    )}
                    <td>{option.usage.summary}</td>
                    <td>
                      <StatusChip tone={option.active ? 'positive' : 'neutral'}>
                        {option.active ? 'Active' : 'Inactive'}
                      </StatusChip>
                    </td>
                    <td className={styles.actionsCell}>
                      <ActionsMenu label={option.label} items={actionsOf(option, index)} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {items.length < total && (
            <div className={styles.more}>
              <p>
                {`Showing ${formatWhole(items.length)} of ${formatWhole(total)} — search to find the others.`}
              </p>
              <Button
                size="sm"
                aria-busy={options.isFetchingNextPage || undefined}
                onClick={() => {
                  if (!options.isFetchingNextPage) {
                    void options.fetchNextPage();
                  }
                }}
              >
                Show more
              </Button>
            </div>
          )}
        </>
      )}
      <p role="status" className="visually-hidden">
        {announcement}
      </p>

      {panel && (
        <OptionPanel attribute={attribute} option={panel.option} onClose={() => setPanel(null)} />
      )}
      {deleting && (
        <SettingDeleteDialog
          kind="option"
          name={deleting.label}
          usage={deleting.usage}
          active={deleting.active}
          remove={() => deleteOption.mutateAsync(deleting.id)}
          setInactive={() => setActive(deleting, false)}
          invalidate={CHANGED_BY_ANY_CHANGE}
          onClose={() => setDeleting(null)}
        />
      )}
      {importing && (
        <ImportOptionsDialog attribute={attribute} onClose={() => setImporting(false)} />
      )}
    </Card>
  );
}
