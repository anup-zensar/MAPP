import { ChevronDown, ChevronRight } from 'lucide-react';
import type { MouseEvent, ReactNode } from 'react';
import { Link } from 'react-router';
import { InfoTip, LevelTag, StatusChip, Tooltip } from '@/components';
import { CONTEXT_LABEL, CONTEXT_TIP } from './allWorkText';
import type { AllWorkRow } from './allWorkTypes';
import { Highlight } from './Highlight';
import { rowHref } from './rowHref';
import styles from './RowName.module.css';
import { statusWord } from './statusWords';
import type { VisibleRow } from './viewState';

/** Names longer than this may end with an ellipsis; the full name shows on hover and focus. */
const LONG_NAME = 36;

export interface RowNameProps {
  visible: VisibleRow;
  /** The searched text, marked in the name and the excerpt (FR-006). */
  search: string;
  /** Shows the status chip after the name, when no column shows it (D-118). */
  showStatus: boolean;
  onToggle: (key: string) => void;
  onOpen: (row: AllWorkRow) => void;
  /** A line under the name (Budget & cost: "Line items exceed the planned cost by …"). */
  children?: ReactNode;
}

/**
 * The name of a row in every view (plan "Grid", FR-011): indented by depth, the Expand or
 * Collapse button, the level tag, the name as a link to the record, the AOP tag, the status chip
 * when needed, "Shown for context" on context rows (FR-005) and the excerpt of a search hit.
 */
export function RowName({ visible, search, showStatus, onToggle, onOpen, children }: RowNameProps) {
  const { row, expanded, childCount } = visible;

  function open(event: MouseEvent<HTMLAnchorElement>) {
    // Ctrl, Shift or the middle button open the page in a new tab as usual.
    if (event.button !== 0 || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) {
      return;
    }
    event.preventDefault();
    onOpen(row);
  }

  const link = (
    <Link to={rowHref(row)} className={styles.link} onClick={open}>
      <Highlight text={row.name} search={search} />
    </Link>
  );

  return (
    <div className={styles.name} style={{ paddingLeft: `${row.depth * 24}px` }}>
      {childCount > 0 ? (
        <button
          type="button"
          className={styles.toggle}
          aria-expanded={expanded}
          aria-label={`${expanded ? 'Collapse' : 'Expand'} ${row.name}`}
          onClick={() => onToggle(row.key)}
        >
          {expanded ? <ChevronDown aria-hidden="true" /> : <ChevronRight aria-hidden="true" />}
        </button>
      ) : (
        <span className={styles.toggleSpace} />
      )}
      <div className={styles.text}>
        <div className={styles.line}>
          {row.level !== 'unassigned' && <LevelTag level={row.level} />}
          {row.name.length > LONG_NAME ? <Tooltip content={row.name}>{link}</Tooltip> : link}
          {row.level === 'plan' && row.year !== undefined && (
            // Plan rows show their year: All work may show several (016 FR-034).
            <span className={styles.year}>{row.year}</span>
          )}
          {row.strategic && <span className={styles.aop}>AOP</span>}
          {showStatus && row.status && (
            <StatusChip tone={row.status.tone}>
              {statusWord(row.status.label, row.status.tone)}
            </StatusChip>
          )}
        </div>
        {row.role === 'context' && (
          <span className={styles.context}>
            {CONTEXT_LABEL}
            <InfoTip label={CONTEXT_LABEL} text={CONTEXT_TIP} />
          </span>
        )}
        {row.searchHit?.excerpt && (
          <span className={styles.excerpt}>
            <Highlight text={row.searchHit.excerpt} search={search} />
          </span>
        )}
        {children}
      </div>
    </div>
  );
}
