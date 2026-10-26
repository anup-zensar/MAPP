import { ListChecks, Plus } from 'lucide-react';
import { useEffect, useState } from 'react';
import { Button, LinkButton } from '@/ui/Button/Button';
import { CanChange } from '@/ui/CanChange/CanChange';
import { Card } from '@/ui/Card/Card';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { ListToolbar } from '@/ui/ListToolbar/ListToolbar';
import { SegmentedFilter } from '@/ui/SegmentedFilter/SegmentedFilter';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { bySection, hasHelpState, hasStatus, lacksHelp, matches } from './attribute-filtering';
import { LEVEL_WORDS, LEVELS, levelMidSentence, lockReason } from './attribute-words';
import type { AttributeLevel } from './attribute-words';
import { useAllShellAttributes, useMoveAttribute } from './api';
import type { AttributeMove, ShellAttribute } from './api';
import { AttributeRow } from './AttributeRow';
import { OrderButton } from './OrderButton';
import styles from './ShellAttributes.module.css';
import { useAttributeActions } from './useAttributeActions';
import type { AttributeFilters } from './useAttributeFilters';
import { useFailureToast } from './useStatusChange';

const ATTRIBUTE_NOUN = { one: 'Shell attribute', many: 'Shell attributes' };

const FILTERED_REASON = 'Clear the search and filters to change the order.';

const DIRECTIONS = ['up', 'down'] as const;

