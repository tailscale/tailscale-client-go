// Package tailscale contains a basic implementation of a client for the Tailscale HTTP api. Documentation is here:
// https://tailscale.com/api
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
	"time"

	"github.com/tailscale/hujson"
	"golang.org/x/oauth2/clientcredentials"
)

type (
	// client type is used to perform actions against the Tailscale API.
	Client interface {
		Tailnet() string

		BuildRequest(ctx context.Context, method, uri string, opts ...RequestOption) (*http.Request, error)

		PerformRequest(req *http.Request, out interface{}) error
	}

	// client type is used to perform actions against the Tailscale API.
	client struct {
		apiKey    string
		http      *http.Client
		baseURL   *url.URL
		tailnet   string
		userAgent string // empty string means Go's default value.
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

	// ClientOption type is a function that is used to modify a Client.
	ClientOption func(c *client) error
)

const baseURL = "https://api.tailscale.com"
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
func NewClient(apiKey, tailnet string, options ...ClientOption) (Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	c := &client{
		baseURL:   u,
		tailnet:   tailnet,
		userAgent: defaultUserAgent,
	}

	if apiKey != "" {
		c.apiKey = apiKey
		c.http = &http.Client{Timeout: defaultHttpClientTimeout}
	}

	for _, option := range options {
		if err = option(c); err != nil {
			return nil, err
		}
	}

	// apiKey or WithOAuthClientCredentials will initialize the http client. Fail here if both are not set.
	if c.apiKey == "" && c.http == nil {
		return nil, errors.New("no authentication credentials provided")
	}

	return c, nil
}

// WithBaseURL sets a custom baseURL for the Tailscale API, this is primarily used for testing purposes.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *client) error {
		u, err := url.Parse(baseURL)
		if err != nil {
			return err
		}

		c.baseURL = u
		return nil
	}
}

// WithOAuthClientCredentials sets the OAuth Client Credentials to use for the Tailscale API.
func WithOAuthClientCredentials(clientID, clientSecret string, scopes []string) ClientOption {
	return func(c *client) error {
		relTokenURL, err := url.Parse("/api/v2/oauth/token")
		if err != nil {
			return err
		}
		oauthConfig := clientcredentials.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			TokenURL:     c.baseURL.ResolveReference(relTokenURL).String(),
			Scopes:       scopes,
		}

		// use context.Background() here, since this is used to refresh the token in the future
		c.http = oauthConfig.Client(context.Background())
		c.http.Timeout = defaultHttpClientTimeout
		return nil
	}
}

// WithUserAgent sets a custom User-Agent header in HTTP requests.
// Passing an empty string will make the client use Go's default value.
func WithUserAgent(ua string) ClientOption {
	return func(c *client) error {
		c.userAgent = ua
		return nil
	}
}

type RequestParams struct {
	headers     map[string]string
	body        any
	contentType string
}

type RequestOption func(*RequestParams)

func RequestBody(body any) RequestOption {
	return func(rof *RequestParams) {
		rof.body = body
	}
}

func RequestHeaders(headers map[string]string) RequestOption {
	return func(rof *RequestParams) {
		rof.headers = headers
	}
}

func RequestContentType(ct string) RequestOption {
	return func(rof *RequestParams) {
		rof.contentType = ct
	}
}

func (c *client) Tailnet() string {
	return c.tailnet
}

func (c *client) BuildRequest(ctx context.Context, method, uri string, opts ...RequestOption) (*http.Request, error) {
	rof := &RequestParams{
		contentType: defaultContentType,
	}
	for _, opt := range opts {
		opt(rof)
	}

	u, err := c.baseURL.Parse(uri)
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

	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
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
	if c.apiKey != "" {
		req.SetBasicAuth(c.apiKey, "")
	}

	return req, nil
}

func (c *client) PerformRequest(req *http.Request, out interface{}) error {
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

// Time wraps a time and allows for unmarshalling timestamps that represent an empty time as an empty string (e.g "")
// this is used by the tailscale API when it returns devices that have no created date, such as its hello service.
type Time struct {
	time.Time
}

// MarshalJSON is an implementation of json.Marshal.
func (t Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Time)
}

// UnmarshalJSON unmarshals the content of data as a time.Time, a blank string will keep the time at its zero value.
func (t *Time) UnmarshalJSON(data []byte) error {
	if string(data) == `""` {
		return nil
	}

	if err := json.Unmarshal(data, &t.Time); err != nil {
		return err
	}

	return nil
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
