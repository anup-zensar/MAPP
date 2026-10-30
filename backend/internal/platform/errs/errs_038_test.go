package errs_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"mapp/internal/platform/errs"
)

func TestErrorCodes_038_FR014_FR054(t *testing.T) {
	t.Parallel()
	attempts := errs.TooManyAttempts("Too many sign-in attempts. Try again in 15 minutes.", 14*time.Minute)
	large := errs.TooLarge("This request is larger than 1 MB.")
	tests := []struct {
		name       string
		err        *errs.Error
		status     int
		code       string
		retryAfter time.Duration
	}{
		{"TooManyAttempts", attempts, http.StatusTooManyRequests, "too_many_attempts", 14 * time.Minute},
		{"TooLarge", large, http.StatusRequestEntityTooLarge, "too_large", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.err.Status != tt.status || tt.err.Code != tt.code || tt.err.RetryAfter != tt.retryAfter {
				t.Errorf("got %d %q retry after %v, want %d %q retry after %v",
					tt.err.Status, tt.err.Code, tt.err.RetryAfter, tt.status, tt.code, tt.retryAfter)
			}
			want := fmt.Sprintf(`{"code":%q,"message":%q,"fields":[]}`, tt.code, tt.err.Message)
			if got := encode(t, tt.err); got != want {
				t.Errorf("JSON =\n%s\nwant\n%s", got, want)
			}
		})
	}
}
