package sling_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"golang.struktur.de/sling"
	"golang.struktur.de/sling/httpmock"
	"golang.struktur.de/sling/slingmock"
	"net/url"
	"strings"
	"testing"
)

var requestURL, _ = url.Parse("http://example.com/doc/")

type errorResponse  struct {
	Message string
}

func (err *errorResponse) Error() string {
	return err.Message
}

func newTestHTTP(t *testing.T) (sling.HTTP, *httpmock.Transport) {
	return slingmock.NewHTTP(t, requestURL.String())
}

func TestJson_RequestUsesTheProvidedMethod(t *testing.T) {
	method := "OPTIONS"
	http, transport := newTestHTTP(t)
	transport.SetResponseStatusOK()
	transport.SetResponseBodyValidJSON()

	if err := http.Do(sling.JSONRequest(method, "")); err != transport.error {
		t.Fatalf("Unexpected error '%v' making request", err)
	}

	transport.AssertRequestMethod(method)
}

func TestJson_RequestUsesThePathRelatativeToTheBaseURL(t *testing.T) {
	path := "/some/path"
	http, transport := newTestHTTP(t)
	transport.SetResponseStatusOK()
	transport.SetResponseBodyValidJSON()

	if err := http.Do(sling.JSONRequest("", path)); err != transport.error {
		t.Fatalf("Unexpected error '%v' making request", err)
	}

	expectedPath := strings.TrimRight(requestURL.Path, "/") + path
	transport.AssertRequestPath(expectedPath)
}

func TestJson_RequestUsesProvidedHeaders(t *testing.T) {
	request := sling.JSONRequest("", "").
		Header("X-Foo-Status", "bar")
	http, transport := newTestHTTP(t)
	transport.SetResponseStatusOK()
	transport.SetResponseBodyValidJSON()

	if err := http.Do(request); err != transport.error {
		t.Fatalf("Unexpected error '%v' making request", err)
	}

	transport.AssertRequestHeader("X-Foo-Status", "bar")
}

func TestJson_RequestSuppliesAppropriateHeaders(t *testing.T) {
	http, transport := newTestHTTP(t)
	transport.SetResponseStatusOK()
	transport.SetResponseBodyValidJSON()
	if err := http.Do(sling.JSONRequest("", "")); err != transport.error {
		t.Fatalf("Unexpected error '%v' making request", err)
	}

	transport.AssertRequestContentType("application/json")
	transport.AssertRequestAccepts("application/json")
}

func TestJson_RequestEncodesRequestAsJSON(t *testing.T) {
	doc := struct {
		Field string `json:"field"`
	}{
		Field: "value",
	}
	http, transport := newTestHTTP(t)
	transport.SetResponseStatusOK()
	transport.SetResponseBodyValidJSON()
	err := http.Do(sling.JSONRequest("", "").Body(doc))

	if err != nil {
		t.Fatalf("Got error %v when submitting request", err)
	}

	transport.AssertRequestBodyJSON(func(decoder *json.Decoder) {
		requestJSON := make(map[string]string)
		if err := decoder.Decode(&requestJSON); err != nil {
			t.Fatalf("Failed to unmarshal request JSON: %v", err)
		}

		if requestJSON["field"] != doc.Field {
			t.Errorf("Request json %v did not contain document properties", requestJSON)
		}
	})
}

func TestJson_RequestDecodesTheResponseAsJSON(t *testing.T) {
	responseJSON := `{"Foo": 56}`
	http, transport := newTestHTTP(t)
	transport.SetResponseStatusOK()
	transport.SetResponseBody(responseJSON)
	responseData := struct {
		Foo int
	}{}

	err := http.Do(sling.JSONRequest("", "").Success(&responseData))

	if err != nil {
		t.Errorf("Expected request to produce error %v, but got %v", nil, err)
	}

	if responseData.Foo != 56 {
		t.Errorf("Expected response JSON %s to have been decoded into %v", responseJSON, responseData)
	}
}

func TestJson_RequestFails(t *testing.T) {
	http, transport := newTestHTTP(t)
	transport.error = errors.New("Failed")
	if err := http.Do(sling.JSONRequest("", "")); err != transport.error {
		t.Errorf("Expected request to produce error %v, but got %v", transport.error, err)
	}
}

func TestJson_RequestDefaultErrorIncludesRequestAndResponseInformation(t *testing.T) {
	method := "OPTIONS"
	path := "/api/v1/to/madness"
	statusCode := 500

	http, transport := newTestHTTP(t)
	transport.SetResponseStatusCode(statusCode)

	err := http.Do(sling.JSONRequest(method, path))
	if err == nil {
		t.Fatal("No error returned for unsuccessful response")
	}

	msg := err.Error()
	if strings.Index(msg, method) == -1 {
		t.Errorf("Expected error '%s' to contain HTTP method '%s'", msg, method)
	}

	if strings.Index(msg, path) == -1 {
		t.Errorf("Expected error '%s' to contain path '%s'", msg, path)
	}

	if strings.Index(msg, requestURL.String()) == -1 {
		t.Errorf("Expected error '%s' to contain base url '%s'", msg, requestURL)
	}

	if strings.Index(msg, fmt.Sprintf("%d", statusCode)) == -1 {
		t.Errorf("Expected error '%s' to contain HTTP response status '%d'", msg, statusCode)
	}
}

func TestJson_RequestReceivesRegisteredError(t *testing.T) {
	http, transport := newTestHTTP(t)
	transport.SetResponseStatusCode(499)
	transport.SetResponseBodyInvalidJSON()
	expectedError := errors.New("HTTP 499")
	err := http.Do(sling.JSONRequest("", "").StatusError(499, expectedError))
	if err != expectedError {
		t.Errorf("Expected registered error '%v' to be returned for error status, but was '%v'", expectedError, err)
	}
}

func TestJson_RequestFailsWithAnErrorResponseBodySet(t *testing.T) {
	message := "Error Message"
	http, transport := newTestHTTP(t)
	transport.SetResponseStatusCode(500)
	transport.SetResponseBodyJSON(&errorResponse{message})
	
	errorResponse := &errorResponse{}
	err := http.Do(sling.JSONRequest("", "").Failure(errorResponse))

	if err.Error() != message {
		t.Errorf("Expected error message '%s' for return of type error, but was '%v'", message, err)
	}
}

func TestJson_RequestRetrievesBadJSON(t *testing.T) {
	http, transport := newTestHTTP(t)

	for _, status := range []int{200, 404} {
		transport.SetResponseStatusCode(status)
		transport.SetResponseBodyInvalidJSON()
		var responseData string
		err := http.Do(sling.JSONRequest("", "").Response(responseData))
		if _, ok := err.(*json.SyntaxError); !ok {
			t.Errorf("Expected a JSON syntax error for a malformed response with HTTP status %d, but was %v", status, err)
		}
	}
}
