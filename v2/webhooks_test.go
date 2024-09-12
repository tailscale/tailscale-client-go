// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tsclient_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

func TestClient_CreateWebhook(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	req := tsclient.CreateWebhookRequest{
		EndpointURL:   "https://example.com/my/endpoint",
		ProviderType:  tsclient.WebhookDiscordProviderType,
		Subscriptions: []tsclient.WebhookSubscriptionType{tsclient.WebhookNodeCreated, tsclient.WebhookNodeApproved},
	}

	expectedSecret := "my-secret"
	expectedWebhook := &tsclient.Webhook{
		EndpointID:       "12345",
		EndpointURL:      req.EndpointURL,
		ProviderType:     req.ProviderType,
		CreatorLoginName: "pretend@example.com",
		Created:          time.Date(2022, 2, 10, 11, 50, 23, 0, time.UTC),
		LastModified:     time.Date(2022, 2, 10, 11, 50, 23, 0, time.UTC),
		Subscriptions:    req.Subscriptions,
		Secret:           &expectedSecret,
	}
	server.ResponseBody = expectedWebhook

	webhook, err := client.Webhooks().Create(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, http.MethodPost, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/webhooks", server.Path)
	assert.Equal(t, expectedWebhook, webhook)
}

func TestClient_Webhooks(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	expectedWebhooks := map[string][]tsclient.Webhook{
		"webhooks": {
			{
				EndpointID:       "12345",
				EndpointURL:      "https://example.com/my/endpoint",
				ProviderType:     "",
				CreatorLoginName: "pretend@example.com",
				Created:          time.Date(2022, 2, 10, 11, 50, 23, 0, time.UTC),
				LastModified:     time.Date(2022, 2, 10, 11, 50, 23, 0, time.UTC),
				Subscriptions:    []tsclient.WebhookSubscriptionType{tsclient.WebhookNodeCreated, tsclient.WebhookNodeApproved},
			},
			{
				EndpointID:       "54321",
				EndpointURL:      "https://example.com/my/endpoint/other",
				ProviderType:     "slack",
				CreatorLoginName: "pretend2@example.com",
				Created:          time.Date(2022, 2, 10, 11, 50, 23, 0, time.UTC),
				LastModified:     time.Date(2022, 2, 10, 11, 50, 23, 0, time.UTC),
				Subscriptions:    []tsclient.WebhookSubscriptionType{tsclient.WebhookNodeApproved},
			},
		},
	}
	server.ResponseBody = expectedWebhooks

	actualWebhooks, err := client.Webhooks().List(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, http.MethodGet, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/webhooks", server.Path)
	assert.Equal(t, expectedWebhooks["webhooks"], actualWebhooks)
}

func TestClient_Webhook(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	expectedWebhook := &tsclient.Webhook{
		EndpointID:       "54321",
		EndpointURL:      "https://example.com/my/endpoint/other",
		ProviderType:     "slack",
		CreatorLoginName: "pretend2@example.com",
		Created:          time.Date(2022, 2, 10, 11, 50, 23, 0, time.UTC),
		LastModified:     time.Date(2022, 2, 10, 11, 50, 23, 0, time.UTC),
		Subscriptions:    []tsclient.WebhookSubscriptionType{tsclient.WebhookNodeApproved},
	}
	server.ResponseBody = expectedWebhook

	actualWebhook, err := client.Webhooks().Get(context.Background(), "54321")
	assert.NoError(t, err)
	assert.Equal(t, http.MethodGet, server.Method)
	assert.Equal(t, "/api/v2/webhooks/54321", server.Path)
	assert.Equal(t, expectedWebhook, actualWebhook)
}

func TestClient_UpdateWebhook(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	subscriptions := []tsclient.WebhookSubscriptionType{tsclient.WebhookNodeCreated, tsclient.WebhookNodeApproved, tsclient.WebhookNodeNeedsApproval}

	expectedWebhook := &tsclient.Webhook{
		EndpointID:       "54321",
		EndpointURL:      "https://example.com/my/endpoint/other",
		ProviderType:     "slack",
		CreatorLoginName: "pretend2@example.com",
		Created:          time.Date(2022, 2, 10, 11, 50, 23, 0, time.UTC),
		LastModified:     time.Date(2022, 2, 10, 11, 50, 23, 0, time.UTC),
		Subscriptions:    subscriptions,
	}
	server.ResponseBody = expectedWebhook

	actualWebhook, err := client.Webhooks().Update(context.Background(), "54321", subscriptions)
	assert.NoError(t, err)
	assert.Equal(t, http.MethodPatch, server.Method)
	assert.Equal(t, "/api/v2/webhooks/54321", server.Path)
	assert.Equal(t, expectedWebhook, actualWebhook)
}

func TestClient_DeleteWebhook(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	err := client.Webhooks().Delete(context.Background(), "54321")
	assert.NoError(t, err)
	assert.Equal(t, http.MethodDelete, server.Method)
	assert.Equal(t, "/api/v2/webhooks/54321", server.Path)
}

func TestClient_TestWebhook(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusAccepted

	err := client.Webhooks().Test(context.Background(), "54321")
	assert.NoError(t, err)
	assert.Equal(t, http.MethodPost, server.Method)
	assert.Equal(t, "/api/v2/webhooks/54321/test", server.Path)
}

func TestClient_RotateWebhookSecret(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	expectedSecret := "my-new-secret"
	expectedWebhook := &tsclient.Webhook{
		EndpointID:       "54321",
		EndpointURL:      "https://example.com/my/endpoint/other",
		ProviderType:     "slack",
		CreatorLoginName: "pretend2@example.com",
		Created:          time.Date(2022, 2, 10, 11, 50, 23, 0, time.UTC),
		LastModified:     time.Date(2022, 2, 10, 11, 50, 23, 0, time.UTC),
		Subscriptions:    []tsclient.WebhookSubscriptionType{tsclient.WebhookNodeApproved},
		Secret:           &expectedSecret,
	}
	server.ResponseBody = expectedWebhook

	actualWebhook, err := client.Webhooks().RotateSecret(context.Background(), "54321")
	assert.NoError(t, err)
	assert.Equal(t, http.MethodPost, server.Method)
	assert.Equal(t, "/api/v2/webhooks/54321/rotate", server.Path)
	assert.Equal(t, expectedWebhook, actualWebhook)
}
