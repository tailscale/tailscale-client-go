// Package tsclient contains a basic implementation of a client for the Tailscale HTTP api. Documentation is here:
// https://github.com/tailscale/tailscale/blob/main/api.md
//
// WARNING - this v2 implementation is under active development, use at your own risk.
package tsclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/tailscale/hujson"
	"golang.org/x/oauth2/clientcredentials"
)

type (
	// Client type is used to perform actions against the Tailscale API.
	Client struct {
		// BaseURL is the base URL for accessing the Tailscale API server. Defaults to https://api.tailscale.com.
		BaseURL *url.URL
		// UserAgent configures the User-Agent HTTP header for requests, defaults to "tailscale-client-go"
		UserAgent string
		// APIKey allows specifying an APIKey to use for authentication.
		APIKey string
		// Tailnet allows specifying a specific Tailnet by name, to which this Client will connect by default.
		Tailnet string

		// http is the http client to use for requests to the API server. If specified, this supercedes the above configuration.
		http *http.Client

		initOnce sync.Once

		// Specific resources
		contacts *ContactsResource
		devices  *DevicesResource
		keys     *KeysResource
		webhooks *WebhooksResource
	}

	// APIError type describes an error as returned by the Tailscale API.
	APIError struct {
		Message string         `json:"message"`
		Data    []APIErrorData `json:"data"`
		status  int
	}

	// APIErrorData type describes elements of the data field within errors returned by the Tailscale API.
	APIErrorData struct {
		User   string   `json:"user"`
		Errors []string `json:"errors"`
	}
)

var defaultBaseURL *url.URL
var oauthRelTokenURL *url.URL

func init() {
	var err error
	defaultBaseURL, err = url.Parse("https://api.tailscale.com")
	if err != nil {
		panic(fmt.Errorf("failed to parse defaultBaseURL: %w", err))
	}

	oauthRelTokenURL, err = url.Parse("/api/v2/oauth/token")
	if err != nil {
		panic(fmt.Errorf("failed to parse oauthRelTokenURL: %s", err))
	}
}

const defaultContentType = "application/json"
const defaultHttpClientTimeout = time.Minute
const defaultUserAgent = "tailscale-client-go"

// NewClient returns a new instance of the Client type that will perform operations against a chosen tailnet and will
// provide the apiKey for authorization. Additional options can be provided, see ClientOption for more details.
//
// To use OAuth Client credentials, call [UseOAuth].
func (c *Client) init() {
	c.initOnce.Do(func() {
		if c.BaseURL == nil {
			c.BaseURL = defaultBaseURL
		}
		if c.UserAgent == "" {
			c.UserAgent = defaultUserAgent
		}
		if c.http == nil {
			c.http = &http.Client{Timeout: defaultHttpClientTimeout}
		}
		c.contacts = &ContactsResource{c}
		c.devices = &DevicesResource{c}
		c.keys = &KeysResource{c}
		c.webhooks = &WebhooksResource{c}
	})
}

// UseOAuth configures the client to use the specified OAuth credentials.
func (c *Client) UseOAuth(clientID, clientSecret string, scopes []string) {
	oauthConfig := clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     c.BaseURL.ResolveReference(oauthRelTokenURL).String(),
		Scopes:       scopes,
	}

	// use context.Background() here, since this is used to refresh the token in the future
	c.http = oauthConfig.Client(context.Background())
	c.http.Timeout = defaultHttpClientTimeout
}

func (c *Client) Contacts() *ContactsResource {
	c.init()
	return c.contacts
}

func (c *Client) Devices() *DevicesResource {
	c.init()
	return c.devices
}

func (c *Client) Keys() *KeysResource {
	c.init()
	return c.keys
}

func (c *Client) Webhooks() *WebhooksResource {
	c.init()
	return c.webhooks
}

type requestParams struct {
	headers     map[string]string
	body        any
	contentType string
}

type requestOption func(*requestParams)

func requestBody(body any) requestOption {
	return func(rof *requestParams) {
		rof.body = body
	}
}

func requestHeaders(headers map[string]string) requestOption {
	return func(rof *requestParams) {
		rof.headers = headers
	}
}

