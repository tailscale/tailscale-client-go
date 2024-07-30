// Package tailscale contains a basic implementation of a client for the Tailscale HTTP api. Documentation is here:
// https://github.com/tailscale/tailscale/blob/main/api.md
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
	// Client type is used to perform actions against the Tailscale API.
	Client struct {
		apiKey  string
		http    *http.Client
		baseURL *url.URL
		tailnet string
		// tailnetPathEscaped is the value of tailnet passed to url.PathEscape.
		// This value should be used when formatting paths that have tailnet as a segment.
		tailnetPathEscaped string
		userAgent          string // empty string means Go's default value.
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
	ClientOption func(c *Client) error
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
func NewClient(apiKey, tailnet string, options ...ClientOption) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	c := &Client{
		baseURL:            u,
		tailnet:            tailnet,
		tailnetPathEscaped: url.PathEscape(tailnet),
		userAgent:          defaultUserAgent,
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
	return func(c *Client) error {
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
	return func(c *Client) error {
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
	return func(c *Client) error {
		c.userAgent = ua
		return nil
	}
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

// SetDNSSearchPaths replaces the list of search paths with the list supplied by the user and returns an error otherwise.
func (c *Client) SetDNSSearchPaths(ctx context.Context, searchPaths []string) error {
	const uriFmt = "/api/v2/tailnet/%v/dns/searchpaths"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnetPathEscaped), requestBody(map[string][]string{
		"searchPaths": searchPaths,
	}))
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// DNSSearchPaths retrieves the list of search paths that is currently set for the given tailnet.
func (c *Client) DNSSearchPaths(ctx context.Context) ([]string, error) {
	const uriFmt = "/api/v2/tailnet/%v/dns/searchpaths"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnetPathEscaped))
	if err != nil {
		return nil, err
	}

	resp := make(map[string][]string)
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return resp["searchPaths"], nil
}

// SetDNSNameservers replaces the list of DNS nameservers for the given tailnet with the list supplied by the user. Note
// that changing the list of DNS nameservers may also affect the status of MagicDNS (if MagicDNS is on).
func (c *Client) SetDNSNameservers(ctx context.Context, dns []string) error {
	const uriFmt = "/api/v2/tailnet/%v/dns/nameservers"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnetPathEscaped), requestBody(map[string][]string{
		"dns": dns,
	}))
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// DNSNameservers lists the DNS nameservers for a tailnet
func (c *Client) DNSNameservers(ctx context.Context) ([]string, error) {
	const uriFmt = "/api/v2/tailnet/%v/dns/nameservers"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnetPathEscaped))
	if err != nil {
		return nil, err
	}

	resp := make(map[string][]string)
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return resp["dns"], nil
}

// SplitDnsRequest is a map from domain names to a list of nameservers.
type SplitDnsRequest map[string][]string

// SplitDnsResponse is a map from domain names to a list of nameservers.
type SplitDnsResponse SplitDnsRequest

