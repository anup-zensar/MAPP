import { Plus, Shapes } from 'lucide-react';
import { useState } from 'react';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { useFormatters } from '@/format/useFormatters';
import { usePageTitle } from '@/ui/usePageTitle';
import { ListToolbar } from '@/ui/ListToolbar/ListToolbar';
import styles from '@/ui/settings-table.module.css';
import { filterRecords, useListFilters } from '@/ui/useListFilters';
import type { ListFilters } from '@/ui/useListFilters';
import { CanChange, SettingsViewOnlyNote } from '@/ui/CanChange/CanChange';
import { useCanChange } from '@/ui/useCanChange';
import { SettingDeleteDialog } from '../SettingDeleteDialog';
import { useSettingStatus } from '../useSettingStatus';
import { modelNames, moreActions } from './activity-type-actions';
import type { ActivityTypeActions } from './activity-type-actions';
import { ActivityTypeDetails } from './ActivityTypeDetails';
import { ActivityTypeForm } from './ActivityTypeForm';
import { ACTIVITY_TYPE_LISTS, setActivityTypeStatus, useDeleteActivityType } from './api';
import type { ActivityType } from './api';
import { useActivityTypes } from '../api';

const TYPE_NOUN = { one: 'activity type', many: 'activity types' };

/** The side-panel form that is open: a new activity type, or one being changed. */
type OpenForm = { kind: 'new' } | { kind: 'edit'; type: ActivityType };

/**
 * Activity types (feature 002, US5, FR-002, FR-013): the kinds of work an activity can be, as the
 * standard CRUD pattern — search and status filter in the address, a table, details in a side
 * panel (`?id=`), New activity type and Edit in a side-panel form, Set inactive / Reactivate, and
 * Delete (unused types only, with Undo).
 */
export function ActivityTypesPage() {
  usePageTitle('Activity types');
  const filters = useListFilters();
  const types = useActivityTypes();
  const deleteType = useDeleteActivityType();
  const [form, setForm] = useState<OpenForm | null>(null);
  const [deleting, setDeleting] = useState<ActivityType | null>(null);
  const status = useSettingStatus({
    kind: 'activityType',
    request: setActivityTypeStatus,
    invalidate: ACTIVITY_TYPE_LISTS,
  });

  const actions: ActivityTypeActions = {
    view: (type) => filters.setOpenId(type.id),
    edit: (type) => setForm({ kind: 'edit', type }),
    setActive: (type, active) => (active ? status.reactivate(type) : status.setInactive(type)),
    remove: (type) => setDeleting(type),
  };
  const newType = () => setForm({ kind: 'new' });
  // The details wait for the list; a type that is not in it any more says so.
  const openType =
    filters.openId && types.data
      ? (types.data.items.find((type) => type.id === filters.openId) ?? null)
      : undefined;

  return (
    <>
      <PageHeader
        breadcrumb={[{ label: 'Settings', to: '/settings' }, { label: 'Activity types' }]}
        title="Activity types"
        description="The kinds of work activities can be."
        actions={
          <CanChange>
            <Button variant="primary" icon={<Plus />} onClick={newType}>
              New activity type
            </Button>
          </CanChange>
        }
      />
      <SettingsViewOnlyNote />
      <ActivityTypesCard types={types} filters={filters} actions={actions} onNew={newType} />

      {form && (
        <ActivityTypeForm
          type={form.kind === 'edit' ? form.type : undefined}
          onClose={() => setForm(null)}
        />
      )}
      {!form && openType !== undefined && (
        <ActivityTypeDetails
          type={openType}
          actions={actions}
          onClose={() => filters.setOpenId('')}
        />
      )}
      {deleting && (
        <SettingDeleteDialog
          kind="activityType"
          name={deleting.name}
          usage={deleting.usage}
          active={deleting.active}
          remove={() => deleteType.mutateAsync(deleting.id)}
          setInactive={() => status.applyInactive(deleting)}
          invalidate={ACTIVITY_TYPE_LISTS}
          onDeleted={() => {
            if (filters.openId === deleting.id) {
              filters.setOpenId('');
            }
          }}
          onClose={() => setDeleting(null)}
        />
      )}
      {status.dialog}
    </>
  );
}

interface ActivityTypesCardProps {
  types: ReturnType<typeof useActivityTypes>;
  filters: ListFilters;
  actions: ActivityTypeActions;
  onNew: () => void;
}

/** The toolbar and the table, or what to show instead: loading, failure, no types yet. */
function ActivityTypesCard({ types, filters, actions, onNew }: ActivityTypesCardProps) {
  if (!types.data) {
    return (
      <Card>
        {types.isError ? (
          <ErrorState error={types.error} onRetry={() => void types.refetch()} />
        ) : (
          <Skeleton lines={5} height="18px" />
        )}
      </Card>
    );
  }

  const all = types.data.items;
  if (all.length === 0) {
    return (
      <Card>
        <EmptyState
          icon={<Shapes />}
          title="No activity types yet — create your first activity type"
          action={
            <CanChange>
              <Button icon={<Plus />} onClick={onNew}>
                New activity type
              </Button>
            </CanChange>
          }
        />
      </Card>
    );
  }

  const { shown, counts } = filterRecords(all, filters, (type) => type.name);

  return (
    <Card>
      <ListToolbar
        search={{
          label: 'Search activity types',
          placeholder: 'Search name',
          value: filters.q,
          onChange: filters.setQ,
        }}
        status={{ value: filters.status, onChange: filters.setStatus, counts }}
        count={{ shown: shown.length, total: all.length, noun: TYPE_NOUN }}
      />
      {shown.length === 0 ? (
        <div className={styles.noMatch}>
          <p>Nothing matches these filters</p>
          <Button onClick={filters.clear}>Clear filters</Button>
        </div>
      ) : (
        <div className={styles.tableScroll}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th scope="col">Name</th>
                <th scope="col" className={styles.number}>
                  Default planned cost
                </th>
                <th scope="col">Allowed in</th>
                <th scope="col">Used by</th>
                <th scope="col">Status</th>
                <th scope="col">
                  <span className="visually-hidden">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {shown.map((type) => (
                <ActivityTypeRow key={type.id} type={type} actions={actions} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}

function ActivityTypeRow({ type, actions }: { type: ActivityType; actions: ActivityTypeActions }) {
  const canChange = useCanChange();
  const { money } = useFormatters();
  return (
    <tr>
      <th scope="row" className={styles.nameCell}>
        <button type="button" className={styles.nameButton} onClick={() => actions.view(type)}>
          {type.name}
        </button>
      </th>
      <td className={styles.number}>
        {type.defaultPlannedCost ? money(type.defaultPlannedCost) : '—'}
      </td>
      <td>{modelNames(type)}</td>
      <td>{type.usage.summary}</td>
      <td className={styles.nowrap}>
        <StatusChip tone={type.active ? 'positive' : 'neutral'}>
          {type.active ? 'Active' : 'Inactive'}
        </StatusChip>
      </td>
      <td className={styles.actionsCell}>
        <ActionsMenu
          label={type.name}
          items={[
            { label: 'View', onSelect: () => actions.view(type) },
            ...(canChange
              ? [
                  { label: 'Edit', onSelect: () => actions.edit(type) },
                  ...moreActions(type, actions),
                ]
              : []),
          ]}
        />
      </td>
    </tr>
  );
}
