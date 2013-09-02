package slingmock

import (
	"golang.struktur.de/sling"
	"golang.struktur.de/sling/httpmock"
	"net/http"
	"net/url"
	"testing"
)

type fakeConnectionPool struct {
	transport *httpmock.Transport
}

// NewConnectionPool returns a connection pool which uses returned mock
// Transport.
func NewConnectionPool(t *testing.T) (sling.ConnectionPool, *httpmock.Transport) {
	transport := httpmock.NewTransport(t)
	return &fakeConnectionPool{
		transport: transport,
	}, transport
}

func (fake *fakeConnectionPool) HTTP(baseURL string) (sling.HTTP, error) {
	url, _ := url.Parse(baseURL)
	return &fakeHTTP{
		Client: &http.Client{
			Transport: fake.transport,
		},
		URL: url,
	}, nil
}

// NewHTTP creates a HTTP client with it's own ConnectionPool which uses the
// returned mock Transport to make requests.
func NewHTTP(t *testing.T, baseURL string) (sling.HTTP, *httpmock.Transport) {
	pool, transport := NewConnectionPool(t)
	http, _ := pool.HTTP(baseURL)
	return http, transport
}
