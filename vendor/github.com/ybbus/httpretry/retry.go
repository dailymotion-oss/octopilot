package httpretry

import (
	"crypto/x509"
	"net/http"
	"net/url"
	"strings"
)

// RetryPolicy decides if a request should be retried.
//
// This is done by examining the response status code and the error message of the last response.
//
// The statusCode may be 0 if there was no response available (e.g. in case of a request error).
type RetryPolicy func(statusCode int, err error) bool

var defaultRetryPolicy RetryPolicy = func(statusCode int, err error) bool {
	// check if error is of type temporary
	t, ok := err.(interface{ Temporary() bool })
	if ok && t.Temporary() {
		return true
	}

	// we cannot know all errors, so we filter errors that should NOT be retried
	switch e := err.(type) {
	case *url.Error:
		switch {
		case
			e.Op == "parse",
			strings.Contains(e.Err.Error(), "stopped after"),
			strings.Contains(e.Error(), "unsupported protocol scheme"),
			strings.Contains(e.Error(), "no Host in request URL"):
			return false
		}
		// check inner error of url.Error
		switch e.Err.(type) {
		case // this errors will not likely change when retrying
			x509.UnknownAuthorityError,
			x509.CertificateInvalidError,
			x509.ConstraintViolationError:
			return false
		}
	case error: // generic error, check for strings if nothing found, retry
		return true
	case nil: // no error, continue
	}

	// most of the codes should not be retried, so we filter status codes that SHOULD be retried
	switch statusCode {
	case // status codes that should be retried
		http.StatusRequestTimeout,
		http.StatusConflict,
		http.StatusLocked,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
		http.StatusInsufficientStorage:
		return true
	case 0: // means we did not get a response. we need to retry
		return true
	default: // on all other status codes we should not retry (e.g. 200, 401 etc.)
		return false
	}
}
