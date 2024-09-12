// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tsclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

func TestClient_TailnetSettings_Get(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	expected := tsclient.TailnetSettings{
		DevicesApprovalOn:                      true,
		DevicesAutoUpdatesOn:                   true,
		DevicesKeyDurationDays:                 5,
		UsersApprovalOn:                        true,
		UsersRoleAllowedToJoinExternalTailnets: tsclient.RoleAllowedToJoinExternalTailnetsMember,
		NetworkFlowLoggingOn:                   true,
		RegionalRoutingOn:                      true,
		PostureIdentityCollectionOn:            true,
	}
	server.ResponseBody = expected

	actual, err := client.TailnetSettings().Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, http.MethodGet, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/settings", server.Path)
	assert.Equal(t, &expected, actual)
}

func TestClient_TailnetSettings_Update(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK
	server.ResponseBody = nil

	updateRequest := tsclient.UpdateTailnetSettingsRequest{
		DevicesApprovalOn:                      tsclient.PointerTo(true),
		DevicesAutoUpdatesOn:                   tsclient.PointerTo(true),
		DevicesKeyDurationDays:                 tsclient.PointerTo(5),
		UsersApprovalOn:                        tsclient.PointerTo(true),
		UsersRoleAllowedToJoinExternalTailnets: tsclient.PointerTo(tsclient.RoleAllowedToJoinExternalTailnetsMember),
		NetworkFlowLoggingOn:                   tsclient.PointerTo(true),
		RegionalRoutingOn:                      tsclient.PointerTo(true),
		PostureIdentityCollectionOn:            tsclient.PointerTo(true),
	}
	err := client.TailnetSettings().Update(context.Background(), updateRequest)
	assert.NoError(t, err)
	assert.Equal(t, http.MethodPatch, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/settings", server.Path)
	var receivedRequest tsclient.UpdateTailnetSettingsRequest
	err = json.Unmarshal(server.Body.Bytes(), &receivedRequest)
	assert.NoError(t, err)
	assert.EqualValues(t, updateRequest, receivedRequest)
}