// UpdateSplitDNS updates the split DNS settings for a tailnet using the
// provided SplitDnsRequest object. This is a PATCH operation that performs
// partial updates of the underlying data structure.
//
// Mapping a domain to a nil slice in the request will unset the nameservers
// associated with that domain. Values provided for domains will overwrite the
// current value associated with the domain. Domains not included in the request
// will remain unchanged.
func (c *Client) UpdateSplitDNS(ctx context.Context, request SplitDnsRequest) (SplitDnsResponse, error) {
	const uriFmt = "/api/v2/tailnet/%v/dns/split-dns"

	req, err := c.buildRequest(ctx, http.MethodPatch, fmt.Sprintf(uriFmt, c.tailnetPathEscaped), requestBody(request))
	if err != nil {
		return nil, err
	}

	var resp SplitDnsResponse
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// SetSplitDNS sets the split DNS settings for a tailnet using the provided
// SplitDnsRequest object. This is a PUT operation that fully replaces the underlying
// data structure.
//
// Passing in an empty SplitDnsRequest will unset all split DNS mappings for the tailnet.
func (c *Client) SetSplitDNS(ctx context.Context, request SplitDnsRequest) error {
	const uriFmt = "/api/v2/tailnet/%v/dns/split-dns"

	req, err := c.buildRequest(ctx, http.MethodPut, fmt.Sprintf(uriFmt, c.tailnetPathEscaped), requestBody(request))
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// SplitDNS retrieves the split DNS configuration for a tailnet.
func (c *Client) SplitDNS(ctx context.Context) (SplitDnsResponse, error) {
	const uriFmt = "/api/v2/tailnet/%v/dns/split-dns"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnetPathEscaped))
	if err != nil {
		return nil, err
	}

	var resp SplitDnsResponse
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

type (
	// ACL contains the schema for a tailnet policy file. More details: https://tailscale.com/kb/1018/acls/
	ACL struct {
		ACLs                []ACLEntry          `json:"acls,omitempty" hujson:"ACLs,omitempty"`
		AutoApprovers       *ACLAutoApprovers   `json:"autoApprovers,omitempty" hujson:"AutoApprovers,omitempty"`
		Groups              map[string][]string `json:"groups,omitempty" hujson:"Groups,omitempty"`
		Hosts               map[string]string   `json:"hosts,omitempty" hujson:"Hosts,omitempty"`
		TagOwners           map[string][]string `json:"tagOwners,omitempty" hujson:"TagOwners,omitempty"`
		DERPMap             *ACLDERPMap         `json:"derpMap,omitempty" hujson:"DerpMap,omitempty"`
		Tests               []ACLTest           `json:"tests,omitempty" hujson:"Tests,omitempty"`
		SSH                 []ACLSSH            `json:"ssh,omitempty" hujson:"SSH,omitempty"`
		NodeAttrs           []NodeAttrGrant     `json:"nodeAttrs,omitempty" hujson:"NodeAttrs,omitempty"`
		DisableIPv4         bool                `json:"disableIPv4,omitempty" hujson:"DisableIPv4,omitempty"`
		OneCGNATRoute       string              `json:"oneCGNATRoute,omitempty" hujson:"OneCGNATRoute,omitempty"`
		RandomizeClientPort bool                `json:"randomizeClientPort,omitempty" hujson:"RandomizeClientPort,omitempty"`

		// Postures and DefaultSourcePosture are for an experimental feature and not yet public or documented as of 2023-08-17.
		// This API is subject to change. Internal bug: corp/13986
		Postures             map[string][]string `json:"postures,omitempty" hujson:"Postures,omitempty"`
		DefaultSourcePosture []string            `json:"defaultSrcPosture,omitempty" hujson:"DefaultSrcPosture,omitempty"`
	}

	ACLAutoApprovers struct {
		Routes   map[string][]string `json:"routes,omitempty" hujson:"Routes,omitempty"`
		ExitNode []string            `json:"exitNode,omitempty" hujson:"ExitNode,omitempty"`
	}

	ACLEntry struct {
		Action      string   `json:"action,omitempty" hujson:"Action,omitempty"`
		Ports       []string `json:"ports,omitempty" hujson:"Ports,omitempty"`
		Users       []string `json:"users,omitempty" hujson:"Users,omitempty"`
		Source      []string `json:"src,omitempty" hujson:"Src,omitempty"`
		Destination []string `json:"dst,omitempty" hujson:"Dst,omitempty"`
		Protocol    string   `json:"proto,omitempty" hujson:"Proto,omitempty"`

		// SourcePosture is for an experimental feature and not yet public or documented as of 2023-08-17.
		SourcePosture []string `json:"srcPosture,omitempty" hujson:"SrcPosture,omitempty"`
	}

	ACLTest struct {
		User   string   `json:"user,omitempty" hujson:"User,omitempty"`
		Allow  []string `json:"allow,omitempty" hujson:"Allow,omitempty"`
		Deny   []string `json:"deny,omitempty" hujson:"Deny,omitempty"`
		Source string   `json:"src,omitempty" hujson:"Src,omitempty"`
		Accept []string `json:"accept,omitempty" hujson:"Accept,omitempty"`
	}

	ACLDERPMap struct {
		Regions            map[int]*ACLDERPRegion `json:"regions" hujson:"Regions"`
		OmitDefaultRegions bool                   `json:"omitDefaultRegions,omitempty" hujson:"OmitDefaultRegions,omitempty"`
	}

	ACLDERPRegion struct {
		RegionID   int            `json:"regionID" hujson:"RegionID"`
		RegionCode string         `json:"regionCode" hujson:"RegionCode"`
		RegionName string         `json:"regionName" hujson:"RegionName"`
		Avoid      bool           `json:"avoid,omitempty" hujson:"Avoid,omitempty"`
		Nodes      []*ACLDERPNode `json:"nodes" hujson:"Nodes"`
	}

	ACLDERPNode struct {
		Name             string `json:"name" hujson:"Name"`
		RegionID         int    `json:"regionID" hujson:"RegionID"`
		HostName         string `json:"hostName" hujson:"HostName"`
		CertName         string `json:"certName,omitempty" hujson:"CertName,omitempty"`
		IPv4             string `json:"ipv4,omitempty" hujson:"IPv4,omitempty"`
		IPv6             string `json:"ipv6,omitempty" hujson:"IPv6,omitempty"`
		STUNPort         int    `json:"stunPort,omitempty" hujson:"STUNPort,omitempty"`
		STUNOnly         bool   `json:"stunOnly,omitempty" hujson:"STUNOnly,omitempty"`
		DERPPort         int    `json:"derpPort,omitempty" hujson:"DERPPort,omitempty"`
		InsecureForTests bool   `json:"insecureForRests,omitempty" hujson:"InsecureForTests,omitempty"`
		STUNTestIP       string `json:"stunTestIP,omitempty" hujson:"STUNTestIP,omitempty"`
	}

	ACLSSH struct {
		Action          string   `json:"action,omitempty" hujson:"Action,omitempty"`
		Users           []string `json:"users,omitempty" hujson:"Users,omitempty"`
		Source          []string `json:"src,omitempty" hujson:"Src,omitempty"`
		Destination     []string `json:"dst,omitempty" hujson:"Dst,omitempty"`
		CheckPeriod     Duration `json:"checkPeriod,omitempty" hujson:"CheckPeriod,omitempty"`
		Recorder        []string `json:"recorder,omitempty" hujson:"Recorder,omitempty"`
		EnforceRecorder bool     `json:"enforceRecorder,omitempty" hujson:"EnforceRecorder,omitempty"`
	}

	NodeAttrGrant struct {
		Target []string                       `json:"target,omitempty" hujson:"Target,omitempty"`
		Attr   []string                       `json:"attr,omitempty" hujson:"Attr,omitempty"`
		App    map[string][]*NodeAttrGrantApp `json:"app,omitempty" hujson:"App,omitempty"`
	}

	NodeAttrGrantApp struct {
		Name       string   `json:"name,omitempty" hujson:"Name,omitempty"`
		Connectors []string `json:"connectors,omitempty" hujson:"Connectors,omitempty"`
		Domains    []string `json:"domains,omitempty" hujson:"Domains,omitempty"`
	}
)

// ACL retrieves the ACL that is currently set for the given tailnet.
func (c *Client) ACL(ctx context.Context) (*ACL, error) {
	const uriFmt = "/api/v2/tailnet/%s/acl"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnetPathEscaped))
	if err != nil {
		return nil, err
	}

	var resp ACL
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// RawACL retrieves the ACL that is currently set for the given tailnet
// as a HuJSON string.
func (c *Client) RawACL(ctx context.Context) (string, error) {
	const uriFmt = "/api/v2/tailnet/%s/acl"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnetPathEscaped), requestContentType("application/hujson"))
	if err != nil {
		return "", err
	}

	var resp []byte
	if err = c.performRequest(req, &resp); err != nil {
		return "", err
	}

	return string(resp), nil
}

