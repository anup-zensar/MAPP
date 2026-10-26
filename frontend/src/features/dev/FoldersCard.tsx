import { FolderOpen, Plus } from 'lucide-react';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import type { SampleFolder, SampleNote } from './api';
import { FolderRow } from './FolderRow';
import styles from './SampleRecordsPage.module.css';

/** What people can do with the folders and notes in the table. */
export interface FolderActions {
  newFolder: () => void;
  edit: (folder: SampleFolder) => void;
  addNote: (folder: SampleFolder) => void;
  deleteFolder: (folder: SampleFolder) => void;
  deleteNote: (note: SampleNote) => void;
}

interface FoldersCardProps {
  folders: SampleFolder[];
  actions: FolderActions;
}

/** The folders as a table, or how to create the first one. */
export function FoldersCard({ folders, actions }: FoldersCardProps) {
  return (
    <Card title="Sample folders">
      {folders.length === 0 ? (
        <EmptyState
          icon={<FolderOpen />}
          title="No sample folders yet — create your first sample folder"
          action={
            // Secondary: the page header already has the one primary button.
            <Button icon={<Plus />} onClick={actions.newFolder}>
              New sample folder
            </Button>
          }
        />
      ) : (
        // On narrow screens the table may scroll sideways inside its card (design system §2).
        <div className={styles.tableScroll}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th scope="col">Name</th>
                <th scope="col">Notes</th>
                <th scope="col">Last changed</th>
                <th scope="col">
                  <span className="visually-hidden">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {folders.map((folder) => (
                <FolderRow key={folder.id} folder={folder} actions={actions} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}
