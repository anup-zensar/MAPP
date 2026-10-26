import { Link, useNavigate } from 'react-router';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { useCanChange } from '@/ui/useCanChange';
import { formatWhole } from '@/format/decimal';
import { lacksHelp } from './attribute-filtering';
import {
  dependsOnText,
  isChoiceType,
  MARKING_WORDS,
  shownWhenText,
  TYPE_WORDS,
} from './attribute-words';
import type { ShellAttribute } from './api';
import { OrderButton } from './OrderButton';
import styles from './ShellAttributes.module.css';
import type { useAttributeActions } from './useAttributeActions';

const DIRECTIONS = ['up', 'down'] as const;

export interface AttributeRowProps {
  attribute: ShellAttribute;
  actions: ReturnType<typeof useAttributeActions>;
  /** Why it can't move one place that way, if it can't. */
  moveReason: (direction: 'up' | 'down') => string | undefined;
  onMove: (direction: 'up' | 'down', buttonId: string) => void;
}

/** One attribute of the list: its move buttons, its facts and its actions. */
export function AttributeRow({ attribute, actions, moveReason, onMove }: AttributeRowProps) {
  const navigate = useNavigate();
  const canChange = useCanChange();
  const page = `/settings/shell-attributes/${attribute.id}`;

  return (
    <tr>
      <td className={styles.orderCell}>
        {DIRECTIONS.map((direction) => (
          <OrderButton
            key={direction}
            id={`move-${attribute.id}-${direction}`}
            label={`Move '${attribute.label}' ${direction}`}
            direction={direction}
            disabledReason={moveReason(direction)}
            onMove={() => onMove(direction, `move-${attribute.id}-${direction}`)}
          />
        ))}
      </td>
      <th scope="row" className={styles.nameCell}>
        <span className={styles.nameLine}>
          <Link to={page} className={styles.name}>
            {attribute.label}
          </Link>
          {lacksHelp(attribute) && <StatusChip tone="review">No help yet</StatusChip>}
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
            { label: 'View', onSelect: () => navigate(page) },
            ...(canChange
              ? [
                  { label: 'Edit', onSelect: () => navigate(`${page}/edit`) },
                  {
                    label: attribute.active ? 'Set inactive' : 'Reactivate',
                    onSelect: () => actions.setActive(attribute, !attribute.active),
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
  );
}
