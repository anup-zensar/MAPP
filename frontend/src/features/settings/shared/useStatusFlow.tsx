import { useRef, useState } from 'react';
import type { ReactElement } from 'react';
import { ConfirmDialog, useToast } from '@/components';

/** The record whose status changes, with the version people saw when they chose the action. */
export interface StatusFlowRecord {
  id: string;
  name: string;
  version: number;
}

/** How the record is named in the question and the messages. */
export interface StatusWords {
  /** At the start of a message: "Model 'Lubricants 1.0' set inactive". */
  title: string;
  /** In the middle of a sentence: "Set model 'Lubricants 1.0' inactive?" */
  noun: string;
  /** People are deactivated ("Deactivate Priya Raman?"); every other record is set inactive. */
  person?: boolean;
}

export interface StatusFlowOptions<T extends StatusFlowRecord> {
  words: StatusWords;
  /**
   * Where the record is used, one sentence each ("Used by 2 line items."); empty when nothing uses
   * it. May load first (a person's details).
   */
  usage: (record: T) => string[] | Promise<string[]>;
  /** What setting it inactive does, said after where it is used. */
  effect: (record: T) => string[];
  /** Sends the change from `version`; answers the record's version after it, for Undo. */
  request: (record: T, active: boolean, version: number) => Promise<{ version: number }>;
  /** Says why a change failed (a conflict names who changed the record first). */
  onFailure: (error: unknown) => void;
}

export interface StatusFlow<T extends StatusFlowRecord> {
  /** Set inactive (or Deactivate): asks first when the record is in use, then offers Undo. */
  setInactive: (record: T) => void;
  /** Set inactive at once, because another question already confirmed it; offers Undo. */
  applyInactive: (record: T) => Promise<boolean>;
  /** Reactivate: needs no question. */
  reactivate: (record: T) => void;
  /** A change is on its way: its button shows it, and another click is ignored. */
  busy: boolean;
  /** The question; render it once on the page. */
  dialog: ReactElement | null;
}

/** The question that waits for its answer. */
interface Pending<T> {
  record: T;
  sentences: string[];
}

/**
 * The one "set inactive" behaviour of Settings, People and Teams (design system §6.7, functional
 * review S-01): a record in use asks first and lists where it is used; one that nothing uses is
 * set inactive at once. Either way the message offers **Undo**, which reactivates it. Reactivating
 * needs no question.
 */
export function useStatusFlow<T extends StatusFlowRecord>({
  words,
  usage,
  effect,
  request,
  onFailure,
}: StatusFlowOptions<T>): StatusFlow<T> {
  const { show } = useToast();
  const [pending, setPending] = useState<Pending<T> | null>(null);
  const [busy, setBusy] = useState(false);
  // A ref as well as the state: two clicks in the same moment see the first one at once.
  const running = useRef(false);
  const inactiveWord = words.person ? 'deactivated' : 'set inactive';

  async function run<R>(work: () => Promise<R>): Promise<R | undefined> {
    if (running.current) {
      return undefined;
    }
    running.current = true;
    setBusy(true);
    try {
      return await work();
    } finally {
      running.current = false;
      setBusy(false);
    }
  }

  async function reactivateFrom(record: T, version: number) {
    const done = await run(async () => {
      try {
        await request(record, true, version);
        show({ message: `${words.title} '${record.name}' reactivated` });
        return true;
      } catch (error) {
        onFailure(error);
        return false;
      }
    });
    return done ?? false;
  }

  async function applyInactive(record: T): Promise<boolean> {
    const done = await run(async () => {
      try {
        const saved = await request(record, false, record.version);
        show({
          message: `${words.title} '${record.name}' ${inactiveWord}`,
          action: {
            label: 'Undo',
            altText: `Reactivate ${record.name} from its details`,
            onClick: () => void reactivateFrom(record, saved.version),
          },
        });
        return true;
      } catch (error) {
        onFailure(error);
        return false;
      }
    });
    return done ?? false;
  }

  async function setInactive(record: T) {
    if (running.current) {
      return;
    }
    let sentences: string[];
    try {
      sentences = await usage(record);
    } catch (error) {
      onFailure(error);
      return;
    }
    if (sentences.length === 0) {
      await applyInactive(record);
      return;
    }
    setPending({ record, sentences: [...sentences, ...effect(record)] });
  }

  const dialog = pending && (
    <ConfirmDialog
      open
      onOpenChange={(open) => !open && setPending(null)}
      title={
        words.person
          ? `Deactivate ${pending.record.name}?`
          : `Set ${words.noun} '${pending.record.name}' inactive?`
      }
      confirmLabel={words.person ? 'Deactivate person' : 'Set inactive'}
      busy={busy}
      // Closed whatever the answer: a failure says why in its own message.
      onConfirm={() => void applyInactive(pending.record).then(() => setPending(null))}
    >
      {pending.sentences.map((sentence) => (
        <p key={sentence}>{sentence}</p>
      ))}
    </ConfirmDialog>
  );

  return {
    setInactive: (record) => void setInactive(record),
    applyInactive,
    reactivate: (record) => void reactivateFrom(record, record.version),
    busy,
    dialog,
  };
}
