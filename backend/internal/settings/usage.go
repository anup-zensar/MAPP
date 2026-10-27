package settings

import (
	"context"
	"encoding/json"
	"fmt"

	"mapp/internal/history"
	"mapp/internal/platform/money"
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
	Currency            string            // line items: their currency; people: their display currency; plans: their plan currency (016)
	PlannedCost         *money.Amount     // line items: planned cost in Currency
	Budget              *money.Amount     // plans: the plan budget in Currency (016 FR-005)
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
	plansByCurr        map[string]*tally // plans planned in a currency (016 FR-010)
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
		lineItemsByCurr: map[string]*tally{}, peopleByCurr: map[string]*tally{}, plansByCurr: map[string]*tally{}, lineItemsByItem: map[string]*tally{},
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
		switch ref.Level {
		case "":
			count(u.peopleByCurr, ref.Currency, ref.InBin)
		case LevelPlan:
			count(u.plansByCurr, ref.Currency, ref.InBin)
		default:
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
	inBin := map[string][]history.Snapshot{}
	for _, kind := range []string{modelKind, attributeKind, optionKind} {
		snapshots, err := s.history.SnapshotsInBin(ctx, kind)
		if err != nil {
			return nil, nil, nil, err
		}
		inBin[kind] = snapshots
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