type setACLParams struct {
	headers map[string]string
}
type SetACLOption func(p *setACLParams)

// WithETag allows passing an ETag value with Set ACL API call that
// will be used in the `If-Match` HTTP request header.
func WithETag(etag string) SetACLOption {
	return func(p *setACLParams) {
		p.headers["If-Match"] = fmt.Sprintf("%q", etag)
	}
}

// SetACL sets the ACL for the given tailnet. `acl` can either be an [ACL],
// or a HuJSON string.
func (c *Client) SetACL(ctx context.Context, acl any, opts ...SetACLOption) error {
	const uriFmt = "/api/v2/tailnet/%s/acl"

	p := &setACLParams{headers: make(map[string]string)}
	for _, opt := range opts {
		opt(p)
	}

	reqOpts := []requestOption{
		requestHeaders(p.headers),
		requestBody(acl),
	}
	switch v := acl.(type) {
	case ACL:
	case string:
		reqOpts = append(reqOpts, requestContentType("application/hujson"))
	default:
		return fmt.Errorf("expected ACL content as a string or as ACL struct; got %T", v)
	}

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnetPathEscaped), reqOpts...)
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// ValidateACL validates the provided ACL via the API. `acl` can either be an [ACL],
// or a HuJSON string.
func (c *Client) ValidateACL(ctx context.Context, acl any) error {
	const uriFmt = "/api/v2/tailnet/%s/acl/validate"

	reqOpts := []requestOption{
		requestBody(acl),
	}
	switch v := acl.(type) {
	case ACL:
	case string:
		reqOpts = append(reqOpts, requestContentType("application/hujson"))
	default:
		return fmt.Errorf("expected ACL content as a string or as ACL struct; got %T", v)
	}

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnetPathEscaped), reqOpts...)
	if err != nil {
		return err
	}

	var response APIError
	if err := c.performRequest(req, &response); err != nil {
		return err
	}
	if response.Message != "" {
		return fmt.Errorf("ACL validation failed: %s; %v", response.Message, response.Data)
	}
	return nil
}

