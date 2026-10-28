import {
  ACTIVITY_STATUS_LABELS,
  CONTAINER_STATUS_LABELS,
  FILTER_LABELS,
  FINANCE_LABELS,
  PLAN_STATUS_LABELS,
  PLAN_TYPE_LABELS,
  SHOW_LEVELS_LABELS,
  STRATEGIC_LABELS,
} from './all-work-text';
import type { AllWorkOptions, FilterValue, Filters, ViewState } from './all-work-types';
import { attributeChipLabel, withAttributeValues } from './attribute-filters';
import { ID_FILTERS, NO_FILTERS, VALUE_FILTERS, choicesOf } from './view-state';
import type { IdFilter } from './view-state';

const ATTRIBUTE_CHIP = 'attributes:';

/** One filter value in use, as a removable chip: "Activity status: Declined". */
export interface Chip {
  key: string;
  label: string;
}

/** Every filter in use but the year, in the filter bar's order, named from its choices. */
export function chipsOf(state: ViewState, options?: AllWorkOptions): Chip[] {
  const { filters } = state;
  const chips: Chip[] = [];
  const add = (filter: keyof Filters, value: string, label: string) =>
    chips.push({ key: `${filter}:${value}`, label: `${FILTER_LABELS[filter]}: ${label}` });
  const idLabel = (filter: IdFilter, value: FilterValue) =>
    optionLabel(choicesOf(filter, options), value);

  filters.plans.forEach((value) => add('plans', value.id, idLabel('plans', value)));
  filters.planStatuses.forEach((value) => add('planStatuses', value, PLAN_STATUS_LABELS[value]));
  filters.containerStatuses.forEach((value) =>
    add('containerStatuses', value, CONTAINER_STATUS_LABELS[value]),
  );
  filters.activityStatuses.forEach((value) =>
    add('activityStatuses', value, ACTIVITY_STATUS_LABELS[value]),
  );
  filters.owners.forEach((value) => add('owners', value.id, idLabel('owners', value)));
  filters.teams.forEach((value) => add('teams', value.id, idLabel('teams', value)));
  filters.businessUnits.forEach((value) =>
    add('businessUnits', value.id, idLabel('businessUnits', value)),
  );
  filters.activityTypes.forEach((value) =>
    add('activityTypes', value.id, idLabel('activityTypes', value)),
  );
  if (filters.strategic !== 'include') {
    add('strategic', filters.strategic, STRATEGIC_LABELS[filters.strategic]);
  }
  if (filters.showLevels !== 'everything') {
    add('showLevels', filters.showLevels, SHOW_LEVELS_LABELS[filters.showLevels]);
  }
  filters.planTypes?.forEach((value) => add('planTypes', value, PLAN_TYPE_LABELS[value]));
  filters.planCurrencies?.forEach((value) => add('planCurrencies', value, value));
  filters.finance?.forEach((value) => add('finance', value, FINANCE_LABELS[value]));
  filters.attributes?.forEach((filter) =>
    chips.push({
      key: `${ATTRIBUTE_CHIP}${filter.attributeId}`,
      label: attributeChipLabel(filter, options?.attributes),
    }),
  );
  return chips;
}

export function removeChip(state: ViewState, chipKey: string): ViewState {
  const separator = chipKey.indexOf(':');
  const filter = chipKey.slice(0, separator);
  const value = chipKey.slice(separator + 1);
  return { ...state, filters: withoutValue(state.filters, filter, value) };
}

/** Clear all: every filter and the search go; the year stays. */
export function clearAll(state: ViewState): ViewState {
  return { ...state, filters: NO_FILTERS, search: '' };
}

export function hasFilters(state: ViewState): boolean {
  return chipsOf(state).length > 0 || state.search.trim() !== '';
}

function optionLabel(
  choices: { id: string; label: string; inactive?: boolean }[],
  value: FilterValue,
): string {
  const choice = choices.find((candidate) => candidate.id === value.id);
  if (!choice) {
    return value.label;
  }
  return choice.inactive ? `${choice.label} (inactive)` : choice.label;
}

function withoutValue(filters: Filters, filter: string, value: string): Filters {
  const idFilter = ID_FILTERS.find((key) => key === filter);
  if (idFilter) {
    return { ...filters, [idFilter]: filters[idFilter].filter((chosen) => chosen.id !== value) };
  }
  const valueFilter = VALUE_FILTERS.find((key) => key === filter);
  if (valueFilter) {
    // The lists hold different words; filtering by text keeps each list's own type.
    const values: string[] = filters[valueFilter] ?? [];
    return { ...filters, [valueFilter]: values.filter((chosen) => chosen !== value) };
  }
  if (filter === 'strategic') {
    return { ...filters, strategic: 'include' };
  }
  if (filter === 'attributes') {
    return { ...filters, attributes: withAttributeValues(filters.attributes, value, []) };
  }
  return filter === 'showLevels' ? { ...filters, showLevels: 'everything' } : filters;
}
