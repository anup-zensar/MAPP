/** What can go wrong when the web app talks to the API, in the words people see (FR-020, research R9). */
import { formatDate, formatTime, localDate } from '@/format/intl';
import type { DateFormat } from '@/format/intl';
import type { components } from './generated/schema';

type ErrorBody = components['schemas']['Error'];
type ErrorDetail = components['schemas']['ErrorDetail'];
type FieldError = components['schemas']['FieldError'];
type Conflict = components['schemas']['Conflict'];
type RecordRef = components['schemas']['RecordRef'];
type Resolution = components['schemas']['Resolution'];
type ReadinessGroup = components['schemas']['ReadinessGroup'];

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
  | 'overBudget'
  | 'tooManyAttempts'
  | 'tooLarge'
  | 'unreachable'
  | 'other';

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
  over_budget: 'overBudget',
  too_many_attempts: 'tooManyAttempts',
  too_large: 'tooLarge',
  internal_error: 'other',
};

const UNREACHABLE =
  "SP&A One can't reach its server right now. Check that it is running and try again.";
const SOMETHING_WENT_WRONG = 'Something went wrong. Please try again.';

interface ApiErrorDetails {
  message?: string;
  status?: number;
  fields?: FieldError[];
  conflict?: Conflict;
  parent?: RecordRef;
  resolution?: Resolution;
  missing?: ReadinessGroup[];
}

export class ApiError extends Error {
  readonly kind: ApiErrorKind;
  readonly status: number;
  readonly fields: FieldError[];
  readonly conflict?: Conflict;
  readonly parent?: RecordRef;
  readonly resolution?: Resolution;
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

  static fromBody(status: number, body: ErrorBody): ApiError {
    const { error } = body;
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

export function conflictMessage(error: ApiError, noun: string, dateFormat?: DateFormat): string {
  if (!error.conflict) {
    return error.message;
  }
  const { changedBy, changedAt } = error.conflict;
  return `Not saved — ${changedBy} changed this ${noun} ${when(changedAt, dateFormat)}`;
}

export function resolvedMessage(error: ApiError, name: string, dateFormat?: DateFormat): string {
  if (!error.resolution) {
    return error.message;
  }
  const { how, by, at } = error.resolution;
  const what =
    how === 'undone'
      ? 'undid the delete'
      : how === 'deletedPermanently'
        ? 'deleted it permanently'
        : 'restored it';
  return `'${name}' is no longer in the recycle bin — ${by.name} ${what} ${when(at, dateFormat)}`;
}

export function asApiError(error: unknown): ApiError {
  return error instanceof ApiError ? error : new ApiError('other');
}

export function isSessionLost(error: ApiError): boolean {
  return error.kind === 'signInRequired' || error.kind === 'sessionExpired';
}

function when(timestamp: string, dateFormat?: DateFormat): string {
  const time = `at ${formatTime(timestamp)}`;
  const day = localDate(timestamp);
  return day === localDate(new Date()) ? time : `on ${formatDate(day, dateFormat)} ${time}`;
}
