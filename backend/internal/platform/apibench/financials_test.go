package apibench_test

import (
	"net/http"
	"testing"

	"mapp/internal/platform/apibench"
)

// BenchmarkGSAPImport checks and imports a 5,000-row GSAP extract of new transactions on the volume.
func BenchmarkGSAPImport(b *testing.B) {
	server := apibench.New(b, apibench.Volume)
	alexClient := server.SignIn(b, alex)
	b.Run("check", func(b *testing.B) {
		alexClient.Measure(b, http.MethodPost, "/api/v1/financials/imports/check", server.GSAPFile(b, 7))
	})
	b.Run("import", func(b *testing.B) {
		alexClient.Measure(b, http.MethodPost, "/api/v1/financials/imports", server.GSAPFile(b, 8))
	})
}
