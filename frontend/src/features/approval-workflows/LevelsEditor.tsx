import { ArrowDown, ArrowUp, Plus, Trash2 } from 'lucide-react';
import { Button, problemOf, TextField } from '@/components';
import type { FieldProblem } from '@/components';
import { PersonField } from '@/features/people';
import { emptyLevel, levelFieldName } from './levelDrafts';
import type { LevelDraft } from './levelDrafts';
import styles from './LevelsEditor.module.css';

export interface LevelsEditorProps {
  levels: LevelDraft[];
  onChange: (levels: LevelDraft[]) => void;
  /** The last save's problems; a level's are keyed "levels.<n>.name" / "levels.<n>.approverIds". */
  problems: FieldProblem[];
}

/**
 * The levels of an approval workflow (FR-002, FR-003): one block per level, in order — its name
 * and its approvers (Required) — with Move up, Move down and Remove named per level, and **Add
 * level**. Every level applies to every submission; any one approver of a level approves it.
 */
export function LevelsEditor({ levels, onChange, problems }: LevelsEditorProps) {
  function change(index: number, update: Partial<LevelDraft>) {
    onChange(levels.map((level, i) => (i === index ? { ...level, ...update } : level)));
  }

  function move(index: number, by: -1 | 1) {
    const next = [...levels];
    [next[index], next[index + by]] = [next[index + by], next[index]];
    onChange(next);
  }

  return (
    <div className={styles.editor}>
      <ol role="list" className={styles.levels}>
        {levels.map((level, index) => {
          const number = index + 1;
          return (
            <li key={level.key} className={styles.level}>
              <div className={styles.header}>
                <h3 className={styles.heading}>{`Level ${number}`}</h3>
                <div className={styles.tools}>
                  {index > 0 && (
                    <Button
                      variant="ghost"
                      size="sm"
                      icon={<ArrowUp />}
                      aria-label={`Move level ${number} up`}
                      onClick={() => move(index, -1)}
                    />
                  )}
                  {index < levels.length - 1 && (
                    <Button
                      variant="ghost"
                      size="sm"
                      icon={<ArrowDown />}
                      aria-label={`Move level ${number} down`}
                      onClick={() => move(index, 1)}
                    />
                  )}
                  {levels.length > 1 && (
                    <Button
                      variant="ghost"
                      size="sm"
                      icon={<Trash2 />}
                      aria-label={`Remove level ${number}`}
                      onClick={() => onChange(levels.filter((_, i) => i !== index))}
                    />
                  )}
                </div>
              </div>
              <TextField
                label="Level name"
                name={levelFieldName(number, 'name')}
                hint={`Shown as 'Level ${number}' when empty`}
                maxLength={60}
                value={level.name}
                onChange={(name) => change(index, { name })}
                error={problemOf(problems, levelFieldName(number, 'name'))}
              />
              <PersonField
                multiple
                label="Approvers"
                marking="Required"
                name={levelFieldName(number, 'approverIds')}
                value={level.approverIds}
                onChange={(approverIds) => change(index, { approverIds })}
                error={problemOf(problems, levelFieldName(number, 'approverIds'))}
              />
            </li>
          );
        })}
      </ol>
      <div>
        <Button icon={<Plus />} onClick={() => onChange([...levels, emptyLevel()])}>
          Add level
        </Button>
      </div>
    </div>
  );
}
