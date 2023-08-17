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
	// The Client type is used to perform actions against the Tailscale API.
	Client struct {
		apiKey  string
		http    *http.Client
		baseURL *url.URL
		tailnet string
	}

	// The APIError type describes an error as returned by the Tailscale API.
	APIError struct {
		Message string         `json:"message"`
		Data    []APIErrorData `json:"data"`
		status  int
	}

	// The APIErrorData type describes elements of the data field within errors returned by the Tailscale API.
	APIErrorData struct {
		User   string   `json:"user"`
		Errors []string `json:"errors"`
	}

	// The ClientOption type is a function that is used to modify a Client.
	ClientOption func(c *Client) error
)

const baseURL = "https://api.tailscale.com"
const contentType = "application/json"
const defaultHttpClientTimeout = time.Minute

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
		baseURL: u,
		tailnet: tailnet,
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

// TODO: consider setting `headers` and `body` via opts to decrease the number of arguments.
func (c *Client) buildRequest(ctx context.Context, method, uri string, headers map[string]string, body interface{}) (*http.Request, error) {
	u, err := c.baseURL.Parse(uri)
	if err != nil {
		return nil, err
	}

	var bodyBytes []byte
	if body != nil {
		bodyBytes, err = json.MarshalIndent(body, "", " ")
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	switch {
	case body == nil:
		req.Header.Set("Accept", contentType)
	default:
		req.Header.Set("Content-Type", contentType)
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

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnet), nil, map[string][]string{
		"searchPaths": searchPaths,
	})
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// DNSSearchPaths retrieves the list of search paths that is currently set for the given tailnet.
func (c *Client) DNSSearchPaths(ctx context.Context) ([]string, error) {
	const uriFmt = "/api/v2/tailnet/%v/dns/searchpaths"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnet), nil, nil)
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

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnet), nil, map[string][]string{
		"dns": dns,
	})
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// DNSNameservers lists the DNS nameservers for a tailnet
func (c *Client) DNSNameservers(ctx context.Context) ([]string, error) {
	const uriFmt = "/api/v2/tailnet/%v/dns/nameservers"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnet), nil, nil)
	if err != nil {
		return nil, err
	}

	resp := make(map[string][]string)
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return resp["dns"], nil
}

type (
	ACL struct {
		ACLs                []ACLEntry          `json:"acls,omitempty" hujson:"ACLs,omitempty"`
		AutoApprovers       *ACLAutoApprovers   `json:"autoapprovers,omitempty" hujson:"AutoApprovers,omitempty"`
		Groups              map[string][]string `json:"groups,omitempty" hujson:"Groups,omitempty"`
		Hosts               map[string]string   `json:"hosts,omitempty" hujson:"Hosts,omitempty"`
		TagOwners           map[string][]string `json:"tagowners,omitempty" hujson:"TagOwners,omitempty"`
		DERPMap             *ACLDERPMap         `json:"derpMap,omitempty" hujson:"DerpMap,omitempty"`
		Tests               []ACLTest           `json:"tests,omitempty" hujson:"Tests,omitempty"`
		SSH                 []ACLSSH            `json:"ssh,omitempty" hujson:"SSH,omitempty"`
		NodeAttrs           []NodeAttrGrant     `json:"nodeAttrs,omitempty" hujson:"NodeAttrs,omitempty"`
		DisableIPv4         bool                `json:"disableIPv4,omitempty" hujson:"DisableIPv4,omitempty"`
		OneCGNATRoute       string              `json:"oneCGNATRoute,omitempty" hujson:"OneCGNATRoute,omitempty"`
		RandomizeClientPort bool                `json:"randomizeClientPort,omitempty" hujson:"RandomizeClientPort,omitempty"`

		// As of Aug 2023 these fields are experimental and subject to change.
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

		// Experimental.
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
		Target []string `json:"target,omitempty" hujson:"Target,omitempty"`
		Attr   []string `json:"attr,omitempty" hujson:"Attr,omitempty"`
	}
)

