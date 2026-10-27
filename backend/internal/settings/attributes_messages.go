package settings

// The limits of a Shell attribute and its options (data model §4, §5).
const (
	maxAttributeLabel = 80
	maxHelpText       = 300
	maxSectionName    = 60
	maxOptionLabel    = 120
	maxOptions        = 5000
	maxListedLabels   = 5 // option labels an impact warning names
)

// Messages of the plan's "User-visible messages" table about creating and changing Shell attributes, and
// about what a change affects.
const (
	enterLabelMessage        = "Enter a label."
	attributeLabelTaken      = "The %s form already has a Shell attribute called '%s'."
	chooseLevelMessage       = "Choose a level."
	chooseTypeMessage        = "Choose a type."
	chooseMarkingMessage     = "Choose a marking."
	publishMarkingMessage    = "At %s level the marking is Optional or Needed to publish."
	submitMarkingMessage     = "At %s level the marking is Optional or Needed to submit."
	yesNoMarkingMessage      = "Yes/no attributes are always optional — for a needed yes-or-no answer, use a single choice with the options Yes and No."
	splitLevelMessage        = "Multiple choice with % split is only for activities and line items, which carry cost."
	addOptionMessage         = "Add at least one option."
	appearsTwiceMessage      = "'%s' appears twice."
	optionLabelMessage       = "Enter a label for every option."
	tooManyOptionsMessage    = "An attribute can have at most 5,000 options."
	onlyChoiceOptionsMessage = "Only choice attributes have options."
	onlyChoiceDependsMessage = "Only choice attributes can depend on another attribute."
	chooseParentMessage      = "Choose a single or multiple choice attribute that comes before this one on the %s form."
	linkOptionMessage        = "Link '%s' to at least one %s option."
	linkParentOnlyMessage    = "'%s' can only be linked to options of %s."
	inactiveLinkMessage      = "'%s' is inactive — link active %s options only."
	conditionOptionsMessage  = "Choose business units for the condition from the Business unit options."
	noBusinessUnitMessage    = "There is no Business unit attribute yet, so conditions can't be set."
	stayBelowMessage         = "%s must stay below %s."
	withOptionsDetails       = "with %s"
	removedOptionsDetails    = "Removed its %s: %s."
	incompleteMessage        = "%s %s no value yet and will show as incomplete. %s"
	activitiesKeepStatus     = "Submitted and approved activities keep their status."
	plansKeepStatus          = "Published plans keep their status."
	optionsRemovedMessage    = "Its %d options will be removed: %s."
	optionRemovedMessage     = "Its option '%s' will be removed."
)

// Messages of the plan's "User-visible messages" table about Shell attributes: not found and locks.
const (
	attributeGoneMessage = "This Shell attribute no longer exists — someone may have deleted it."
	typeLockedMessage    = "Type can't change once records use this attribute — create a new attribute instead."
	levelLockedMessage   = "Level can't change once records use this attribute — create a new attribute instead."
	parentTypeLock       = "%s depends on it, so its type can't change."
	levelDependencyLock  = "Level can't change while it depends on, or is depended on by, another attribute."
	businessUnitLock     = "Business unit can't be deleted, set inactive, moved or given another type, because other " +
		"attributes are shown or hidden by it."
	dependentActiveLock  = "Set '%s' inactive first — it depends on '%s'."
	reactivateParentLock = "Reactivate '%s' first — '%s' depends on it."
)

// businessUnitLocks are the fields and actions the Business unit attribute never allows, because the
// conditions of other attributes depend on it (FR-023, D-055).
var businessUnitLocks = []string{"type", "level", "section", "position", "condition", "parentAttributeId", "active", "delete"}
