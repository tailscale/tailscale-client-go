// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tsclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

func TestClient_DevicePosture_CreateIntegration(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	req := tsclient.CreatePostureIntegrationRequest{
		Provider:     tsclient.PostureIntegrationProviderIntune,
		CloudID:      "cloudid",
		ClientID:     "clientid",
		TenantID:     "tenantid",
		ClientSecret: "clientsecret",
	}

	resp := &tsclient.PostureIntegration{
		ID:       "1",
		Provider: tsclient.PostureIntegrationProviderIntune,
		CloudID:  "cloudid",
		ClientID: "clientid",
		TenantID: "tenantid",
	}
	server.ResponseBody = resp

	integration, err := client.DevicePosture().CreateIntegration(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/posture/integrations", server.Path)
	assert.Equal(t, resp, integration)

	var actualRequest tsclient.CreatePostureIntegrationRequest
	err = json.Unmarshal(server.Body.Bytes(), &actualRequest)
	require.NoError(t, err)
	assert.Equal(t, req, actualRequest)
}

func TestClient_DevicePosture_UpdateIntegration(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	req := tsclient.UpdatePostureIntegrationRequest{
		CloudID:      "cloudid",
		ClientID:     "clientid",
		TenantID:     "tenantid",
		ClientSecret: tsclient.PointerTo("clientsecret"),
	}

	resp := &tsclient.PostureIntegration{
		ID:       "1",
		Provider: tsclient.PostureIntegrationProviderIntune,
		CloudID:  "cloudid",
		ClientID: "clientid",
		TenantID: "tenantid",
	}
	server.ResponseBody = resp

	actualResp, err := client.DevicePosture().UpdateIntegration(context.Background(), "1", req)
	assert.NoError(t, err)
	assert.Equal(t, http.MethodPatch, server.Method)
	assert.Equal(t, "/api/v2/posture/integrations/1", server.Path)
	assert.Equal(t, resp, actualResp)

	var actualRequest tsclient.UpdatePostureIntegrationRequest
	err = json.Unmarshal(server.Body.Bytes(), &actualRequest)
	require.NoError(t, err)
	assert.Equal(t, req, actualRequest)
}

func TestClient_DevicePosture_DeleteIntegration(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	err := client.DevicePosture().DeleteIntegration(context.Background(), "1")
	assert.NoError(t, err)
	assert.Equal(t, http.MethodDelete, server.Method)
	assert.Equal(t, "/api/v2/posture/integrations/1", server.Path)
}

func TestClient_DevicePosture_GetIntegration(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	resp := &tsclient.PostureIntegration{
		ID:       "1",
		Provider: tsclient.PostureIntegrationProviderIntune,
		CloudID:  "cloudid1",
		ClientID: "clientid1",
		TenantID: "tenantid1",
	}
	server.ResponseBody = resp

	actualResp, err := client.DevicePosture().GetIntegration(context.Background(), "1")
	assert.NoError(t, err)
	assert.Equal(t, http.MethodGet, server.Method)
	assert.Equal(t, "/api/v2/posture/integrations/1", server.Path)
	assert.Equal(t, resp, actualResp)
}

func TestClient_DevicePosture_ListIntegrations(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	resp := []tsclient.PostureIntegration{
		{
			ID:       "1",
			Provider: tsclient.PostureIntegrationProviderIntune,
			CloudID:  "cloudid1",
			ClientID: "clientid1",
			TenantID: "tenantid1",
		},
		{
			ID:       "2",
			Provider: tsclient.PostureIntegrationProviderJamfPro,
			CloudID:  "cloudid2",
			ClientID: "clientid2",
			TenantID: "tenantid2",
		},
	}
	server.ResponseBody = map[string][]tsclient.PostureIntegration{
		"integrations": resp,
	}

	actualResp, err := client.DevicePosture().ListIntegrations(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, http.MethodGet, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/posture/integrations", server.Path)
	assert.Equal(t, resp, actualResp)
}
