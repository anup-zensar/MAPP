import { ArrowDown, ArrowUp, ListChecks, Plus } from 'lucide-react';
import { useEffect, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { Button, LinkButton } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { SegmentedFilter } from '@/ui/SegmentedFilter/SegmentedFilter';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { Tooltip } from '@/ui/Tooltip/Tooltip';
import { formatWhole } from '@/format/decimal';
import { usePageTitle } from '@/ui/usePageTitle';
import { ListToolbar } from '../shared/ListToolbar';
import {
  dependsOnText,
  isChoiceType,
  isLevel,
  LEVEL_WORDS,
  LEVELS,
  levelMidSentence,
  lockReason,
  MARKING_WORDS,
  shownWhenText,
  TYPE_WORDS,
} from './attributeWords';
import type { AttributeLevel } from './attributeWords';
import { useAllShellAttributes, useMoveAttribute } from './shellAttributesApi';
import type { AttributeMove, ShellAttribute } from './shellAttributesApi';
import styles from './ShellAttributes.module.css';
import { useAttributeActions } from './useAttributeActions';
import { useFailureToast } from './useStatusChange';
import { CanChange, SettingsViewOnlyNote } from '../shared/CanChange';
import { useCanChange } from '../shared/useCanChange';

type StatusFilter = 'active' | 'inactive' | 'all';

type HelpFilter = 'all' | 'missing';

const ATTRIBUTE_NOUN = { one: 'Shell attribute', many: 'Shell attributes' };

const FILTERED_REASON = 'Clear the search and filters to change the order.';

/** The list's filters, kept in the address so Back and links keep them (plan "Routes"). */
function useFilters() {
  const [params, setParams] = useSearchParams();
  const levelParam = params.get('level');
  const statusParam = params.get('status');
  const level: AttributeLevel = isLevel(levelParam) ? levelParam : 'plan';
  const status: StatusFilter =
    statusParam === 'inactive' || statusParam === 'all' ? statusParam : 'active';
  const q = params.get('q') ?? '';
  // Only the attributes without help (016 FR-048), as Settings › Field help links here.
  const help: HelpFilter = params.get('help') === 'missing' ? 'missing' : 'all';

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
    /** The address names a level; without one, the help filter may choose it. */
    levelGiven: isLevel(levelParam),
    status,
    q,
    help,
    // With the help filter on, the level stays in the address, so it is not chosen again.
    setLevel: (value: AttributeLevel) => change('level', value, help === 'missing' ? '' : 'plan'),
    /**
     * Turns the help filter on or off. The level on screen (`shown`), which the filter may have
     * chosen, goes into the address so the list stays on it.
     */
    setHelp: (value: HelpFilter, shown: AttributeLevel) =>
      setParams(
        (current) => {
          const next = new URLSearchParams(current);
          if (value === 'all') {
            next.delete('help');
          } else {
            next.set('help', value);
          }
          keepLevel(next, shown);
          return next;
        },
        { replace: true },
      ),
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
          keepLevel(next, shown);
          return next;
        },
        { replace: true },
      ),
  };
}

/** Puts the level on screen into the address, unless it is the default (plan) level. */
function keepLevel(params: URLSearchParams, shown: AttributeLevel) {
  if (shown === 'plan') {
    params.delete('level');
  } else {
    params.set('level', shown);
  }
}

/** True when the attribute has no help text, so its ⓘ is missing (016 FR-048). */
function lacksHelp(attribute: ShellAttribute): boolean {
  return attribute.helpText.trim() === '';
}

/** True when the attribute passes the help filter. */
function hasHelpState(attribute: ShellAttribute, help: HelpFilter): boolean {
  return help === 'all' || lacksHelp(attribute);
}

/** True when the attribute has the status the filter asks for. */
function hasStatus(attribute: ShellAttribute, status: StatusFilter): boolean {
  return status === 'all' || attribute.active === (status === 'active');
}

/** True when the label or the section contains the search text, ignoring letter case. */
function matches(attribute: ShellAttribute, q: string): boolean {
  const text = q.trim().toLowerCase();
  return (
    text === '' ||
    attribute.label.toLowerCase().includes(text) ||
    attribute.section.toLowerCase().includes(text)
  );
}

/** The attributes in display order, as one group per section (data model §4.2). */
function bySection(attributes: ShellAttribute[]): { name: string; items: ShellAttribute[] }[] {
  const sections: { name: string; items: ShellAttribute[] }[] = [];
  for (const attribute of attributes) {
    const last = sections[sections.length - 1];
    if (last && last.name === attribute.section) {
      last.items.push(attribute);
    } else {
      sections.push({ name: attribute.section, items: [attribute] });
    }
  }
  return sections;
}

/**
 * Shell attributes (feature 002, US2, FR-016 – FR-019, SC-009): one level's attributes in the
 * order of its form, grouped by section, with the move buttons that set that order.
 */