type DNSPreferences struct {
	MagicDNS bool `json:"magicDNS"`
}

// DNSPreferences retrieves the DNS preferences that are currently set for the given tailnet. Supply the tailnet of
// interest in the path.
func (c *Client) DNSPreferences(ctx context.Context) (*DNSPreferences, error) {
	const uriFmt = "/api/v2/tailnet/%s/dns/preferences"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnetPathEscaped))
	if err != nil {
		return nil, err
	}

	var resp DNSPreferences
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SetDNSPreferences replaces the DNS preferences for a tailnet, specifically, the MagicDNS setting. Note that MagicDNS
// is dependent on DNS servers.
func (c *Client) SetDNSPreferences(ctx context.Context, preferences DNSPreferences) error {
	const uriFmt = "/api/v2/tailnet/%s/dns/preferences"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnetPathEscaped), requestBody(preferences))
	if err != nil {
		return nil
	}

	return c.performRequest(req, nil)
}

type (
	DeviceRoutes struct {
		Advertised []string `json:"advertisedRoutes"`
		Enabled    []string `json:"enabledRoutes"`
	}
)

// SetDeviceSubnetRoutes sets which subnet routes are enabled to be routed by a device by replacing the existing list
// of subnet routes with the supplied routes. Routes can be enabled without a device advertising them (e.g. for preauth).
func (c *Client) SetDeviceSubnetRoutes(ctx context.Context, deviceID string, routes []string) error {
	const uriFmt = "/api/v2/device/%s/routes"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, deviceID), requestBody(map[string][]string{
		"routes": routes,
	}))
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// DeviceSubnetRoutes Retrieves the list of subnet routes that a device is advertising, as well as those that are
// enabled for it. Enabled routes are not necessarily advertised (e.g. for pre-enabling), and likewise, advertised
// routes are not necessarily enabled.
func (c *Client) DeviceSubnetRoutes(ctx context.Context, deviceID string) (*DeviceRoutes, error) {
	const uriFmt = "/api/v2/device/%s/routes"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, deviceID))
	if err != nil {
		return nil, err
	}

	var resp DeviceRoutes
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
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

type Device struct {
	Addresses                 []string `json:"addresses"`
	Name                      string   `json:"name"`
	ID                        string   `json:"id"`
	Authorized                bool     `json:"authorized"`
	User                      string   `json:"user"`
	Tags                      []string `json:"tags"`
	KeyExpiryDisabled         bool     `json:"keyExpiryDisabled"`
	BlocksIncomingConnections bool     `json:"blocksIncomingConnections"`
	ClientVersion             string   `json:"clientVersion"`
	Created                   Time     `json:"created"`
	Expires                   Time     `json:"expires"`
	Hostname                  string   `json:"hostname"`
	IsExternal                bool     `json:"isExternal"`
	LastSeen                  Time     `json:"lastSeen"`
	MachineKey                string   `json:"machineKey"`
	NodeKey                   string   `json:"nodeKey"`
	OS                        string   `json:"os"`
	UpdateAvailable           bool     `json:"updateAvailable"`
}

// Devices lists the devices in a tailnet.
func (c *Client) Devices(ctx context.Context) ([]Device, error) {
	const uriFmt = "/api/v2/tailnet/%s/devices"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnetPathEscaped))
	if err != nil {
		return nil, err
	}

	resp := make(map[string][]Device)
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return resp["devices"], nil
}

