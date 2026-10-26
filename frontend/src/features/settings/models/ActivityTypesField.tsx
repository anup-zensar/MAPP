import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import type { ChoiceOption } from '@/ui/ChoiceField/ChoiceField';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import type { FieldBinding } from '@/ui/FormField/FormField';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { useActivityTypes } from '../api';
import type { Model } from './api';

export interface ActivityTypesFieldProps extends FieldBinding<string[]> {
  /** The model being edited: its allowed types stay offered even when inactive. */
  model: Model | null;
}

/** Allowed activity types: the active types, plus those the model already allows. */
export function ActivityTypesField({
  model,
  name,
  value,
  onChange,
  error,
}: ActivityTypesFieldProps) {
  const activityTypes = useActivityTypes();

  if (!activityTypes.data) {
    return activityTypes.isError ? (
      <ErrorState error={activityTypes.error} onRetry={() => void activityTypes.refetch()} />
    ) : (
      <Skeleton lines={3} />
    );
  }

  const options: ChoiceOption[] = activityTypes.data.items.map((type) => ({
    id: type.id,
    label: type.name,
    active: type.active,
  }));
  for (const allowed of model?.activityTypes ?? []) {
    if (!options.some((option) => option.id === allowed.id)) {
      options.push({ id: allowed.id, label: allowed.name, active: allowed.active });
    }
  }

  return (
    <ChoiceField
      label="Allowed activity types"
      name={name}
      marking="Required"
      hint="Activities in plans using this model can be of these types."
      multiple
      options={options}
      value={value}
      onChange={onChange}
      error={error}
    />
  );
}
