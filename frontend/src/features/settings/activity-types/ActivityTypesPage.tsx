import { Plus, Shapes } from 'lucide-react';
import { useState } from 'react';
import { useActivityTypes } from '@/api/settingsQueries';
import {
  ActionsMenu,
  Breadcrumb,
  Button,
  Card,
  EmptyState,
  ErrorState,
  PageHeader,
  SearchField,
  SegmentedFilter,
  Skeleton,
  StatusChip,
} from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import { usePageTitle } from '@/lib/usePageTitle';
import { SettingDeleteDialog } from '../shared/SettingDeleteDialog';
import styles from '../shared/SettingsTable.module.css';
import { filterRecords, useListFilters } from '../shared/useListFilters';
import type { ListFilters } from '../shared/useListFilters';
import { useSettingStatus } from '../shared/useSettingStatus';
import { modelNames, moreActions } from './activityTypeActions';
import type { ActivityTypeActions } from './activityTypeActions';
import { ActivityTypeDetails } from './ActivityTypeDetails';
import { ActivityTypeForm } from './ActivityTypeForm';
import {
  ACTIVITY_TYPE_LISTS,
  setActivityTypeStatus,
  useDeleteActivityType,
} from './activityTypesApi';
import type { ActivityType } from './activityTypesApi';

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
  const changeStatus = useSettingStatus({
    kind: 'activityType',
    request: setActivityTypeStatus,
    invalidate: ACTIVITY_TYPE_LISTS,
  });

  const actions: ActivityTypeActions = {
    view: (type) => filters.setOpenId(type.id),
    edit: (type) => setForm({ kind: 'edit', type }),
    setActive: (type, active) => void changeStatus(type, active),
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
      <Breadcrumb items={[{ label: 'Settings', to: '/settings' }, { label: 'Activity types' }]} />
      <PageHeader
        eyebrow="Settings"
        title="Activity types"
        description="The kinds of work activities can be."
        actions={
          <Button variant="primary" icon={<Plus />} onClick={newType}>
            New activity type
          </Button>
        }
      />
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
          setInactive={() => changeStatus(deleting, false)}
          invalidate={ACTIVITY_TYPE_LISTS}
          onDeleted={() => {
            if (filters.openId === deleting.id) {
              filters.setOpenId('');
            }
          }}
          onClose={() => setDeleting(null)}
        />
      )}
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
            <Button icon={<Plus />} onClick={onNew}>
              New activity type
            </Button>
          }
        />
      </Card>
    );
  }

  const { shown, counts } = filterRecords(all, filters, (type) => type.name);

  return (
    <Card>
      <div className={styles.toolbar}>
        <SearchField
          label="Search activity types"
          placeholder="Search name"
          value={filters.q}
          onChange={filters.setQ}
        />
        <SegmentedFilter
          label="Status"
          options={[
            { value: 'active', label: 'Active', count: counts.active },
            { value: 'inactive', label: 'Inactive', count: counts.inactive },
            { value: 'all', label: 'All', count: counts.all },
          ]}
          value={filters.status}
          onChange={filters.setStatus}
        />
        <p className={styles.resultCount} aria-live="polite">
          {`${shown.length} of ${all.length}`}
        </p>
      </div>
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
                <th scope="col">Default planned cost</th>
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
  const { money } = useFormatters();
  return (
    <tr>
      <th scope="row">
        <button type="button" className={styles.nameButton} onClick={() => actions.view(type)}>
          {type.name}
        </button>
      </th>
      <td className={styles.numeric}>
        {type.defaultPlannedCost ? money(type.defaultPlannedCost) : '—'}
      </td>
      <td>{modelNames(type)}</td>
      <td>{type.usage.summary}</td>
      <td>
        <StatusChip tone={type.active ? 'positive' : 'neutral'}>
          {type.active ? 'Active' : 'Inactive'}
        </StatusChip>
      </td>
      <td className={styles.actionsCell}>
        <ActionsMenu
          label={type.name}
          items={[
            { label: 'View', onSelect: () => actions.view(type) },
            { label: 'Edit', onSelect: () => actions.edit(type) },
            ...moreActions(type, actions),
          ]}
        />
      </td>
    </tr>
  );
}
