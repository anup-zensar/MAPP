import { Calculator, Plus } from 'lucide-react';
import { useState } from 'react';
import { useModels } from '@/api/settingsQueries';
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
import { formatPercent } from '@/lib/decimal';
import { useFormatters } from '@/lib/useFormatters';
import { usePageTitle } from '@/lib/usePageTitle';
import { SettingDeleteDialog } from '../shared/SettingDeleteDialog';
import styles from '../shared/SettingsTable.module.css';
import { filterRecords, useListFilters } from '../shared/useListFilters';
import type { ListFilters } from '../shared/useListFilters';
import { useSettingStatus } from '../shared/useSettingStatus';
import { moreActions } from './modelActions';
import type { ModelActions } from './modelActions';
import { ModelDetails } from './ModelDetails';
import { ModelForm } from './ModelForm';
import type { ModelFormMode } from './ModelForm';
import { MODEL_LISTS, setModelStatus, useDeleteModel } from './modelsApi';
import type { Model } from './modelsApi';

/**
 * Models (feature 002, US1, FR-002, FR-009 – FR-012): the rulebooks plans use, as the standard CRUD
 * pattern — search and status filter in the address, a table, details in a side panel (`?id=`),
 * New model, Edit and Duplicate in a side-panel form, Set inactive / Reactivate, and Delete (unused
 * models only, with Undo).
 */
export function ModelsPage() {
  usePageTitle('Models');
  const filters = useListFilters();
  const models = useModels();
  const deleteModel = useDeleteModel();
  const [form, setForm] = useState<ModelFormMode | null>(null);
  const [deleting, setDeleting] = useState<Model | null>(null);
  const changeStatus = useSettingStatus({
    kind: 'model',
    request: setModelStatus,
    invalidate: MODEL_LISTS,
  });

  const actions: ModelActions = {
    view: (model) => filters.setOpenId(model.id),
    edit: (model) => setForm({ kind: 'edit', model }),
    duplicate: (model) => setForm({ kind: 'duplicate', model }),
    setActive: (model, active) => void changeStatus(model, active),
    remove: (model) => setDeleting(model),
  };
  const newModel = () => setForm({ kind: 'new' });
  // The details wait for the list; a model that is not in it any more says so.
  const openModel =
    filters.openId && models.data
      ? (models.data.items.find((model) => model.id === filters.openId) ?? null)
      : undefined;

  return (
    <>
      <Breadcrumb items={[{ label: 'Settings', to: '/settings' }, { label: 'Models' }]} />
      <PageHeader
        eyebrow="Settings"
        title="Models"
        description="The rulebooks plans use. Changes apply to every plan using the model."
        actions={
          <Button variant="primary" icon={<Plus />} onClick={newModel}>
            New model
          </Button>
        }
      />
      <ModelsCard models={models} filters={filters} actions={actions} onNew={newModel} />

      {form && <ModelForm mode={form} onView={filters.setOpenId} onClose={() => setForm(null)} />}
      {!form && openModel !== undefined && (
        <ModelDetails model={openModel} actions={actions} onClose={() => filters.setOpenId('')} />
      )}
      {deleting && (
        <SettingDeleteDialog
          kind="model"
          name={deleting.name}
          usage={deleting.usage}
          active={deleting.active}
          remove={() => deleteModel.mutateAsync(deleting.id)}
          setInactive={() => changeStatus(deleting, false)}
          invalidate={MODEL_LISTS}
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

interface ModelsCardProps {
  models: ReturnType<typeof useModels>;
  filters: ListFilters;
  actions: ModelActions;
  onNew: () => void;
}

/** The toolbar and the table, or what to show instead: loading, failure, no models yet. */
function ModelsCard({ models, filters, actions, onNew }: ModelsCardProps) {
  if (!models.data) {
    return (
      <Card>
        {models.isError ? (
          <ErrorState error={models.error} onRetry={() => void models.refetch()} />
        ) : (
          <Skeleton lines={5} height="18px" />
        )}
      </Card>
    );
  }

  const all = models.data.items;
  if (all.length === 0) {
    return (
      <Card>
        <EmptyState
          icon={<Calculator />}
          title="No models yet — create your first model"
          action={
            <Button icon={<Plus />} onClick={onNew}>
              New model
            </Button>
          }
        />
      </Card>
    );
  }

  const { shown, counts } = filterRecords(all, filters, (model) => model.name);

  return (
    <Card>
      <div className={styles.toolbar}>
        <SearchField
          label="Search models"
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
                <th scope="col">Rates</th>
                <th scope="col">Average deal size</th>
                <th scope="col">Allowed activity types</th>
                <th scope="col">Used by</th>
                <th scope="col">Status</th>
                <th scope="col">
                  <span className="visually-hidden">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {shown.map((model) => (
                <ModelRow key={model.id} model={model} actions={actions} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}

function ModelRow({ model, actions }: { model: Model; actions: ModelActions }) {
  const { money } = useFormatters();
  return (
    <tr>
      <th scope="row">
        <button type="button" className={styles.nameButton} onClick={() => actions.view(model)}>
          {model.name}
        </button>
      </th>
      <td className={styles.numeric}>
        {`${formatPercent(model.inqToMqlRate)} → ${formatPercent(model.mqlToDealRate)}`}
      </td>
      <td className={styles.numeric}>{money(model.averageDealSize)}</td>
      <td className={styles.numeric}>{model.activityTypes.length}</td>
      <td>{model.usage.summary}</td>
      <td>
        <StatusChip tone={model.active ? 'positive' : 'neutral'}>
          {model.active ? 'Active' : 'Inactive'}
        </StatusChip>
      </td>
      <td className={styles.actionsCell}>
        <ActionsMenu
          label={model.name}
          items={[
            { label: 'View', onSelect: () => actions.view(model) },
            { label: 'Edit', onSelect: () => actions.edit(model) },
            ...moreActions(model, actions),
          ]}
        />
      </td>
    </tr>
  );
}
