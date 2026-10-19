/**
 * The ids that were in `before` and are not in `after`: the teams a person form removed, or the
 * members a team form removed. Only then does the form ask the API's impact check before saving
 * (plan "Save flow", FR-013).
 */
export function removedIds(before: string[], after: string[]): string[] {
  return before.filter((id) => !after.includes(id));
}
