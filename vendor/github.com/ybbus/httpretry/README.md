[![Go Report Card](https://goreportcard.com/badge/github.com/ybbus/httpretry)](https://goreportcard.com/report/github.com/ybbus/httpretry)
[![Go build](https://github.com/ybbus/httpretry/actions/workflows/go.yml/badge.svg)](https://github.com/ybbus/httpretry)
[![Codecov](https://codecov.io/github/ybbus/httpretry/branch/master/graph/badge.svg?token=ARYOQ8R1DT)](https://codecov.io/github/ybbus/httpretry)
[![GoDoc](https://godoc.org/github.com/ybbus/httpretry?status.svg)](https://godoc.org/github.com/ybbus/httpretry)
[![GitHub license](https://img.shields.io/github/license/mashape/apistatus.svg)]()

# httpRetry

Enriches the standard go http client with retry functionality using a wrapper around the Roundtripper interface.

The advantage of this library is that it makes use of the default http.Client.
This means you can provide it to any library that accepts the go standard http.Client.
This in turn gives you the possibility to add resilience to a lot of http based go libraries with just a single line of code.
Of course it can also be used as standalone http client in your own projects.

## Installation

```sh
go get -u github.com/ybbus/httpretry
```

## Quickstart

To get a standard http client with retry functionality:

```golang
client := httpretry.NewDefaultClient()
// use this as usual when working with http.Client
```

This single line of code returns a default http.Client that uses an exponential backoff and sends up to 5 retries if the request was not successful.
Requests will be retried if the error seems to be temporary or the requests returns a status code that may change over time (e.g. GetwayTimeout).

### Modify / customize the Roundtripper (http.Transport)

Since httpretry wraps the actual Roundtripper of the http.Client, you should not try to replace / modify the client.Transport field after creation.

You either configure the http.Client upfront and then "make" it retryable like in this code:

```golang
customHttpClient := &http.Client{}
customHttpClient.Transport = &http.Transport{...}

retryClient := httpretry.NewCustomClient(cumstomHttpClient)
```

or you use one of the available helper functions to gain access to the underlying Roundtripper / http.Transport:

```golang
// replaces the original roundtripper
httpretry.ReplaceOriginalRoundtripper(retryClient, myRoundTripper)

// modifies the embedded http.Transport by providing a function that receives the client.Transport as parameter
httpretry.ModifyOriginalTransport(retryClient, func(t *http.Transport) { t.TLSHandshakeTimeout = 5 * time.Second })

// returns the embedded Roundtripper
httpretry.GetOriginalRoundtripper(retryClient)

// returns the embedded Roundtripper as http.Transport if it is of that type
httpretry.GetOriginalTransport(retryClient)
```

### Customize retry settings

You may provide your own Backoff- and RetryPolicy.

```golang
client := httpretry.NewDefaultClient(
    // retry up to 5 times
    httpretry.WithMaxRetryCount(5),
    // retry on status >= 500, if err != nil, or if response was nil (status == 0)
    httpretry.WithRetryPolicy(func(statusCode int, err error) bool {
      return err != nil || statusCode >= 500 || statusCode == 0
    }),
    // every retry should wait one more second
    httpretry.WithBackoffPolicy(func(attemptNum int) time.Duration {
      return time.Duration(attemptNum+1) * 1 * time.Second
    }),
)
```
