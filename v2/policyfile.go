package tailscale

import (
	"context"
	"fmt"
	"net/http"
)

type PolicyFile struct {
	*Client
}

func (c *Client) PolicyFile() *PolicyFile {
	return c.policyFile
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
func (c *PolicyFile) ACL(ctx context.Context) (*ACL, error) {
	const uriFmt = "/api/v2/tailnet/%s/acl"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnet))
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
func (c *PolicyFile) RawACL(ctx context.Context) (string, error) {
	const uriFmt = "/api/v2/tailnet/%s/acl"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnet), requestContentType("application/hujson"))
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
func (c *PolicyFile) SetACL(ctx context.Context, acl any, opts ...SetACLOption) error {
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

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnet), reqOpts...)
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// ValidateACL validates the provided ACL via the API. `acl` can either be an [ACL],
// or a HuJSON string.
func (c *PolicyFile) ValidateACL(ctx context.Context, acl any) error {
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

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnet), reqOpts...)
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
