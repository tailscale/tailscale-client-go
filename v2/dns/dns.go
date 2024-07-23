// Package dns provides a client to the Tailscale API's DNS functions.
package dns

import (
	"context"
	"fmt"
	"net/http"

	"github.com/tailscale/tailscale-client-go/v2"
)

type Client struct {
	tailscale.Client
}

func With(c tailscale.Client) *Client {
	return &Client{c}
}

// SetDNSSearchPaths replaces the list of search paths with the list supplied by the user and returns an error otherwise.
func (c *Client) SetDNSSearchPaths(ctx context.Context, searchPaths []string) error {
	const uriFmt = "/api/v2/tailnet/%v/dns/searchpaths"

	req, err := c.BuildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.Tailnet()), tailscale.RequestBody(map[string][]string{
		"searchPaths": searchPaths,
	}))
	if err != nil {
		return err
	}

	return c.PerformRequest(req, nil)
}

// DNSSearchPaths retrieves the list of search paths that is currently set for the given tailnet.
func (c *Client) DNSSearchPaths(ctx context.Context) ([]string, error) {
	const uriFmt = "/api/v2/tailnet/%v/dns/searchpaths"

	req, err := c.BuildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.Tailnet()))
	if err != nil {
		return nil, err
	}

	resp := make(map[string][]string)
	if err = c.PerformRequest(req, &resp); err != nil {
		return nil, err
	}

	return resp["searchPaths"], nil
}

// SetDNSNameservers replaces the list of DNS nameservers for the given tailnet with the list supplied by the user. Note
// that changing the list of DNS nameservers may also affect the status of MagicDNS (if MagicDNS is on).
func (c *Client) SetDNSNameservers(ctx context.Context, dns []string) error {
	const uriFmt = "/api/v2/tailnet/%v/dns/nameservers"

	req, err := c.BuildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.Tailnet()), tailscale.RequestBody(map[string][]string{
		"dns": dns,
	}))
	if err != nil {
		return err
	}

	return c.PerformRequest(req, nil)
}

// DNSNameservers lists the DNS nameservers for a tailnet
func (c *Client) DNSNameservers(ctx context.Context) ([]string, error) {
	const uriFmt = "/api/v2/tailnet/%v/dns/nameservers"

	req, err := c.BuildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.Tailnet()))
	if err != nil {
		return nil, err
	}

	resp := make(map[string][]string)
	if err = c.PerformRequest(req, &resp); err != nil {
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

	req, err := c.BuildRequest(ctx, http.MethodPatch, fmt.Sprintf(uriFmt, c.Tailnet()), tailscale.RequestBody(request))
	if err != nil {
		return nil, err
	}

	var resp SplitDnsResponse
	if err = c.PerformRequest(req, &resp); err != nil {
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

	req, err := c.BuildRequest(ctx, http.MethodPut, fmt.Sprintf(uriFmt, c.Tailnet()), tailscale.RequestBody(request))
	if err != nil {
		return err
	}

	return c.PerformRequest(req, nil)
}

// SplitDNS retrieves the split DNS configuration for a tailnet.
func (c *Client) SplitDNS(ctx context.Context) (SplitDnsResponse, error) {
	const uriFmt = "/api/v2/tailnet/%v/dns/split-dns"

	req, err := c.BuildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.Tailnet()))
	if err != nil {
		return nil, err
	}

	var resp SplitDnsResponse
	if err = c.PerformRequest(req, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

type Preferences struct {
	MagicDNS bool `json:"magicDNS"`
}

// DNSPreferences retrieves the DNS preferences that are currently set for the given tailnet. Supply the tailnet of
// interest in the path.
func (c *Client) DNSPreferences(ctx context.Context) (*Preferences, error) {
	const uriFmt = "/api/v2/tailnet/%s/dns/preferences"

	req, err := c.BuildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.Tailnet()))
	if err != nil {
		return nil, err
	}

	var resp Preferences
	if err = c.PerformRequest(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SetDNSPreferences replaces the DNS preferences for a tailnet, specifically, the MagicDNS setting. Note that MagicDNS
// is dependent on DNS servers.
func (c *Client) SetDNSPreferences(ctx context.Context, preferences Preferences) error {
	const uriFmt = "/api/v2/tailnet/%s/dns/preferences"

	req, err := c.BuildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.Tailnet()), tailscale.RequestBody(preferences))
	if err != nil {
		return nil
	}

	return c.PerformRequest(req, nil)
}
