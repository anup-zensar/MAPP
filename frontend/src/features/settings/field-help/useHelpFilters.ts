import { useSearchParams } from 'react-router';
import type { FieldHelpLevel } from './api';
import { withChoice } from '../search-params';
import { helpChoicesOf } from './schemas';
import type { Shown } from './schemas';

export type HelpFilters = ReturnType<typeof useHelpFilters>;

/** The page's choices, kept in the address so Back and links keep them. */
export function useHelpFilters() {
  const [params, setParams] = useSearchParams();

  function change(name: string, value: string, defaultValue: string) {
    setParams((current) => withChoice(current, name, value, defaultValue), { replace: true });
  }

  return {
    ...helpChoicesOf(params),
    setLevel: (value: FieldHelpLevel) => change('level', value, 'plan'),
    setShown: (value: Shown) => change('show', value, 'all'),
    setQ: (value: string) => change('q', value, ''),
    setOpenKey: (value: string) => change('key', value, ''),
    clear: () =>
      setParams(
        (current) => {
          const next = new URLSearchParams(current);
          next.delete('q');
          next.delete('show');
          return next;
        },
        { replace: true },
      ),
  };
}
