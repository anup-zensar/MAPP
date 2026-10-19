package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"northstar/internal/history"
	"northstar/internal/platform/money"
)

// Reference is one record outside Settings and the Settings records it uses (data model §10, D-042):
// one per plan, campaign, initiative, activity, line item and person, live or in the recycle bin. Features
// 003–007 report them through a ReferenceSource, so Settings counts "Used by", refuses in-use deletes and
// states impacts without importing those features.
type Reference struct {
	Record              history.RecordRef // {type: "activity", id: "A-1003", name: "Launch film"}; plans carry their name
	InBin               bool              // true while the record is in the recycle bin
	Level               Level             // its Shell-attribute level; "" for people
	PlanID              string            // its plan (a plan: its own id); "" for people
	ActivityID          string            // an activity: its own id; a line item: its activity; "" otherwise
	Attributes          AttributeValues   // its Shell attribute values; nil when none
	ModelID             string            // plans
	ActivityTypeID      string            // activities
	Currency            string            // line items: their currency; people: their display currency
	PlannedCost         *money.Amount     // line items: planned cost in Currency
	FinanceBudgetID     string            // plans: default finance budget
	FinanceBudgetItemID string            // line items: linked finance budget item
}

// ReferenceSource lists every record of one kind, live and in the recycle bin (read with
// history.SnapshotsInBin), inside the caller's transaction, which ctx carries. Features 003–007 each provide
// one; internal/app registers them with (*Service).AddReferenceSource.
type ReferenceSource func(ctx context.Context) ([]Reference, error)

// The kinds of usage parts (contract UsagePart.kind) that are not a level.
const (
	usedByPerson       = "person"
	usedByModel        = "model"
	usedByDependent    = "dependentAttribute"
	usedByCondition    = "condition"
	usedByLinkedOption = "linkedOption"
)

// notUsedYet is the summary of a record nothing uses.
const notUsedYet = "Not used yet"

// maxUsageRecords is how many plans Usage.Records lists at most.
const maxUsageRecords = 50

// levelNouns are the singular and plural nouns of the records of each level, as "Used by" counts them.
var levelNouns = map[Level][2]string{
	LevelPlan:       {"plan", "plans"},
	LevelCampaign:   {"campaign", "campaigns"},
	LevelInitiative: {"initiative", "initiatives"},
	LevelActivity:   {"activity", "activities"},
	LevelLineItem:   {"line item", "line items"},
}

// tally counts the records that use a Settings record, and how many of them are in the recycle bin.
type tally struct {
	count, inBin int
}

// add counts one record.
func (t *tally) add(inBin bool) {
	t.count++
	if inBin {
		t.inBin++
	}
}

// binned is a Settings record that may be in the recycle bin: Settings records in the bin count in "Used
// by" the same way (data model §10), so a restore never meets a missing setting.
type binned[T any] struct {
	record T
	inBin  bool
}

// usageIndex answers "Used by" for every Settings record (FR-003, data model §10). It is built once per
// request by (*Service).usage, inside the request's transaction, from the references of features 003–007
// and the Settings records themselves, live and in the recycle bin.
type usageIndex struct {
	refs       []Reference
	attributes map[string]Attribute // every attribute, live and in the bin, by id

	plansByModel       map[string]*tally
	activitiesByType   map[string]*tally
	modelsByType       map[string]*tally
	valuesByAttribute  map[string]*tally
	valuesByOption     map[string]*tally
	dependents         map[string][]binned[Attribute] // parent attribute id → the attributes depending on it
	conditions         tally                          // attributes with a business-unit condition
	conditionsByOption map[string]*tally
	linkedOptions      map[string]map[string]*tally // parent option id → dependent attribute id → its options linked to it
	lineItemsByCurr    map[string]*tally
	peopleByCurr       map[string]*tally
	lineItemsByItem    map[string]*tally
	plansByBudget      map[string]*tally
}