/** The toolbar and the table, once the attributes have arrived. */
export function AttributeList({ filters }: { filters: AttributeFilters }) {
  const attributes = useAllShellAttributes();
  const actions = useAttributeActions();
  const move = useMoveAttribute();
  const showFailure = useFailureToast('Shell attribute');
  const [announcement, setAnnouncement] = useState('');
  // The move button that was used last: the focus stays on it (SC-009).
  const [focusTarget, setFocusTarget] = useState<string | null>(null);
  const { status, q, help } = filters;

  // A moved row is taken out of the page and put back in its new place, which loses the focus:
  // it goes back to the button that moved it.
  useEffect(() => {
    const focusLost = document.activeElement === null || document.activeElement === document.body;
    if (focusTarget && focusLost) {
      document.getElementById(focusTarget)?.focus();
    }
  }, [focusTarget, attributes.data]);

  if (!attributes.data) {
    return (
      <Card>
        {attributes.isError ? (
          <ErrorState error={attributes.error} onRetry={() => attributes.refetch()} />
        ) : (
          <Skeleton lines={6} height="18px" />
        )}
      </Card>
    );
  }

  const all = attributes.data.items;
  // Arriving from Field help without a level: the first level with attributes lacking help.
  const level: AttributeLevel =
    help === 'missing' && !filters.levelGiven
      ? (LEVELS.find((each) =>
          all.some(
            (attribute) =>
              attribute.level === each && hasStatus(attribute, status) && lacksHelp(attribute),
          ),
        ) ?? filters.level)
      : filters.level;
  const ofLevel = all.filter((attribute) => attribute.level === level);
  const shown = ofLevel.filter(
    (attribute) =>
      hasStatus(attribute, status) && hasHelpState(attribute, help) && matches(attribute, q),
  );
  const withoutHelp = ofLevel.filter(
    (attribute) => hasStatus(attribute, status) && lacksHelp(attribute),
  ).length;
  // The order can change only while every attribute of the level is in view (SC-009).
  const orderLocked = shown.length < ofLevel.length ? FILTERED_REASON : undefined;
  const levelSections = bySection(ofLevel);

  async function runMove(request: AttributeMove, buttonId: string, name: string) {
    if (move.isPending) {
      return;
    }
    setFocusTarget(buttonId);
    try {
      await move.mutateAsync(request);
      setAnnouncement(`Moved '${name}' ${request.direction}.`);
    } catch (error) {
      showFailure(error);
    }
  }

  /** Why the attribute can't move one place up or down, if it can't. */
  function attributeMoveReason(attribute: ShellAttribute, direction: 'up' | 'down') {
    const locked = orderLocked ?? lockReason(attribute, 'position');
    if (locked) {
      return locked;
    }
    const section = levelSections.find((each) => each.items.includes(attribute))?.items ?? [];
    const edge = direction === 'up' ? section[0] : section[section.length - 1];
    if (edge === attribute) {
      const place = direction === 'up' ? 'first' : 'last';
      return `'${attribute.label}' is already ${place} in its section.`;
    }
    return undefined;
  }

  /** Why the section can't move one place up or down, if it can't. */
  function sectionMoveReason(name: string, direction: 'up' | 'down') {
    if (orderLocked) {
      return orderLocked;
    }
    const index = levelSections.findIndex((section) => section.name === name);
    if (direction === 'up' && index === 0) {
      return 'This section is already first.';
    }
    if (direction === 'down' && index === levelSections.length - 1) {
      return 'This section is already last.';
    }
    return undefined;
  }

  const levelCounts = LEVELS.map((each) => ({
    value: each,
    label: LEVEL_WORDS[each],
    count: all.filter(
      (attribute) =>
        attribute.level === each && hasStatus(attribute, status) && hasHelpState(attribute, help),
    ).length,
  }));
  const statusCounts = {
    active: ofLevel.filter((attribute) => hasStatus(attribute, 'active')).length,
    inactive: ofLevel.filter((attribute) => hasStatus(attribute, 'inactive')).length,
    all: ofLevel.length,
  };

  return (
    <div className={styles.stack}>
      {/* The list toolbar sits inside the list's card, as on every Settings list (§2.4 A). */}
      <Card>
        <div className={styles.levels}>
          <SegmentedFilter
            label="Level"
            options={levelCounts}
            value={level}
            onChange={filters.setLevel}
          />
        </div>
        <ListToolbar
          search={{
            label: 'Search label or section',
            placeholder: 'Search label or section',
            value: q,
            onChange: filters.setQ,
          }}
          status={{ value: status, onChange: filters.setStatus, counts: statusCounts }}
          filters={
            <SegmentedFilter
              label="Help"
              options={[
                { value: 'all', label: 'Any help' },
                { value: 'missing', label: 'No help yet', count: withoutHelp },
              ]}
              value={help}
              onChange={(value) => filters.setHelp(value, level)}
            />
          }
          count={{ shown: shown.length, total: ofLevel.length, noun: ATTRIBUTE_NOUN }}
        />
        {ofLevel.length === 0 ? (
          <EmptyState
            icon={<ListChecks />}
            title={`No ${levelMidSentence(level)} attributes yet — create the first one`}
            action={
              <CanChange>
                <LinkButton icon={<Plus />} to={`/settings/shell-attributes/new?level=${level}`}>
                  New Shell attribute
                </LinkButton>
              </CanChange>
            }
          />
        ) : shown.length === 0 ? (
          <EmptyState
            title="Nothing matches these filters"
            action={<Button onClick={() => filters.clear(level)}>Clear filters</Button>}
          />
        ) : (
          <div className={styles.tableScroll}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th scope="col">Order</th>
                  <th scope="col">Label</th>
                  <th scope="col">Type</th>
                  <th scope="col">Marking</th>
                  <th scope="col" className={styles.number}>
                    Options
                  </th>
                  <th scope="col">Depends on / shown when</th>
                  <th scope="col">Used by</th>
                  <th scope="col">Status</th>
                  <th scope="col">
                    <span className="visually-hidden">Actions</span>
                  </th>
                </tr>
              </thead>
              {bySection(shown).map((section) => {
                const first = ofLevel.find((attribute) => attribute.section === section.name);
                return (
                  <tbody key={section.name}>
                    <tr className={styles.sectionRow}>
                      <th scope="colgroup" colSpan={9}>
                        <div className={styles.sectionHeader}>
                          <span className={styles.sectionName}>{section.name}</span>
                          {first &&
                            DIRECTIONS.map((direction) => (
                              <OrderButton
                                key={direction}
                                id={`move-section-${first.id}-${direction}`}
                                label={`Move section '${section.name}' ${direction}`}
                                direction={direction}
                                disabledReason={sectionMoveReason(section.name, direction)}
                                onMove={() =>
                                  runMove(
                                    { attribute: first, what: 'section', direction },
                                    `move-section-${first.id}-${direction}`,
                                    section.name,
                                  )
                                }
                              />
                            ))}
                        </div>
                      </th>
                    </tr>
                    {section.items.map((attribute) => (
                      <AttributeRow
                        key={attribute.id}
                        attribute={attribute}
                        actions={actions}
                        moveReason={(direction) => attributeMoveReason(attribute, direction)}
                        onMove={(direction, buttonId) =>
                          runMove(
                            { attribute, what: 'attribute', direction },
                            buttonId,
                            attribute.label,
                          )
                        }
                      />
                    ))}
                  </tbody>
                );
              })}
            </table>
          </div>
        )}
      </Card>
      {/* Says where a moved row went; the focus stays on its button (SC-009). */}
      <p role="status" className="visually-hidden">
        {announcement}
      </p>
      {actions.dialog}
    </div>
  );
}
