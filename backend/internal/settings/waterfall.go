package settings

// Waterfall works out the goals a model gives for inq inquiries (domain model §4.5, data model §2.1,
// FR-010): MQL = the INQ→MQL rate of the inquiries and deals = the MQL→deal rate of the MQL, each rounded
// half-up to a whole number; revenue = deals × the average deal size, exactly, in USD. A revenue beyond
// what money keeps is money.ErrOutOfRange. The web app's lib/waterfall.ts repeats it for the live example;
// both are tested with the same table. Working a goal backwards is feature 004's rule.
func Waterfall(inq int64, m Model) (Goals, error) {
	mql := m.InqToMqlRate.OfWhole(inq)
	deals := m.MqlToDealRate.OfWhole(mql)
	revenue, err := m.AverageDealSize.Amount.MulInt(deals)
	if err != nil {
		return Goals{}, err
	}
	return Goals{INQ: inq, MQL: mql, Deals: deals, Revenue: revenue}, nil
}