// ACL retrieves the ACL that is currently set for the given tailnet.
func (c *Client) ACL(ctx context.Context) (*ACL, error) {
	const uriFmt = "/api/v2/tailnet/%s/acl"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnet), nil, nil)
	if err != nil {
		return nil, err
	}

	var resp ACL
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
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

// SetACL sets the ACL for the given tailnet.
func (c *Client) SetACL(ctx context.Context, acl ACL, opts ...SetACLOption) error {
	const uriFmt = "/api/v2/tailnet/%s/acl"

	p := &setACLParams{headers: make(map[string]string)}
	for _, opt := range opts {
		opt(p)
	}

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnet), p.headers, acl)
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// ValidateACL validates the provided ACL via the API.
func (c *Client) ValidateACL(ctx context.Context, acl ACL) error {
	const uriFmt = "/api/v2/tailnet/%s/acl/validate"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnet), nil, acl)
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

type DNSPreferences struct {
	MagicDNS bool `json:"magicDNS"`
}

// DNSPreferences retrieves the DNS preferences that are currently set for the given tailnet. Supply the tailnet of
// interest in the path.
func (c *Client) DNSPreferences(ctx context.Context) (*DNSPreferences, error) {
	const uriFmt = "/api/v2/tailnet/%s/dns/preferences"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnet), nil, nil)
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

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnet), nil, preferences)
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

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, deviceID), nil, map[string][]string{
		"routes": routes,
	})
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

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, deviceID), nil, nil)
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

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnet), nil, nil)
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

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, deviceID), nil, map[string]bool{
		"authorized": authorized,
	})
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// DeleteDevice deletes the device given its deviceID.
func (c *Client) DeleteDevice(ctx context.Context, deviceID string) error {
	const uriFmt = "/api/v2/device/%s"
	req, err := c.buildRequest(ctx, http.MethodDelete, fmt.Sprintf(uriFmt, deviceID), nil, nil)
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

type (
	// The KeyCapabilities type describes the capabilities of an authentication key.
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

	// The CreateKeyRequest type describes the definition of an authentication key to create.
	CreateKeyRequest struct {
		Capabilities  KeyCapabilities `json:"capabilities"`
		ExpirySeconds int64           `json:"expirySeconds"`
		Description   string          `json:"description"`
	}

	// The CreateKeyOption type is a function that is used to modify a CreateKeyRequest.
	CreateKeyOption func(c *CreateKeyRequest) error

	// The Key type describes an authentication key within the tailnet.
	Key struct {
		ID           string          `json:"id"`
		Key          string          `json:"key"`
		Description  string          `json:"description"`
		Created      time.Time       `json:"created"`
		Expires      time.Time       `json:"expires"`
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

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnet), nil, ckr)
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

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnet, id), nil, nil)
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

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnet), nil, nil)
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

	req, err := c.buildRequest(ctx, http.MethodDelete, fmt.Sprintf(uriFmt, c.tailnet, id), nil, nil)
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// SetDeviceTags updates the tags of a target device.
func (c *Client) SetDeviceTags(ctx context.Context, deviceID string, tags []string) error {
	const uriFmt = "/api/v2/device/%s/tags"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, deviceID), nil, map[string][]string{
		"tags": tags,
	})
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

type (
	// The DeviceKey type represents the properties of the key of an individual device within
	// the tailnet.
	DeviceKey struct {
		KeyExpiryDisabled bool `json:"keyExpiryDisabled"` // Whether or not this device's key will ever expire.
	}
)

// SetDeviceKey updates the properties of a device's key.
func (c *Client) SetDeviceKey(ctx context.Context, deviceID string, key DeviceKey) error {
	const uriFmt = "/api/v2/device/%s/key"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, deviceID), nil, key)
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

// The Duration type wraps a time.Duration, allowing it to be JSON marshalled as a string like "20h" rather than
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
