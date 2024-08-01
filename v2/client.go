// Package tailscale contains a basic implementation of a client for the Tailscale HTTP api. Documentation is here:
// https://github.com/tailscale/tailscale/blob/main/api.md
//
// WARNING - this v2 implementation is under active development, use at your own risk.
package tailscale

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

		http *http.Client
		// tailnetPathEscaped is the value of tailnet passed to url.PathEscape.
		// This value should be used when formatting paths that have tailnet as a segment.
		tailnetPathEscaped string

		initOnce sync.Once

		// Specific resources
		contacts   *ContactsResource
		devices    *DevicesResource
		dns        *DNSResource
		keys       *KeysResource
		policyFile *PolicyFileResource
		webhooks   *WebhooksResource
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

func init() {
	var err error
	defaultBaseURL, err = url.Parse("https://api.tailscale.com")
	if err != nil {
		panic(fmt.Errorf("failed to parse baseURL: %w", err))
	}
}

const defaultContentType = "application/json"
const defaultHttpClientTimeout = time.Minute
const defaultUserAgent = "tailscale-client-go"

// NewClient returns a new instance of the Client type that will perform operations against a chosen tailnet and will
// provide the apiKey for authorization. Additional options can be provided, see ClientOption for more details.
//
// To use OAuth Client credentials pass an empty string as apiKey and use WithOAuthClientCredentials() as below:
//
//	client, err := tailscale.NewClient(
//	"",
//	tailnet,
//	tailscale.WithOAuthClientCredentials(oauthClientID, oauthClientSecret, oauthScopes),
//	)
func (c *Client) init() {
	c.initOnce.Do(func() {
		if c.BaseURL == nil {
			c.BaseURL = defaultBaseURL
		}
		c.tailnetPathEscaped = url.PathEscape(c.Tailnet)
		if c.UserAgent == "" {
			c.UserAgent = defaultUserAgent
		}
		c.http = &http.Client{Timeout: defaultHttpClientTimeout}
		c.contacts = &ContactsResource{c}
		c.devices = &DevicesResource{c}
		c.dns = &DNSResource{c}
		c.keys = &KeysResource{c}
		c.policyFile = &PolicyFileResource{c}
		c.webhooks = &WebhooksResource{c}
	})
}

func (c *Client) Contacts() *ContactsResource {
	c.init()
	return c.contacts
}

func (c *Client) Devices() *DevicesResource {
	c.init()
	return c.devices
}

func (c *Client) DNS() *DNSResource {
	c.init()
	return c.dns
}

func (c *Client) Keys() *KeysResource {
	c.init()
	return c.keys
}

func (c *Client) PolicyFile() *PolicyFileResource {
	c.init()
	return c.policyFile
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

func (c *Client) buildRequest(ctx context.Context, method, uri string, opts ...requestOption) (*http.Request, error) {
	rof := &requestParams{
		contentType: defaultContentType,
	}
	for _, opt := range opts {
		opt(rof)
	}

	u, err := c.BaseURL.Parse(uri)
	if err != nil {
		return nil, err
	}

	var bodyBytes []byte
	if rof.body != nil {
		switch body := rof.body.(type) {
		case string:
			bodyBytes = []byte(body)
		case []byte:
			bodyBytes = body
		default:
			bodyBytes, err = json.MarshalIndent(rof.body, "", " ")
			if err != nil {
				return nil, err
			}
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewBuffer(bodyBytes))
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

	// c.apiKey will not be set on the client was configured with WithOAuthClientCredentials()
	if c.APIKey != "" {
		req.SetBasicAuth(c.APIKey, "")
	}

	return req, nil
}

func (c *Client) performRequest(req *http.Request, out interface{}) error {
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
		if err = json.Unmarshal(body, &apiErr); err != nil {
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
