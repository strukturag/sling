package slingmock

import (
	"golang.struktur.de/sling"
	"net/http"
	"net/url"
)

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
