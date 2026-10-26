import type { UseFormReturn } from 'react-hook-form';
import { isChoiceType, isLevel, markingsFor, typesFor } from './attribute-words';
import type { AttributeValues } from './schemas';

type AttributeForm = UseFormReturn<AttributeValues>;

export const NONE = 'none';

const CHANGE = { shouldDirty: true } as const;

/** NONE is sent as ""; links belong to the parent's options, so another parent starts afresh. */
export function chooseParent(form: AttributeForm, next: string) {
  form.setValue('parentAttributeId', next === NONE ? '' : next, CHANGE);
  form.setValue('optionLinks', {}, CHANGE);
  form.setValue(
    'options',
    form.getValues('options').map((row) => ({ ...row, parentOptionIds: [] })),
    CHANGE,
  );
}

/** A level offers its own types, markings and parents; a choice it doesn't offer is cleared. */
export function chooseLevel(form: AttributeForm, next: string) {
  if (!isLevel(next)) {
    return;
  }
  const { type, marking } = form.getValues();
  const types: string[] = typesFor(next);
  const markings: string[] = markingsFor(next);
  const keptType = types.includes(type) ? type : '';
  form.setValue('level', next, CHANGE);
  form.setValue('type', keptType, CHANGE);
  form.setValue(
    'marking',
    keptType === 'yesNo' || markings.includes(marking) ? marking : '',
    CHANGE,
  );
  chooseParent(form, NONE);
}

/** Yes/no attributes are always optional (FR-017); only choice attributes depend on others. */
export function chooseType(form: AttributeForm, type: string) {
  form.setValue('type', type, CHANGE);
  if (type === 'yesNo') {
    form.setValue('marking', 'optional', CHANGE);
  }
  if (!isChoiceType(type) && form.getValues('parentAttributeId') !== '') {
    chooseParent(form, NONE);
  }
}
