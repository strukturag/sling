package sling

import (
	"testing"
)

func TestHTTP_newFailsForMalformedURLs(t *testing.T) {
	if _, err := newHTTP(":/", nil); err == nil {
		t.Error("No error returned for bad database url")
	}
}

func TestHTTP_newFailsForInvalidProtocols(t *testing.T) {
	if _, err := newHTTP("ftp://example.com", nil); err.Error() != "Only http and https are supported" {
		t.Errorf("Expected error to be '%v', but was '%v'", "Only http and https are supported", err)
	}
}

func TestHTTP_newEnsuresBaseURLHasATrailingSlash(t *testing.T) {
	http, err := newHTTP("http://example.com", nil)
	if err != nil {
		t.Fatalf("Error '%v' returned for valid url", err)
	}

	// HACK(lcooper): This isn't the best, figure out how to fix this,
	// or move it to an integration test or something.
	if expectedURL, actualURL := "http://example.com/", http.(*httpClient).URL.String(); expectedURL != actualURL {
		t.Errorf("Expected processed url to be %s, but was %s", expectedURL, actualURL)
	}
}