func requestContentType(ct string) requestOption {
	return func(rof *requestParams) {
		rof.contentType = ct
	}
}

// buildURL builds a url to /api/v2/... using the given pathElements.
// It url escapes each path element, so the caller doesn't need to worry about that.
func (c *Client) buildURL(pathElements ...any) *url.URL {
	elem := make([]string, 1, len(pathElements)+1)
	elem[0] = "/api/v2"
	for _, pathElement := range pathElements {
		elem = append(elem, url.PathEscape(fmt.Sprint(pathElement)))
	}
	return c.BaseURL.JoinPath(elem...)
}

// buildTailnetURL builds a url to /api/v2/tailnet/<tailnet>/... using the given pathElements.
// It url escapes each path element, so the caller doesn't need to worry about that.
func (c *Client) buildTailnetURL(pathElements ...any) *url.URL {
	allElements := make([]any, 2, len(pathElements)+2)
	allElements[0] = "tailnet"
	allElements[1] = c.Tailnet
	allElements = append(allElements, pathElements...)
	return c.buildURL(allElements...)
}

func (c *Client) buildRequest(ctx context.Context, method string, uri *url.URL, opts ...requestOption) (*http.Request, error) {
	rof := &requestParams{
		contentType: defaultContentType,
	}
	for _, opt := range opts {
		opt(rof)
	}

	var bodyBytes []byte
	if rof.body != nil {
		switch body := rof.body.(type) {
		case string:
			bodyBytes = []byte(body)
		case []byte:
			bodyBytes = body
		default:
			var err error
			bodyBytes, err = json.MarshalIndent(rof.body, "", " ")
			if err != nil {
				return nil, err
			}
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, uri.String(), bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	for k, v := range rof.headers {
		req.Header.Set(k, v)
	}

	switch {
	case rof.body == nil:
		req.Header.Set("Accept", rof.contentType)
	default:
		req.Header.Set("Content-Type", rof.contentType)
	}

	if c.APIKey != "" {
		req.SetBasicAuth(c.APIKey, "")
	}

	return req, nil
}

func (c *Client) do(req *http.Request, out interface{}) error {
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode >= http.StatusOK && res.StatusCode < http.StatusMultipleChoices {
		// If we don't care about the response body, leave. This check is required as some
		// API responses have empty bodies, so we don't want to try and standardize them for
		// parsing.
		if out == nil {
			return nil
		}

		// If we're expected to write result into a []byte, do not attempt to parse it.
		if o, ok := out.(*[]byte); ok {
			*o = bytes.Clone(body)
			return nil
		}

		// If we've got hujson back, convert it to JSON, so we can natively parse it.
		if !json.Valid(body) {
			body, err = hujson.Standardize(body)
			if err != nil {
				return err
			}
		}

		return json.Unmarshal(body, out)
	}

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err != nil {
			return err
		}

		apiErr.status = res.StatusCode
		return apiErr
	}

	return nil
}

func (err APIError) Error() string {
	return fmt.Sprintf("%s (%v)", err.Message, err.status)
}

// IsNotFound returns true if the provided error implementation is an APIError with a status of 404.
func IsNotFound(err error) bool {
	var apiErr APIError
	if errors.As(err, &apiErr) {
		return apiErr.status == http.StatusNotFound
	}

	return false
}

// ErrorData returns the contents of the APIError.Data field from the provided error if it is of type APIError. Returns
// a nil slice if the given error is not of type APIError.
func ErrorData(err error) []APIErrorData {
	var apiErr APIError
	if errors.As(err, &apiErr) {
		return apiErr.Data
	}

	return nil
}

// Duration type wraps a time.Duration, allowing it to be JSON marshalled as a string like "20h" rather than
// a numeric value.
type Duration time.Duration

func (d Duration) String() string {
	return time.Duration(d).String()
}

func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

func (d *Duration) UnmarshalText(b []byte) error {
	text := string(b)
	if text == "" {
		text = "0s"
	}
	pd, err := time.ParseDuration(text)
	if err != nil {
		return err
	}
	*d = Duration(pd)
	return nil
}
