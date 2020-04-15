package httpretry

import (
	"errors"
	"net/http"
)

const (
	defaultMaxRetryCount = 5
)

// NewDefaultClient returns a default http client with retry functionality wrapped around the Roundtripper (client.Transport).
//
// You should not replace the client.Transport field, otherwise you will lose the retry functionality.
//
// If you need to set / change the original client.Transport field you have two options:
//
// 1. create your own http client and use NewCustomClient() function to enrich the client with retry functionality.
//   client := &http.Client{}
//   client.Transport = &http.Transport{ ... }
//   retryClient := httpretry.NewCustomClient(client)
// 2. use one of the helper functions (e.g. httpretry.ModifyOriginalTransport(retryClient)) to retrieve and change the Transport.
//   retryClient := httpretry.NewDefaultClient()
//   err := httpretry.ModifyOriginalTransport(retryClient, func(t *http.Transport){t.TLSHandshakeTimeout = 5 * time.Second})
//   if err != nil { ... } // will be nil if embedded Roundtripper was not of type http.Transport
func NewDefaultClient(opts ...Option) *http.Client {
	return NewCustomClient(&http.Client{}, opts...)
}

// NewCustomClient returns the provided http client with retry functionality wrapped around the Roundtripper (client.Transport).
//
// You should not replace the client.Transport field after creating the retry client, otherwise you will lose the retry functionality.
//
// If you need to change the original client.Transport field you may use the helper functions:
//
//   err := httpretry.ModifyTransport(retryClient, func(t *http.Transport){t.TLSHandshakeTimeout = 5 * time.Second})
//   if err != nil { ... } // will be nil if embedded Roundtripper was not of type http.Transport
func NewCustomClient(client *http.Client, opts ...Option) *http.Client {
	if client == nil {
		panic("client must not be nil")
	}

	nextRoundtripper := client.Transport
	if nextRoundtripper == nil {
		nextRoundtripper = http.DefaultTransport
	}

	// set defaults
	retryRoundtripper := &RetryRoundtripper{
		Next:             nextRoundtripper,
		MaxRetryCount:    defaultMaxRetryCount,
		ShouldRetry:      defaultRetryPolicy,
		CalculateBackoff: defaultBackoffPolicy,
	}

	// overwrite defaults with user provided configuration
	for _, o := range opts {
		o(retryRoundtripper)
	}

	client.Transport = retryRoundtripper

	return client
}

// GetOriginalRoundtripper returns the original roundtripper that was embedded in the retry roundtripper.
func GetOriginalRoundtripper(client *http.Client) http.RoundTripper {
	if client == nil {
		panic("client must not be nil")
	}

	switch r := client.Transport.(type) {
	case *RetryRoundtripper:
		return r.Next
	default: // also catches Transport == nil
		return client.Transport
	}
}

// ReplaceOriginalRoundtripper replaces the original roundtripper that was embedded in the retry roundtripper
func ReplaceOriginalRoundtripper(client *http.Client, roundtripper http.RoundTripper) error {
	if client == nil {
		panic("client must not be nil")
	}

	switch r := client.Transport.(type) {
	case *RetryRoundtripper:
		r.Next = roundtripper
		return nil
	default:
		client.Transport = roundtripper
		return nil
	}
}

// GetOriginalTransport retrieves the original http.Transport that was mebedded in the retry roundtripper.
func GetOriginalTransport(client *http.Client) (*http.Transport, error) {
	if client == nil {
		panic("client must not be nil")
	}

	switch r := client.Transport.(type) {
	case *RetryRoundtripper:
		switch t := r.Next.(type) {
		case *http.Transport:
			return t, nil
		case nil:
			return nil, nil
		default:
			return nil, errors.New("embedded roundtripper is not of type *http.Transport")
		}
	case *http.Transport:
		return r, nil
	case nil:
		return nil, nil
	default:
		return nil, errors.New("roundtripper is not of type *http.Transport")
	}
}

// ModifyOriginalTransport allows to modify the original http.Transport that was embedded in the retry roundtipper.
func ModifyOriginalTransport(client *http.Client, f func(transport *http.Transport)) error {
	if client == nil {
		panic("client must not be nil")
	}

	switch r := client.Transport.(type) {
	case *http.Transport:
		f(r)
		return nil
	case *RetryRoundtripper:
		switch t := r.Next.(type) {
		case nil:
			return errors.New("embedded transport was nil")
		case *http.Transport:
			f(t)
			return nil
		default:
			return errors.New("embedded roundtripper is not of type *http.Transport")
		}
	case nil:
		return errors.New("transport was nil")
	default:
		return errors.New("transport is not of type *http.Transport")
	}
}
