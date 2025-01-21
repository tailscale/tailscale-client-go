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
		DestinationType:      tsclient.LogstreamCriblEndpoint,
		URL:                  "http://example.com",
		User:                 "my-user",
		Token:                "my-token",
		S3Bucket:             "my-bucket",
		S3Region:             "us-west-2",
		S3KeyPrefix:          "logs/",
		S3AuthenticationType: tsclient.S3AccessKeyAuthentication,
		S3AccessKeyID:        "my-access-key-id",
		S3SecretAccessKey:    "my-secret-access-key",
		S3RoleARN:            "my-role-arn",
		S3ExternalID:         "my-external-id",
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

func TestClient_CreateOrGetAwsExternalId(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	wantExternalID := &tsclient.AWSExternalID{
		ExternalID:            "external-id",
		TailscaleAWSAccountID: "account-id",
	}
	server.ResponseBody = wantExternalID

	gotExternalID, err := client.Logging().CreateOrGetAwsExternalId(context.Background(), true)
	assert.NoError(t, err)
	assert.Equal(t, server.Method, http.MethodPost)
	assert.Equal(t, server.Path, "/api/v2/tailnet/example.com/aws-external-id")
	assert.Equal(t, gotExternalID, wantExternalID)

	gotRequest := make(map[string]bool)
	err = json.Unmarshal(server.Body.Bytes(), &gotRequest)
	assert.NoError(t, err)
	assert.EqualValues(t, gotRequest, map[string]bool{"reusable": true})
}

func TestClient_ValidateAWSTrustPolicy(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	roleARN := "arn:aws:iam::123456789012:role/example-role"

	err := client.Logging().ValidateAWSTrustPolicy(context.Background(), "external-id-0000-0000", roleARN)
	assert.NoError(t, err)
	assert.Equal(t, server.Method, http.MethodPost)
	assert.Equal(t, server.Path, "/api/v2/tailnet/example.com/aws-external-id/external-id-0000-0000/validate-aws-trust-policy")

	gotRequest := make(map[string]string)
	err = json.Unmarshal(server.Body.Bytes(), &gotRequest)
	assert.NoError(t, err)
	assert.EqualValues(t, gotRequest, map[string]string{"roleArn": roleARN})
}
