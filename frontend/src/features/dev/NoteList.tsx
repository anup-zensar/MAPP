import { Trash2 } from 'lucide-react';
import { Button } from '@/ui/Button/Button';
import { Tooltip } from '@/ui/Tooltip/Tooltip';
import type { SampleNote } from './api';
import styles from './SampleRecordsPage.module.css';

interface NoteListProps {
  notes: SampleNote[];
  onDelete: (note: SampleNote) => void;
}

/** A folder's notes, oldest first, each with its own delete button. */
export function NoteList({ notes, onDelete }: NoteListProps) {
  return (
    // role="list": some screen readers drop the list when its bullets are styled away.
    <ul role="list" className={styles.notes}>
      {notes.map((note) => (
        <li key={note.id} className={styles.note}>
          <span>{note.title}</span>
          <Tooltip content={`Delete note '${note.title}'`}>
            <Button
              variant="tertiary"
              size="sm"
              icon={<Trash2 />}
              aria-label={`Delete note '${note.title}'`}
              onClick={() => onDelete(note)}
            />
          </Tooltip>
        </li>
      ))}
    </ul>
  );
}
