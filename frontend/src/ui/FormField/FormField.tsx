import type { ReactNode } from 'react';
import { useController } from 'react-hook-form';
import type { Control, FieldPath, FieldPathValue, FieldValues } from 'react-hook-form';

/** What a field component (TextField, MoneyField, PersonPicker, …) needs from its form. */
export interface FieldBinding<V> {
  name: string;
  value: V;
  onChange: (value: V) => void;
  /** The message of the field's problem, shown under it. */
  error?: string;
}

export interface FormFieldProps<T extends FieldValues, N extends FieldPath<T>> {
  control: Control<T>;
  name: N;
  render: (field: FieldBinding<FieldPathValue<T, N>>) => ReactNode;
}

/**
 * Binds one field of a React Hook Form form to a field component:
 *
 *   <FormField control={control} name="email" render={(field) => <TextField label="E-mail" {...field} />} />
 */
export function FormField<T extends FieldValues, N extends FieldPath<T>>({
  control,
  name,
  render,
}: FormFieldProps<T, N>) {
  const { field, fieldState } = useController({ control, name });
  return render({
    name,
    value: field.value,
    onChange: field.onChange,
    error: fieldState.error?.message,
  });
}
