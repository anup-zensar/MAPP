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

	// The note of the "restored" log entry when a restore ends the round that was open at the delete
	// (FR-023, 008-P3).
	cancelledNote = "Approval round %d was cancelled because the activity was deleted — submit it for approval again."

	// A notification that is not the signed-in person's, or none (FR-034).
	notificationGoneMessage = "This notification no longer exists."

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
	recalledTitle         = "Submission recalled"
	restartedTitle        = "Approval restarted"
	submittedText         = "%s submitted %s"
	resubmittedText       = "%s re-submitted %s"
	nextLevelWaitsText    = "%s approved level %d of %s — level %d waits for you"
	levelApprovedText     = "%s approved level %d of %d of %s"
	approvedText          = "%s approved %s"
	declinedText          = "%s declined %s: \"%s\""
	recalledText          = "%s recalled %s"
	restartedText         = "%s changed %s — it needs approval again"

	// Feature 017: approval rules (data-model §1–§3, plan "User-visible messages").
	chooseSourceMessage          = "Level %d: choose who approves."
	chooseRoleMessage            = "Level %d: choose a role."
	roleFromListMessage          = "Level %d: choose a role from the list."
	roleCannotApproveMessage     = "Level %d: people with the role '%s' can't approve activities. Choose a role that approves."
	chooseModeMessage            = "Level %d: choose 'Any one approves' or 'All must approve'."
	emptyAppliesWhenMessage      = "Level %d: 'Only needed when' needs at least one condition."
	emptyAutoApproveMessage      = "Level %d: 'Approve automatically when' needs at least one condition."
	amountsMessage               = "Level %d: enter amounts of 0.00 or more with at most two decimals."
	amountOrderMessage           = "Level %d: the 'at least' amount must be lower than the 'below' amount."
	businessUnitsMessage         = "Level %d: choose business units from the list."
	activityTypesMessage         = "Level %d: choose activity types from the list."
	backupRequiredMessage        = "Level %d: choose a backup approver, or turn escalation off."
	backupFromListMessage        = "Level %d: choose the backup approver from the list."
	backupInactiveMessage        = "Level %d: %s is inactive — choose an active person."
	backupCannotApproveMessage   = "Level %d: %s can't approve activities. Give them a role that approves first."
	backupIsApproverMessage      = "Level %d: %s is already an approver of this level — choose someone else."
	approverCannotApproveMessage = "%s can't approve activities. Give them a role that approves first."
	reminderDaysMessage          = "Remind after: enter a whole number of days from 1 to 60."
	escalationDaysMessage        = "Escalate after: enter a whole number of days from 1 to 90."
	escalationOrderMessage       = "Escalate after must be more days than Remind after."
	backupInactiveWarning        = "Level %d: the backup approver %s is inactive"
	backupCannotApproveWarning   = "Level %d: the backup approver %s can't approve activities"
	roleGoneWarning              = "Level %d: role '%s' no longer exists"
	remindAfterLabel             = "Remind after"
	escalateAfterLabel           = "Escalate after"
	offText                      = "Off"

	// Resolution problems and the no-approver advice (data-model §3).
	noLineManagerProblem   = "Level %d: %s has no line manager. Ask an administrator to set one in Settings › People."
	roleGoneProblem        = "Level %d: the role of this level no longer exists. Ask an administrator to change the approval workflow."
	nobodyHoldsRoleProblem = "Level %d: nobody holds the role '%s' for this plan."
	onlyApproverAdvice     = "You are the only approver of level %d. An administrator will have to approve on your behalf."
	nobodyElseAdvice       = "Nobody else can approve level %d. Someone allowed to decide on behalf, such as an administrator, will have to approve it."

	// Who may decide (data-model §6; 014 messages 5, 6a and 6b).
	onlyApproversMessage        = "Only the approvers of level %d (%s) can approve or decline this activity."
	ownActivityMessage          = "You own this activity, so someone else must approve it."
	submittedActivityMessage    = "You submitted this activity, so someone else must approve it."
	alreadyApprovedLevelMessage = "You already approved level %d. Waiting for %s."
	nobodyText                  = "nobody"
	noLevelAppliesNote          = "no level applies"

	// Approval delegations (data-model §7).
	choosePersonMessage          = "Choose a person from the list."
	personInactiveMessage        = "%s is inactive — choose an active person."
	chooseDelegateMessage        = "Choose who approves for you."
	delegateCannotApproveMessage = "%s can't approve activities. Choose someone with a role that approves."
	delegateSelfMessage          = "You can't delegate to yourself."
	delegateThemselvesMessage    = "%s can't delegate to themselves."
	enterFirstDayMessage         = "Enter the first day."
	firstDayTodayMessage         = "The first day must be today or later."
	startedMessage               = "The delegation has started, so its first day can't change."
	enterLastDayMessage          = "Enter the last day."
	lastAfterFirstMessage        = "The last day must be on or after the first day."
	lastDayTodayMessage          = "The last day must be today or later."
	maxDaysMessage               = "A delegation can last at most 90 days."
	noteTooLongMessage           = "Use 500 characters or fewer."
	overlapMessage               = "You already delegate to %s on some of these days. Change that delegation instead."
	overlapOtherMessage          = "%s already delegates to %s on some of these days. Change that delegation instead."
	changeDelegationMessage      = "Only %s or someone who can change approval workflows can change this delegation."
	delegationEndedMessage       = "Ended and cancelled delegations can't be changed. They are kept for the audit trail."
	delegationGoneMessage        = "This delegation no longer exists."
	chooseScopeMessage           = "Choose mine or all."
	cancelledStatusText          = "Cancelled"

	// Notifications of feature 017 (plan "User-visible messages").
	delegateSuffix      = " — you approve as delegate of %s"
	autoApprovedTitle   = "Activity approved automatically"
	autoApprovedText    = "%s was approved automatically: %s"
	reminderTitle       = "Reminder — approval waiting %s"
	reminderText        = "%s has waited %s for you at level %d"
	reminderAsDelegate  = " — as delegate of %s"
	escalatedToYouTitle = "Approval escalated to you"
	escalatedToYouText  = "%s waited %s at level %d. You can now approve it as backup approver."
	escalatedTitle      = "Approval escalated"
	escalatedText       = "%s waited %s at level %d and was escalated to %s."
)
