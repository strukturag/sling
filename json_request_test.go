package sling

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func newResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       ioutil.NopCloser(strings.NewReader(body)),
	}
}

type fakeHTTP struct {
	request  *http.Request
	response *http.Response
	error
}

func (fake *fakeHTTP) Do(requestable HTTPRequestable) error {
	if fake.error != nil {
		return fake.error
	}

	req, responder, err := requestable.HTTPRequest(requestURL)
	if err != nil {
		return err
	}

	res := fake.response
	if err != nil {
		return err
	}

	fake.request = req

	return responder.OnHTTPResponse(res)
}

func (fake *fakeHTTP) assertRequestHeader(t *testing.T, name, value string) {
	if fake.request == nil {
		t.Fatal("No http request was made")
	}

	if header := fake.request.Header.Get(name); header != value {
		t.Errorf("Expected request to have value \"%s\" for the \"%s\" header, but was \"%s\"", value, name, header)
	}
}

func (fake *fakeHTTP) assertRequestContentType(t *testing.T, contentType string) {
	fake.assertRequestHeader(t, "Content-Type", contentType)
}

func (fake *fakeHTTP) assertRequestAccepts(t *testing.T, contentType string) {
	fake.assertRequestHeader(t, "Accept", contentType)
}

var requestURL, _ = url.Parse("http://example.com/doc/")

func newTestJson() (HTTP, *fakeHTTP) {
	httpClient := &fakeHTTP{}
	return httpClient, httpClient
}

func TestJson_RequestUsesTheProvidedMethod(t *testing.T) {
	method := "OPTIONS"
	jsonClient, httpClient := newTestJson()
	httpClient.response = newResponse(http.StatusOK, "{}")

	if err := jsonClient.Do(JSONRequest(method, "")); err != httpClient.error {
		t.Fatalf("Unexpected error '%v' making request", err)
	}

	if actualMethod := httpClient.request.Method; method != actualMethod {
		t.Errorf("Expected request method to be %s, but was %s", method, actualMethod)
	}
}

func TestJson_RequestUsesThePathRelatativeToTheBaseURL(t *testing.T) {
	path := "/some/path"
	jsonClient, httpClient := newTestJson()
	httpClient.response = newResponse(http.StatusOK, "{}")

	if err := jsonClient.Do(JSONRequest("", path)); err != httpClient.error {
		t.Fatalf("Unexpected error '%v' making request", err)
	}

	expectedPath, actualPath := strings.TrimRight(requestURL.Path, "/")+path, httpClient.request.URL.Path
	if expectedPath != actualPath {
		t.Errorf("Expected request path to be %s, but was %s", expectedPath, actualPath)
	}
}

func TestJson_RequestSuppliesAppropriateHeaders(t *testing.T) {
	jsonClient, httpClient := newTestJson()
	httpClient.response = newResponse(http.StatusOK, "{}")
	if err := jsonClient.Do(JSONRequest("", "")); err != httpClient.error {
		t.Fatalf("Unexpected error '%v' making request", err)
	}

	httpClient.assertRequestContentType(t, "application/json")
	httpClient.assertRequestAccepts(t, "application/json")
	//	httpClient.assertRequestHeader(t, "Connection", "keep-alive")
}

func TestJson_RequestEncodesRequestAsJSON(t *testing.T) {
	doc := struct {
		Field string `json:"field"`
	}{
		Field: "value",
	}
	jsonClient, httpClient := newTestJson()
	httpClient.response = newResponse(http.StatusOK, `{}`)
	err := jsonClient.Do(JSONRequest("", "").Body(doc))

	if err != nil {
		t.Fatalf("Got error %v when submitting request", err)
	}

	if httpClient.request.Body == nil {
		t.Fatal("No request body was provided")
	}

	requestJSON := make(map[string]string)
	decoder := json.NewDecoder(httpClient.request.Body)
	if err := decoder.Decode(&requestJSON); err != nil {
		t.Fatalf("Failed to unmarshal request JSON: %v", err)
	}

	if requestJSON["field"] != doc.Field {
		t.Errorf("Request json %v did not contain document properties", requestJSON)
	}
}

func TestJson_RequestDecodesTheResponseAsJSON(t *testing.T) {
	responseJSON := `{"Foo": 56}`
	jsonClient, httpClient := newTestJson()
	httpClient.response = newResponse(http.StatusOK, responseJSON)

	responseData := struct {
		Foo int
	}{}

	err := jsonClient.Do(JSONRequest("", "").Success(&responseData))

	if err != nil {
		t.Errorf("Expected request to produce error %v, but got %v", nil, err)
	}

	if responseData.Foo != 56 {
		t.Errorf("Expected response JSON %s to have been decoded into %v", responseJSON, responseData)
	}
}

func TestJson_RequestFails(t *testing.T) {
	jsonClient, httpClient := newTestJson()
	httpClient.error = errors.New("Failed")
	if err := jsonClient.Do(JSONRequest("", "")); err != httpClient.error {
		t.Errorf("Expected request to produce error %v, but got %v", httpClient.error, err)
	}
}

func TestJson_RequestDefaultErrorIncludesRequestAndResponseInformation(t *testing.T) {
	method := "OPTIONS"
	path := "/api/v1/to/madness"
	baseURL, _ := url.Parse("http://no.such.domain.xxx")
	statusCode := 500
	requestBuilder := JSONRequest(method, path)
	_, responder, err := requestBuilder.HTTPRequest(baseURL)
	if err != nil {
		t.Fatalf("Unexpected error creating HTTP request: %v", err)
	}

	err = responder.OnHTTPResponse(newResponse(statusCode, ""))

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

	if strings.Index(msg, baseURL.String()) == -1 {
		t.Errorf("Expected error '%s' to contain base url '%s'", msg, baseURL)
	}

	if strings.Index(msg, fmt.Sprintf("%d", statusCode)) == -1 {
		t.Errorf("Expected error '%s' to contain HTTP response status '%d'", msg, statusCode)
	}
}

func TestJson_RequestReceivesRegisteredError(t *testing.T) {
	jsonClient, httpClient := newTestJson()
	httpClient.response = newResponse(499, `{[`)
	expectedError := errors.New("HTTP 499")
	err := jsonClient.Do(JSONRequest("", "").StatusError(499, expectedError))
	if err != expectedError {
		t.Errorf("Expected registered error '%v' to be returned for error status, but was '%v'", expectedError, err)
	}
}

func TestJson_RequestRetrievesBadJSON(t *testing.T) {
	jsonClient, httpClient := newTestJson()
	statuses := []int{http.StatusOK, http.StatusNotFound}

	for _, status := range statuses {
		httpClient.response = newResponse(status, `{[`)

		var responseData string
		err := jsonClient.Do(JSONRequest("", "").Response(responseData))
		if _, ok := err.(*json.SyntaxError); !ok {
			t.Errorf("Expected a JSON syntax error for a malformed response with HTTP status %d, but was %v", status, err)
		}
	}
}
