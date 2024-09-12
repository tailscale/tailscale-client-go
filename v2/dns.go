// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tsclient

import (
	"context"
	"net/http"
)

// DNSResource provides access to https://tailscale.com/api#tag/dns.
type DNSResource struct {
	*Client
}

type (
	// SplitDNSRequest is a map from domain names to a list of nameservers.
	SplitDNSRequest map[string][]string

	// SplitDNSResponse is a map from domain names to a list of nameservers.
	SplitDNSResponse SplitDNSRequest

	DNSPreferences struct {
		MagicDNS bool `json:"magicDNS"`
	}
)

// SetSearchPaths replaces the list of search paths with the list supplied by the user and returns an error otherwise.
func (dr *DNSResource) SetSearchPaths(ctx context.Context, searchPaths []string) error {
	req, err := dr.buildRequest(ctx, http.MethodPost, dr.buildTailnetURL("dns", "searchpaths"), requestBody(map[string][]string{
		"searchPaths": searchPaths,
	}))
	if err != nil {
		return err
	}

	return dr.do(req, nil)
}

// SearchPaths retrieves the list of search paths that is currently set for the given tailnet.
func (dr *DNSResource) SearchPaths(ctx context.Context) ([]string, error) {
	req, err := dr.buildRequest(ctx, http.MethodGet, dr.buildTailnetURL("dns", "searchpaths"))
	if err != nil {
		return nil, err
	}

	resp := make(map[string][]string)
	if err = dr.do(req, &resp); err != nil {
		return nil, err
	}

	return resp["searchPaths"], nil
}

// SetNameservers replaces the list of DNS nameservers for the given tailnet with the list supplied by the user. Note
// that changing the list of DNS nameservers may also affect the status of MagicDNS (if MagicDNS is on).
func (dr *DNSResource) SetNameservers(ctx context.Context, dns []string) error {
	req, err := dr.buildRequest(ctx, http.MethodPost, dr.buildTailnetURL("dns", "nameservers"), requestBody(map[string][]string{
		"dns": dns,
	}))
	if err != nil {
		return err
	}

	return dr.do(req, nil)
}

// Nameservers lists the DNS nameservers for the tailnet
func (dr *DNSResource) Nameservers(ctx context.Context) ([]string, error) {
	req, err := dr.buildRequest(ctx, http.MethodGet, dr.buildTailnetURL("dns", "nameservers"))
	if err != nil {
		return nil, err
	}

	resp := make(map[string][]string)
	if err = dr.do(req, &resp); err != nil {
		return nil, err
	}

	return resp["dns"], nil
}

// UpdateSplitDNS updates the split DNS settings for the tailnet using the
// provided [SplitDNSRequest] object. This is a PATCH operation that performs
// partial updates of the underlying data structure.
//
// Mapping a domain to a nil slice in the request will unset the nameservers
// associated with that domain. Values provided for domains will overwrite the
// current value associated with the domain. Domains not included in the request
// will remain unchanged.
func (dr *DNSResource) UpdateSplitDNS(ctx context.Context, request SplitDNSRequest) (SplitDNSResponse, error) {
	req, err := dr.buildRequest(ctx, http.MethodPatch, dr.buildTailnetURL("dns", "split-dns"), requestBody(request))
	if err != nil {
		return nil, err
	}

	var resp SplitDNSResponse
	if err := dr.do(req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// SetSplitDNS sets the split DNS settings for the tailnet using the provided
// [SplitDNSRequest] object. This is a PUT operation that fully replaces the underlying
// data structure.
//
// Passing in an empty [SplitDNSRequest] will unset all split DNS mappings for the tailnet.
func (dr *DNSResource) SetSplitDNS(ctx context.Context, request SplitDNSRequest) error {
	req, err := dr.buildRequest(ctx, http.MethodPut, dr.buildTailnetURL("dns", "split-dns"), requestBody(request))
	if err != nil {
		return err
	}

	return dr.do(req, nil)
}

// SplitDNS retrieves the split DNS configuration for the tailnet.
func (dr *DNSResource) SplitDNS(ctx context.Context) (SplitDNSResponse, error) {
	req, err := dr.buildRequest(ctx, http.MethodGet, dr.buildTailnetURL("dns", "split-dns"))
	if err != nil {
		return nil, err
	}

	var resp SplitDNSResponse
	if err := dr.do(req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// Preferences retrieves the DNS preferences that are currently set for the given tailnet.
func (dr *DNSResource) Preferences(ctx context.Context) (*DNSPreferences, error) {
	req, err := dr.buildRequest(ctx, http.MethodGet, dr.buildTailnetURL("dns", "preferences"))
	if err != nil {
		return nil, err
	}

	return body[DNSPreferences](dr, req)
}

// SetPreferences replaces the DNS preferences for the tailnet, specifically, the MagicDNS setting. Note that MagicDNS
// is dependent on DNS servers.
func (dr *DNSResource) SetPreferences(ctx context.Context, preferences DNSPreferences) error {
	req, err := dr.buildRequest(ctx, http.MethodPost, dr.buildTailnetURL("dns", "preferences"), requestBody(preferences))
	if err != nil {
		return nil
	}

	return dr.do(req, nil)
}
