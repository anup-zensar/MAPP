package apibench_test

import (
	"net/http"
	"testing"

	"mapp/internal/platform/apibench"
)

func BenchmarkPurchaseRequestsPage(b *testing.B) {
	server := apibench.New(b, apibench.PurchaseRequests)
	lea := server.SignIn(b, "lea.santos@shell.com")
	for name, query := range map[string]string{
		"first page":       "?pageSize=50",
		"search HX key":    "?q=hx1003&pageSize=50",
		"one vendor":       "?vendor=volume%20vendor%2007&pageSize=50",
		"waiting for GSAP": "?waitingForGsap=true",
	} {
		b.Run(name, func(b *testing.B) {
			lea.Measure(b, http.MethodGet, "/api/v1/purchase-requests"+query, nil)
		})
	}
}
