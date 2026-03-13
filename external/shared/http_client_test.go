package shared

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestRequest(method string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, "http://example.com", body)
	if err != nil {
		panic(err)
	}
	return req
}

func newJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func setupClient() *APIClient {
	return NewAPIClient(WithDumpEnabled(true))
}

func TestAPIClient_Request(t *testing.T) {
	tests := map[string]struct {
		setupClient  func() *APIClient
		setupRequest func() *http.Request
		transport    http.RoundTripper
		opts         []RequestOption
		wantBody     []byte
		wantStatus   int
		wantErr      bool
		checkErr     func(t *testing.T, err error)
	}{
		"GET request": {
			setupClient:  setupClient,
			setupRequest: func() *http.Request { return newTestRequest(http.MethodGet, nil) },
			transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodGet, r.Method)
				return newJSONResponse(http.StatusOK, `{"status":"ok"}`), nil
			}),
			wantBody:   []byte(`{"status":"ok"}`),
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		"with authorization header": {
			setupClient:  func() *APIClient { return NewAPIClient() },
			setupRequest: func() *http.Request { return newTestRequest(http.MethodGet, nil) },
			transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				return newJSONResponse(http.StatusOK, `{"authenticated":true}`), nil
			}),
			opts:       []RequestOption{WithAuthorization("Bearer test-token")},
			wantBody:   []byte(`{"authenticated":true}`),
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		"server error response": {
			setupClient:  func() *APIClient { return NewAPIClient() },
			setupRequest: func() *http.Request { return newTestRequest(http.MethodGet, nil) },
			transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				return newJSONResponse(http.StatusInternalServerError, `{"error":"internal server error"}`), nil
			}),
			wantBody:   []byte(`{"error":"internal server error"}`),
			wantStatus: http.StatusInternalServerError,
			wantErr:    false,
		},
		"POST request with body": {
			setupClient: setupClient,
			setupRequest: func() *http.Request {
				return newTestRequest(http.MethodPost, bytes.NewBufferString(`{"name":"test"}`))
			},
			transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				body, _ := io.ReadAll(r.Body)
				assert.Equal(t, `{"name":"test"}`, string(body))
				return newJSONResponse(http.StatusCreated, `{"id":123}`), nil
			}),
			wantBody:   []byte(`{"id":123}`),
			wantStatus: http.StatusCreated,
			wantErr:    false,
		},
		"network timeout": {
			setupClient:  func() *APIClient { return NewAPIClient() },
			setupRequest: func() *http.Request { return newTestRequest(http.MethodGet, nil) },
			transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				return nil, fakeTimeoutError{}
			}),
			wantStatus: http.StatusRequestTimeout,
			wantErr:    true,
			checkErr: func(t *testing.T, err error) {
				var netErr net.Error
				assert.True(t, errors.As(err, &netErr))
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := tc.setupClient()
			if tc.transport != nil {
				client.Client.Transport = tc.transport
			}
			req := tc.setupRequest()
			body, statusCode, err := client.Request(req, tc.opts...)

			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, body)
				if tc.checkErr != nil {
					tc.checkErr(t, err)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantBody, body)
			}
			assert.Equal(t, tc.wantStatus, statusCode)
		})
	}
}

type fakeTimeoutError struct{}

func (fakeTimeoutError) Error() string   { return "timeout" }
func (fakeTimeoutError) Timeout() bool   { return true }
func (fakeTimeoutError) Temporary() bool { return true }
