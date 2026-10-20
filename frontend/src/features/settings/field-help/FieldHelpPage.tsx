import { Info } from 'lucide-react';
import { useState } from 'react';
import { Link, useSearchParams } from 'react-router';
import {
  ActionsMenu,
  Button,
  Card,
  EmptyState,
  ErrorState,
  PageHeader,
  SegmentedFilter,
  Skeleton,
  StatusChip,
  Tabs,
  useToast,
} from '@/components';
import { formatWhole } from '@/lib/decimal';
import { localDate } from '@/lib/format';
import { useFormatters } from '@/lib/useFormatters';
import { usePageTitle } from '@/lib/usePageTitle';
import { ListToolbar } from '../shared/ListToolbar';
import settingsStyles from '../shared/SettingsTable.module.css';
import { useFieldHelpEntries } from './fieldHelpApi';
import type { FieldHelpEntry, FieldHelpLevel } from './fieldHelpApi';
import { FieldHelpPanel } from './FieldHelpPanel';
import {
  attributesWithoutHelp,
  HELP_LEVEL_WORDS,
  HELP_LEVELS,
  isHelpLevel,
  MISSING_HELP_PATH,
} from './fieldHelpWords';
import { ResetHelpDialog } from './ResetHelpDialog';
import styles from './fieldHelp.module.css';
import { CanChange, SettingsViewOnlyNote } from '../shared/CanChange';

const FIELD_NOUN = { one: 'field', many: 'fields' };

/** Why Reset to default does nothing for a text that was never edited. */
const ALREADY_DEFAULT = 'This field already uses the default text.';

type Shown = 'all' | 'edited';

