package access

// Every text the access API says, from plan "User-visible messages" (numbers in brackets) and the field rules
// of the data model. "%s" and "%d" are filled in at run time.

// Refusals (409 not_allowed, in_use, not_found).
const (
	msgOnlyAdministrator = "%s is the only administrator. Give someone else the Administrator role first." // 10
	msgAdministratorLock = "The Administrator role always has every permission and can't be changed."      // 11
	msgRoleInUseOne      = "'%s' is given to 1 person. Take it away from them first."                      // 13
	msgRoleInUseMany     = "'%s' is given to %d people. Take it away from them first."                     // 13
	msgRoleGone          = "This role no longer exists — someone may have deleted it."
	msgPersonGone        = "This person doesn't exist."
	msgRestoredName      = "Renamed to '%s' because the name was taken."
	msgUndoNameTaken     = "Can't undo — there is now another role called '%s'. Restore it from the recycle bin instead."
)

// Field problems (data model §1.1–§1.3).
const (
	msgAdministratorAllMarkets = "The Administrator role always applies to all markets."  // 12
	msgRoleNameTaken           = "A role called '%s' already exists."                     // 14
	msgChoosePermission        = "Choose at least one permission."                        // 15
	msgChooseTeam              = "Choose at least one team."                              // 16
	msgNewPersonNotAdmin       = "New people can't be made administrators automatically." // 17
	msgEnterRoleName           = "Enter a name."
	msgUse60                   = "Use 60 characters or fewer."
	msgUse300                  = "Use 300 characters or fewer."
	msgUnknownPermission       = "'%s' is not a permission."
	msgChooseDefaultScope      = "Choose All markets or My teams."
	msgChooseScope             = "Choose All markets, My teams or Chosen teams."
	msgChooseRole              = "Choose a role from the list."
	msgRoleTwice               = "'%s' is listed twice."
	msgChooseTeamsFromList     = "Choose teams from the list."
	msgReload                  = "Reload the page and try again."
)

// Warnings of the roles impact check (FR-022).
const (
	msgApproverCannotApprove    = "%s is an approver in %s and %s. Without a role that approves, an administrator will have to decide on their behalf." // 18
	msgRemovesOwnAdministration = "You will no longer be able to change Settings, people or access."                                                    // 19
)

// Texts of the activity log (data model §3).
const (
	rulesRecordName     = "Access rules"
	labelNoSelfApproval = "Nobody approves an activity they own or submitted"
	labelNewPersonRole  = "Role for new people"
	labelPermissions    = "Permissions"
	labelDefaultScope   = "Default scope"
	onText              = "On"
	offText             = "Off"
)
