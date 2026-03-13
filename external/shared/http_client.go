package shared

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	APITimeout = 30 * time.Second
)

type APIConnector interface {
	Request(req *http.Request, opts ...RequestOption) ([]byte, int, error)
}

type APIClient struct {
	Client       *http.Client
	dumpEnabled  bool
	dumpLogLevel slog.Level
	dumpPretty   bool
}

// APIClientOption applies a configuration to an APIClient.
type APIClientOption func(*APIClient)

// WithDumpEnabled controls whether request/response dump logging is active.
func WithDumpEnabled(enabled bool) APIClientOption {
	return func(c *APIClient) { c.dumpEnabled = enabled }
}

// WithDumpLogLevel sets the log level for request/response dump output.
func WithDumpLogLevel(level slog.Level) APIClientOption {
	return func(c *APIClient) { c.dumpLogLevel = level }
}

// WithDumpPretty controls whether request/response body is pretty-printed.
func WithDumpPretty(pretty bool) APIClientOption {
	return func(c *APIClient) { c.dumpPretty = pretty }
}

// RequestOption applies a modification to an http.Request before it is sent.
type RequestOption func(req *http.Request)

// Authorizationヘッダ名の定数
const defaultAuthorizationHeader = "Authorization"

// WithAuthorization returns a RequestOption that sets the Authorization header.
func WithAuthorization(value string) RequestOption {
	return func(req *http.Request) {
		if value != "" {
			req.Header.Set(defaultAuthorizationHeader, value)
		}
	}
}

func NewAPIClient(opts ...APIClientOption) *APIClient {
	c := &APIClient{
		Client: &http.Client{
			Timeout: APITimeout,
		},
		dumpLogLevel: slog.LevelDebug,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *APIClient) Request(req *http.Request, opts ...RequestOption) ([]byte, int, error) {
	// Option適用
	for _, opt := range opts {
		opt(req)
	}

	if c.dumpEnabled {
		c.DumpRequest(req)
	}

	res, err := c.Client.Do(req)
	if err != nil {
		var ne net.Error
		if ok := errors.As(err, &ne); ok {
			return nil, http.StatusRequestTimeout, err
		}

		return nil, http.StatusInternalServerError, err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			slog.Warn("[APIClient] Failed to close response body", "error", err)
		}
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if c.dumpEnabled {
		c.DumpResponse(res.StatusCode, body)
	}

	return body, res.StatusCode, nil
}

func (c *APIClient) DumpRequest(req *http.Request) {
	if req == nil {
		slog.Log(context.Background(), c.dumpLogLevel, "[APIClient] Request is nil")
		return
	}

	body := "(empty)"
	if req.Body != nil {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			body = fmt.Sprintf("Failed to read body: %s", err)
		} else {
			req.Body = io.NopCloser(bytes.NewBuffer(b))
			body = c.humanReadableBody(b)
		}
	}

	slog.Log(context.Background(), c.dumpLogLevel, "[APIClient] Request >>>", //nolint:gosec // G706: false positive, slog attributes are not user-controlled
		"url", req.URL.String(),
		"method", req.Method,
		"body", body,
	)
}

func (c *APIClient) DumpResponse(code int, b []byte) {
	slog.Log(context.Background(), c.dumpLogLevel, "[APIClient] Response <<<", //nolint:gosec // G706: false positive, slog attributes are not user-controlled
		"statusCode", code,
		"body", c.humanReadableBody(b),
	)
}

func (c *APIClient) humanReadableBody(b []byte) string {
	bodyStr := string(b)
	if len(bodyStr) == 0 {
		return "(empty)"
	}
	if !c.dumpPretty {
		return bodyStr
	}
	tab := "  "
	var js any
	if json.Unmarshal(b, &js) == nil {
		// JSONであればインデント付きで出力
		pretty, err := json.MarshalIndent(js, "", tab)
		if err == nil {
			return "\n" + string(pretty)
		}
	}
	// JSONでなければ従来通り
	readableBody := "\n" + tab + bodyStr
	readableBody = strings.ReplaceAll(readableBody, "\n", "\n"+tab)
	readableBody = strings.ReplaceAll(readableBody, "\r", "")
	readableBody = strings.ReplaceAll(readableBody, "\t", tab)
	return readableBody
}