// AuthorizeDevice marks the specified device identifier as authorized to join the tailnet.
func (c *Client) AuthorizeDevice(ctx context.Context, deviceID string) error {
	return c.SetDeviceAuthorized(ctx, deviceID, true)
}

// SetDeviceAuthorized marks the specified device as authorized or not.
func (c *Client) SetDeviceAuthorized(ctx context.Context, deviceID string, authorized bool) error {
	const uriFmt = "/api/v2/device/%s/authorized"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, deviceID), requestBody(map[string]bool{
		"authorized": authorized,
	}))
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// DeleteDevice deletes the device given its deviceID.
func (c *Client) DeleteDevice(ctx context.Context, deviceID string) error {
	const uriFmt = "/api/v2/device/%s"
	req, err := c.buildRequest(ctx, http.MethodDelete, fmt.Sprintf(uriFmt, deviceID))
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

type (
	// KeyCapabilities type describes the capabilities of an authentication key.
	KeyCapabilities struct {
		Devices struct {
			Create struct {
				Reusable      bool     `json:"reusable"`
				Ephemeral     bool     `json:"ephemeral"`
				Tags          []string `json:"tags"`
				Preauthorized bool     `json:"preauthorized"`
			} `json:"create"`
		} `json:"devices"`
	}

	// CreateKeyRequest type describes the definition of an authentication key to create.
	CreateKeyRequest struct {
		Capabilities  KeyCapabilities `json:"capabilities"`
		ExpirySeconds int64           `json:"expirySeconds"`
		Description   string          `json:"description"`
	}

	// CreateKeyOption type is a function that is used to modify a CreateKeyRequest.
	CreateKeyOption func(c *CreateKeyRequest) error

	// Key type describes an authentication key within the tailnet.
	Key struct {
		ID           string          `json:"id"`
		Key          string          `json:"key"`
		Description  string          `json:"description"`
		Created      time.Time       `json:"created"`
		Expires      time.Time       `json:"expires"`
		Revoked      time.Time       `json:"revoked"`
		Invalid      bool            `json:"invalid"`
		Capabilities KeyCapabilities `json:"capabilities"`
	}
)

// WithKeyExpiry sets how long the key is valid for.
func WithKeyExpiry(e time.Duration) CreateKeyOption {
	return func(c *CreateKeyRequest) error {
		c.ExpirySeconds = int64(e.Seconds())
		return nil
	}
}

// WithKeyDescription sets the description for the key.
func WithKeyDescription(desc string) CreateKeyOption {
	return func(c *CreateKeyRequest) error {
		c.Description = desc
		return nil
	}
}

// CreateKey creates a new authentication key with the capabilities selected via the KeyCapabilities type. Returns
// the generated key if successful.
func (c *Client) CreateKey(ctx context.Context, capabilities KeyCapabilities, opts ...CreateKeyOption) (Key, error) {
	const uriFmt = "/api/v2/tailnet/%s/keys"

	ckr := &CreateKeyRequest{
		Capabilities: capabilities,
	}

	for _, opt := range opts {
		if err := opt(ckr); err != nil {
			return Key{}, err
		}
	}

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnetPathEscaped), requestBody(ckr))
	if err != nil {
		return Key{}, err
	}

	var key Key
	return key, c.performRequest(req, &key)
}

// GetKey returns all information on a key whose identifier matches the one provided. This will not return the
// authentication key itself, just the metadata.
func (c *Client) GetKey(ctx context.Context, id string) (Key, error) {
	const uriFmt = "/api/v2/tailnet/%s/keys/%s"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnetPathEscaped, id))
	if err != nil {
		return Key{}, err
	}

	var key Key
	return key, c.performRequest(req, &key)
}

// Keys returns all keys within the tailnet. The only fields set for each key will be its identifier. The keys returned
// are relative to the user that owns the API key used to authenticate the client.
func (c *Client) Keys(ctx context.Context) ([]Key, error) {
	const uriFmt = "/api/v2/tailnet/%s/keys"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnetPathEscaped))
	if err != nil {
		return nil, err
	}

	resp := make(map[string][]Key)
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return resp["keys"], nil
}

