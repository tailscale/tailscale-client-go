package tsclient

import (
	"context"
	"net/http"
)

// TailnetDevicePostureResource provides access to https://tailscale.com/api#tag/deviceposture.
type TailnetDevicePostureResource struct {
	*TailnetClient
}

const (
	PostureIntegrationProviderFalcon      PostureIntegrationProvider = "falcon"
	PostureIntegrationProviderIntune      PostureIntegrationProvider = "intune"
	PostureIntegrationProviderJamfPro     PostureIntegrationProvider = "jamfpro"
	PostureIntegrationProviderKandji      PostureIntegrationProvider = "kandji"
	PostureIntegrationProviderKolide      PostureIntegrationProvider = "kolide"
	PostureIntegrationProviderSentinelOne PostureIntegrationProvider = "sentinelone"
)

type (
	// PostureIntegrationProvider identifies a supported posture integration data provider.
	PostureIntegrationProvider string

	// PostureIntegration is a configured posture integration.
	PostureIntegration struct {
		ID       string                     `json:"id,omitempty"`
		Provider PostureIntegrationProvider `json:"provider,omitempty"`
		CloudID  string                     `json:"cloudId,omitempty"`
		ClientID string                     `json:"clientId,omitempty"`
		TenantID string                     `json:"tenantId,omitempty"`
	}

	// CreatePostureIntegrationRequest is a request to create a posture integration.
	CreatePostureIntegrationRequest struct {
		Provider     PostureIntegrationProvider `json:"provider,omitempty"`
		CloudID      string                     `json:"cloudId,omitempty"`
		ClientID     string                     `json:"clientId,omitempty"`
		TenantID     string                     `json:"tenantId,omitempty"`
		ClientSecret string                     `json:"clientSecret,omitempty"`
	}

	// UpdatePostureIntegrationRequest is a request to update a posture integration.
	UpdatePostureIntegrationRequest struct {
		CloudID  string `json:"cloudId,omitempty"`
		ClientID string `json:"clientId,omitempty"`
		TenantID string `json:"tenantId,omitempty"`
		// ClientSecret may be omitted to preserve the existing value
		ClientSecret *string `json:"clientSecret,omitempty"`
	}
)

// List lists every configured [PostureIntegration].
func (pr *TailnetDevicePostureResource) ListIntegrations(ctx context.Context) ([]PostureIntegration, error) {
	req, err := pr.buildRequest(ctx, http.MethodGet, pr.buildTailnetURL("posture", "integrations"))
	if err != nil {
		return nil, err
	}

	m := make(map[string][]PostureIntegration)
	err = pr.do(req, &m)
	if err != nil {
		return nil, err
	}

	return m["integrations"], nil
}

// CreateIntegration creates a new posture integration, returning the resulting [PostureIntegration].
func (pr *TailnetDevicePostureResource) CreateIntegration(ctx context.Context, intg CreatePostureIntegrationRequest) (*PostureIntegration, error) {
	req, err := pr.buildRequest(ctx, http.MethodPost, pr.buildTailnetURL("posture", "integrations"), requestBody(intg))
	if err != nil {
		return nil, err
	}

	var resp PostureIntegration
	return &resp, pr.do(req, &resp)
}

// UpdateIntegration updates the existing posture integration identified by id, returning the resulting [PostureIntegration].
func (pr *TailnetDevicePostureResource) UpdateIntegration(ctx context.Context, id string, intg UpdatePostureIntegrationRequest) (*PostureIntegration, error) {
	req, err := pr.buildRequest(ctx, http.MethodPatch, pr.buildURL("posture", "integrations", id), requestBody(intg))
	if err != nil {
		return nil, err
	}

	var resp PostureIntegration
	return &resp, pr.do(req, &resp)
}

// DeleteIntegration deletes the posture integration identified by id.
func (pr *TailnetDevicePostureResource) DeleteIntegration(ctx context.Context, id string) error {
	req, err := pr.buildRequest(ctx, http.MethodDelete, pr.buildURL("posture", "integrations", id))
	if err != nil {
		return err
	}

	return pr.do(req, nil)
}

// GetIntegration gets the posture integration identified by id.
func (pr *TailnetDevicePostureResource) GetIntegration(ctx context.Context, id string) (*PostureIntegration, error) {
	req, err := pr.buildRequest(ctx, http.MethodGet, pr.buildURL("posture", "integrations", id))
	if err != nil {
		return nil, err
	}

	var resp PostureIntegration
	return &resp, pr.do(req, &resp)
}
