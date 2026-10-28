package seed

import "mapp/internal/settings"

const (
	purchaseRequestSection = "Purchase request"
	prLineSection          = "PR line"
)

// prAttribute is one row of 032's PR Field Inventory: needed marks it needed to submit.
type prAttribute struct {
	label, help string
	typ         settings.AttributeType
	needed      bool
	role        string
	options     []string
}

// purchaseRequestAttributes are the PR Field Inventory of 032's spec, in its order: the Purchase
// request level, then the PR line level; MAPP reads the ones with a role. Shell's Email group names are
// configuration (D-325); the PR recipients and OA vendors are fictitious.
func purchaseRequestAttributes() []demoAttribute {
	text, long, multiple := settings.TypeText, settings.TypeLongText, settings.TypeMultipleChoice
	header := []prAttribute{
		{"Material or service", "Whether the purchase is for goods or for a service.", multiple, true, "", []string{"Material", "Service"}},
		{"PR type", "How the requisitioner raises it in GSAP.", multiple, true, settings.RolePRType,
			[]string{"Standard PR/PO", "Manual PR/PO", "FI Payment"}},
		{"Short text", "What is bought, in a few words; sent to the requisitioner.", text, true, settings.RoleShortText, nil},
		{"Requestor user ID", "The GSAP user ID of the person requesting.", text, false, "", nil},
		{"Behalf of requestor", "The person the purchase is requested for, if someone else.", text, false, "", nil},
		{"Invoice approver name", "Who approves the vendor's invoices.", text, true, "", nil},
		{"Vendor", "The supplier's name; the Purchase requests page groups by it.", text, true, settings.RoleVendor, nil},
		{"Vendor code", "The supplier's GSAP vendor number, on the PR form.", text, false, settings.RoleVendorCode, nil},
		{"Vendor contact name", "The supplier's contact person.", text, false, "", nil},
		{"Vendor email address", "The supplier contact's e-mail address.", text, false, settings.RoleVendorEmail, nil},
		{"Vendor address", "The supplier's postal address.", long, false, "", nil},
		{"Contract holder", "Who holds the contract with the supplier.", text, false, "", nil},
		{"Outline agreement", "The SAP outline agreement number, if the purchase falls under one.", text, false, "", nil},
		{"CP lead", "The contracting and procurement lead.", text, false, "", nil},
		{"Email group", "The requisitioner team that receives the PR form.", settings.TypeSingleChoice, false, settings.RoleEmailGroup,
			[]string{"AT_RETAIL", "BE_LUBES", "DE_LUBES", "FR_LUBES", "G_GB_RETAIL", "CN_MKT_MO_LUBES"}},
		{"PR recipients", "Other people who receive the PR form.", multiple, false, settings.RolePRRecipients,
			[]string{leasEmail, "pr.desk.manila@shell.com", "pr.desk.krakow@shell.com"}},
		{"OA vendor (China specific)", "The outline-agreement vendor, for China operating units.", multiple, false, "",
			[]string{"Huaxin Printing Co.", "Lumen Events Shanghai", "Redpoint Media"}},
		{"Sub category (China specific)", "The purchase's sub category, for China operating units.", multiple, false, "",
			[]string{"Anti-counterfeit", "B2B Sensor", "Branded item", "Cinema", "Creative", "Digital marketing"}},
	}
	lines := []prAttribute{
		{"Material group", "The GSAP material group of the line.", text, false, "", nil},
		{"Remarks / comments", "Anything the requisitioner should know about the line.", long, false, "", nil},
		{"Material / service code", "The GSAP material or service number of the line.", text, false, "", nil},
	}
	list := make([]demoAttribute, 0, len(header)+len(lines))
	for _, a := range header {
		list = append(list, a.attribute(settings.LevelPurchaseRequest, purchaseRequestSection))
	}
	for _, a := range lines {
		list = append(list, a.attribute(settings.LevelPurchaseRequestLine, prLineSection))
	}
	return list
}

func (a prAttribute) attribute(level settings.Level, section string) demoAttribute {
	marking := settings.MarkingOptional
	if a.needed {
		marking = settings.MarkingNeededToSubmit
	}
	return demoAttribute{level: level, label: a.label, help: a.help, typ: a.typ, marking: marking, section: section,
		role: a.role, options: opts(a.options...)}
}
