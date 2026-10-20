import { formatWhole } from '@/lib/decimal';
import { SettingsCount } from '../shared/SettingsCount';
import { useFieldHelpEntries } from './fieldHelpApi';

/** "3 edited" on the Field help card of the Settings page (016 plan, "Pages"). */
export function FieldHelpCount() {
  const entries = useFieldHelpEntries();
  return (
    <SettingsCount
      loading={entries.isPending}
      count={entries.data?.items.filter((entry) => !entry.isDefault).length}
      text={(count) => (count === 0 ? 'All default' : `${formatWhole(count)} edited`)}
    />
  );
}