// DeleteKey removes an authentication key from the tailnet.
func (c *Client) DeleteKey(ctx context.Context, id string) error {
	const uriFmt = "/api/v2/tailnet/%s/keys/%s"

	req, err := c.buildRequest(ctx, http.MethodDelete, fmt.Sprintf(uriFmt, c.tailnetPathEscaped, id))
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// SetDeviceTags updates the tags of a target device.
func (c *Client) SetDeviceTags(ctx context.Context, deviceID string, tags []string) error {
	const uriFmt = "/api/v2/device/%s/tags"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, deviceID), requestBody(map[string][]string{
		"tags": tags,
	}))
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

type (
	// DeviceKey type represents the properties of the key of an individual device within
	// the tailnet.
	DeviceKey struct {
		KeyExpiryDisabled bool `json:"keyExpiryDisabled"` // Whether or not this device's key will ever expire.
	}
)

// SetDeviceKey updates the properties of a device's key.
func (c *Client) SetDeviceKey(ctx context.Context, deviceID string, key DeviceKey) error {
	const uriFmt = "/api/v2/device/%s/key"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, deviceID), requestBody(key))
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// SetDeviceIPv4Address sets the Tailscale IPv4 address of the device.
func (c *Client) SetDeviceIPv4Address(ctx context.Context, deviceID string, ipv4Address string) error {
	const uriFmt = "/api/v2/device/%s/ip"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, deviceID), requestBody(map[string]string{
		"ipv4": ipv4Address,
	}))
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

const (
	WebhookEmptyProviderType      WebhookProviderType = ""
	WebhookSlackProviderType      WebhookProviderType = "slack"
	WebhookMattermostProviderType WebhookProviderType = "mattermost"
	WebhookGoogleChatProviderType WebhookProviderType = "googlechat"
	WebhookDiscordProviderType    WebhookProviderType = "discord"
)

const (
	WebhookNodeCreated                    WebhookSubscriptionType = "nodeCreated"
	WebhookNodeNeedsApproval              WebhookSubscriptionType = "nodeNeedsApproval"
	WebhookNodeApproved                   WebhookSubscriptionType = "nodeApproved"
	WebhookNodeKeyExpiringInOneDay        WebhookSubscriptionType = "nodeKeyExpiringInOneDay"
	WebhookNodeKeyExpired                 WebhookSubscriptionType = "nodeKeyExpired"
	WebhookNodeDeleted                    WebhookSubscriptionType = "nodeDeleted"
	WebhookPolicyUpdate                   WebhookSubscriptionType = "policyUpdate"
	WebhookUserCreated                    WebhookSubscriptionType = "userCreated"
	WebhookUserNeedsApproval              WebhookSubscriptionType = "userNeedsApproval"
	WebhookUserSuspended                  WebhookSubscriptionType = "userSuspended"
	WebhookUserRestored                   WebhookSubscriptionType = "userRestored"
	WebhookUserDeleted                    WebhookSubscriptionType = "userDeleted"
	WebhookUserApproved                   WebhookSubscriptionType = "userApproved"
	WebhookUserRoleUpdated                WebhookSubscriptionType = "userRoleUpdated"
	WebhookSubnetIPForwardingNotEnabled   WebhookSubscriptionType = "subnetIPForwardingNotEnabled"
	WebhookExitNodeIPForwardingNotEnabled WebhookSubscriptionType = "exitNodeIPForwardingNotEnabled"
)

type (
	// WebhookProviderType defines the provider type for a Webhook destination.
	WebhookProviderType string

	// WebhookSubscriptionType defines events in tailscale to subscribe a Webhook to.
	WebhookSubscriptionType string

	// Webhook type defines a webhook endpoint within a tailnet.
	Webhook struct {
		EndpointID       string                    `json:"endpointId"`
		EndpointURL      string                    `json:"endpointUrl"`
		ProviderType     WebhookProviderType       `json:"providerType"`
		CreatorLoginName string                    `json:"creatorLoginName"`
		Created          time.Time                 `json:"created"`
		LastModified     time.Time                 `json:"lastModified"`
		Subscriptions    []WebhookSubscriptionType `json:"subscriptions"`
		// Secret is only populated on Webhook creation and after secret rotation.
		Secret *string `json:"secret,omitempty"`
	}

	// CreateWebhookRequest type describes the configuration for creating a Webhook.
	CreateWebhookRequest struct {
		EndpointURL   string                    `json:"endpointUrl"`
		ProviderType  WebhookProviderType       `json:"providerType"`
		Subscriptions []WebhookSubscriptionType `json:"subscriptions"`
	}
)

