// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tsclient

import (
	"context"
	"net/http"
)

// LoggingResource provides access to https://tailscale.com/api#tag/logging.
type LoggingResource struct {
	*Client
}

const (
	LogstreamSplunkEndpoint  LogstreamEndpointType = "splunk"
	LogstreamElasticEndpoint LogstreamEndpointType = "elastic"
	LogstreamPantherEndpoint LogstreamEndpointType = "panther"
	LogstreamCriblEndpoint   LogstreamEndpointType = "cribl"
	LogstreamDatadogEndpoint LogstreamEndpointType = "datadog"
	LogstreamAxiomEndpoint   LogstreamEndpointType = "axiom"
	LogstreamS3Endpoint      LogstreamEndpointType = "s3"
)

const (
	LogTypeConfig  LogType = "configuration"
	LogTypeNetwork LogType = "network"
)

const (
	S3AccessKeyAuthentication S3AuthenticationType = "accesskey"
	S3RoleARNAuthentication   S3AuthenticationType = "rolearn"
)

// LogstreamConfiguration type defines a log stream entity in tailscale.
type LogstreamConfiguration struct {
	LogType              LogType               `json:"logType,omitempty"`
	DestinationType      LogstreamEndpointType `json:"destinationType,omitempty"`
	URL                  string                `json:"url,omitempty"`
	User                 string                `json:"user,omitempty"`
	S3Bucket             string                `json:"s3Bucket,omitempty"`
	S3Region             string                `json:"s3Region,omitempty"`
	S3KeyPrefix          string                `json:"s3KeyPrefix,omitempty"`
	S3AuthenticationType S3AuthenticationType  `json:"s3AuthenticationType,omitempty"`
	S3AccessKeyID        string                `json:"s3AccessKeyId,omitempty"`
	S3RoleARN            string                `json:"s3RoleArn,omitempty"`
	S3ExternalID         string                `json:"s3ExternalId,omitempty"`
}

// SetLogstreamConfigurationRequest type defines a request for setting a LogstreamConfiguration.
type SetLogstreamConfigurationRequest struct {
	DestinationType      LogstreamEndpointType `json:"destinationType,omitempty"`
	URL                  string                `json:"url,omitempty"`
	User                 string                `json:"user,omitempty"`
	Token                string                `json:"token,omitempty"`
	S3Bucket             string                `json:"s3Bucket,omitempty"`
	S3Region             string                `json:"s3Region,omitempty"`
	S3KeyPrefix          string                `json:"s3KeyPrefix,omitempty"`
	S3AuthenticationType S3AuthenticationType  `json:"s3AuthenticationType,omitempty"`
	S3AccessKeyID        string                `json:"s3AccessKeyId,omitempty"`
	S3SecretAccessKey    string                `json:"s3SecretAccessKey,omitempty"`
	S3RoleARN            string                `json:"s3RoleArn,omitempty"`
	S3ExternalID         string                `json:"s3ExternalId,omitempty"`
}

// LogstreamEndpointType describes the type of the endpoint.
type LogstreamEndpointType string

// LogType describes the type of logging.
type LogType string

// S3AuthenticationType describes the type of authentication used to stream logs to a LogstreamS3Endpoint.
type S3AuthenticationType string

// LogstreamConfiguration retrieves the tailnet's [LogstreamConfiguration] for the given [LogType].
func (lr *LoggingResource) LogstreamConfiguration(ctx context.Context, logType LogType) (*LogstreamConfiguration, error) {
	req, err := lr.buildRequest(ctx, http.MethodGet, lr.buildTailnetURL("logging", logType, "stream"))
	if err != nil {
		return nil, err
	}

	return body[LogstreamConfiguration](lr, req)
}

// SetLogstreamConfiguration sets the tailnet's [LogstreamConfiguration] for the given [LogType].
func (lr *LoggingResource) SetLogstreamConfiguration(ctx context.Context, logType LogType, request SetLogstreamConfigurationRequest) error {
	req, err := lr.buildRequest(ctx, http.MethodPut, lr.buildTailnetURL("logging", logType, "stream"), requestBody(request))
	if err != nil {
		return err
	}

	return lr.do(req, nil)
}

// DeleteLogstreamConfiguration deletes the tailnet's [LogstreamConfiguration] for the given [LogType].
func (lr *LoggingResource) DeleteLogstreamConfiguration(ctx context.Context, logType LogType) error {
	req, err := lr.buildRequest(ctx, http.MethodDelete, lr.buildTailnetURL("logging", logType, "stream"))
	if err != nil {
		return err
	}

	return lr.do(req, nil)
}

// AWSExternalID represents an AWS External ID that Tailscale can use to stream logs from a
// particular Tailscale AWS account to a LogstreamS3Endpoint that uses S3RoleARNAuthentication.
type AWSExternalID struct {
	ExternalID            string `json:"externalId,omitempty"`
	TailscaleAWSAccountID string `json:"tailscaleAwsAccountId,omitempty"`
}

// CreateOrGetAwsExternalId gets an AWS External ID that Tailscale can use to stream logs to
// a LogstreamS3Endpoint using S3RoleARNAuthentication, creating a new one for this tailnet
// when necessary.
func (lr *LoggingResource) CreateOrGetAwsExternalId(ctx context.Context, reusable bool) (*AWSExternalID, error) {
	req, err := lr.buildRequest(ctx, http.MethodPost, lr.buildTailnetURL("aws-external-id"), requestBody(map[string]bool{
		"reusable": reusable,
	}))
	if err != nil {
		return nil, err
	}
	return body[AWSExternalID](lr, req)
}

// ValidateAWSTrustPolicy validates that Tailscale can assume your AWS IAM role with (and only
// with) the given AWS External ID.
func (lr *LoggingResource) ValidateAWSTrustPolicy(ctx context.Context, awsExternalID string, roleARN string) error {
	req, err := lr.buildRequest(ctx, http.MethodPost, lr.buildTailnetURL("aws-external-id", awsExternalID, "validate-aws-trust-policy"), requestBody(map[string]string{
		"roleArn": roleARN,
	}))
	if err != nil {
		return err
	}
	return lr.do(req, nil)
}
