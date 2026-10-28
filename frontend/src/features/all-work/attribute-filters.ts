import { FILTER_LABELS } from './all-work-text';
import type { AttributeFilter, FilterableAttribute } from './all-work-types';

export const NOT_FILLED_IN = 'none';
export const NOT_FILLED_IN_LABEL = 'Not filled in';

const BUILT_IN_VALUES = new Set(['yes', 'no', NOT_FILLED_IN]);

export function attrParams(filters: AttributeFilter[] | undefined): string[] {
  return (filters ?? [])
    .filter((filter) => filter.values.length > 0)
    .map((filter) => `${filter.attributeId}:${filter.values.join(',')}`);
}

export function withAttributeValues(
  filters: AttributeFilter[] | undefined,
  attributeId: string,
  values: string[],
): AttributeFilter[] {
  const others = (filters ?? []).filter((filter) => filter.attributeId !== attributeId);
  return values.length > 0 ? [...others, { attributeId, values }] : others;
}

export function orderedValues(
  attribute: FilterableAttribute | undefined,
  values: string[],
): string[] {
  const order = (value: string) => {
    const index = attribute?.values.findIndex((option) => option.id === value) ?? -1;
    return value === NOT_FILLED_IN ? Number.MAX_SAFE_INTEGER : index;
  };
  return [...values].sort((a, b) => order(a) - order(b));
}

export function valueLabel(attribute: FilterableAttribute | undefined, value: string): string {
  if (value === NOT_FILLED_IN) {
    return NOT_FILLED_IN_LABEL;
  }
  return attribute?.values.find((option) => option.id === value)?.label ?? value;
}

export function attributeChipLabel(
  filter: AttributeFilter,
  attributes: FilterableAttribute[] | undefined,
): string {
  const attribute = attributes?.find((candidate) => candidate.id === filter.attributeId);
  const values = orderedValues(attribute, filter.values).map((value) =>
    valueLabel(attribute, value),
  );
  return `${attribute?.label ?? filter.attributeId}: ${values.join(', ')}`;
}

export function canonicalAttributes(filters: AttributeFilter[] | undefined): AttributeFilter[] {
  return (filters ?? [])
    .map((filter) => ({ attributeId: filter.attributeId, values: [...filter.values].sort() }))
    .sort((a, b) => a.attributeId.localeCompare(b.attributeId));
}

/** A saved view's filters without attributes and options no longer offered (names: what went). */
export function knownAttributeFilters(
  filters: AttributeFilter[] | undefined,
  attributes: FilterableAttribute[] | undefined,
): { filters: AttributeFilter[] | undefined; names: string[] } {
  if (!filters || !attributes) {
    return { filters, names: [] };
  }
  const names: string[] = [];
  const kept = filters.flatMap((filter) => {
    const attribute = attributes.find((candidate) => candidate.id === filter.attributeId);
    if (!attribute) {
      names.push(`${FILTER_LABELS.attributes}: ${filter.attributeId}`);
      return [];
    }
    const values = filter.values.filter((value) => isOffered(attribute, value));
    filter.values
      .filter((value) => !values.includes(value))
      .forEach((value) => names.push(`${attribute.label}: ${value}`));
    return values.length > 0 ? [{ attributeId: attribute.id, values }] : [];
  });
  return { filters: kept, names };
}

function isOffered(attribute: FilterableAttribute, value: string): boolean {
  return BUILT_IN_VALUES.has(value) || attribute.values.some((option) => option.id === value);
}
