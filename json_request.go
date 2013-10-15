package sling

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// JSON is a placeholder type for items which will be serialized to JSON.
type JSON interface{}

// Errorable is a marker interface which may be implemented by deserialized
// error responses to control their conversion into a returned error.
//
// This is useful if you use a single type for all responses and do not wish
// it to be an error directly, or if error returns require a custom mapping
// to error constants.
type Errorable interface {
	AsError() error
}

// JSONRequestBuilder instances allow the construction of a HTTP request
// whose response is a JSON document.
type JSONRequestBuilder interface {
	// Header sets an optional HTTP request header.
	Header(name, value string) JSONRequestBuilder

	// Body sets an optional object which will be serialized as JSON
	// to create the body of the HTTP request.
	Body(JSON) JSONRequestBuilder

	// Response sets an optional object to which any response will be
	// deserialized as JSON.
	Response(JSON) JSONRequestBuilder

	// Success sets an optional object to which successful responses
	// will be deserialized.
	Success(JSON) JSONRequestBuilder

	// Failure sets an optional object to which unsuccessful responses
	// will be deserialized.
	//
	// If the object implements Errorable, the error returned by AsError()
	// will be returned as the response.
	//
	// If the object implements error, it will be returned directly unless
	// it is also an Errorable.
	Failure(JSON) JSONRequestBuilder

	// StatusError sets the error return for responses with HTTP status
	// statusCode to err.
	//
	// Note that it is presently undefined in combination with an Errorable
	// response or when used for 1XX or 2XX statuses.
	StatusError(statusCode int, err error) JSONRequestBuilder

	// HTTPRequestable methods may be used to initiate a request/response cycle.
	HTTPRequestable
}

type jsonRequest struct {
	method, path           string
	body, success, failure JSON
	statusErrors           map[int]error
	headers http.Header
	*url.URL
}

// JSONRequest creates a new builder for a request with the
// given HTTP method and path.
//
// Note that while neither are currently validated, this is subject to change.
func JSONRequest(method, path string) JSONRequestBuilder {
	return &jsonRequest{
		method:       method,
		path:         path,
		statusErrors: make(map[int]error),
		headers: make(http.Header),
	}
}

func (request *jsonRequest) Header(name, value string) JSONRequestBuilder {
	request.headers.Add(name, value)
	return request
}

func (request *jsonRequest) Body(body JSON) JSONRequestBuilder {
	request.body = body
	return request
}

func (request *jsonRequest) Response(body JSON) JSONRequestBuilder {
	request.success = body
	request.failure = body
	return request
}

func (request *jsonRequest) Success(body JSON) JSONRequestBuilder {
	request.success = body
	return request
}

func (request *jsonRequest) Failure(body JSON) JSONRequestBuilder {
	request.failure = body
	return request
}

func (request *jsonRequest) StatusError(statusCode int, err error) JSONRequestBuilder {
	request.statusErrors[statusCode] = err
	return request
}

func (request *jsonRequest) HTTPRequest(baseURL *url.URL) (*http.Request, HTTPResponder, error) {
	requestedURL, _ := url.Parse(strings.TrimLeft(request.path, "/"))
	request.URL = baseURL.ResolveReference(requestedURL)

	body := new(bytes.Buffer)
	if request.body != nil {
		if err := json.NewEncoder(body).Encode(request.body); err != nil {
			return nil, nil, err
		}
	}

	req, err := http.NewRequest(request.method, request.URL.String(), body)
	if err != nil {
		return nil, nil, err
	}

	for name, values := range request.headers {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return req, request, nil
}

func (responder *jsonRequest) OnHTTPResponse(res *http.Response) error {
	decoder := json.NewDecoder(res.Body)
	if res.StatusCode < http.StatusBadRequest {
		if responder.success != nil {
			return decoder.Decode(responder.success)
		}
		return nil
	} else {
		err := fmt.Errorf("request %s %s as JSON returned status %d", responder.method, responder.URL, res.StatusCode)

		// TODO(lcooper): We should also decode the failure body if provided.
		// Unclear what should be returned if there's both a StatusError
		// and the response is Errorable.
		// TODO(lcooper): It should be possible to register errors
		// for 1XX and 2XX statuses.
		if err, ok := responder.statusErrors[res.StatusCode]; ok {
			return err
		}

		if responder.failure == nil {
			return err
		}

		if err := decoder.Decode(responder.failure); err != nil {
			return err
		}

		switch v := responder.failure.(type) {
		case Errorable:
			return v.AsError()
		case error:
			return v
		default:
			return err
		}
	}
}