// usage reads every reference source once, the Settings records and the Settings records in the recycle
// bin, inside the caller's transaction, and returns the index that answers "Used by".
func (s *Service) usage(ctx context.Context) (*usageIndex, error) {
	u := &usageIndex{
		attributes:   map[string]Attribute{},
		plansByModel: map[string]*tally{}, activitiesByType: map[string]*tally{}, modelsByType: map[string]*tally{},
		valuesByAttribute: map[string]*tally{}, valuesByOption: map[string]*tally{},
		dependents: map[string][]binned[Attribute]{}, conditionsByOption: map[string]*tally{},
		linkedOptions:   map[string]map[string]*tally{},
		lineItemsByCurr: map[string]*tally{}, peopleByCurr: map[string]*tally{}, lineItemsByItem: map[string]*tally{},
		plansByBudget: map[string]*tally{},
	}
	refs, err := s.references(ctx)
	if err != nil {
		return nil, err
	}
	u.refs = refs
	models, attributes, options, err := s.settingsInUse(ctx)
	if err != nil {
		return nil, err
	}
	for _, a := range attributes {
		u.attributes[a.record.ID] = a.record
	}

	for _, m := range models {
		for _, typeID := range m.record.ActivityTypeIDs {
			count(u.modelsByType, typeID, m.inBin)
		}
	}
	for _, a := range attributes {
		if a.record.ParentAttributeID != "" {
			u.dependents[a.record.ParentAttributeID] = append(u.dependents[a.record.ParentAttributeID], a)
		}
		if len(a.record.Condition) > 0 {
			u.conditions.add(a.inBin)
		}
		for _, optionID := range unique(a.record.Condition) {
			count(u.conditionsByOption, optionID, a.inBin)
		}
	}
	for _, o := range options {
		for _, parentID := range unique(o.record.ParentOptionIDs) {
			if u.linkedOptions[parentID] == nil {
				u.linkedOptions[parentID] = map[string]*tally{}
			}
			count(u.linkedOptions[parentID], o.record.AttributeID, o.inBin)
		}
	}
	for _, ref := range u.refs {
		u.addReference(ref)
	}
	return u, nil
}

// references reads every reference source once, inside the caller's transaction: the records of features
// 003–007 that use Settings records, live and in the recycle bin.
func (s *Service) references(ctx context.Context) ([]Reference, error) {
	refs := []Reference{}
	for _, source := range s.sources {
		listed, err := source(ctx)
		if err != nil {
			return nil, err
		}
		refs = append(refs, listed...)
	}
	return refs, nil
}

// addReference counts what one record outside Settings uses.
func (u *usageIndex) addReference(ref Reference) {
	if ref.ModelID != "" {
		count(u.plansByModel, ref.ModelID, ref.InBin)
	}
	if ref.ActivityTypeID != "" {
		count(u.activitiesByType, ref.ActivityTypeID, ref.InBin)
	}
	if ref.Currency != "" {
		if ref.Level == "" {
			count(u.peopleByCurr, ref.Currency, ref.InBin)
		} else {
			count(u.lineItemsByCurr, ref.Currency, ref.InBin)
		}
	}
	if ref.FinanceBudgetItemID != "" {
		count(u.lineItemsByItem, ref.FinanceBudgetItemID, ref.InBin)
	}
	if ref.FinanceBudgetID != "" {
		count(u.plansByBudget, ref.FinanceBudgetID, ref.InBin)
	}
	for attributeID, raw := range ref.Attributes {
		attribute, known := u.attributes[attributeID]
		if !known || attribute.Level != ref.Level || isEmptyValue(raw) {
			continue
		}
		count(u.valuesByAttribute, attributeID, ref.InBin)
		for _, optionID := range optionIDsOf(attribute, raw) {
			count(u.valuesByOption, optionID, ref.InBin)
		}
	}
}

// count adds one record to the tally of key in tallies.
func count(tallies map[string]*tally, key string, inBin bool) {
	if tallies[key] == nil {
		tallies[key] = &tally{}
	}
	tallies[key].add(inBin)
}

