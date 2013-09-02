// Package httpmock provides utilities for mocking the Go HTTP client's request/response cycle.
package httpmock

import (
	"encoding/json"
	"errors"
	"golang.struktur.de/sling"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// DefaultURLString provides a suitable non-existant but valid URL
// for use when the URL may be parsed.
var DefaultURLString = "http://httpmock.default.url.tld"

// Transport implements http.RoundTripper, allowing it to replace the default
// Transport of http.Client. It also provides a variety of methods for configuring
// the response provided when a request is made and assertions about the properties
// of the recieved request.
//
// Note that Transport is only designed to handle a single request/response cycle.
type Transport struct {
	request  *http.Request
	response http.Response
	error
	t *testing.T
}

type fakeHTTP struct {
	*http.Client
	*url.URL
}

func (fake *fakeHTTP) Do(requestable sling.HTTPRequestable) error {
	req, responder, err := requestable.HTTPRequest(fake.URL)
	if err != nil {
		return err
	}

	res, err := fake.Client.Do(req)
	if err != nil {
		return err
	}
	res.Body.Close()

	return responder.OnHTTPResponse(res)
}

type fakeConnectionPool struct {
	transport *Transport
}

// NewConnectionPool returns a connection pool which uses returned mock
// Transport.
func NewConnectionPool(t *testing.T) (sling.ConnectionPool, *Transport) {
	transport := NewTransport(t)
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

// NewTransport creates a new Transport instance which reports assertion errors
// to the given testing.T.
//
// All request assertions will fail the test with t.Fatal() if no HTTP request
// was made, otherwise all reporting is at the t.Error() level.
func NewTransport(t *testing.T) *Transport {
	return &Transport{t: t}
}

// RoundTrip implements http.RoundTripper using the configured response data
// and stores the provided request for later assertion usage.
func (fake *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	fake.request = req
	if fake.error != nil {
		return nil, fake.error
	}
	if fake.response.Body == nil {
		fake.SetResponseBody("")
	}
	return &fake.response, nil
}

// SetResponseStatusCode sets the HTTP status code of the response to statusCode.
func (fake *Transport) SetResponseStatusCode(statusCode int) {
	fake.response.StatusCode = statusCode
}

// SetResponseBody sets the response body to a reader against body.
func (fake *Transport) SetResponseBody(body string) {
	fake.response.Body = newClosableStringReader(body)
}

// SetResponseBodyJSON marshals data as JSON and sets the result as the response body.
//
// Note that the test will be failed with t.Fatal() if marshalling fails.
func (fake *Transport) SetResponseBodyJSON(data interface{}) {
	if result, err := json.Marshal(data); err != nil {
		fake.t.Fatalf("Failed to set response body JSON: %v", err)
	} else {
		fake.SetResponseBody(string(result))
	}
}

// SetResponseBodyInvalidJSON sets the response body to a string which all JSON parsers will reject.
func (fake *Transport) SetResponseBodyInvalidJSON() {
	fake.SetResponseBody("{[")
}

// SetResponseError forces an error return from the the Transport to simulate a connection error.
func (fake *Transport) SetResponseError() {
	if fake.error == nil {
		fake.error = errors.New("Fake HTTP transport error")
	}
}

func (fake *Transport) assertRequestMade() {
	if fake.request == nil {
		fake.t.Fatalf("No HTTP requests were made.")
	}
}

// AssertRequestMethod tests the the HTTP request used method as it's HTTP method.
func (fake *Transport) AssertRequestMethod(method string) {
	fake.assertRequestMade()
	if fake.request.Method != method {
		fake.t.Errorf("Expected HTTP request to be made with method %s, but was %s", method, fake.request.Method)
	}
}

// AssertRequestProtocolIsHTTP tests that the requested URL used the 'http' scheme.
func (fake *Transport) AssertRequestProtocolIsHTTP() {
	if expectedProto, proto := "http", fake.request.URL.Scheme; expectedProto != proto {
		fake.t.Errorf("Expected HTTP request to have been made with protocol '%s', but was '%s'", expectedProto, proto)
	}
}

// AssertRequestHost tests that the requested url has the given host.
func (fake *Transport) AssertRequestHost(host string) {
	if actualHost := fake.request.URL.Host; actualHost != host {
		fake.t.Errorf("Expected HTTP request to have been made to host '%s', but was '%s'", host, actualHost)
	}
}

// AssertRequestPath tests that the requested url has the given path.
func (fake *Transport) AssertRequestPath(path string) {
	fake.assertRequestMade()
	requestUrl := fake.request.URL
	if requestUrl.Path != path {
		fake.t.Errorf("Expected HTTP request to have path '%s', but was '%s'", path, requestUrl.Path)
	}
}

// AssertRequestContentType tests that the request has a Content-Type header equal to contentType.
func (fake *Transport) AssertRequestContentType(contentType string) {
	fake.assertRequestMade()

	if requestContentType := fake.request.Header.Get("Content-Type"); requestContentType != contentType {
		fake.t.Errorf("Expected HTTP request to be made with content type '%s', but was '%s'", contentType, requestContentType)
	}
}

// AssertRequestAccepts asserts that the Accept header of the request contains contentType.
//
// Note that presently 'contains' means 'equals', this will change in the future.
func (fake *Transport) AssertRequestAccepts(contentType string) {
	fake.assertRequestMade()

	// TODO(lcooper): Actually parse rather then just match.
	if requestAccepts := fake.request.Header.Get("Accept"); requestAccepts != contentType {
		fake.t.Errorf("Expected HTTP request to accept a response of content type '%s', but accepts '%s' instead", contentType, requestAccepts)
	}
}

// AssertRequestBodyJSON tests that a request body was provided, and if so, creates a JSON
// decoder using it, and passes it to cb for further tests.
//
// Presently no validation of the JSON is done, this is likely to change in the future.
func (fake *Transport) AssertRequestBodyJSON(cb func(*json.Decoder)) {
	if fake.request.Body == nil {
		fake.t.Error("HTTP request was made without a body")
	} else {
		// TODO(lcooper): Verify that the JSON is valid.
		cb(json.NewDecoder(fake.request.Body))
	}
}

// AssertResponseBodyClosed tests that the function under test closed the response body reader.
func (fake *Transport) AssertResponseBodyClosed() {
	if src, ok := fake.response.Body.(*stringReaderCloser); ok {
		if !src.closed {
			fake.t.Error("HTTP response body was not closed")
		}
	} else {
		fake.t.Log("Cannot determine response body closed status, skipping")
	}
}

type stringReaderCloser struct {
	*strings.Reader
	closed bool
}

func newClosableStringReader(s string) io.ReadCloser {
	return &stringReaderCloser{Reader: strings.NewReader(s)}
}

func (src *stringReaderCloser) Close() error {
	src.closed = true
	return nil
}
