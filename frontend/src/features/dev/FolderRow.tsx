import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { useFormatters } from '@/format/useFormatters';
import type { SampleFolder } from './api';
import type { FolderActions } from './FoldersCard';
import { NoteList } from './NoteList';
import styles from './SampleRecordsPage.module.css';

/** One folder: its name and ID, its notes, its last change and its "⋯" menu. */
export function FolderRow({ folder, actions }: { folder: SampleFolder; actions: FolderActions }) {
  const { dateTime } = useFormatters();
  return (
    <tr>
      <th scope="row">
        <span className={styles.name}>{folder.name}</span>{' '}
        <span className={styles.id}>{folder.id}</span>
      </th>
      <td>
        {/* An empty value shows "—" (design system §6.3). */}
        {folder.notes.length > 0 ? (
          <NoteList notes={folder.notes} onDelete={actions.deleteNote} />
        ) : (
          '—'
        )}
      </td>
      <td>{`${folder.updatedBy.name}, ${dateTime(folder.updatedAt)}`}</td>
      <td className={styles.actionsCell}>
        <ActionsMenu
          label={folder.name}
          items={[
            { label: 'Edit', onSelect: () => actions.edit(folder) },
            { label: 'Add note', onSelect: () => actions.addNote(folder) },
            {
              label: 'Delete',
              tone: 'danger',
              separatorBefore: true,
              onSelect: () => actions.deleteFolder(folder),
            },
          ]}
        />
      </td>
    </tr>
  );
}
