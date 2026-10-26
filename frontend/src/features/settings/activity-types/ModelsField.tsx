import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import type { ChoiceOption } from '@/ui/ChoiceField/ChoiceField';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import type { FieldBinding } from '@/ui/FormField/FormField';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { useModels } from '../api';
import type { ActivityType } from './api';

export interface ModelsFieldProps extends FieldBinding<string[]> {
  /** The type being edited: the models allowing it stay offered even when inactive. */
  type?: ActivityType;
}

/** Allowed in models: the active models, plus those that already allow the type. */
export function ModelsField({ type, name, value, onChange, error }: ModelsFieldProps) {
  const models = useModels();

  if (!models.data) {
    return models.isError ? (
      <ErrorState error={models.error} onRetry={() => void models.refetch()} />
    ) : (
      <Skeleton lines={3} />
    );
  }

  const options: ChoiceOption[] = models.data.items.map((model) => ({
    id: model.id,
    label: model.name,
    active: model.active,
  }));
  for (const allowing of type?.models ?? []) {
    if (!options.some((option) => option.id === allowing.id)) {
      options.push({ id: allowing.id, label: allowing.name, active: allowing.active });
    }
  }

  return (
    <ChoiceField
      label="Allowed in models"
      name={name}
      hint="Plans using these models can choose this type."
      multiple
      options={options}
      value={value}
      onChange={onChange}
      error={error}
    />
  );
}
