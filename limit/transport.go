/*

Package limit provides an http RoundTripper for limiting
the concurrency of client HTTP requests.

*/
package limit

import (
	"net/http"
	"sync"
)

// Transport provides an http RoundTripper to limit the
// concurrency of client requests according to arbitrary criteria
// that can depend on information in the Request.
type Transport struct {
	// The locker returned by Locker will be locked
	// for the duration of each request, from before
	// the request is sent until the response header
	// is read.
	// Locker must be safe to call concurrently
	// in multiple goroutines.
	Locker func(*http.Request) sync.Locker

	// The transport used to perform requests.
	// If nil, http.DefaultTransport is used.
	Transport http.RoundTripper
}

func (t *Transport) transport() http.RoundTripper {
	if t.Transport == nil {
		return http.DefaultTransport
	}
	return t.Transport
}

// RoundTrip satisfies http.RoundTripper.
func (t *Transport) RoundTrip(r *http.Request) (*http.Response, error) {
	l := t.Locker(r)
	l.Lock()
	defer l.Unlock()
	return t.transport().RoundTrip(r)
}

// CancelRequest calls CancelRequest on the underlying
// RoundTripper if possible.
func (t *Transport) CancelRequest(r *http.Request) {
	type canceler interface {
		CancelRequest(*http.Request)
	}
	if c, ok := t.transport().(canceler); ok {
		c.CancelRequest(r)
	}
}

// NewTransportByHost returns a Transport that limits
// the number of concurrent requests to each host.
// It uses field Host from the request URL,
// not from the Request struct itself.
func NewTransportByHost(maxReqsPerHost int) *Transport {
	t := &tab{n: maxReqsPerHost, f: func(r *http.Request) interface{} {
		return r.URL.Host
	}}
	return &Transport{Locker: t.locker}
}

// NewTransport returns a Transport that limits
// the number of concurrent requests.
func NewTransport(maxReqs int) *Transport {
	t := &tab{n: maxReqs, f: func(r *http.Request) interface{} {
		type globalKey struct{}
		return globalKey{}
	}}
	return &Transport{Locker: t.locker}
}
