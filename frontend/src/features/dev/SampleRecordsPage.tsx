import { Plus } from 'lucide-react';
import { useState } from 'react';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { SummaryCard } from '@/ui/SummaryCard/SummaryCard';
import { usePageTitle } from '@/ui/usePageTitle';
import {
  useDeleteFolder,
  useDeleteNote,
  useRecentActivity,
  useRecycleBin,
  useSampleFolders,
} from './api';
import type { SampleFolder, SampleNote } from './api';
import { FolderPanel } from './FolderPanel';
import { NotePanel } from './NotePanel';
import { ActivityLogCard } from './ActivityLogCard';
import { DeleteDialog } from './DeleteDialog';
import { FoldersCard } from './FoldersCard';
import type { FolderActions } from './FoldersCard';
import { RecentlyDeletedCard } from './RecentlyDeletedCard';
import { whatGoesWith } from './sample-text';
import styles from './SampleRecordsPage.module.css';

/** The side panel that is open. */
type OpenPanel =
  | { kind: 'newFolder' }
  | { kind: 'editFolder'; folder: SampleFolder }
  | { kind: 'addNote'; folder: SampleFolder };

/** The delete that waits for its confirmation. */
type PendingDelete = { kind: 'folder'; folder: SampleFolder } | { kind: 'note'; note: SampleNote };

/**
 * The developer page of User Story 5 (research R23, D-036): the test bench for the shared create,
 * edit, delete, Undo, restore and activity-log behaviour, with sample folders and the notes in
 * them. Only in local and development environments (DevOnly in router.tsx).
 *
 * On purpose not the full standard CRUD pattern — no search, no details view — because sample
 * records are not records people manage (plan, Complexity Tracking).
 */
export function SampleRecordsPage() {
  usePageTitle('Sample records');
  const [panel, setPanel] = useState<OpenPanel | null>(null);
  const [pendingDelete, setPendingDelete] = useState<PendingDelete | null>(null);
  const deleteFolder = useDeleteFolder();
  const deleteNote = useDeleteNote();

  const actions: FolderActions = {
    newFolder: () => setPanel({ kind: 'newFolder' }),
    edit: (folder) => setPanel({ kind: 'editFolder', folder }),
    addNote: (folder) => setPanel({ kind: 'addNote', folder }),
    deleteFolder: (folder) => setPendingDelete({ kind: 'folder', folder }),
    deleteNote: (note) => setPendingDelete({ kind: 'note', note }),
  };
  const closePanel = () => setPanel(null);
  const closeDeleteDialog = () => setPendingDelete(null);

  return (
    <>
      <PageHeader
        eyebrow="Developer"
        title="Sample records"
        description="Try the shared create, edit, delete, Undo and activity-log behaviour. Not part of the Plan module."
        actions={
          <Button variant="primary" icon={<Plus />} onClick={actions.newFolder}>
            New sample folder
          </Button>
        }
      />
      <SampleLists actions={actions} />

      {panel?.kind === 'newFolder' && <FolderPanel onClose={closePanel} />}
      {panel?.kind === 'editFolder' && <FolderPanel folder={panel.folder} onClose={closePanel} />}
      {panel?.kind === 'addNote' && <NotePanel folder={panel.folder} onClose={closePanel} />}

      {pendingDelete?.kind === 'folder' && (
        <DeleteDialog
          title={`Delete sample folder '${pendingDelete.folder.name}'?`}
          text={whatGoesWith(pendingDelete.folder)}
          confirmLabel="Delete sample folder"
          remove={() => deleteFolder.mutateAsync(pendingDelete.folder.id)}
          onClose={closeDeleteDialog}
        />
      )}
      {pendingDelete?.kind === 'note' && (
        <DeleteDialog
          title={`Delete sample note '${pendingDelete.note.title}'?`}
          text="It goes to the recycle bin. You can restore it."
          confirmLabel="Delete sample note"
          remove={() => deleteNote.mutateAsync(pendingDelete.note.id)}
          onClose={closeDeleteDialog}
        />
      )}
    </>
  );
}

/** The summary cards and the page's cards, once its three lists have arrived. */
function SampleLists({ actions }: { actions: FolderActions }) {
  const folders = useSampleFolders();
  const bin = useRecycleBin();
  const log = useRecentActivity();

  // A list that is being refreshed keeps its last answer on screen (design system §5).
  if (folders.data && bin.data && log.data) {
    return (
      <div className={styles.sections}>
        <SummaryCards folders={folders.data.items} inRecycleBin={bin.data.total} />
        <FoldersCard folders={folders.data.items} actions={actions} />
        {/* Shown once they have something to show. */}
        {bin.data.items.length > 0 && <RecentlyDeletedCard entries={bin.data.items} />}
        {log.data.items.length > 0 && <ActivityLogCard entries={log.data.items} />}
      </div>
    );
  }

  const failed = [folders, bin, log].filter((list) => list.isError);
  const [firstFailed] = failed;
  if (firstFailed) {
    return (
      <Card>
        <ErrorState
          error={firstFailed.error}
          onRetry={() => failed.forEach((list) => list.refetch())}
        />
      </Card>
    );
  }
  return <Skeleton lines={6} height="18px" />;
}

interface SummaryCardsProps {
  folders: SampleFolder[];
  /** How many entries the recycle bin holds. */
  inRecycleBin: number;
}

/** Sample folders, sample notes and what is in the recycle bin, counted. */
function SummaryCards({ folders, inRecycleBin }: SummaryCardsProps) {
  const notes = folders.reduce((count, folder) => count + folder.notes.length, 0);

  return (
    <div className={styles.summary}>
      <SummaryCard label="Sample folders" value={folders.length} />
      <SummaryCard label="Sample notes" value={notes} />
      <SummaryCard label="In the recycle bin" value={inRecycleBin} />
    </div>
  );
}
