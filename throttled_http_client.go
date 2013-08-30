package sling

import (
	"net/http"
)

type netHTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type throttledHTTPClient struct {
	semaphore
	netHTTPClient
}

func newThrottledHTTPClient(client netHTTPClient, maxRequests int) netHTTPClient {
	return &throttledHTTPClient{
		semaphore:     make(semaphore, maxRequests),
		netHTTPClient: client,
	}
}

func (throttledClient *throttledHTTPClient) Do(req *http.Request) (*http.Response, error) {
	throttledClient.Lock()
	defer throttledClient.Unlock()
	return throttledClient.netHTTPClient.Do(req)
}

type nothing struct{}
type semaphore chan nothing

func (s semaphore) Lock() {
	n := nothing{}
	s <- n
}

func (s semaphore) Unlock() {
	<-s
}
