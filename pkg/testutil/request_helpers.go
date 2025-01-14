// Copyright 2019 DeepMap, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package testutil

// This is a set of fluent request builders for tests, which help us to
// simplify constructing and unmarshaling test objects. For example, to post
// a body and return a response, you would do something like:
//
//   var body RequestBody
//   var response ResponseBody
//   t is *testing.T, from a unit test
//   e is *echo.Echo
//   response := NewRequest().Post("/path").WithJsonBody(body).Go(t, e)
//   err := response.UnmarshalBodyToObject(&response)
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func NewRequest() *RequestBuilder {
	return &RequestBuilder{
		Headers: make(map[string]string),
	}
}

// This structure caches request settings as we build up the request.
type RequestBuilder struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    []byte
	Error   error
	Cookies []*http.Cookie
}

// Path operations
func (r *RequestBuilder) WithMethod(method string, path string) *RequestBuilder {
	r.Method = method
	r.Path = path
	return r
}

func (r *RequestBuilder) Get(path string) *RequestBuilder {
	return r.WithMethod("GET", path)
}

func (r *RequestBuilder) Post(path string) *RequestBuilder {
	return r.WithMethod("POST", path)
}

func (r *RequestBuilder) Put(path string) *RequestBuilder {
	return r.WithMethod("PUT", path)
}

func (r *RequestBuilder) Delete(path string) *RequestBuilder {
	return r.WithMethod("DELETE", path)
}

// Header operations
func (r *RequestBuilder) WithHeader(header, value string) *RequestBuilder {
	r.Headers[header] = value
	return r
}

func (r *RequestBuilder) WithContentType(value string) *RequestBuilder {
	return r.WithHeader("Content-Type", value)
}

func (r *RequestBuilder) WithJsonContentType() *RequestBuilder {
	return r.WithContentType("application/json")
}

func (r *RequestBuilder) WithAccept(value string) *RequestBuilder {
	return r.WithHeader("Accept", value)
}

func (r *RequestBuilder) WithAcceptJson() *RequestBuilder {
	return r.WithAccept("application/json")
}

func (r *RequestBuilder) WithAcceptScim() *RequestBuilder {
	return r.WithAccept("application/scim+json")
}

// Request body operations

func (r *RequestBuilder) WithBody(body []byte) *RequestBuilder {
	r.Body = body
	return r
}

// This function takes an object as input, marshals it to JSON, and sends it
// as the body with Content-Type: application/json
func (r *RequestBuilder) WithJsonBody(obj interface{}) *RequestBuilder {
	var err error
	r.Body, err = json.Marshal(obj)
	if err != nil {
		r.Error = fmt.Errorf("failed to marshal json object: %s", err)
	}
	return r.WithJsonContentType()
}

// Cookie operations
func (r *RequestBuilder) WithCookie(c *http.Cookie) *RequestBuilder {
	r.Cookies = append(r.Cookies, c)
	return r
}

func (r *RequestBuilder) WithCookieNameValue(name, value string) *RequestBuilder {
	return r.WithCookie(&http.Cookie{Name: name, Value: value})
}

// This function performs the request, it takes a pointer to a testing context
// to print messages, and a pointer to an echo context for request handling.
func (r *RequestBuilder) Go(t *testing.T, e *echo.Echo) *CompletedRequest {
	if r.Error != nil {
		// Fail the test if we had an error
		t.Errorf("error constructing request: %s", r.Error)
		return nil
	}
	var bodyReader io.Reader
	if r.Body != nil {
		bodyReader = bytes.NewReader(r.Body)
	}

	req := httptest.NewRequest(r.Method, r.Path, bodyReader)
	for h, v := range r.Headers {
		req.Header.Add(h, v)
	}
	for _, c := range r.Cookies {
		req.AddCookie(c)
	}

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	return &CompletedRequest{
		Recorder: rec,
	}
}

// This is the result of calling Go() on the request builder. We're wrapping the
// ResponseRecorder with some nice helper functions.
type CompletedRequest struct {
	Recorder *httptest.ResponseRecorder
}

// This function takes a destination object as input, and unmarshals the object
// in the response based on the Content-Type header.
func (c *CompletedRequest) UnmarshalBodyToObject(obj interface{}) error {
	ctype := c.Recorder.Header().Get("Content-Type")

	// Content type can have an annotation after ;
	contentParts := strings.Split(ctype, ";")

	switch strings.TrimSpace(contentParts[0]) {
	case "application/json":
		return json.Unmarshal(c.Recorder.Body.Bytes(), obj)
	case "application/scim+json":
		return json.Unmarshal(c.Recorder.Body.Bytes(), obj)
	default:
		return fmt.Errorf("no Content-Type on response")
	}
}

// This function assumes that the response contains JSON and unmarshals it
// into the specified object.
func (c *CompletedRequest) UnmarshalJsonToObject(obj interface{}) error {
	return json.Unmarshal(c.Recorder.Body.Bytes(), obj)
}

// Shortcut for response code
func (c *CompletedRequest) Code() int {
	return c.Recorder.Code
}
