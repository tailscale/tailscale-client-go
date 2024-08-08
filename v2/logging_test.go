package tsclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

func TestClient_LogstreamConfiguration(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	expectedLogstream := &tsclient.LogstreamConfiguration{}
	server.ResponseBody = expectedLogstream

	actualWebhook, err := client.Logging().LogstreamConfiguration(context.Background(), tsclient.LogTypeConfig)
	assert.NoError(t, err)
	assert.Equal(t, http.MethodGet, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/logging/configuration/stream", server.Path)
	assert.Equal(t, expectedLogstream, actualWebhook)
}

func TestClient_SetLogstreamConfiguration(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	logstreamRequest := tsclient.SetLogstreamConfigurationRequest{
		DestinationType: tsclient.LogstreamCriblEndpoint,
		URL:             "http://example.com",
		User:            "my-user",
		Token:           "my-token",
	}
	server.ResponseBody = nil

	err := client.Logging().SetLogstreamConfiguration(context.Background(), tsclient.LogTypeNetwork, logstreamRequest)
	assert.NoError(t, err)
	assert.Equal(t, http.MethodPut, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/logging/network/stream", server.Path)
	var receivedRequest tsclient.SetLogstreamConfigurationRequest
	err = json.Unmarshal(server.Body.Bytes(), &receivedRequest)
	assert.NoError(t, err)
	assert.EqualValues(t, logstreamRequest, receivedRequest)
}

func TestClient_DeleteLogstream(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	err := client.Logging().DeleteLogstreamConfiguration(context.Background(), tsclient.LogTypeConfig)
	assert.NoError(t, err)
	assert.Equal(t, http.MethodDelete, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/logging/configuration/stream", server.Path)
}
