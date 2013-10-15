package sling

import (
	"net/http"
	"sync"
	"testing"
	"time"
)

type fakeSleepingHttpClient struct {
	requests []*http.Request
}

func (fake *fakeSleepingHttpClient) Do(req *http.Request) (*http.Response, error) {
	// NOTE(lcooper): Totally not an abuse of the ContentLength field ;)
	time.Sleep(time.Duration(req.ContentLength) * 1000)
	if fake.requests == nil {
		fake.requests = make([]*http.Request, 0, 1)
	}
	fake.requests = append(fake.requests, req)

	return nil, nil
}

func TestThrottledHTTPClient_Do(t *testing.T) {
	queries := []struct {
		request *http.Request
	}{
		{&http.Request{ContentLength: 100}},
		{&http.Request{ContentLength: 100}},
		{&http.Request{ContentLength: 0}},
	}

	wrappedClient := &fakeSleepingHttpClient{}
	client := newThrottledHTTPClient(wrappedClient, len(queries)-1)

	queryExecutors := &sync.WaitGroup{}
	for _, query := range queries {
		queryExecutors.Add(1)
		go func(request *http.Request) {
			client.Do(request)
			queryExecutors.Done()
		}(query.request)
	}
	queryExecutors.Wait()

	expected, executed := len(queries), len(wrappedClient.requests)
	if expected != executed {
		t.Fatalf("Expected %d requests to have been executed, but %d were executed", expected, executed)
	}

	expectedId, actualId := queries[expected-1].request, wrappedClient.requests[executed-1]
	if expectedId != actualId {
		t.Errorf("Expected %p to be the last request executed, but was %p", expectedId, actualId)
	}
}
