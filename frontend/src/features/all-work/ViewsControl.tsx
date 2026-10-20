import { useState } from 'react';
import { useToast } from '@/components';
import { useSavedViews, useUpdateView } from './allWorkApi';
import type { SavedView, ViewState } from './allWorkTypes';
import { ManageViewsPanel } from './ManageViewsPanel';
import { SaveViewDialog } from './SaveViewDialog';
import { useViewRefusal } from './useViewRefusal';
import { sameState, savedStateOf } from './viewState';
import { ViewsMenu } from './ViewsMenu';

export interface ViewsControlProps {
  /** The state on screen; null while the visit starts. */
  state: ViewState | null;
  /** The saved view applied last (FR-030). */
  appliedViewId: string | null;
  onApply: (view: SavedView) => void;
  /** A view was saved from what is on screen: it becomes the active one. */
  onSaved: (view: SavedView) => void;
  onStandard: () => void;
}

/**
 * Saved views on the page header (US6, FR-029 – FR-031): the Views menu with "Modified" when the
 * state differs from the active view, and its dialogs — Save current view and Your saved views.
 */
export function ViewsControl({
  state,
  appliedViewId,
  onApply,
  onSaved,
  onStandard,
}: ViewsControlProps) {
  const views = useSavedViews();
  const update = useUpdateView();
  const refused = useViewRefusal();
  const { show } = useToast();
  const [saving, setSaving] = useState(false);
  const [managing, setManaging] = useState(false);
  const items = views.data?.items;
  const active = items?.find((view) => view.id === appliedViewId) ?? null;
  const modified = Boolean(active && state && !sameState(state, active.state));

  /** Update view: the active view keeps what is on screen now. */
  function updateActive() {
    if (!active || !state) return;
    update.mutate(
      { viewId: active.id, body: { state: savedStateOf(state) } },
      {
        onSuccess: (view) => show({ message: `View '${view.name}' updated` }),
        onError: (error) => refused(error, 'name'),
      },
    );
  }

  return (
    <>
      <ViewsMenu
        views={items}
        active={active}
        modified={modified}
        onApply={onApply}
        onSave={() => setSaving(true)}
        onUpdate={updateActive}
        onSaveAsNew={() => setSaving(true)}
        onManage={() => setManaging(true)}
        onStandard={onStandard}
      />
      {saving && state && (
        <SaveViewDialog
          onOpenChange={setSaving}
          views={items ?? []}
          state={state}
          onSaved={onSaved}
        />
      )}
      <ManageViewsPanel open={managing} onOpenChange={setManaging} views={items ?? []} />
    </>
  );
}
