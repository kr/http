package limit

import (
	"net/http"
	"testing"
)

func TestSanity(t *testing.T) {
	f := func(r *http.Request) interface{} { return 0 }
	tr := &Transport{
		Locker:    By(f, 10),
		Transport: dummyTransport{},
	}
	resp, err := tr.RoundTrip(nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("StatusCode = %d want 200", resp.StatusCode)
	}
}

type dummyTransport struct{}

func (dummyTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
	}
	return resp, nil
}
