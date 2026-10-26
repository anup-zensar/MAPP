import { Calculator, Plus } from 'lucide-react';
import { useState } from 'react';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { formatPercent } from '@/format/decimal';
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
import { moreActions } from './model-actions';
import type { ModelActions } from './model-actions';
import { ModelDetails } from './ModelDetails';
import { ModelForm } from './ModelForm';
import type { ModelFormMode } from './model-values';
import { MODEL_LISTS, setModelStatus, useDeleteModel } from './api';
import type { Model } from './api';
import { useModels } from '../api';

const MODEL_NOUN = { one: 'model', many: 'models' };

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
  const status = useSettingStatus({
    kind: 'model',
    request: setModelStatus,
    invalidate: MODEL_LISTS,
  });

  const actions: ModelActions = {
    view: (model) => filters.setOpenId(model.id),
    edit: (model) => setForm({ kind: 'edit', model }),
    duplicate: (model) => setForm({ kind: 'duplicate', model }),
    setActive: (model, active) => (active ? status.reactivate(model) : status.setInactive(model)),
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
      <PageHeader
        breadcrumb={[{ label: 'Settings', to: '/settings' }, { label: 'Models' }]}
        title="Models"
        description="The rulebooks plans use. Changes apply to every plan using the model."
        actions={
          <CanChange>
            <Button variant="primary" icon={<Plus />} onClick={newModel}>
              New model
            </Button>
          </CanChange>
        }
      />
      <SettingsViewOnlyNote />
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
          setInactive={() => status.applyInactive(deleting)}
          invalidate={MODEL_LISTS}
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
            <CanChange>
              <Button icon={<Plus />} onClick={onNew}>
                New model
              </Button>
            </CanChange>
          }
        />
      </Card>
    );
  }

  const { shown, counts } = filterRecords(all, filters, (model) => model.name);

  return (
    <Card>
      <ListToolbar
        search={{
          label: 'Search models',
          placeholder: 'Search name',
          value: filters.q,
          onChange: filters.setQ,
        }}
        status={{ value: filters.status, onChange: filters.setStatus, counts }}
        count={{ shown: shown.length, total: all.length, noun: MODEL_NOUN }}
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
                  Rates
                </th>
                <th scope="col" className={styles.number}>
                  Average deal size
                </th>
                <th scope="col" className={styles.number}>
                  Allowed activity types
                </th>
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
  const canChange = useCanChange();
  const { money } = useFormatters();
  return (
    <tr>
      <th scope="row" className={styles.nameCell}>
        <button type="button" className={styles.nameButton} onClick={() => actions.view(model)}>
          {model.name}
        </button>
      </th>
      <td className={styles.number}>
        {`${formatPercent(model.inqToMqlRate)} → ${formatPercent(model.mqlToDealRate)}`}
      </td>
      <td className={styles.number}>{money(model.averageDealSize)}</td>
      <td className={styles.number}>{model.activityTypes.length}</td>
      <td>{model.usage.summary}</td>
      <td className={styles.nowrap}>
        <StatusChip tone={model.active ? 'positive' : 'neutral'}>
          {model.active ? 'Active' : 'Inactive'}
        </StatusChip>
      </td>
      <td className={styles.actionsCell}>
        <ActionsMenu
          label={model.name}
          items={[
            { label: 'View', onSelect: () => actions.view(model) },
            ...(canChange
              ? [
                  { label: 'Edit', onSelect: () => actions.edit(model) },
                  ...moreActions(model, actions),
                ]
              : []),
          ]}
        />
      </td>
    </tr>
  );
}
