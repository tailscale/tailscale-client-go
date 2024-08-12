package tsclient_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

func TestClient_Users_List(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	expectedUsers := map[string][]tsclient.User{
		"users": {
			{
				ID:                 "12345",
				DisplayName:        "Jane Doe",
				LoginName:          "janedoe",
				ProfilePicURL:      "http://example.com/users/janedoe",
				TailnetID:          "1",
				Created:            time.Date(2022, 2, 10, 11, 50, 23, 0, time.UTC),
				Type:               tsclient.UserTypeMember,
				Role:               tsclient.UserRoleOwner,
				Status:             tsclient.UserStatusActive,
				DeviceCount:        2,
				LastSeen:           time.Date(2022, 2, 10, 12, 50, 23, 0, time.UTC),
				CurrentlyConnected: true,
			},
			{
				ID:                 "12346",
				DisplayName:        "John Doe",
				LoginName:          "johndoe",
				ProfilePicURL:      "http://example.com/users/johndoe",
				TailnetID:          "2",
				Created:            time.Date(2022, 2, 10, 11, 50, 23, 12, time.UTC),
				Type:               tsclient.UserTypeShared,
				Role:               tsclient.UserRoleMember,
				Status:             tsclient.UserStatusIdle,
				DeviceCount:        2,
				LastSeen:           time.Date(2022, 2, 10, 12, 50, 23, 12, time.UTC),
				CurrentlyConnected: true,
			},
		},
	}
	server.ResponseBody = expectedUsers

	actualUsers, err := client.Users().List(
		context.Background(),
		tsclient.PointerTo(tsclient.UserTypeMember),
		tsclient.PointerTo(tsclient.UserRoleAdmin))
	assert.NoError(t, err)
	assert.Equal(t, http.MethodGet, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/users", server.Path)
	assert.Equal(t, url.Values{"type": []string{"member"}, "role": []string{"admin"}}, server.Query)
	assert.Equal(t, expectedUsers["users"], actualUsers)
}

func TestClient_Users_Get(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	expectedUser := &tsclient.User{
		ID:                 "12345",
		DisplayName:        "Jane Doe",
		LoginName:          "janedoe",
		ProfilePicURL:      "http://example.com/users/janedoe",
		TailnetID:          "1",
		Created:            time.Date(2022, 2, 10, 11, 50, 23, 0, time.UTC),
		Type:               tsclient.UserTypeMember,
		Role:               tsclient.UserRoleOwner,
		Status:             tsclient.UserStatusActive,
		DeviceCount:        2,
		LastSeen:           time.Date(2022, 2, 10, 12, 50, 23, 0, time.UTC),
		CurrentlyConnected: true,
	}
	server.ResponseBody = expectedUser

	actualUser, err := client.Users().Get(context.Background(), "12345")
	assert.NoError(t, err)
	assert.Equal(t, http.MethodGet, server.Method)
	assert.Equal(t, "/api/v2/users/12345", server.Path)
	assert.Equal(t, expectedUser, actualUser)
}
