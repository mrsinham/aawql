package oauth2

import (
	"context"
	"net/http"

	"golang.org/x/oauth2"
)

type transport struct {
	underlyingTransport http.RoundTripper
	developerToken      string
}

// RoundTrip implements the net/http.RoundTripper interface.
func (t *transport) RoundTrip(r *http.Request) (*http.Response, error) {
	r2 := cloneRequest(r)
	r2.Header.Add("developerToken", t.developerToken)
	return t.underlyingTransport.RoundTrip(r2)
}

func cloneRequest(r *http.Request) *http.Request {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
	}
	return r2
}

// NewClient returns a new instance of http.Client.
func NewClient(ctx context.Context, source oauth2.TokenSource, developerToken string) *http.Client {
	c := oauth2.NewClient(ctx, source)
	c.Transport = &transport{underlyingTransport: c.Transport, developerToken: developerToken}
	return c
}
