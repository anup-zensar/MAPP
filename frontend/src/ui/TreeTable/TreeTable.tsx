import { ChevronDown, ChevronRight } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import type { KeyboardEvent, ReactNode } from 'react';
import { Link, useNavigate } from 'react-router';
import { cx } from '../cx';
import { ActionsMenu } from '../ActionsMenu/ActionsMenu';
import type { ActionsMenuItem } from '../ActionsMenu/ActionsMenu';
import { LevelTag } from '../LevelTag/LevelTag';
import type { PlanningLevel } from '../LevelTag/LevelTag';
import styles from './TreeTable.module.css';

/** One row of the tree and the rows under it. */
export interface TreeNode {
  id: string;
  level: PlanningLevel;
  name: string;
  /** Where the name leads (the record's page or panel); Enter on the row goes there too. */
  href?: string;
  /** Opens the record when it has no address (Enter on the row, a click on the name). */
  onOpen?: () => void;
  /**
   * The cells by column key. The first column's cell is shown after the name (an ID, a count of
   * children).
   */
  cells: Record<string, ReactNode>;
  children: TreeNode[];
  /** The row's ⋯ menu; its button is named "Actions for <name>". */
  actions?: ActionsMenuItem[];
  /** The hover grey, for the record just created, moved or copied; it is scrolled into view. */
  highlighted?: boolean;
}

export interface TreeTableColumn {
  key: string;
  header: string;
  /** 'end' right-aligns the column and keeps its values on one line, for money (design system §4). */
  align?: 'start' | 'end';
  /** The narrowest the column may become, so a person or team never squeezes: "160px". */
  minWidth?: string;
}

/** The totals row under the tree, by column key. */
export type TreeTableFooter = Record<string, ReactNode>;

export interface TreeTableProps {
  /** Names the tree for screen readers: "Plan tree". */
  label: string;
  /** The columns; the first holds the names. */
  columns: TreeTableColumn[];
  nodes: TreeNode[];
  /** The rows whose children are shown. */
  expandedIds: Set<string>;
  onToggle: (id: string) => void;
  /** Shown when there are no rows. */
  emptyText?: string;
  footer?: TreeTableFooter;
}

interface Row {
  node: TreeNode;
  depth: number;
  position: number;
  siblings: number;
  parentId?: string;
}

/**
 * The tree table of plans (feature 004, design system §5): a level bar and tag on every row,
 * names as links, money right-aligned, a ⋯ menu per row and an optional totals row. It is an ARIA
 * treegrid with one row in the tab order: ↑ ↓ move, → opens a row or goes to its first child,
 * ← closes it or goes to its parent, Home and End go to the first and last row, Enter opens the
 * record. Each row with children has an "Expand <name>" / "Collapse <name>" button.
 */
