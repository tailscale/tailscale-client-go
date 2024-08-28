package tsclient

import (
	"context"
	"net/http"
	"time"
)

// TailnetWebhooksResource provides access to https://tailscale.com/api#tag/webhooks.
type TailnetWebhooksResource struct {
	*TailnetClient
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

// Create creates a new [Webhook] with the specifications provided in the [CreateWebhookRequest].
// Returns the created [Webhook] if successful.
func (wr *TailnetWebhooksResource) Create(ctx context.Context, request CreateWebhookRequest) (*Webhook, error) {
	req, err := wr.buildRequest(ctx, http.MethodPost, wr.buildTailnetURL("webhooks"), requestBody(request))
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	return &webhook, wr.do(req, &webhook)
}

// List lists every [Webhook] in the tailnet.
func (wr *TailnetWebhooksResource) List(ctx context.Context) ([]Webhook, error) {
	req, err := wr.buildRequest(ctx, http.MethodGet, wr.buildTailnetURL("webhooks"))
	if err != nil {
		return nil, err
	}

	resp := make(map[string][]Webhook)
	if err = wr.do(req, &resp); err != nil {
		return nil, err
	}

	return resp["webhooks"], nil
}

// Get retrieves a specific [Webhook].
func (wr *TailnetWebhooksResource) Get(ctx context.Context, endpointID string) (*Webhook, error) {
	req, err := wr.buildRequest(ctx, http.MethodGet, wr.buildURL("webhooks", endpointID))
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	return &webhook, wr.do(req, &webhook)
}

// Update updates an existing webhook's subscriptions. Returns the updated [Webhook] on success.
func (wr *TailnetWebhooksResource) Update(ctx context.Context, endpointID string, subscriptions []WebhookSubscriptionType) (*Webhook, error) {
	req, err := wr.buildRequest(ctx, http.MethodPatch, wr.buildURL("webhooks", endpointID), requestBody(map[string][]WebhookSubscriptionType{
		"subscriptions": subscriptions,
	}))
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	return &webhook, wr.do(req, &webhook)
}

// Delete deletes a specific webhook.
func (wr *TailnetWebhooksResource) Delete(ctx context.Context, endpointID string) error {
	req, err := wr.buildRequest(ctx, http.MethodDelete, wr.buildURL("webhooks", endpointID))
	if err != nil {
		return err
	}

	return wr.do(req, nil)
}

// Test queues a test event to be sent to a specific webhook.
// Sending the test event is an asynchronous operation which will
// typically happen a few seconds after using this method.
func (wr *TailnetWebhooksResource) Test(ctx context.Context, endpointID string) error {
	req, err := wr.buildRequest(ctx, http.MethodPost, wr.buildURL("webhooks", endpointID, "test"))
	if err != nil {
		return err
	}

	return wr.do(req, nil)
}

// RotateSecret rotates the secret associated with a webhook.
// A new secret will be generated and set on the returned [Webhook].
func (wr *TailnetWebhooksResource) RotateSecret(ctx context.Context, endpointID string) (*Webhook, error) {
	req, err := wr.buildRequest(ctx, http.MethodPost, wr.buildURL("webhooks", endpointID, "rotate"))
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	return &webhook, wr.do(req, &webhook)
}
