package httpretry

// Option is a function type to modify the RetryRoundtripper configuration
type Option func(*RetryRoundtripper)

// WithMaxRetryCount sets the maximum number of retries if an http request was not successful.
//
// Default: 5
func WithMaxRetryCount(maxRetryCount int) Option {
	if maxRetryCount < 0 {
		maxRetryCount = 0
	}
	return func(roundtripper *RetryRoundtripper) {
		roundtripper.MaxRetryCount = maxRetryCount
	}
}

// WithRetryPolicy sets the user defined retry policy.
//
// Default: RetryPolicy checks for some common errors that are likely not retryable and for status codes
// that should be retried.
//
// For example:
//  - url parsing errors
//  - too many redirects
//  - certificate errors
//  - BadGateway
//  - ServiceUnavailable
//  - etc.
func WithRetryPolicy(retryPolicy RetryPolicy) Option {
	return func(roundtripper *RetryRoundtripper) {
		roundtripper.ShouldRetry = retryPolicy
	}
}

// WithBackoffPolicy sets the user defined backoff policy.
//
// Default: ExponentialBackoff(1*time.Second, 30*time.Second, 200*time.Millisecond)
func WithBackoffPolicy(backoffPolicy BackoffPolicy) Option {
	return func(roundtripper *RetryRoundtripper) {
		roundtripper.CalculateBackoff = backoffPolicy
	}
}
