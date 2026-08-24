package testx

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

// HTTPRequest describes the request sent to an HTTP handler.
type HTTPRequest struct {
	Method string
	Target string
	Body   []byte
	Header http.Header
}

// HTTPResponse describes the response expected from an HTTP handler.
// Header is treated as a subset: only listed header values are compared.
type HTTPResponse struct {
	Status int
	Body   string
	Header http.Header
}

// HTTPPreset executes httptest requests against a handler.
type HTTPPreset struct {
	handler http.Handler
}

// HTTP creates a preset for an http.Handler.
func HTTP(handler http.Handler) HTTPPreset {
	return HTTPPreset{handler: handler}
}

// Run executes HTTP request/response cases.
func (p HTTPPreset) Run(t *testing.T, cases ...Case[HTTPRequest, HTTPResponse]) {
	t.Helper()
	if p.handler == nil {
		t.Fatal("testx: HTTP handler is nil")
	}
	for i := range cases {
		c := cases[i]
		name := c.Name
		if name == "" {
			name = "case"
		}
		t.Run(name, func(t *testing.T) {
			if c.skip {
				t.Skip(c.skipReason)
			}
			if c.parallel {
				t.Parallel()
			}
			method := c.Input.Method
			if method == "" {
				method = http.MethodGet
			}
			target := c.Input.Target
			if target == "" {
				target = "/"
			}
			req, err := http.NewRequest(method, target, bytes.NewReader(c.Input.Body))
			if err != nil {
				t.Fatalf("testx: invalid HTTP request: %v", err)
			}
			req.Header = c.Input.Header.Clone()
			recorder := httptest.NewRecorder()
			p.handler.ServeHTTP(recorder, req)
			got := HTTPResponse{Status: recorder.Code, Body: recorder.Body.String(), Header: recorder.Header().Clone()}
			assertHTTPResponse(t, c.Expect, got)
		})
	}
}

func assertHTTPResponse(t testing.TB, want, got HTTPResponse) {
	t.Helper()
	wantStatus := want.Status
	if wantStatus == 0 {
		wantStatus = http.StatusOK
	}
	if wantStatus != got.Status {
		t.Errorf("HTTP status: want %d, got %d", wantStatus, got.Status)
	}
	if want.Body != got.Body {
		t.Errorf("HTTP body: want %q, got %q", want.Body, got.Body)
	}
	for key, wantValues := range want.Header {
		gotValues, ok := got.Header[http.CanonicalHeaderKey(key)]
		if !ok || !reflect.DeepEqual(wantValues, gotValues) {
			t.Errorf("HTTP header %q: want %v, got %v", key, wantValues, gotValues)
		}
	}
}