export function TreeTable({
  label,
  columns,
  nodes,
  expandedIds,
  onToggle,
  emptyText,
  footer,
}: TreeTableProps) {
  const navigate = useNavigate();
  const table = useRef<HTMLTableElement>(null);
  const [focusedId, setFocusedId] = useState<string | null>(null);
  const rows = visibleRows(nodes, expandedIds);
  const hasActions = rows.some((row) => row.node.actions && row.node.actions.length > 0);
  const tabbableId = rows.some((row) => row.node.id === focusedId) ? focusedId : rows[0]?.node.id;
  const highlightedId = rows.find((row) => row.node.highlighted)?.node.id;
  const [nameColumn, ...otherColumns] = columns;

  useEffect(() => {
    // The record just created, moved or copied comes into view (feature 005, FR-004).
    if (highlightedId) {
      rowOf(table.current, highlightedId)?.scrollIntoView({ block: 'center' });
    }
  }, [highlightedId]);

  function focusRow(id: string | undefined) {
    if (id) {
      setFocusedId(id);
      rowOf(table.current, id)?.focus();
    }
  }

  function open(node: TreeNode) {
    if (node.onOpen) {
      node.onOpen();
    } else if (node.href) {
      void navigate(node.href);
    }
  }

  function onKeyDown(event: KeyboardEvent<HTMLTableRowElement>, row: Row, index: number) {
    // Keys typed in a control inside the row (the ⋯ menu) are that control's.
    if (event.target !== event.currentTarget) {
      return;
    }
    const { node } = row;
    const hasChildren = node.children.length > 0;
    const expanded = expandedIds.has(node.id);
    const keys: Record<string, () => void> = {
      ArrowDown: () => focusRow(rows[index + 1]?.node.id),
      ArrowUp: () => focusRow(rows[index - 1]?.node.id),
      Home: () => focusRow(rows[0]?.node.id),
      End: () => focusRow(rows[rows.length - 1]?.node.id),
      ArrowRight: () => {
        if (hasChildren && !expanded) {
          onToggle(node.id);
        } else if (hasChildren) {
          focusRow(node.children[0].id);
        }
      },
      ArrowLeft: () => {
        if (hasChildren && expanded) {
          onToggle(node.id);
        } else {
          focusRow(row.parentId);
        }
      },
      Enter: () => open(node),
    };
    const action = keys[event.key];
    if (action) {
      event.preventDefault();
      action();
    }
  }

  return (
    <div className={styles.scroll}>
      <table ref={table} role="treegrid" aria-label={label} className={styles.table}>
        <thead>
          <tr>
            {columns.map((column) => (
              <th
                key={column.key}
                scope="col"
                className={cx(column.align === 'end' && styles.end)}
                style={column.minWidth ? { minWidth: column.minWidth } : undefined}
              >
                {column.header}
              </th>
            ))}
            {hasActions && (
              <th scope="col" className={styles.actionsColumn}>
                <span className="visually-hidden">Actions</span>
              </th>
            )}
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 && emptyText && (
            <tr>
              <td colSpan={columns.length + (hasActions ? 1 : 0)} className={styles.empty}>
                {emptyText}
              </td>
            </tr>
          )}
          {rows.map((row, index) => {
            const { node } = row;
            const hasChildren = node.children.length > 0;
            const expanded = hasChildren && expandedIds.has(node.id);
            return (
              <tr
                key={node.id}
                data-row-id={node.id}
                aria-level={row.depth}
                aria-posinset={row.position}
                aria-setsize={row.siblings}
                aria-expanded={hasChildren ? expanded : undefined}
                tabIndex={node.id === tabbableId ? 0 : -1}
                onFocus={(event) => {
                  if (event.target === event.currentTarget) {
                    setFocusedId(node.id);
                  }
                }}
                onKeyDown={(event) => onKeyDown(event, row, index)}
                className={cx(styles.row, node.highlighted && styles.highlighted)}
              >
                <td className={cx(styles.nameCell, styles[node.level])}>
                  <div className={styles.name} style={{ paddingLeft: `${(row.depth - 1) * 20}px` }}>
                    {hasChildren ? (
                      <button
                        type="button"
                        tabIndex={-1}
                        className={styles.chevron}
                        aria-label={`${expanded ? 'Collapse' : 'Expand'} ${node.name}`}
                        onClick={() => onToggle(node.id)}
                      >
                        {expanded ? <ChevronDown /> : <ChevronRight />}
                      </button>
                    ) : (
                      <span className={styles.chevronSpace} />
                    )}
                    <LevelTag level={node.level} />
                    <span className={styles.title}>
                      {node.href ? (
                        <Link to={node.href} tabIndex={-1} className={styles.link}>
                          {node.name}
                        </Link>
                      ) : node.onOpen ? (
                        <button
                          type="button"
                          tabIndex={-1}
                          className={styles.linkButton}
                          onClick={node.onOpen}
                        >
                          {node.name}
                        </button>
                      ) : (
                        <span className={styles.text}>{node.name}</span>
                      )}
                      {nameColumn && node.cells[nameColumn.key] !== undefined && (
                        <span className={styles.extra}>{node.cells[nameColumn.key]}</span>
                      )}
                    </span>
                  </div>
                </td>
                {otherColumns.map((column) => (
                  <td key={column.key} className={cx(column.align === 'end' && styles.end)}>
                    {node.cells[column.key]}
                  </td>
                ))}
                {hasActions && (
                  <td className={styles.actionsCell}>
                    {node.actions && node.actions.length > 0 && (
                      <ActionsMenu label={node.name} items={node.actions} />
                    )}
                  </td>
                )}
              </tr>
            );
          })}
        </tbody>
        {footer && (
          <tfoot>
            <tr className={styles.footer}>
              {columns.map((column) => (
                <td key={column.key} className={cx(column.align === 'end' && styles.end)}>
                  {footer[column.key]}
                </td>
              ))}
              {hasActions && <td />}
            </tr>
          </tfoot>
        )}
      </table>
    </div>
  );
}

/** The rows on screen: every node whose parents are all expanded, in tree order. */
function visibleRows(nodes: TreeNode[], expandedIds: Set<string>): Row[] {
  const rows: Row[] = [];
  function walk(level: TreeNode[], depth: number, parentId?: string) {
    level.forEach((node, index) => {
      rows.push({ node, depth, position: index + 1, siblings: level.length, parentId });
      if (expandedIds.has(node.id)) {
        walk(node.children, depth + 1, node.id);
      }
    });
  }
  walk(nodes, 1);
  return rows;
}

/** The row of a record in the table. */
function rowOf(table: HTMLTableElement | null, id: string): HTMLTableRowElement | undefined {
  const all = table?.querySelectorAll<HTMLTableRowElement>('tr[data-row-id]') ?? [];
  return [...all].find((element) => element.dataset.rowId === id);
}
