package tsclient

import (
	"context"
	"fmt"
	"net/http"
)

// PolicyFileResource provides access to https://tailscale.com/api#tag/policyfile.
type PolicyFileResource struct {
	*Client
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

// Get retrieves the [ACL] that is currently set for the tailnet.
func (pr *PolicyFileResource) Get(ctx context.Context) (*ACL, error) {
	req, err := pr.buildRequest(ctx, http.MethodGet, pr.buildTailnetURL("acl"))
	if err != nil {
		return nil, err
	}

	return body[ACL](pr, req)
}

// Raw retrieves the [ACL] that is currently set for the tailnet as a HuJSON string.
func (pr *PolicyFileResource) Raw(ctx context.Context) (string, error) {
	req, err := pr.buildRequest(ctx, http.MethodGet, pr.buildTailnetURL("acl"), requestContentType("application/hujson"))
	if err != nil {
		return "", err
	}

	var resp []byte
	if err = pr.do(req, &resp); err != nil {
		return "", err
	}

	return string(resp), nil
}

// Set sets the [ACL] for the tailnet. acl can either be an [ACL], or a HuJSON string.
// etag is an optional value that, if supplied, will be used in the "If-Match" HTTP request header.
func (pr *PolicyFileResource) Set(ctx context.Context, acl any, etag string) error {
	headers := make(map[string]string)
	if etag != "" {
		headers["If-Match"] = fmt.Sprintf("%q", etag)
	}

	reqOpts := []requestOption{
		requestHeaders(headers),
		requestBody(acl),
	}
	switch v := acl.(type) {
	case ACL:
	case string:
		reqOpts = append(reqOpts, requestContentType("application/hujson"))
	default:
		return fmt.Errorf("expected ACL content as a string or as ACL struct; got %T", v)
	}

	req, err := pr.buildRequest(ctx, http.MethodPost, pr.buildTailnetURL("acl"), reqOpts...)
	if err != nil {
		return err
	}

	return pr.do(req, nil)
}

// Validate validates the provided ACL via the API. acl can either be an [ACL], or a HuJSON string.
func (pr *PolicyFileResource) Validate(ctx context.Context, acl any) error {
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

	req, err := pr.buildRequest(ctx, http.MethodPost, pr.buildTailnetURL("acl", "validate"), reqOpts...)
	if err != nil {
		return err
	}

	var response APIError
	if err := pr.do(req, &response); err != nil {
		return err
	}
	if response.Message != "" {
		return fmt.Errorf("ACL validation failed: %s; %v", response.Message, response.Data)
	}
	return nil
}
