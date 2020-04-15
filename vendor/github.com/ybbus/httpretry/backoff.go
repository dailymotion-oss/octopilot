package httpretry

import (
	"math"
	"math/rand"
	"time"
)

// BackoffPolicy is used to calculate the time to wait, before executing another request.
//
// The backoff can be calculated by taking the current number of retries into consideration.
type BackoffPolicy func(attemptCount int) time.Duration

var (
	// defaultBackoffPolicy uses ExponentialBackoff with 1 second minWait, 30 seconds max wait and 200ms jitter
	defaultBackoffPolicy = ExponentialBackoff(1*time.Second, 30*time.Second, 200*time.Millisecond)

	// ConstantBackoff waits for the exact same duration after a failed retry.
	//
	// constantWait: the constant backoff
	//
	// maxJitter: random interval [0, maxJitter) added to the exponential backoff
	//
	// Example:
	//   minWait = 2 * time.Seconds
	//   maxJitter = 0 * time.Seconds
	//
	//   Backoff will be: 2, 2, 2, ...
	ConstantBackoff = func(constantWait time.Duration, maxJitter time.Duration) BackoffPolicy {
		if constantWait < 0 {
			constantWait = 0
		}
		if maxJitter < 0 {
			maxJitter = 0
		}

		return func(attemptCount int) time.Duration {
			return constantWait + randJitter(maxJitter)
		}
	}

	// LinearBackoff increases the backoff time by multiplying the minWait duration by the number of attempts.
	//
	// minWait: the initial backoff
	//
	// maxWait: sets an upper bound on the maximum time to wait between two requests. set to 0 for no upper bound
	//
	// maxJitter: random interval [0, maxJitter) added to the linear backoff
	//
	// Example:
	//   minWait = 1 * time.Seconds
	//   maxWait = 5 * time.Seconds
	//   maxJitter = 0 * time.Seconds
	//
	//   Backoff will be: 1, 2, 3, 4, 5, 5, 5, ...
	LinearBackoff = func(minWait time.Duration, maxWait time.Duration, maxJitter time.Duration) BackoffPolicy {
		if minWait < 0 {
			minWait = 0
		}
		if maxJitter < 0 {
			maxJitter = 0
		}
		if maxWait < minWait {
			maxWait = 0
		}
		return func(attemptCount int) time.Duration {
			nextWait := time.Duration(attemptCount)*minWait + randJitter(maxJitter)
			if maxWait > 0 {
				return minDuration(nextWait, maxWait)
			}
			return nextWait
		}
	}

	// ExponentialBackoff increases the backoff exponentially by multiplying the minWait with 2^attemptCount
	//
	// minWait: the initial backoff
	//
	// maxWait: sets an upper bound on the maximum time to wait between two requests. set to 0 for no upper bound
	//
	// maxJitter: random interval [0, maxJitter) added to the exponential backoff
	//
	// Example:
	//   minWait = 1 * time.Seconds
	//   maxWait = 60 * time.Seconds
	//   maxJitter = 0 * time.Seconds
	//
	//   Backoff will be: 1, 2, 4, 8, 16, 32, 60, 60, ...
	ExponentialBackoff = func(minWait time.Duration, maxWait time.Duration, maxJitter time.Duration) BackoffPolicy {
		if minWait < 0 {
			minWait = 0
		}
		if maxJitter < 0 {
			maxJitter = 0
		}
		if maxWait < minWait {
			maxWait = 0
		}
		return func(attemptCount int) time.Duration {
			nextWait := time.Duration(math.Pow(2, float64(attemptCount-1)))*minWait + randJitter(maxJitter)
			if maxWait > 0 {
				return minDuration(nextWait, maxWait)
			}
			return nextWait
		}
	}
)

// minDuration returns the minimum of two durations
func minDuration(duration1 time.Duration, duration2 time.Duration) time.Duration {
	if duration1 < duration2 {
		return duration1
	}
	return duration2
}

// randJitter returns a random duration in the interval [0, maxJitter)
//
// if maxJitter is <= 0, a duration of 0 is returned
func randJitter(maxJitter time.Duration) time.Duration {
	if maxJitter <= 0 {
		return 0
	}

	return time.Duration(rand.Intn(int(maxJitter)))
}