export function ShellAttributesPage() {
  usePageTitle('Shell attributes');
  const filters = useFilters();

  return (
    <>
      <PageHeader
        breadcrumb={[{ label: 'Settings', to: '/settings' }, { label: 'Shell attributes' }]}
        title="Shell attributes"
        description="Shell's own fields on each level's form. Changes apply to every form straight away."
        actions={
          <CanChange>
            <LinkButton
              variant="primary"
              icon={<Plus />}
              to={`/settings/shell-attributes/new?level=${filters.level}`}
            >
              New Shell attribute
            </LinkButton>
          </CanChange>
        }
      />
      <SettingsViewOnlyNote />
      <AttributeList filters={filters} />
    </>
  );
}

type Filters = ReturnType<typeof useFilters>;

/** The toolbar and the table, once the attributes have arrived. */
function AttributeList({ filters }: { filters: Filters }) {
  const attributes = useAllShellAttributes();
  const navigate = useNavigate();
  const actions = useAttributeActions();
  const canChange = useCanChange();
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
                            (['up', 'down'] as const).map((direction) => (
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
                      <tr key={attribute.id}>
                        <td className={styles.orderCell}>
                          {(['up', 'down'] as const).map((direction) => (
                            <OrderButton
                              key={direction}
                              id={`move-${attribute.id}-${direction}`}
                              label={`Move '${attribute.label}' ${direction}`}
                              direction={direction}
                              disabledReason={attributeMoveReason(attribute, direction)}
                              onMove={() =>
                                runMove(
                                  { attribute, what: 'attribute', direction },
                                  `move-${attribute.id}-${direction}`,
                                  attribute.label,
                                )
                              }
                            />
                          ))}
                        </td>
                        <th scope="row" className={styles.nameCell}>
                          <span className={styles.nameLine}>
                            <Link
                              to={`/settings/shell-attributes/${attribute.id}`}
                              className={styles.name}
                            >
                              {attribute.label}
                            </Link>
                            {lacksHelp(attribute) && (
                              <StatusChip tone="review">No help yet</StatusChip>
                            )}
                          </span>
                        </th>
                        <td className={styles.nowrap}>{TYPE_WORDS[attribute.type]}</td>
                        <td className={styles.nowrap}>{MARKING_WORDS[attribute.marking]}</td>
                        <td className={styles.number}>
                          {isChoiceType(attribute.type) ? formatWhole(attribute.optionCount) : '—'}
                        </td>
                        <td>
                          {dependsOnText(attribute) && (
                            <span className={styles.line}>{dependsOnText(attribute)}</span>
                          )}
                          <span className={styles.line}>{shownWhenText(attribute)}</span>
                        </td>
                        <td>{attribute.usage.summary}</td>
                        <td className={styles.nowrap}>
                          <StatusChip tone={attribute.active ? 'positive' : 'neutral'}>
                            {attribute.active ? 'Active' : 'Inactive'}
                          </StatusChip>
                        </td>
                        <td className={styles.actionsCell}>
                          <ActionsMenu
                            label={attribute.label}
                            items={[
                              {
                                label: 'View',
                                onSelect: () =>
                                  navigate(`/settings/shell-attributes/${attribute.id}`),
                              },
                              ...(canChange
                                ? [
                                    {
                                      label: 'Edit',
                                      onSelect: () =>
                                        navigate(`/settings/shell-attributes/${attribute.id}/edit`),
                                    },
                                    {
                                      label: attribute.active ? 'Set inactive' : 'Reactivate',
                                      onSelect: () =>
                                        actions.setActive(attribute, !attribute.active),
                                    },
                                    {
                                      label: 'Delete',
                                      tone: 'danger',
                                      separatorBefore: true,
                                      onSelect: () => actions.askDelete(attribute),
                                    } as const,
                                  ]
                                : []),
                            ]}
                          />
                        </td>
                      </tr>
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

interface OrderButtonProps {
  id: string;
  /** "Move 'Region' up", "Move section 'Lubricants attributes' down". */
  label: string;
  direction: 'up' | 'down';
  /** Why it can't move now; the button stays focusable and its tip says why (SC-009). */
  disabledReason?: string;
  onMove: () => void;
}

/** One move button. At an edge or while filtering it is aria-disabled, never removed. */
function OrderButton({ id, label, direction, disabledReason, onMove }: OrderButtonProps) {
  return (
    <Tooltip content={disabledReason ?? label}>
      <Button
        id={id}
        variant="tertiary"
        size="sm"
        icon={direction === 'up' ? <ArrowUp /> : <ArrowDown />}
        aria-label={label}
        aria-disabled={disabledReason ? true : undefined}
        className={disabledReason ? styles.unavailable : undefined}
        onClick={() => {
          if (!disabledReason) {
            onMove();
          }
        }}
      />
    </Tooltip>
  );
}
