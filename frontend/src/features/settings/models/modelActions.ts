import type { ActionsMenuItem } from '@/components';
import type { Model } from './modelsApi';

/** What people can do with a model, from its row or its details. */
export interface ModelActions {
  view: (model: Model) => void;
  edit: (model: Model) => void;
  duplicate: (model: Model) => void;
  setActive: (model: Model, active: boolean) => void;
  remove: (model: Model) => void;
}

/** Duplicate · Set inactive or Reactivate · Delete (last, after a line — design system §6.1). */
export function moreActions(model: Model, actions: ModelActions): ActionsMenuItem[] {
  return [
    { label: 'Duplicate', onSelect: () => actions.duplicate(model) },
    model.active
      ? { label: 'Set inactive', onSelect: () => actions.setActive(model, false) }
      : { label: 'Reactivate', onSelect: () => actions.setActive(model, true) },
    {
      label: 'Delete',
      tone: 'danger',
      separatorBefore: true,
      onSelect: () => actions.remove(model),
    },
  ];
}
