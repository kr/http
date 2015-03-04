package limit_test

import (
	"net/http"

	"github.com/kr/http/limit"
)

// Allow at most 10 concurrent requests to each URL.
func ExampleBy() {
	url := func(r *http.Request) interface{} {
		return r.URL.String()
	}
	http.DefaultTransport = &limit.Transport{
		Locker:    limit.By(url, 10),
		Transport: http.DefaultTransport,
	}
}
