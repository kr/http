/*

Package limit provides an http RoundTripper for limiting
the concurrency of client HTTP requests.

*/
package limit

import (
	"errors"
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

	cmu      sync.Mutex // protects the following
	canceler map[*http.Request]func()
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

	ready := make(chan struct{})
	cancel := make(chan struct{})
	t.setCanceler(r, func() { close(cancel) })
	go func() {
		l.Lock()
		select {
		case ready <- struct{}{}:
		case <-cancel:
			l.Unlock()
		}
	}()
	select {
	case <-ready:
		t.setCanceler(r, nil)
	case <-cancel:
		return nil, errors.New("canceled")
	}

	defer l.Unlock()
	return t.transport().RoundTrip(r)
}

// CancelRequest cancels a request r
// by abandoning it if it is still waiting to execute,
// and by calling CancelRequest
// on the underlying RoundTripper.
func (t *Transport) CancelRequest(r *http.Request) {
	cancel := t.setCanceler(r, nil)
	if cancel != nil {
		cancel()
	}
	type canceler interface {
		CancelRequest(*http.Request)
	}
	if c, ok := t.transport().(canceler); ok {
		c.CancelRequest(r)
	}
}

// returns the old canceler
func (t *Transport) setCanceler(r *http.Request, f func()) func() {
	t.cmu.Lock()
	defer t.cmu.Unlock()
	if t.canceler == nil {
		t.canceler = make(map[*http.Request]func())
	}
	old := t.canceler[r]
	if f != nil {
		t.canceler[r] = f
	} else {
		delete(t.canceler, r)
	}
	return old
}

// NewTransportByHost returns a Transport that limits
// the number of concurrent requests to each host.
// It uses field Host from the request URL,
// not from the Request struct itself.
func NewTransportByHost(maxReqsPerHost int) *Transport {
	host := func(r *http.Request) interface{} {
		return r.URL.Host
	}
	return &Transport{Locker: By(host, maxReqsPerHost)}
}

// NewTransport returns a Transport that limits
// the number of concurrent requests.
func NewTransport(maxReqs int) *Transport {
	global := func(r *http.Request) interface{} {
		return 0
	}
	return &Transport{Locker: By(global, maxReqs)}
}
