# Success Criteria Review: Activity Log and Recycle Bin

**Purpose**: Record the check of the success criteria that no task names (tasks.md TC99): SC-003, SC-004
and SC-007.
**Checked**: 2026-10-20, branch lane-011b, at the final review (T049)
**Feature**: [spec.md](../spec.md)

None of the three criteria is timed, so each was checked against the automated tests that prove it. These
were run at the review: `go test ./...` in `backend/`, `npm run check` in `frontend/` and every
end-to-end file, twice, against freshly seeded demo data. Each criterion lists its tests.

## SC-003: restores bring back every record of the entry with the same IDs and values

- [x] Every record of the entry comes back with the same ID and values. The only exceptions are the
  status and name changes the confirmation states: `TestRestore_SameIDsAndValues_FR021`,
  `TestRestoreWithParents_ParentsFirstOneStep_FR023` and `TestSeed_DeletedInitiative_Restores_011`
  (backend/internal/history, backend/seed).
- [x] Undo stays an exact restore: `TestUndo_StillExact_FR027`.
- [x] In the browser: the seeded initiative "Trade shows" returns with its activity under the same ID.
  An approved activity comes back as Created, as its dialog said it would. An activity and its initiative
  come back together with **Restore both** (`e2e/recycle-bin.spec.ts`). An activity deleted from its page
  returns to the same page (`e2e/activity-log-and-recycle-bin.spec.ts`).

## SC-004: restore confirmations name the destination, the contents and every status change

- [x] The preview gives the destination, the contents and each status change, and it changes nothing:
  `TestPreviewRestore_StatusChanges_FR018_FR019`, `TestPreviewRestore_NameClash_FR022`,
  `TestPreviewRestore_DestinationIsCurrentPlace_EdgeCase` and `TestPreviewRestore_ChangesNothing_FR018`.
- [x] The dialog shows all three before the person confirms: `RestoreDialog.test.tsx` ("says where it
  returns, what changes and the notes, then restores"). In the browser, the dialog shows "It returns to
  … with 1 activity and 1 line item." and "… must be submitted for approval again" before **Restore**
  (`e2e/recycle-bin.spec.ts`, `e2e/activity-log-and-recycle-bin.spec.ts`).

## SC-007: nothing is permanently deleted without a confirmation; restores, permanent deletes and bin empties are logged with the person

- [x] **Delete permanently** and **Empty recycle bin** each open a confirmation with **Cancel**
  focused, and they delete only after it: `PurgeDialog.test.tsx` and `EmptyBinDialog.test.tsx`. In the
  browser they were checked with axe (`e2e/recycle-bin.spec.ts`, `e2e/activity-log-and-recycle-bin.spec.ts`).
- [x] Each restore, permanent delete and bin empty writes a log entry with the person who did it:
  `TestRestore_LogsActorAndStatusChanges_FR019_FR026`, `TestPurge_RemovesForGoodAndLogs_FR030`,
  `TestEmptyBin_OneEntryWithItems_FR029_FR030` and `TestLog_EntriesNeverChangeOrDisappear_FR010_FR030`.
  In the browser, the log shows "Mei Lin restored …", "Mei Lin permanently deleted …" and "… emptied the
  recycle bin" with their date and time (`e2e/activity-log-and-recycle-bin.spec.ts`, `e2e/recycle-bin.spec.ts`).

## Notes

- A moderated session with Shell users can still confirm that people understand these steps. Nothing
  above depends on one.
