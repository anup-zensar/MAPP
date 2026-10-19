/**
 * The words each kind of Settings record is called by on screen (feature 002 plan, "User-visible
 * messages"): in the middle of a sentence, at the start of one, what setting it inactive does, and
 * what its details say when it is gone.
 */
export type SettingKind =
  'model' | 'activityType' | 'shellAttribute' | 'option' | 'currency' | 'financeBudgetItem';

export interface SettingNouns {
  /** In the middle of a sentence: "Delete model 'Lubricants 1.0'?" */
  noun: string;
  /** At the start of a sentence: "Model 'Lubricants 1.0' saved". */
  title: string;
  /** What setting it inactive does, said before people choose Set inactive. */
  inactiveEffect: string;
  /** Its details when someone else deleted it. */
  notFound: string;
}

export const SETTING_NOUNS: Record<SettingKind, SettingNouns> = {
  model: {
    noun: 'model',
    title: 'Model',
    inactiveEffect:
      "New plans can't choose an inactive model; plans using it keep it and still calculate goals.",
    notFound: 'This model no longer exists — someone may have deleted it.',
  },
  activityType: {
    noun: 'activity type',
    title: 'Activity type',
    inactiveEffect: "New activities can't choose an inactive type; activities using it keep it.",
    notFound: 'This activity type no longer exists — someone may have deleted it.',
  },
  shellAttribute: {
    noun: 'Shell attribute',
    title: 'Shell attribute',
    inactiveEffect:
      'Forms stop offering it; values already on records stay visible as "(inactive)" and never block publishing or submitting.',
    notFound: 'This Shell attribute no longer exists — someone may have deleted it.',
  },
  option: {
    noun: 'option',
    title: 'Option',
    inactiveEffect: 'Forms stop offering it; records that have it keep it as "(inactive)".',
    notFound: 'This option no longer exists — someone may have deleted it.',
  },
  currency: {
    noun: 'currency',
    title: 'Currency',
    inactiveEffect:
      "Line items and people can't choose an inactive currency; those using it keep it and amounts still convert.",
    notFound: 'This currency no longer exists — someone may have deleted it.',
  },
  financeBudgetItem: {
    noun: 'finance budget item',
    title: 'Finance budget item',
    inactiveEffect: "Line items can't link to an inactive item; those linked keep it.",
    notFound: 'This finance budget item no longer exists — someone may have deleted it.',
  },
};
