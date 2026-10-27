package access

import "mapp/internal/platform/authz"

func homeRoutes() map[string]Rule {
	return map[string]Rule{
		// 009 Home: filtered to what the person can see
		"GET /api/v1/home/summary":        signedIn(),
		"GET /api/v1/home/plans":          signedIn(),
		"GET /api/v1/home/attention":      signedIn(),
		"GET /api/v1/home/waiting":        signedIn(),
		"GET /api/v1/home/my-activities":  signedIn(),
		"GET /api/v1/home/recent-changes": signedIn(),
	}
}

func allWorkRoutes() map[string]Rule {
	return map[string]Rule{
		// 010 All work: filtered; export needs data.export
		"GET /api/v1/all-work":                    signedIn(),
		"GET /api/v1/all-work/options":            signedIn(),
		"GET /api/v1/all-work/export":             inPlan(authz.DataExport),
		"GET /api/v1/all-work/records/{recordId}": view(),     // a record the person cannot see answers 403
		"GET /api/v1/all-work/preferences":        signedIn(), // own
		"PUT /api/v1/all-work/preferences":        signedIn(), // own
		"GET /api/v1/saved-views":                 signedIn(), // own
		"POST /api/v1/saved-views":                signedIn(), // own
		"PATCH /api/v1/saved-views/{viewId}":      signedIn(), // own
		"DELETE /api/v1/saved-views/{viewId}":     signedIn(), // own
	}
}

func historyRoutes() map[string]Rule {
	return map[string]Rule{
		// 011 Activity log and recycle bin (D-238)
		"GET /api/v1/activity-log":                          signedIn(), // filtered
		"GET /api/v1/activity-log/summary":                  signedIn(), // filtered
		"GET /api/v1/recycle-bin":                           signedIn(), // filtered, rows answer access
		"POST /api/v1/recycle-bin/{entryId}/undo":           custom(),   // history.Access.CanRestore
		"POST /api/v1/recycle-bin/{entryId}/restore":        custom(),
		"GET /api/v1/recycle-bin/{entryId}/restore-preview": custom(),
		"DELETE /api/v1/recycle-bin/{entryId}":              need(authz.RecycleBinPurge),
		"POST /api/v1/recycle-bin/empty":                    need(authz.RecycleBinPurge),
	}
}

func financeRoutes() map[string]Rule {
	return map[string]Rule{
		// 015 Financial transactions (consolidation D-301)
		"GET /api/v1/financials/records/{recordId}":                   inPlan(authz.FinanceView), // View of the record + finance.view in its plan
		"GET /api/v1/financials/transactions":                         need(authz.FinanceView),   // filtered
		"GET /api/v1/financials/transactions/{transactionId}":         need(authz.FinanceView),
		"POST /api/v1/financials/transactions/{transactionId}/link":   inPlan(authz.FinanceLink),
		"POST /api/v1/financials/transactions/{transactionId}/unlink": inPlan(authz.FinanceLink),
		"GET /api/v1/financials/overview":                             need(authz.FinanceView), // filtered
		"GET /api/v1/financials/imports":                              need(authz.FinanceView),
		"POST /api/v1/financials/imports":                             need(authz.FinanceImport),
		"POST /api/v1/financials/imports/check":                       need(authz.FinanceImport),
		"GET /api/v1/financials/imports/template":                     need(authz.FinanceImport),
	}
}
