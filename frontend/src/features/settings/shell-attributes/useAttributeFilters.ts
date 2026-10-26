import { useSearchParams } from 'react-router';
import { withChoice } from '../search-params';
import type { AttributeLevel } from './attribute-words';
import { attributeChoicesOf } from './schemas';
import type { HelpFilter, StatusFilter } from './schemas';

export type AttributeFilters = ReturnType<typeof useAttributeFilters>;

export function useAttributeFilters() {
  const [params, setParams] = useSearchParams();
  const choices = attributeChoicesOf(params);

  function change(name: string, value: string, defaultValue: string) {
    setParams((current) => withChoice(current, name, value, defaultValue), { replace: true });
  }

  return {
    ...choices,
    // With the help filter on, the level stays in the address, so it is not chosen again.
    setLevel: (value: AttributeLevel) =>
      change('level', value, choices.help === 'missing' ? '' : 'plan'),
    // The level on screen (`shown`), which the help filter may have chosen, stays in the address.
    setHelp: (value: HelpFilter, shown: AttributeLevel) =>
      setParams((current) => keepLevel(withChoice(current, 'help', value, 'all'), shown), {
        replace: true,
      }),
    setStatus: (value: StatusFilter) => change('status', value, 'active'),
    setQ: (value: string) => change('q', value, ''),
    // Shows every attribute of the level on screen, so the order can be changed again.
    clear: (shown: AttributeLevel) =>
      setParams(
        (current) => {
          const next = new URLSearchParams(current);
          next.delete('q');
          next.delete('help');
          next.set('status', 'all');
          return keepLevel(next, shown);
        },
        { replace: true },
      ),
  };
}

function keepLevel(params: URLSearchParams, shown: AttributeLevel): URLSearchParams {
  return withChoice(params, 'level', shown, 'plan');
}
