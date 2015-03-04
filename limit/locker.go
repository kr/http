package limit

import (
	"net/http"
	"sync"
)

// By returns a function that produces Lockers.
// The returned function uses f to map each request
// to a key value. For each distinct key value,
// it produces a Locker that acts as a counting semaphore,
// allowing n concurrent holders of the lock.
func By(f func(*http.Request) interface{}, n int) func(*http.Request) sync.Locker {
	t := &tab{f: f, n: n, m: map[interface{}]*sem{}}
	return t.locker
}

type tab struct {
	n int
	f func(*http.Request) interface{}

	mu sync.Mutex // protects the following
	m  map[interface{}]*sem
}

func (t *tab) locker(r *http.Request) sync.Locker {
	t.mu.Lock()
	defer t.mu.Unlock()
	k := t.f(r)
	s := t.m[k]
	if s == nil {
		s = &sem{k: k, v: t.n}
		s.c.L = &s.m
		t.m[k] = s
	}
	s.q++
	return s
}

type sem struct {
	k interface{}

	c sync.Cond
	m sync.Mutex // protects the following
	v int

	t *tab // t.mu protects the following
	q int  // # of goroutines waiting to acquire lock
}

func (s *sem) P() {
	s.c.L.Lock()
	defer s.c.L.Unlock()
	for s.v == 0 {
		s.c.Wait()
	}
	s.v--

	s.t.mu.Lock()
	defer s.t.mu.Unlock()
	s.q--
}

func (s *sem) V() {
	s.c.L.Lock()
	defer s.c.L.Unlock()
	defer s.c.Signal()
	s.v++

	s.t.mu.Lock()
	defer s.t.mu.Unlock()
	if s.v == s.t.n && s.q == 0 {
		delete(s.t.m, s.k)
	}
}

// for sync.Locker
func (s *sem) Lock()   { s.P() }
func (s *sem) Unlock() { s.V() }
