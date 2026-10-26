/** The address with one choice changed; a choice back at its default leaves the address. */
export function withChoice(
  params: URLSearchParams,
  name: string,
  value: string,
  defaultValue: string,
): URLSearchParams {
  const next = new URLSearchParams(params);
  if (value === defaultValue) {
    next.delete(name);
  } else {
    next.set(name, value);
  }
  return next;
}
