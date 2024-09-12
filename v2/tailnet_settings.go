// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tsclient

import (
	"context"
	"net/http"
)

// TailnetSettingsResource provides access to https://tailscale.com/api#tag/tailnetsettings.
type TailnetSettingsResource struct {
	*Client
}

type (
	// TailnetSettings represents the current settings of a tailnet.
	// See https://tailscale.com/api#model/tailnetsettings.
	TailnetSettings struct {
		DevicesApprovalOn      bool `json:"devicesApprovalOn"`
		DevicesAutoUpdatesOn   bool `json:"devicesAutoUpdatesOn"`
		DevicesKeyDurationDays int  `json:"devicesKeyDurationDays"` // days before device key expiry

		UsersApprovalOn                        bool                              `json:"usersApprovalOn"`
		UsersRoleAllowedToJoinExternalTailnets RoleAllowedToJoinExternalTailnets `json:"usersRoleAllowedToJoinExternalTailnets"`

		NetworkFlowLoggingOn        bool `json:"networkFlowLoggingOn"`
		RegionalRoutingOn           bool `json:"regionalRoutingOn"`
		PostureIdentityCollectionOn bool `json:"postureIdentityCollectionOn"`
	}

	// UpdateTailnetSettingsRequest is a request to update the settings of a tailnet.
	// Nil values indicate that the existing setting should be left unchanged.
	UpdateTailnetSettingsRequest struct {
		DevicesApprovalOn      *bool `json:"devicesApprovalOn,omitempty"`
		DevicesAutoUpdatesOn   *bool `json:"devicesAutoUpdatesOn,omitempty"`
		DevicesKeyDurationDays *int  `json:"devicesKeyDurationDays,omitempty"` // days before device key expiry

		UsersApprovalOn                        *bool                              `json:"usersApprovalOn,omitempty"`
		UsersRoleAllowedToJoinExternalTailnets *RoleAllowedToJoinExternalTailnets `json:"usersRoleAllowedToJoinExternalTailnets,omitempty"`

		NetworkFlowLoggingOn        *bool `json:"networkFlowLoggingOn,omitempty"`
		RegionalRoutingOn           *bool `json:"regionalRoutingOn,omitempty"`
		PostureIdentityCollectionOn *bool `json:"postureIdentityCollectionOn,omitempty"`
	}

	// RoleAllowedToJoinExternalTailnets constrains which users are allowed to join external tailnets
	// based on their role.
	RoleAllowedToJoinExternalTailnets string
)

const (
	RoleAllowedToJoinExternalTailnetsNone   RoleAllowedToJoinExternalTailnets = "none"
	RoleAllowedToJoinExternalTailnetsAdmin  RoleAllowedToJoinExternalTailnets = "admin"
	RoleAllowedToJoinExternalTailnetsMember RoleAllowedToJoinExternalTailnets = "member"
)

// Get retrieves the current [TailnetSettings].
// See https://tailscale.com/api#tag/tailnetsettings/GET/tailnet/{tailnet}/settings.
func (tsr *TailnetSettingsResource) Get(ctx context.Context) (*TailnetSettings, error) {
	req, err := tsr.buildRequest(ctx, http.MethodGet, tsr.buildTailnetURL("settings"))
	if err != nil {
		return nil, err
	}

	return body[TailnetSettings](tsr, req)
}

// Update updates the tailnet settings.
// See https://tailscale.com/api#tag/tailnetsettings/PATCH/tailnet/{tailnet}/settings.
func (tsr *TailnetSettingsResource) Update(ctx context.Context, request UpdateTailnetSettingsRequest) error {
	req, err := tsr.buildRequest(ctx, http.MethodPatch, tsr.buildTailnetURL("settings"), requestBody(request))
	if err != nil {
		return err
	}

	return tsr.do(req, nil)
}