// CreateWebhook creates a new webhook with the specifications provided in the CreateWebhookRequest.
// Returns a Webhook if successful.
func (c *Client) CreateWebhook(ctx context.Context, request CreateWebhookRequest) (*Webhook, error) {
	const uriFmt = "/api/v2/tailnet/%s/webhooks"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnetPathEscaped), requestBody(request))
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	return &webhook, c.performRequest(req, &webhook)
}

// Webhooks lists the webhooks in a tailnet.
func (c *Client) Webhooks(ctx context.Context) ([]Webhook, error) {
	const uriFmt = "/api/v2/tailnet/%s/webhooks"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnetPathEscaped))
	if err != nil {
		return nil, err
	}

	resp := make(map[string][]Webhook)
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return resp["webhooks"], nil
}

// Webhook retrieves a specific webhook.
func (c *Client) Webhook(ctx context.Context, endpointID string) (*Webhook, error) {
	const uriFmt = "/api/v2/webhooks/%s"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, endpointID))
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	return &webhook, c.performRequest(req, &webhook)
}

// UpdateWebhook updates an existing webhook's subscriptions.
// Returns a Webhook on success.
func (c *Client) UpdateWebhook(ctx context.Context, endpointID string, subscriptions []WebhookSubscriptionType) (*Webhook, error) {
	const uriFmt = "/api/v2/webhooks/%s"

	req, err := c.buildRequest(ctx, http.MethodPatch, fmt.Sprintf(uriFmt, endpointID), requestBody(map[string][]WebhookSubscriptionType{
		"subscriptions": subscriptions,
	}))
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	return &webhook, c.performRequest(req, &webhook)
}

// DeleteWebhook deletes a specific webhook.
func (c *Client) DeleteWebhook(ctx context.Context, endpointID string) error {
	const uriFmt = "/api/v2/webhooks/%s"

	req, err := c.buildRequest(ctx, http.MethodDelete, fmt.Sprintf(uriFmt, endpointID))
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// TestWebhook queues a test event to be sent to a specific webhook.
// Sending the test event is an asynchronous operation which will
// typically happen a few seconds after using this method.
func (c *Client) TestWebhook(ctx context.Context, endpointID string) error {
	const uriFmt = "/api/v2/webhooks/%s/test"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, endpointID))
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// RotateWebhookSecret rotates the secret associated with a webhook.
// A new secret will be generated and set on the returned Webhook.
func (c *Client) RotateWebhookSecret(ctx context.Context, endpointID string) (*Webhook, error) {
	const uriFmt = "/api/v2/webhooks/%s/rotate"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, endpointID))
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	return &webhook, c.performRequest(req, &webhook)
}

const (
	ContactAccount  ContactType = "account"
	ContactSupport  ContactType = "support"
	ContactSecurity ContactType = "security"
)

type (
	// ContactType defines the type of contact.
	ContactType string

	// Contacts type defines the object returned when retrieving contacts.
	Contacts struct {
		Account  Contact `json:"account"`
		Support  Contact `json:"support"`
		Security Contact `json:"security"`
	}

	// Contact type defines the structure of an individual contact for the tailnet.
	Contact struct {
		Email string `json:"email"`
		// FallbackEmail is the email used when Email has not been verified.
		FallbackEmail string `json:"fallbackEmail,omitempty"`
		// NeedsVerification is true if Email needs to be verified.
		NeedsVerification bool `json:"needsVerification"`
	}

	// UpdateContactRequest type defines the structure of a request to update a Contact.
	UpdateContactRequest struct {
		Email *string `json:"email,omitempty"`
	}
)

// Contacts retieves the contact information for a tailnet.
func (c *Client) Contacts(ctx context.Context) (*Contacts, error) {
	const uriFmt = "/api/v2/tailnet/%s/contacts"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnetPathEscaped))
	if err != nil {
		return nil, err
	}

	var contacts Contacts
	return &contacts, c.performRequest(req, &contacts)
}

// UpdateContact updates the email for the specified ContactType within the tailnet.
// If the email address changes, the system will send a verification email to confirm the change.
func (c *Client) UpdateContact(ctx context.Context, contactType ContactType, contact UpdateContactRequest) error {
	const uriFmt = "/api/v2/tailnet/%s/contacts/%s"

	req, err := c.buildRequest(ctx, http.MethodPatch, fmt.Sprintf(uriFmt, c.tailnetPathEscaped, contactType), requestBody(contact))
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
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
