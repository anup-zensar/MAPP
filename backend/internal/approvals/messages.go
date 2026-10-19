package approvals

// Messages of the plan's "User-visible messages" table that the API sends. Code and tests quote them.
const (
	// Records that are gone.
	activityGoneMessage = "This activity no longer exists — someone may have deleted it."
	workflowGoneMessage = "This approval workflow no longer exists — someone may have deleted it."
	roundNotOfActivity  = "This approval round doesn't belong to this activity. Reload the page and try again."
	reloadMessage       = "Reload the page and try again."
	chooseTabMessage    = "Choose waiting, open or decided."

	// Submitting (FR-009, FR-010).
	onlyCreatedOrDeclinedMessage = "Only Created or Declined activities can be submitted for approval."
	workflowNeededMessage        = "Approval workflow — needed to submit"
	workflowInactiveMessage      = "Approval workflow '%s' is inactive — choose an active one."

	// Recall of an approved activity (FR-021).
	approvedNoRecallMessage = "Approved activities can't be recalled. " +
		"Changing its planned cost, dates or line-item amounts sends it back for approval."

	// Comments.
	declineCommentMessage = "Add a comment explaining why the activity is declined."
	tooLong2000Message    = "Use 2,000 characters or fewer."

	// A decision on a level or round that has moved on (FR-018). The web app adds the time.
	alreadyApprovedMessage = "Level %d was already approved by %s"
	alreadyDeclinedMessage = "Level %d was already declined by %s"
	recalledByMessage      = "The submission was recalled by %s"
	restartedByMessage     = "Approval was restarted by %s"
	cancelledMessage       = "The approval round was cancelled because the activity was deleted"

	// Approval workflows (FR-002, FR-003, FR-008, data model §1).
	enterNameMessage         = "Enter a name."
	nameTooLongMessage       = "Use 120 characters or fewer."
	nameTakenMessage         = "There is already an approval workflow called '%s'. Choose another name."
	addLevelMessage          = "Add at least one level."
	levelNameTooLongMessage  = "Level %d: use 60 characters or fewer."
	chooseApproverMessage    = "Level %d: choose at least one approver."
	unknownApproverMessage   = "Level %d: choose approvers from the list."
	listedTwiceMessage       = "Level %d: %s is listed twice."
	inactiveApproverMessage  = "Level %d: %s is inactive — choose an active person."
	noActiveApproverWarning  = "Level %d has no active approver"
	inUseOneMessage          = "Used by 1 activity — set it inactive instead."
	inUseManyMessage         = "Used by %d activities — set it inactive instead."
	undoNameTakenMessage     = "Can't undo — there is now another approval workflow called '%s'. Restore it from the recycle bin instead."
	renamedOnRestoreMessage  = "Renamed to '%s' because the name was taken."
	activeText, inactiveText = "Active", "Inactive"

	// Notifications: titles and texts, fixed when they are created (data model §4).
	approvalNeededTitle   = "Approval needed"
	levelApprovedTitle    = "Level %d approved"
	activityApprovedTitle = "Activity approved"
	activityDeclinedTitle = "Activity declined"
	submittedText         = "%s submitted %s"
	resubmittedText       = "%s re-submitted %s"
	nextLevelWaitsText    = "%s approved level %d of %s — level %d waits for you"
	levelApprovedText     = "%s approved level %d of %d of %s"
	approvedText          = "%s approved %s"
	declinedText          = "%s declined %s: \"%s\""
)
