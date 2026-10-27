package allwork

import "mapp/internal/platform/money"

// seedYear is 2026 of the demo data on 18 Oct 2026: two plans, EUR and GBP line items, a balance of
// 20,000.00, one of 0.00, one of −500.00, an activity without dates and an ongoing one.
//
//	P-1001 FY2026 Lubricants Europe (Published, 1,200,000.00, Priya, T-1001, Lubricants)
//	  C-1001 Helix launch (700,000.00, Priya)
//	    I-1001 Digital & social (300,000.00)
//	      A-1001 Launch film (Created, Mar–May, 120,000.00; L-1001 50,000.00 USD, L-1002 46,000.00 EUR = 50,000.00)
//	      A-1002 Social always-on (Complete, Jan–Jun, 80,000.00; L-1003 15,800.00 GBP = 20,000.00, L-1004 5,000.00)
//	    I-1002 Print & outdoor (250,000.00)
//	      A-1003 Billboard winter wave (Complete, Jan–Feb, 45,000.00; L-1005 9,200.00 EUR = 10,000.00)
//	      A-1004 Trade magazine ads (Submitted, Apr–Sep, 30,000.00; L-1006 30,000.00 custom, deposit in Mar)
//	  C-1002 Rimula fleet (450,000.00, Daniel)
//	    I-1003 Events (200,000.00, Daniel)
//	      A-1005 Partner co-marketing fund (In progress, ongoing from Mar, 100,000.00)
//	    I-1004 Retail events (150,000.00, Daniel)
//	      A-1006 Fleet roadshow (Declined, Jul–Sep, 60,000.00)
//	      A-1007 Christmas trade gifting (Approved, Nov–Dec, 25,000.00, owner Daniel)
//	P-1002 FY2026 Mobility China (Draft, 800,000.00, Mei, T-1002, no business unit)
//	  C-1003 Retail growth (600,000.00, Mei)
//	    I-1005 Station promotions (350,000.00)
//	      A-1008 Grand opening Shanghai (Created, Apr–Jun, 40,000.00; L-1007 1,000.00, L-1008 39,500.00 → −500.00)
//	      A-1009 Lunar new year promo (Created, no dates, no planned cost, no owner)
//	    I-1006 Fleet cards (200,000.00, no dates)
//	      A-1010 Fleet card launch (Submitted, May–Dec, 150,000.00)
//	    I-1007 Loyalty (no activities)
func seedYear() *yearBuilder {
	b := newYear()
	lub := b.plan("P-1001", "FY2026 Lubricants Europe", status("published"), budget("1200000.00"), owner(priya),
		teams("T-1001"), units("SO-1002"), describe("Lubricants marketing in Europe"),
		field("plan.model", Value{Text: "Lubricants 1.0"}), field("attr.SA-1001", choices("Lubricants")))
	helix := b.campaign(lub, "C-1001", "Helix launch", owner(priya), dates("2026-01-01", "2026-12-31"),
		budget("700000.00"), progress("inProgress", 2, 4), timing("running"),
		field("attr.SA-1003", choices("Lubricants – Europe")), field("attr.SA-1004", choices("LUBES – DACH")))
	digital := b.initiative(helix, "I-1001", "Digital & social", owner(priya), dates("2026-01-01", "2026-06-30"),
		budget("300000.00"), progress("inProgress", 1, 2), timing("finished"))
	film := b.activity(digital, "A-1001", "Launch film", owner(priya), dates("2026-03-01", "2026-05-31"),
		planned("120000.00"), goals(250000, 5000, "2500000.00"), activityType("AT-1001"), externalName("Helix Ultra film"),
		describe("A hero film for the Helix Ultra relaunch"), field("activity.type", Value{Text: "Content production"}),
		field("attr.SA-1020", split("Germany", "60.00", "France", "40.00")), field("attr.SA-1021", Value{Number: ptr(int64(1200))}))
	b.lineItem(film, "L-1001", "Agency fee", "50000.00", owner(priya))
	b.lineItem(film, "L-1002", "Film production", "50000.00", owner(priya), local("46000.00", "EUR"),
		field("lineItem.localPlannedCost", Value{Money: &money.Money{Amount: amountOf("46000.00"), Currency: "EUR"}}),
		field("lineItem.currency", Value{Text: "EUR"}))
	social := b.activity(digital, "A-1002", "Social always-on", owner(priya), dates("2026-01-01", "2026-06-30"),
		planned("80000.00"), status("complete"), goals(40000, 800, "80000.00"), activityType("AT-1002"))
	b.lineItem(social, "L-1003", "Venue hire", "20000.00", owner(priya), local("15800.00", "GBP"))
	b.lineItem(social, "L-1004", "Speaker fees", "5000.00", owner(priya))
	printing := b.initiative(helix, "I-1002", "Print & outdoor", owner(priya), dates("2026-01-01", "2026-12-31"),
		budget("250000.00"), progress("inProgress", 1, 2), timing("running"))
	billboard := b.activity(printing, "A-1003", "Billboard winter wave", owner(priya), dates("2026-01-05", "2026-02-28"),
		planned("45000.00"), status("complete"), activityType("AT-1003"))
	b.lineItem(billboard, "L-1005", "Stand build", "10000.00", owner(priya), local("9200.00", "EUR"), swbs("readyToSend"))
	ads := b.activity(printing, "A-1004", "Trade magazine ads", owner(priya), dates("2026-04-01", "2026-09-30"),
		planned("30000.00"), status("submitted"), goals(20000, 400, "40000.00"), activityType("AT-1001"))
	b.lineItem(ads, "L-1006", "Media booking", "30000.00", owner(priya))
	b.months("L-1006", "2026-03", "3000.00", "2026-04", "4500.00", "2026-05", "4500.00", "2026-06", "4500.00",
		"2026-07", "4500.00", "2026-08", "4500.00", "2026-09", "4500.00")
	rimula := b.campaign(lub, "C-1002", "Rimula fleet", owner(daniel), dates("2026-02-01", "2026-12-31"),
		budget("450000.00"), progress("inProgress", 2, 3), timing("running"),
		field("attr.SA-1003", choices("Lubricants – Europe")), field("attr.SA-1004", choices("LUBES – BENELUX!")))
	events := b.initiative(rimula, "I-1003", "Events", owner(daniel), dates("2026-03-01", "2026-09-30"),
		budget("200000.00"), progress("allApproved", 1, 1), timing("finished"))
	b.activity(events, "A-1005", "Partner co-marketing fund", owner(priya), ongoingFrom("2026-03-01"),
		planned("100000.00"), status("inProgress"), activityType("AT-1004"))
	retail := b.initiative(rimula, "I-1004", "Retail events", owner(daniel), dates("2026-06-01", "2026-12-31"),
		budget("150000.00"), progress("inProgress", 1, 2), timing("running"))
	b.activity(retail, "A-1006", "Fleet roadshow", owner(priya), dates("2026-07-01", "2026-09-30"),
		planned("60000.00"), status("declined"), activityType("AT-1005"))
	b.activity(retail, "A-1007", "Christmas trade gifting", owner(daniel), dates("2026-11-01", "2026-12-15"),
		planned("25000.00"), status("approved"), activityType("AT-1005"))

	mob := b.plan("P-1002", "FY2026 Mobility China", budget("800000.00"), owner(mei), teams("T-1002"), missing(1))
	growth := b.campaign(mob, "C-1003", "Retail growth", owner(mei), dates("2026-01-01", "2026-12-31"),
		budget("600000.00"), progress("inProgress", 0, 3), timing("running"), missing(2))
	stations := b.initiative(growth, "I-1005", "Station promotions", owner(mei), dates("2026-03-01", "2026-09-30"),
		budget("350000.00"), progress("inProgress", 0, 2), timing("finished"))
	opening := b.activity(stations, "A-1008", "Grand opening Shanghai", owner(mei), dates("2026-04-01", "2026-06-30"),
		planned("40000.00"), activityType("AT-1005"), missing(1), field("attr.SA-1020", split("China", "70.00", "Macau!", "30.00")))
	b.lineItem(opening, "L-1007", "Station signage", "1000.00", owner(mei), missing(1))
	b.months("L-1007", "2026-04", "900.00")
	b.lineItem(opening, "L-1008", "Fleet card promotion", "39500.00", owner(mei), local("280450.00", "CNY"))
	b.activity(stations, "A-1009", "Lunar new year promo", activityType("AT-1002"), missing(5))
	cards := b.initiative(growth, "I-1006", "Fleet cards", owner(mei), budget("200000.00"), progress("inProgress", 0, 1),
		missing(1))
	b.activity(cards, "A-1010", "Fleet card launch", owner(mei), dates("2026-05-01", "2026-12-31"),
		planned("150000.00"), status("submitted"), activityType("AT-1006"),
		field("activity.type", Value{Text: "MOBILITY – Long-term contract", Inactive: true}))
	b.initiative(growth, "I-1007", "Loyalty", owner(sara), dates("2026-07-01", "2026-12-31"), timing("running"))
	return b
}