/** The page's choices, kept in the address so Back and links keep them. */
function useHelpFilters() {
  const [params, setParams] = useSearchParams();
  const levelParam = params.get('level');
  const level: FieldHelpLevel = isHelpLevel(levelParam) ? levelParam : 'plan';
  const shown: Shown = params.get('show') === 'edited' ? 'edited' : 'all';
  const q = params.get('q') ?? '';
  const openKey = params.get('key') ?? '';

  function change(name: string, value: string, defaultValue: string) {
    setParams(
      (current) => {
        const next = new URLSearchParams(current);
        if (value === defaultValue) {
          next.delete(name);
        } else {
          next.set(name, value);
        }
        return next;
      },
      { replace: true },
    );
  }

  return {
    level,
    shown,
    q,
    openKey,
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

/** True when the label or the text contains the search, ignoring letter case. */
function matches(entry: FieldHelpEntry, q: string): boolean {
  const text = q.trim().toLowerCase();
  return (
    text === '' ||
    entry.label.toLowerCase().includes(text) ||
    entry.text.toLowerCase().includes(text)
  );
}

/**
 * Settings › Field help (016 US7, FR-045 – FR-048): every built-in field and calculated value of
 * the planning levels and pages, grouped by level, searchable, each with the help its ⓘ shows and
 * a "Default" or "Edited" chip; **Edit** opens the text in a side panel, **Reset to default** asks
 * first. A note counts the Shell attributes without help, whose help stays on their own pages.
 */
export function FieldHelpPage() {
  usePageTitle('Field help');
  const filters = useHelpFilters();
  const entries = useFieldHelpEntries();

  return (
    <>
      <PageHeader
        breadcrumb={[{ label: 'Settings', to: '/settings' }, { label: 'Field help' }]}
        title="Field help"
        description="What each built-in field means, shown in the ⓘ beside it. A change shows on the next page anyone opens."
      />
      <SettingsViewOnlyNote />
      {!entries.data ? (
        <Card>
          {entries.isError ? (
            <ErrorState error={entries.error} onRetry={() => void entries.refetch()} />
          ) : (
            <Skeleton lines={6} height="18px" />
          )}
        </Card>
      ) : (
        <HelpList
          items={entries.data.items}
          withoutHelp={entries.data.shellAttributesWithoutHelp}
          filters={filters}
        />
      )}
    </>
  );
}

interface HelpListProps {
  items: FieldHelpEntry[];
  withoutHelp: number;
  filters: ReturnType<typeof useHelpFilters>;
}

/** The note, the level tabs and the selected level's fields. */
function HelpList({ items, withoutHelp, filters }: HelpListProps) {
  const [resetting, setResetting] = useState<FieldHelpEntry | null>(null);
  const { level, shown, q } = filters;
  const wanted = (entry: FieldHelpEntry) =>
    matches(entry, q) && (shown === 'all' || !entry.isDefault);
  const ofLevel = items.filter((entry) => entry.level === level);
  const rows = ofLevel.filter(wanted);
  const open = filters.openKey ? items.find((entry) => entry.key === filters.openKey) : undefined;

  const tabs = HELP_LEVELS.filter((each) => items.some((entry) => entry.level === each)).map(
    (each) => ({
      value: each,
      label: HELP_LEVEL_WORDS[each],
      // The fields of each level that match the search and the Show choice.
      badge: (
        <span className={styles.tabCount}>
          {formatWhole(items.filter((entry) => entry.level === each && wanted(entry)).length)}
        </span>
      ),
    }),
  );

  return (
    <Card>
      <p className={styles.note}>
        <Info aria-hidden="true" />
        {withoutHelp > 0 ? (
          <span>
            {`${attributesWithoutHelp(withoutHelp)} — `}
            <Link to={MISSING_HELP_PATH}>view them</Link>. Shell attributes keep their help on their
            own pages.
          </span>
        ) : (
          <span>
            Every Shell attribute has help. Shell attributes keep their help on their own pages in{' '}
            <Link to="/settings/shell-attributes">Shell attributes</Link>.
          </span>
        )}
      </p>
      <Tabs label="Level" tabs={tabs} value={level} onChange={filters.setLevel}>
        <ListToolbar
          search={{
            label: 'Search field or help text',
            placeholder: 'Search field or help text',
            value: q,
            onChange: filters.setQ,
          }}
          filters={
            <SegmentedFilter
              label="Show"
              options={[
                { value: 'all', label: 'All' },
                {
                  value: 'edited',
                  label: 'Edited',
                  count: items.filter((entry) => !entry.isDefault).length,
                },
              ]}
              value={shown}
              onChange={filters.setShown}
            />
          }
          count={{ shown: rows.length, total: ofLevel.length, noun: FIELD_NOUN }}
        />
        {rows.length === 0 ? (
          <EmptyState
            title="Nothing matches these filters"
            action={<Button onClick={filters.clear}>Clear filters</Button>}
          />
        ) : (
          <div className={settingsStyles.tableScroll}>
            <table className={settingsStyles.table}>
              <thead>
                <tr>
                  <th scope="col">Field</th>
                  <th scope="col">Help text</th>
                  <th scope="col">Status</th>
                  <th scope="col">
                    <span className="visually-hidden">Actions</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {rows.map((entry) => (
                  <HelpRow
                    key={entry.key}
                    entry={entry}
                    onEdit={() => filters.setOpenKey(entry.key)}
                    onReset={() => setResetting(entry)}
                  />
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Tabs>
      {open && (
        <FieldHelpPanel key={open.key} entry={open} onClose={() => filters.setOpenKey('')} />
      )}
      {resetting && (
        <ResetHelpDialog
          entry={resetting}
          onDone={() => setResetting(null)}
          onCancel={() => setResetting(null)}
        />
      )}
    </Card>
  );
}

interface HelpRowProps {
  entry: FieldHelpEntry;
  onEdit: () => void;
  onReset: () => void;
}

function HelpRow({ entry, onEdit, onReset }: HelpRowProps) {
  const { date } = useFormatters();
  const { show } = useToast();
  return (
    <tr>
      <th scope="row" className={styles.fieldCell}>
        <button type="button" className={settingsStyles.nameButton} onClick={onEdit}>
          {entry.label}
        </button>
      </th>
      <td className={styles.textCell}>
        {entry.text === '' ? (
          <span className={styles.empty}>Empty — no ⓘ is shown</span>
        ) : (
          entry.text
        )}
      </td>
      <td className={settingsStyles.nowrap}>
        {entry.isDefault ? (
          <StatusChip tone="neutral">Default</StatusChip>
        ) : (
          <>
            <StatusChip tone="active">Edited</StatusChip>
            {entry.updatedBy && entry.updatedAt && (
              <span className={settingsStyles.secondLine}>
                {`${entry.updatedBy.name}, ${date(localDate(entry.updatedAt))}`}
              </span>
            )}
          </>
        )}
      </td>
      <td className={settingsStyles.actionsCell}>
        <CanChange>
          <ActionsMenu
            label={entry.label}
            items={[
              { label: 'Edit', onSelect: onEdit },
              {
                label: 'Reset to default',
                hint: entry.isDefault ? ALREADY_DEFAULT : undefined,
                onSelect: () => (entry.isDefault ? show({ message: ALREADY_DEFAULT }) : onReset()),
              },
            ]}
          />
        </CanChange>
      </td>
    </tr>
  );
}
