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
)

const (
	LogTypeConfig  LogType = "configuration"
	LogTypeNetwork LogType = "network"
)

type (
	// LogstreamConfiguration type defines a log stream entity in tailscale.
	LogstreamConfiguration struct {
		LogType         LogType               `json:"logType,omitempty"`
		DestinationType LogstreamEndpointType `json:"destinationType,omitempty"`
		URL             string                `json:"url,omitempty"`
		User            string                `json:"user,omitempty"`
	}

	// SetLogstreamConfigurationRequest type defines a request for setting a LogstreamConfiguration.
	SetLogstreamConfigurationRequest struct {
		DestinationType LogstreamEndpointType `json:"destinationType,omitempty"`
		URL             string                `json:"url,omitempty"`
		User            string                `json:"user,omitempty"`
		Token           string                `json:"token,omitempty"`
	}

	// LogstreamEndpointType describes the type of the endpoint.
	LogstreamEndpointType string

	// LogType describes the type of logging.
	LogType string
)

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
