package people

// Every text the people API says, from the plan's "User-visible messages" table (003 plan). Code and tests
// use these constants, so each text is written once. "%s" and "%d" are filled in at run time.

// Field problems (003 data model §1–§2, FR-002, FR-011). Every problem of a request is reported at once.
const (
	enterNameMessage          = "Enter a name."
	use120Message             = "Use 120 characters or fewer." // a person's name and job title
	use80Message              = "Use 80 characters or fewer."  // a team's name
	use300Message             = "Use 300 characters or fewer." // a team's description
	use254Message             = "Use 254 characters or fewer." // an e-mail address
	enterEmailMessage         = "Enter an e-mail address."
	invalidEmailMessage       = "Enter a valid e-mail address."
	emailTakenMessage         = "%s already belongs to %s."
	emailTakenInactiveMessage = "%s already belongs to %s (inactive) — reactivate them instead."
	teamNameTakenMessage      = "A team called '%s' already exists."
	chooseColourMessage       = "Choose one of the colours."
	inactiveTeamMessage       = "%s is inactive, so people can't be added to it. Reactivate the team first."
	unknownTeamMessage        = "Choose teams from the list."
	inactivePersonMessage     = "%s is inactive and can't be added. Reactivate them first."
	unknownPersonMessage      = "Choose people from the list."
	chooseStatusMessage       = "Choose active or inactive."
	reloadMessage             = "Reload the page and try again."
	chooseHomeScopeMessage    = "Choose My teams or Everything." // Home's switch (009 FR-003)
)

// Preference problems (003 data model §4, FR-023, FR-024).
const (
	chooseCurrencyMessage   = "Choose one of the active currencies."
	chooseDateFormatMessage = "Choose one of the date formats."
)

// Refusals (not_found, not_allowed, in_use, name_taken) and the note of a renaming restore.
const (
	personGoneMessage         = "This person doesn't exist."
	teamGoneMessage           = "This team no longer exists — someone may have deleted it."
	deactivateYourselfMessage = "You can't deactivate yourself — ask someone else to do it."
	teamInUseOneMessage       = "%s is used by 1 plan, so it can't be deleted. Set it inactive instead."
	teamInUseManyMessage      = "%s is used by %d plans, so it can't be deleted. Set it inactive instead."
	undoNameTakenMessage      = "Can't undo — there is now another team called '%s'. Restore it from the recycle bin instead."
	renamedNote               = "Renamed to '%s' because the name was taken."
)

// The owner warning of the two impact checks (FR-013): "<person> owns <n> plan(s) that <uses what>: <plans>.
// <first name> will no longer belong to …". The middle part says which teams the plans use.
const (
	ownerWarningOneMessage  = "%s owns 1 plan that %s: %s. %s will no longer belong to its teams, so its owner will need attention."
	ownerWarningManyMessage = "%s owns %d plans that %s: %s. %s will no longer belong to their teams, so their owner will need attention."
	usesThisTeam            = "uses this team"         // a team's impact check, one plan
	useThisTeam             = "use this team"          // a team's impact check, several plans
	usesTeamNamed           = "uses %s"                // a person's impact check with one removed team, one plan
	useTeamNamed            = "use %s"                 // … several plans
	usesTheRemovedTeams     = "uses the removed teams" // a person's impact check with several removed teams, one plan
	useTheRemovedTeams      = "use the removed teams"  // … several plans
)

// Texts of the activity log (003 data model §6).
const (
	verbDeactivated = "deactivated"
	verbReactivated = "reactivated"
	emailLabel      = "E-mail"
	jobTitleLabel   = "Job title"
	teamsLabel      = "Teams"
	rolesLabel      = "Roles" // a person's roles and scopes (014)
	membersLabel    = "Members"
	colourLabel     = "Colour"
	statusLabel     = "Status"
	activeText      = "Active"
	inactiveText    = "Inactive"
	noTeamText      = "No team"
	noMembersText   = "No members"
)