// settingsInUse returns the models, attributes and options that may use other Settings records: those
// stored and those in the recycle bin.
func (s *Service) settingsInUse(ctx context.Context) ([]binned[Model], []binned[Attribute], []binned[Option], error) {
	inBin, err := s.snapshotsInBin(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	storedModels, err := s.repo.ListModels(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	storedAttributes, err := s.repo.ListAttributes(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	storedOptions, err := s.repo.ListOptions(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	models, err := withBinned(storedModels, inBin[modelKind])
	if err != nil {
		return nil, nil, nil, err
	}
	attributes, err := withBinned(storedAttributes, inBin[attributeKind])
	if err != nil {
		return nil, nil, nil, err
	}
	options, err := withBinned(storedOptions, inBin[optionKind])
	if err != nil {
		return nil, nil, nil, err
	}
	return models, attributes, options, nil
}

// snapshotsInBin returns the snapshots of the records in the recycle bin by kind, oldest entry first. An
// entry that was undone or restored holds no snapshots any more.
func (s *Service) snapshotsInBin(ctx context.Context) (map[string][]history.Snapshot, error) {
	entries, err := s.bin.BinEntries(ctx)
	if err != nil {
		return nil, err
	}
	byKind := map[string][]history.Snapshot{}
	for _, entry := range entries {
		if entry.Resolution != nil {
			continue
		}
		for _, snapshot := range entry.Snapshots {
			byKind[snapshot.Kind] = append(byKind[snapshot.Kind], snapshot)
		}
	}
	return byKind, nil
}

// withBinned returns the stored records followed by the records of the snapshots, marked as in the bin.
func withBinned[T any](stored []T, snapshots []history.Snapshot) ([]binned[T], error) {
	list := make([]binned[T], 0, len(stored)+len(snapshots))
	for _, record := range stored {
		list = append(list, binned[T]{record: record})
	}
	for _, snapshot := range snapshots {
		var record T
		if err := json.Unmarshal(snapshot.Data, &record); err != nil {
			return nil, fmt.Errorf("settings: read %s %s in the recycle bin: %w", snapshot.Kind, snapshot.ID, err)
		}
		list = append(list, binned[T]{record: record, inBin: true})
	}
	return list, nil
}

// part returns one usage part: text, followed by " (<n> in the recycle bin)" when some of the records
// counted are in the bin.
func part(kind string, t tally, text string) UsagePart {
	if t.inBin > 0 {
		text += fmt.Sprintf(" (%d in the recycle bin)", t.inBin)
	}
	return UsagePart{Kind: kind, Text: text, Count: t.count, InRecycleBin: t.inBin}
}

// newUsage returns the usage of parts: in use when there is any part, the parts joined with " · " as its
// summary, or "Not used yet". records lists plans for models and finance budgets; it is never nil.
func newUsage(parts []UsagePart, records []history.RecordRef) Usage {
	if records == nil {
		records = []history.RecordRef{}
	}
	if len(parts) == 0 {
		return Usage{InUse: false, Summary: notUsedYet, Parts: []UsagePart{}, Records: records}
	}
	texts := make([]string, 0, len(parts))
	for _, p := range parts {
		texts = append(texts, p.Text)
	}
	return Usage{InUse: true, Summary: strings.Join(texts, " · "), Parts: parts, Records: records}
}

// usedBy returns the part "Used by <n> <records>" of tallies[key], or false when nothing is counted.
func usedBy(tallies map[string]*tally, key, kind, singular, pluralNoun string) (UsagePart, bool) {
	t := tallies[key]
	if t == nil {
		return UsagePart{}, false
	}
	return part(kind, *t, "Used by "+plural(t.count, singular, pluralNoun)), true
}

// model returns the usage of the model id: the plans using it, and the live ones among them as records.
func (u *usageIndex) model(id string) Usage {
	var parts []UsagePart
	if p, used := usedBy(u.plansByModel, id, string(LevelPlan), "plan", "plans"); used {
		parts = append(parts, p)
	}
	return newUsage(parts, u.livePlans(func(ref Reference) bool { return ref.ModelID == id }))
}

// activityType returns the usage of the activity type id: the activities of that type and the models that
// allow it.
func (u *usageIndex) activityType(id string) Usage {
	var parts []UsagePart
	if p, used := usedBy(u.activitiesByType, id, string(LevelActivity), "activity", "activities"); used {
		parts = append(parts, p)
	}
	if t := u.modelsByType[id]; t != nil {
		parts = append(parts, part(usedByModel, *t, "Allowed in "+plural(t.count, "model", "models")))
	}
	return newUsage(parts, nil)
}

// attribute returns the usage of the Shell attribute a: the records of its level with a value, the
// attributes depending on it and, for the Business unit attribute, the attributes with a condition.
func (u *usageIndex) attribute(a Attribute) Usage {
	var parts []UsagePart
	nouns := levelNouns[a.Level]
	if p, used := usedBy(u.valuesByAttribute, a.ID, string(a.Level), nouns[0], nouns[1]); used {
		parts = append(parts, p)
	}
	if dependents := u.dependents[a.ID]; len(dependents) > 0 {
		var t tally
		for _, d := range dependents {
			t.add(d.inBin)
		}
		text := fmt.Sprintf("%d attributes depend on it", t.count)
		if t.count == 1 {
			text = dependents[0].record.Label + " depends on it"
		}
		parts = append(parts, part(usedByDependent, t, text))
	}
	if a.Role == RoleBusinessUnit && u.conditions.count > 0 {
		parts = append(parts, part(usedByCondition, u.conditions, "Condition of "+plural(u.conditions.count, "attribute", "attributes")))
	}
	return newUsage(parts, nil)
}

// option returns the usage of the option o of the attribute a: the records whose value has it, the options
// of dependent attributes linked to it, and the attributes whose condition has it.
func (u *usageIndex) option(o Option, a Attribute) Usage {
	var parts []UsagePart
	nouns := levelNouns[a.Level]
	if p, used := usedBy(u.valuesByOption, o.ID, string(a.Level), nouns[0], nouns[1]); used {
		parts = append(parts, p)
	}
	linked := u.linkedOptions[o.ID]
	for _, attributeID := range slices.SortedFunc(maps.Keys(linked), compareIDs) {
		t := linked[attributeID]
		label := u.attributes[attributeID].Label
		parts = append(parts, part(usedByLinkedOption, *t,
			fmt.Sprintf("Linked to %s", plural(t.count, label+" option", label+" options"))))
	}
	if t := u.conditionsByOption[o.ID]; t != nil {
		parts = append(parts, part(usedByCondition, *t, "Condition of "+plural(t.count, "attribute", "attributes")))
	}
	return newUsage(parts, nil)
}

// currency returns the usage of the currency code: the line items in it and the people showing amounts in
// it.
func (u *usageIndex) currency(code string) Usage {
	var parts []UsagePart
	if p, used := usedBy(u.lineItemsByCurr, code, string(LevelLineItem), "line item", "line items"); used {
		parts = append(parts, p)
	}
	if t := u.peopleByCurr[code]; t != nil {
		parts = append(parts, part(usedByPerson, *t, "Display currency of "+plural(t.count, "person", "people")))
	}
	return newUsage(parts, nil)
}

// item returns the usage of the finance budget item id: the line items linked to it.
func (u *usageIndex) item(id string) Usage {
	var parts []UsagePart
	if t := u.lineItemsByItem[id]; t != nil {
		parts = append(parts, part(string(LevelLineItem), *t, "Linked to "+plural(t.count, "line item", "line items")))
	}
	return newUsage(parts, nil)
}

// budget returns the usage of the finance budget id: the plans using it as their default finance budget,
// and the live ones among them as records. It never blocks anything: budgets are not deleted (D-047).
func (u *usageIndex) budget(id string) Usage {
	var parts []UsagePart
	if t := u.plansByBudget[id]; t != nil {
		parts = append(parts, part(string(LevelPlan), *t, "Default finance budget of "+plural(t.count, "plan", "plans")))
	}
	return newUsage(parts, u.livePlans(func(ref Reference) bool { return ref.FinanceBudgetID == id }))
}

// livePlans lists the plans outside the recycle bin that match, sorted by name, at most 50.
func (u *usageIndex) livePlans(match func(Reference) bool) []history.RecordRef {
	plans := []history.RecordRef{}
	for _, ref := range u.refs {
		if ref.Level == LevelPlan && !ref.InBin && match(ref) {
			plans = append(plans, ref.Record)
		}
	}
	slices.SortStableFunc(plans, func(a, b history.RecordRef) int { return compareNames(a.Name, b.Name) })
	return plans[:min(len(plans), maxUsageRecords)]
}
