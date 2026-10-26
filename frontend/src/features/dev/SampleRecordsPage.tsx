import { FolderOpen, Plus, RotateCcw, Trash2 } from 'lucide-react';
import { useState } from 'react';
import { ApiError, resolvedMessage } from '@/api/errors';
import type { components } from '@/api/generated/schema';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { SummaryCard } from '@/ui/SummaryCard/SummaryCard';
import { useToast } from '@/ui/Toast/useToast';
import { Tooltip } from '@/ui/Tooltip/Tooltip';
import { useFormatters } from '@/lib/useFormatters';
import { usePageTitle } from '@/lib/usePageTitle';
import { useUndoableDelete } from '@/lib/useUndoableDelete';
import {
  SAMPLE_PAGE_LISTS,
  useDeleteFolder,
  useDeleteNote,
  useRecentActivity,
  useRecycleBin,
  useRestoreFromBin,
  useSampleFolders,
} from './sampleApi';
import type { ActivityLogEntry, RecycleBinEntry, SampleFolder, SampleNote } from './sampleApi';
import { FolderPanel, NotePanel } from './SamplePanels';
import styles from './SampleRecordsPage.module.css';

type DeleteResult = components['schemas']['DeleteResult'];

/** The side panel that is open. */
type OpenPanel =
  | { kind: 'newFolder' }
  | { kind: 'editFolder'; folder: SampleFolder }
  | { kind: 'addNote'; folder: SampleFolder };

/** The delete that waits for its confirmation. */
type PendingDelete = { kind: 'folder'; folder: SampleFolder } | { kind: 'note'; note: SampleNote };

/** What people can do with the folders and notes in the table. */
interface FolderActions {
  newFolder: () => void;
  edit: (folder: SampleFolder) => void;
  addNote: (folder: SampleFolder) => void;
  deleteFolder: (folder: SampleFolder) => void;
  deleteNote: (note: SampleNote) => void;
}

/** "Sample folder 'Ideas' moved to the recycle bin" with Undo, restorable on this page too. */
const UNDO_OPTIONS = {
  invalidate: SAMPLE_PAGE_LISTS,
  altText: 'Restore it from Recently deleted on this page',
};

