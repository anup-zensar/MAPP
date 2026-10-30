// Package performance holds the records behind every performance figure (feature 046).
package performance

import "mapp/internal/platform/record"

type Category string

const (
	CategoryWebAnalytics        Category = "webAnalytics"
	CategoryMarketingAutomation Category = "marketingAutomation"
	CategoryCRM                 Category = "crm"
	CategoryLeadProgression     Category = "leadProgression"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

type DeliveryRoute string

const (
	RouteUpload    DeliveryRoute = "upload"
	RouteSimulated DeliveryRoute = "simulated"
)

type Cadence string

const (
	CadenceDaily    Cadence = "daily"
	CadenceWeekly   Cadence = "weekly"
	CadenceMonthly  Cadence = "monthly"
	CadenceOnDemand Cadence = "onDemand"
)

// Freshness is derived from the latest load and the cadence: a stored value would go stale on its own.
type Freshness string

const (
	FreshnessCurrent  Freshness = "current"
	FreshnessStale    Freshness = "stale"
	FreshnessFailed   Freshness = "failed"
	FreshnessInactive Freshness = "inactive"
)

type Source struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	OwnerID       string        `json:"ownerId"`
	Category      Category      `json:"category"`
	Status        Status        `json:"status"`
	DeliveryRoute DeliveryRoute `json:"deliveryRoute"`
	Cadence       Cadence       `json:"cadence"`
	IntegrationID string        `json:"integrationId"`
	record.Meta
}

func (s Source) CanPull() bool {
	return s.Status == StatusActive && s.DeliveryRoute == RouteSimulated
}

type Meaning string

const (
	MeaningKey       Meaning = "key"
	MeaningReference Meaning = "reference"
	MeaningFactDate  Meaning = "factDate"
	MeaningStage     Meaning = "stage"
	MeaningRevenue   Meaning = "revenue"
	MeaningCurrency  Meaning = "currency"
	MeaningWebMetric Meaning = "webMetric"
	MeaningJourney   Meaning = "journey"
)

// WebMetric is the web-analytics vocabulary (046-C11): counts by month, never money, and never a stage.
type WebMetric string

const (
	MetricSessions          WebMetric = "sessions"
	MetricVisits            WebMetric = "visits"
	MetricEngagementTouches WebMetric = "engagementTouches"
)

type Mapping struct {
	ID         string    `json:"id"`
	SourceID   string    `json:"sourceId"`
	ColumnName string    `json:"columnName"`
	Meaning    Meaning   `json:"meaning"`
	Stage      string    `json:"stage"`
	WebMetric  WebMetric `json:"webMetric"`
	ModelID    string    `json:"modelId"`
	record.Meta
}

// SuppliedBy marks what FR-003 lets only one active source do for a model.
func (m Mapping) SuppliedBy() bool {
	return m.Meaning == MeaningStage || m.Meaning == MeaningRevenue
}
