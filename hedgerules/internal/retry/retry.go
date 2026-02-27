package retry

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/aws/smithy-go"
)

// throttleCodes are AWS error codes that indicate rate limiting or throttling.
var throttleCodes = map[string]bool{
	"Throttling":                            true,
	"ThrottlingException":                   true,
	"RequestThrottled":                      true,
	"TooManyRequestsException":              true,
	"ProvisionedThroughputExceededException": true,
	"TransactionInProgressException":        true,
	"RequestLimitExceeded":                  true,
	"BandwidthLimitExceeded":                true,
	"LimitExceededException":                true,
}

// IsThrottle reports whether err is an AWS throttling or rate-limiting error.
func IsThrottle(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return throttleCodes[apiErr.ErrorCode()]
	}
	return false
}

// Do calls fn and retries up to maxRetries times if fn returns a throttle error.
// Non-throttle errors are returned immediately without retrying.
// Retries use exponential backoff with jitter, starting at 1s and capping at 30s.
func Do(maxRetries int, fn func() error) error {
	for attempt := 0; ; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		if !IsThrottle(err) {
			return err
		}
		if attempt >= maxRetries {
			return err
		}

		backoff := time.Duration(1<<uint(attempt)) * time.Second
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
		jitter := time.Duration(rand.Int63n(int64(backoff/2 + 1)))
		sleep := backoff + jitter

		fmt.Fprintf(os.Stderr, "Rate limited by AWS, retrying in %s (attempt %d/%d)...\n",
			sleep.Round(time.Millisecond), attempt+1, maxRetries)
		time.Sleep(sleep)
	}
}
