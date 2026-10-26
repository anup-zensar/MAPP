/**
 * What can go wrong when the web app talks to the API, in the words people see (FR-020).
 * Technical details never reach the screen: every failure becomes an ApiError with a kind (for the
 * code to choose what to show) and a plain-language message (research R9, R15).
 */
import { formatDate, formatTime, localDate } from '@/lib/format';
import type { DateFormat } from '@/lib/format';
import type { components } from './generated/schema';

type ErrorBody = components['schemas']['Error'];
type ErrorDetail = components['schemas']['ErrorDetail'];
type FieldError = components['schemas']['FieldError'];
type Conflict = components['schemas']['Conflict'];
type RecordRef = components['schemas']['RecordRef'];
type Resolution = components['schemas']['Resolution'];
type ReadinessGroup = components['schemas']['ReadinessGroup'];

/** What went wrong, as the web app needs to know it. */
export type ApiErrorKind =
  | 'validation'
  | 'badRequest'
  | 'conflict'
  | 'notFound'
  | 'signInRequired'
  | 'sessionExpired'
  | 'incorrectCredentials'
  | 'restoreParentFirst'
  | 'undoExpired'
  | 'alreadyResolved'
  | 'nameTaken'
  | 'inUse'
  | 'cannotRestore'
  | 'notAllowed'
  | 'notReady'
  | 'parentDeletedPermanently'
  | 'forbidden'
  | 'unreachable'
  | 'other';

/** The kind of each error code the API sends (research R9). */
const KIND_BY_CODE: Record<ErrorDetail['code'], ApiErrorKind> = {
  validation_failed: 'validation',
  bad_request: 'badRequest',
  sign_in_required: 'signInRequired',
  session_expired: 'sessionExpired',
  incorrect_credentials: 'incorrectCredentials',
  not_found: 'notFound',
  version_conflict: 'conflict',
  restore_parent_first: 'restoreParentFirst',
  undo_expired: 'undoExpired',
  already_resolved: 'alreadyResolved',
  name_taken: 'nameTaken',
  in_use: 'inUse',
  cannot_restore: 'cannotRestore',
  not_allowed: 'notAllowed',
  not_ready: 'notReady',
  parent_deleted_permanently: 'parentDeletedPermanently',
  forbidden: 'forbidden',
  internal_error: 'other',
};

const UNREACHABLE =
  "MAPP can't reach its server right now. Check that it is running and try again.";
const SOMETHING_WENT_WRONG = 'Something went wrong. Please try again.';

/** Everything an ApiError carries besides its kind; all optional. */
interface ApiErrorDetails {
  /** The message for people; a general one for the kind when left out. */
  message?: string;
  status?: number;
  fields?: FieldError[];
  conflict?: Conflict;
  parent?: RecordRef;
  resolution?: Resolution;
  missing?: ReadinessGroup[];
}

/** A failed API call, ready to be shown to people. */
export class ApiError extends Error {
  /** What went wrong, so the code can choose what to show. */
  readonly kind: ApiErrorKind;
  /** The HTTP status of the API's answer; 0 when no answer arrived. */
  readonly status: number;
  /** Every field problem at once, for the fields and the error summary (validation). */
  readonly fields: FieldError[];
  /** Who changed the record and when (conflict); show it with conflictMessage(). */
  readonly conflict?: Conflict;
  /** The record to restore first (restoreParentFirst). */
  readonly parent?: RecordRef;
  /** How, by whom and when it left the recycle bin (alreadyResolved); see resolvedMessage(). */
  readonly resolution?: Resolution;
  /**
   * Every needed field that stops publishing or submitting, grouped by record (notReady, feature
   * 004); empty otherwise.
   */
  readonly missing: ReadinessGroup[];

  constructor(kind: ApiErrorKind, details: ApiErrorDetails = {}) {
    super(details.message || (kind === 'unreachable' ? UNREACHABLE : SOMETHING_WENT_WRONG));
    this.name = 'ApiError';
    this.kind = kind;
    this.status = details.status ?? 0;
    this.fields = details.fields ?? [];
    this.conflict = details.conflict;
    this.parent = details.parent;
    this.resolution = details.resolution;
    this.missing = details.missing ?? [];
  }

  /** The error for an error answer of the API: the code gives the kind, the message is kept. */
  static fromBody(status: number, body: ErrorBody): ApiError {
    const { error } = body;
    // A code this version of the web app does not know yet is shown like any other failure.
    return new ApiError(KIND_BY_CODE[error.code] ?? 'other', {
      status,
      message: error.message,
      fields: error.fields,
      conflict: error.conflict,
      parent: error.parent,
      resolution: error.resolution,
      missing: error.missing,
    });
  }
}

/**
 * "Not saved — Daniel Okoro changed this sample folder at 14:32" (FR-029), in the user's own time,
 * with "on 12 Mar 2026" (in the person's date format, feature 003) when the change was not today.
 * The API's message has no time in it, because only the browser knows the user's time zone
 * (research R5).
 */
export function conflictMessage(error: ApiError, noun: string, dateFormat?: DateFormat): string {
  if (!error.conflict) {
    return error.message;
  }
  const { changedBy, changedAt } = error.conflict;
  return `Not saved — ${changedBy} changed this ${noun} ${when(changedAt, dateFormat)}`;
}

/**
 * "'Ideas' is no longer in the recycle bin — Priya Raman restored it at 14:05", or "… undid the
 * delete at 14:05" (research R8), in the user's own time, with the date when it was not today.
 */
export function resolvedMessage(error: ApiError, name: string, dateFormat?: DateFormat): string {
  if (!error.resolution) {
    return error.message;
  }
  const { how, by, at } = error.resolution;
  const what =
    how === 'undone'
      ? 'undid the delete'
      : how === 'deletedPermanently'
        ? 'deleted it permanently' // feature 011 (FR-032)
        : 'restored it';
  return `'${name}' is no longer in the recycle bin — ${by.name} ${what} ${when(at, dateFormat)}`;
}

/** "at 14:32" today, otherwise "on 12 Mar 2026 at 14:32", in the user's own time zone. */
function when(timestamp: string, dateFormat?: DateFormat): string {
  const time = `at ${formatTime(timestamp)}`;
  const day = localDate(timestamp);
  return day === localDate(new Date()) ? time : `on ${formatDate(day, dateFormat)} ${time}`;
}