/** Joins "2 sample notes" and "1 campaign" as "2 sample notes and 1 campaign". */
const AND_LIST = new Intl.ListFormat('en-GB', { type: 'conjunction' });

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
  if (failed.length > 0) {
    return (
      <Card>
        <ErrorState
          error={failed[0].error}
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

interface FoldersCardProps {
  folders: SampleFolder[];
  actions: FolderActions;
}

/** The folders as a table, or how to create the first one. */
function FoldersCard({ folders, actions }: FoldersCardProps) {
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

/** One folder: its name and ID, its notes, its last change and its "⋯" menu. */
function FolderRow({ folder, actions }: { folder: SampleFolder; actions: FolderActions }) {
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

interface NoteListProps {
  notes: SampleNote[];
  onDelete: (note: SampleNote) => void;
}

/** A folder's notes, oldest first, each with its own delete button. */
function NoteList({ notes, onDelete }: NoteListProps) {
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

interface DeleteDialogProps {
  /** The question: "Delete sample folder 'Ideas'?" */
  title: string;
  /** What goes to the recycle bin with it. */
  text: string;
  confirmLabel: string;
  /** Deletes the record and answers its recycle-bin entry. */
  remove: () => Promise<DeleteResult>;
  onClose: () => void;
}

/**
 * Asks before a delete (design system §6.5); afterwards the message offers Undo (FR-026), or says
 * why the delete did not work.
 */
function DeleteDialog({ title, text, confirmLabel, remove, onClose }: DeleteDialogProps) {
  const [busy, setBusy] = useState(false);
  const afterDelete = useUndoableDelete(UNDO_OPTIONS);
  const showFailure = useFailureMessage();

  async function removeAndTell() {
    try {
      afterDelete(await remove());
    } catch (error) {
      showFailure(error, removeAndTell);
    }
  }

  async function confirm() {
    setBusy(true);
    await removeAndTell();
    onClose();
  }

  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
      title={title}
      confirmLabel={confirmLabel}
      tone="danger"
      busy={busy}
      onConfirm={confirm}
    >
      {text}
    </ConfirmDialog>
  );
}

/** What a folder takes with it to the recycle bin (plan "User-visible messages"). */
function whatGoesWith(folder: SampleFolder): string {
  const count = folder.notes.length;
  if (count === 0) {
    return 'It goes to the recycle bin. You can restore it.';
  }
  const notes = count === 1 ? 'its sample note' : `its ${count} sample notes`;
  return `It goes to the recycle bin together with ${notes}. You can restore it.`;
}

/** What was deleted and is not restored yet, newest first, each with Restore (FR-025). */
function RecentlyDeletedCard({ entries }: { entries: RecycleBinEntry[] }) {
  const format = useFormatters();
  const { show } = useToast();
  const showFailure = useFailureMessage();
  const restoreFromBin = useRestoreFromBin();

  async function restore(entry: RecycleBinEntry) {
    // One restore at a time: a second click while one is on its way is ignored.
    if (restoreFromBin.isPending) {
      return;
    }
    try {
      const { details } = await restoreFromBin.mutateAsync(entry.id);
      // The details say what the restore changed, such as a new name (FR-025).
      const restored = `${recordTitle(entry)} restored`;
      show({ message: details ? `${restored}. ${details}` : restored });
    } catch (error) {
      if (error instanceof ApiError && error.kind === 'alreadyResolved') {
        show({ message: resolvedMessage(error, entry.recordName, format.dateFormat) });
      } else {
        showFailure(error, () => restore(entry));
      }
    }
  }

  return (
    <Card title="Recently deleted">
      <ul role="list" className={styles.list}>
        {entries.map((entry) => (
          <li key={entry.id} className={styles.binEntry}>
            <p>{deletedText(entry, format.dateTime)}</p>
            <Button
              size="sm"
              icon={<RotateCcw />}
              aria-label={`Restore ${entry.recordTypeLabel} ${entry.recordName}`}
              onClick={() => restore(entry)}
            >
              Restore
            </Button>
          </li>
        ))}
      </ul>
    </Card>
  );
}

/**
 * "Sample folder 'Ideas' (SF-1001), with 2 sample notes — deleted by Priya Raman, 12 Mar 2026,
 * 14:32".
 */
function deletedText(entry: RecycleBinEntry, dateTime: (timestamp: string) => string): string {
  const contents = entry.contents.map((content) => `${content.count} ${content.label}`);
  const withContents = contents.length > 0 ? `, with ${AND_LIST.format(contents)}` : '';
  return (
    `${recordTitle(entry)} (${entry.recordId})${withContents}` +
    ` — deleted by ${entry.deletedBy.name}, ${dateTime(entry.deletedAt)}`
  );
}

/** The newest activity-log entries (FR-022): who did what to which record, where and when. */
function ActivityLogCard({ entries }: { entries: ActivityLogEntry[] }) {
  return (
    <Card title="Activity log">
      <ol role="list" className={styles.list}>
        {entries.map((entry) => (
          <LogEntry key={entry.id} entry={entry} />
        ))}
      </ol>
    </Card>
  );
}

/**
 * "Priya Raman changed sample folder 'Ideas 2' (SF-1001)", then where the record was ("in Ideas")
 * and when, then each changed field: "Name: Ideas → Ideas 2".
 */
function LogEntry({ entry }: { entry: ActivityLogEntry }) {
  const { dateTime } = useFormatters();
  const place = entry.path.map((record) => record.name).join(' › ');

  return (
    <li className={styles.logEntry}>
      <p>{`${entry.actor.name} ${entry.verb} ${entry.recordTypeLabel} '${entry.recordName}' (${entry.recordId})`}</p>
      <p className={styles.meta}>
        {place && `in ${place} · `}
        <time dateTime={entry.at}>{dateTime(entry.at)}</time>
      </p>
      {entry.changes.length > 0 && (
        <ul className={styles.changes}>
          {entry.changes.map((change) => (
            <li key={change.field}>{`${change.label}: ${change.oldText} → ${change.newText}`}</li>
          ))}
        </ul>
      )}
    </li>
  );
}

/**
 * "Sample folder 'Ideas'": the record type, with a capital letter as it starts the sentence
 * (data model §5), and the record's name.
 */
function recordTitle(entry: RecycleBinEntry): string {
  const recordType = entry.recordTypeLabel;
  return `${recordType.charAt(0).toUpperCase()}${recordType.slice(1)} '${entry.recordName}'`;
}

/**
 * Shows why an action failed, in plain words (FR-020): with Try again when the server could not
 * be reached or failed, without it when the API refused for good ("Restore 'Ideas' first"). A
 * session that has ended is left to the sign-in page, which says so.
 */
function useFailureMessage() {
  const { show } = useToast();

  return function showFailure(error: unknown, tryAgain: () => void) {
    const failure = error instanceof ApiError ? error : new ApiError('other');
    if (failure.kind === 'signInRequired' || failure.kind === 'sessionExpired') {
      return;
    }
    const mayWorkLater = failure.kind === 'unreachable' || failure.kind === 'other';
    show({
      message: failure.message,
      action: mayWorkLater
        ? { label: 'Try again', altText: 'Try again', onClick: tryAgain }
        : undefined,
    });
  };
}
