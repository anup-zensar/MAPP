package access

import "mapp/internal/platform/authz"

func homeRoutes() map[string]Rule {
	return map[string]Rule{
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
		"GET /api/v1/all-work":                    signedIn(),
		"GET /api/v1/all-work/options":            signedIn(),
		"GET /api/v1/all-work/export":             inPlan(authz.DataExport),
		"GET /api/v1/all-work/records/{recordId}": view(), // a record the person cannot see answers 403
		"GET /api/v1/all-work/preferences":        signedIn(),
		"PUT /api/v1/all-work/preferences":        signedIn(),
		"GET /api/v1/saved-views":                 signedIn(),
		"POST /api/v1/saved-views":                signedIn(),
		"PATCH /api/v1/saved-views/{viewId}":      signedIn(),
		"DELETE /api/v1/saved-views/{viewId}":     signedIn(),
	}
}

func historyRoutes() map[string]Rule {
	return map[string]Rule{
		"GET /api/v1/activity-log":                          signedIn(),
		"GET /api/v1/activity-log/summary":                  signedIn(),
		"GET /api/v1/recycle-bin":                           signedIn(),
		"POST /api/v1/recycle-bin/{entryId}/undo":           custom(), // history.Access.CanRestore
		"POST /api/v1/recycle-bin/{entryId}/restore":        custom(),
		"GET /api/v1/recycle-bin/{entryId}/restore-preview": custom(),
		"DELETE /api/v1/recycle-bin/{entryId}":              need(authz.RecycleBinPurge),
		"POST /api/v1/recycle-bin/empty":                    need(authz.RecycleBinPurge),
	}
}

func financeRoutes() map[string]Rule {
	return map[string]Rule{
		"GET /api/v1/financials/records/{recordId}":                   inPlan(authz.FinanceView),
		"GET /api/v1/financials/transactions":                         need(authz.FinanceView),
		"GET /api/v1/financials/transactions/{transactionId}":         need(authz.FinanceView),
		"POST /api/v1/financials/transactions/{transactionId}/link":   need(authz.FinanceView), // the service checks finance.link after scope
		"POST /api/v1/financials/transactions/{transactionId}/unlink": need(authz.FinanceView),
		"GET /api/v1/financials/overview":                             need(authz.FinanceView),
		"GET /api/v1/financials/imports":                              need(authz.FinanceView),
		"POST /api/v1/financials/imports":                             need(authz.FinanceImport),
		"POST /api/v1/financials/imports/check":                       need(authz.FinanceImport),
		"GET /api/v1/financials/imports/template":                     need(authz.FinanceImport),
	}
}
