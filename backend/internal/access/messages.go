package access

// The texts the access API says (plan "User-visible messages"); "%s" and "%d" are filled in at run time.

const (
	msgOnlyAdministrator = "%s is the only administrator. Give someone else the Administrator role first."
	msgAdministratorLock = "The Administrator role always has every permission and can't be changed."
	msgRoleInUseOne      = "'%s' is given to 1 person. Take it away from them first."
	msgRoleInUseMany     = "'%s' is given to %d people. Take it away from them first."
	msgRoleGone          = "This role no longer exists — someone may have deleted it."
	msgPersonGone        = "This person doesn't exist."
	msgRestoredName      = "Renamed to '%s' because the name was taken."
	msgUndoNameTaken     = "Can't undo — there is now another role called '%s'. Restore it from the recycle bin instead."
)

const (
	msgAdministratorAllMarkets = "The Administrator role always applies to all markets."
	msgRoleNameTaken           = "A role called '%s' already exists."
	msgChoosePermission        = "Choose at least one permission."
	msgChooseTeam              = "Choose at least one team."
	msgNewPersonNotAdmin       = "New people can't be made administrators automatically."
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
	msgLinkNeedsImport         = "Linking financial transactions needs Import GSAP extracts too — only people who import see transactions that are not linked."
)

const (
	msgApproverCannotApprove    = "%s is an approver in %s and %s. Without a role that approves, an administrator will have to decide on their behalf."
	msgRemovesOwnAdministration = "You will no longer be able to change Settings, people or access."
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
