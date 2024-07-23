package tailscale

import (
	"context"
	"fmt"
	"net/http"
)

// SetDNSSearchPaths replaces the list of search paths with the list supplied by the user and returns an error otherwise.
func (c *Client) SetDNSSearchPaths(ctx context.Context, searchPaths []string) error {
	const uriFmt = "/api/v2/tailnet/%v/dns/searchpaths"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnet), requestBody(map[string][]string{
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

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnet))
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

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnet), requestBody(map[string][]string{
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

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnet))
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

	req, err := c.buildRequest(ctx, http.MethodPatch, fmt.Sprintf(uriFmt, c.tailnet), requestBody(request))
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

	req, err := c.buildRequest(ctx, http.MethodPut, fmt.Sprintf(uriFmt, c.tailnet), requestBody(request))
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// SplitDNS retrieves the split DNS configuration for a tailnet.
func (c *Client) SplitDNS(ctx context.Context) (SplitDnsResponse, error) {
	const uriFmt = "/api/v2/tailnet/%v/dns/split-dns"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnet))
	if err != nil {
		return nil, err
	}

	var resp SplitDnsResponse
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

type DNSPreferences struct {
	MagicDNS bool `json:"magicDNS"`
}

// DNSPreferences retrieves the DNS preferences that are currently set for the given tailnet. Supply the tailnet of
// interest in the path.
func (c *Client) DNSPreferences(ctx context.Context) (*DNSPreferences, error) {
	const uriFmt = "/api/v2/tailnet/%s/dns/preferences"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnet))
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

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnet), requestBody(preferences))
	if err != nil {
		return nil
	}

	return c.performRequest(req, nil)
}
