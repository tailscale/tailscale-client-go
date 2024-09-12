// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tsclient_test

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tailscale/hujson"
	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

var (
	//go:embed testdata/acl.json
	jsonACL []byte
	//go:embed testdata/acl.hujson
	huJSONACL []byte
)

func TestACL_Unmarshal(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name          string
		ACLContent    []byte
		Expected      tsclient.ACL
		UnmarshalFunc func(data []byte, v interface{}) error
	}{
		{
			Name:          "It should handle JSON ACLs",
			ACLContent:    jsonACL,
			UnmarshalFunc: json.Unmarshal,
			Expected: tsclient.ACL{
				ACLs: []tsclient.ACLEntry{
					{
						Action:      "accept",
						Ports:       []string(nil),
						Users:       []string(nil),
						Source:      []string{"autogroup:members"},
						Destination: []string{"autogroup:self:*"},
						Protocol:    "",
					},
					{
						Action:      "accept",
						Ports:       []string(nil),
						Users:       []string(nil),
						Source:      []string{"group:dev"},
						Destination: []string{"tag:dev:*"},
						Protocol:    "",
					},
					{
						Action:      "accept",
						Ports:       []string(nil),
						Users:       []string(nil),
						Source:      []string{"group:devops"},
						Destination: []string{"tag:prod:*"},
						Protocol:    "",
					},
					{
						Action:      "accept",
						Ports:       []string(nil),
						Users:       []string(nil),
						Source:      []string{"autogroup:members"},
						Destination: []string{"tag:monitoring:80,443"},
						Protocol:    "",
					},
				},
				Groups: map[string][]string{
					"group:dev":    {"alice@example.com", "bob@example.com"},
					"group:devops": {"carl@example.com"},
				},
				Hosts: map[string]string(nil),
				TagOwners: map[string][]string{
					"tag:dev":        {"group:devops"},
					"tag:monitoring": {"group:devops"},
					"tag:prod":       {"group:devops"},
				},
				DERPMap: (*tsclient.ACLDERPMap)(nil),
				Tests: []tsclient.ACLTest{
					{
						User:   "",
						Allow:  []string(nil),
						Deny:   []string(nil),
						Source: "carl@example.com",
						Accept: []string{"tag:prod:80"},
					},
					{
						User:   "",
						Allow:  []string(nil),
						Deny:   []string{"tag:prod:80"},
						Source: "alice@example.com",
						Accept: []string{"tag:dev:80"}},
				},
				SSH: []tsclient.ACLSSH{
					{
						Action:      "accept",
						Source:      []string{"autogroup:members"},
						Destination: []string{"autogroup:self"},
						Users:       []string{"root", "autogroup:nonroot"},
					},
					{
						Action:      "accept",
						Source:      []string{"autogroup:members"},
						Destination: []string{"tag:prod"},
						Users:       []string{"root", "autogroup:nonroot"},
					},
					{
						Action:      "accept",
						Source:      []string{"tag:logging"},
						Destination: []string{"tag:prod"},
						Users:       []string{"root", "autogroup:nonroot"},
						CheckPeriod: tsclient.Duration(time.Hour * 20),
					},
				},
			},
		},
		{
			Name:       "It should handle HuJSON ACLs",
			ACLContent: huJSONACL,
			UnmarshalFunc: func(b []byte, v interface{}) error {
				b = append([]byte{}, b...)
				b, err := hujson.Standardize(b)
				if err != nil {
					return err
				}
				return json.Unmarshal(b, v)
			},
			Expected: tsclient.ACL{
				ACLs: []tsclient.ACLEntry{
					{
						Action:      "accept",
						Ports:       []string(nil),
						Users:       []string(nil),
						Source:      []string{"autogroup:members"},
						Destination: []string{"autogroup:self:*"},
						Protocol:    "",
					},
					{
						Action:      "accept",
						Ports:       []string(nil),
						Users:       []string(nil),
						Source:      []string{"group:dev"},
						Destination: []string{"tag:dev:*"},
						Protocol:    "",
					},
					{
						Action:      "accept",
						Ports:       []string(nil),
						Users:       []string(nil),
						Source:      []string{"group:devops"},
						Destination: []string{"tag:prod:*"},
						Protocol:    "",
					},
					{
						Action:      "accept",
						Ports:       []string(nil),
						Users:       []string(nil),
						Source:      []string{"autogroup:members"},
						Destination: []string{"tag:monitoring:80,443"},
						Protocol:    "",
					},
				},
				Groups: map[string][]string{
					"group:dev":    {"alice@example.com", "bob@example.com"},
					"group:devops": {"carl@example.com"},
				},
				Hosts: map[string]string(nil),
				TagOwners: map[string][]string{
					"tag:dev":        {"group:devops"},
					"tag:monitoring": {"group:devops"},
					"tag:prod":       {"group:devops"},
				},
				DERPMap: (*tsclient.ACLDERPMap)(nil),
				SSH: []tsclient.ACLSSH{
					{
						Action:      "accept",
						Source:      []string{"autogroup:members"},
						Destination: []string{"autogroup:self"},
						Users:       []string{"root", "autogroup:nonroot"},
					},
					{
						Action:      "accept",
						Source:      []string{"autogroup:members"},
						Destination: []string{"tag:prod"},
						Users:       []string{"root", "autogroup:nonroot"},
					},
					{
						Action:      "accept",
						Source:      []string{"tag:logging"},
						Destination: []string{"tag:prod"},
						Users:       []string{"root", "autogroup:nonroot"},
						CheckPeriod: tsclient.Duration(time.Hour * 20),
					},
				},
				Tests: []tsclient.ACLTest{
					{
						User:   "",
						Allow:  []string(nil),
						Deny:   []string(nil),
						Source: "carl@example.com",
						Accept: []string{"tag:prod:80"},
					},
					{
						User:   "",
						Allow:  []string(nil),
						Deny:   []string{"tag:prod:80"},
						Source: "alice@example.com",
						Accept: []string{"tag:dev:80"}},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			var actual tsclient.ACL

			assert.NoError(t, tc.UnmarshalFunc(tc.ACLContent, &actual))
			assert.EqualValues(t, tc.Expected, actual)
		})
	}
}

func TestClient_SetACL(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK
	expectedACL := tsclient.ACL{
		ACLs: []tsclient.ACLEntry{
			{
				Action: "accept",
				Ports:  []string{"*:*"},
				Users:  []string{"*"},
			},
		},
		TagOwners: map[string][]string{
			"tag:example": {"group:example"},
		},
		Hosts: map[string]string{
			"example-host-1": "100.100.100.100",
			"example-host-2": "100.100.101.100/24",
		},
		Groups: map[string][]string{
			"group:example": {
				"user1@example.com",
				"user2@example.com",
			},
		},
		Tests: []tsclient.ACLTest{
			{
				User:  "user1@example.com",
				Allow: []string{"example-host-1:22", "example-host-2:80"},
				Deny:  []string{"exapmle-host-2:100"},
			},
			{
				User:  "user2@example.com",
				Allow: []string{"100.60.3.4:22"},
			},
		},
	}

	assert.NoError(t, client.PolicyFile().Set(context.Background(), expectedACL, ""))
	assert.Equal(t, http.MethodPost, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/acl", server.Path)
	assert.Equal(t, "", server.Header.Get("If-Match"))
	assert.EqualValues(t, "application/json", server.Header.Get("Content-Type"))

	var actualACL tsclient.ACL
	assert.NoError(t, json.Unmarshal(server.Body.Bytes(), &actualACL))
	assert.EqualValues(t, expectedACL, actualACL)
}
func TestClient_SetACL_HuJSON(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	assert.NoError(t, client.PolicyFile().Set(context.Background(), string(huJSONACL), ""))
	assert.Equal(t, http.MethodPost, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/acl", server.Path)
	assert.Equal(t, "", server.Header.Get("If-Match"))
	assert.EqualValues(t, "application/hujson", server.Header.Get("Content-Type"))
	assert.EqualValues(t, huJSONACL, server.Body.Bytes())
}

func TestClient_SetACLWithETag(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK
	expectedACL := tsclient.ACL{
		ACLs: []tsclient.ACLEntry{
			{
				Action: "accept",
				Ports:  []string{"*:*"},
				Users:  []string{"*"},
			},
		},
	}

	assert.NoError(t, client.PolicyFile().Set(context.Background(), expectedACL, "test-etag"))
	assert.Equal(t, http.MethodPost, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/acl", server.Path)
	assert.Equal(t, `"test-etag"`, server.Header.Get("If-Match"))

	var actualACL tsclient.ACL
	assert.NoError(t, json.Unmarshal(server.Body.Bytes(), &actualACL))
	assert.EqualValues(t, expectedACL, actualACL)
}

func TestClient_ACL(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)

	server.ResponseCode = http.StatusOK
	server.ResponseBody = &tsclient.ACL{
		ACLs: []tsclient.ACLEntry{
			{
				Action: "accept",
				Ports:  []string{"*:*"},
				Users:  []string{"*"},
			},
		},
		TagOwners: map[string][]string{
			"tag:example": {"group:example"},
		},
		Hosts: map[string]string{
			"example-host-1": "100.100.100.100",
			"example-host-2": "100.100.101.100/24",
		},
		Groups: map[string][]string{
			"group:example": {
				"user1@example.com",
				"user2@example.com",
			},
		},
		Tests: []tsclient.ACLTest{
			{
				User:  "user1@example.com",
				Allow: []string{"example-host-1:22", "example-host-2:80"},
				Deny:  []string{"exapmle-host-2:100"},
			},
			{
				User:  "user2@example.com",
				Allow: []string{"100.60.3.4:22"},
			},
		},
	}

	acl, err := client.PolicyFile().Get(context.Background())
	assert.NoError(t, err)
	assert.EqualValues(t, acl, server.ResponseBody)
	assert.EqualValues(t, http.MethodGet, server.Method)
	assert.EqualValues(t, "application/json", server.Header.Get("Accept"))
	assert.EqualValues(t, "/api/v2/tailnet/example.com/acl", server.Path)
}

func TestClient_RawACL(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)

	server.ResponseCode = http.StatusOK
	server.ResponseBody = huJSONACL

	acl, err := client.PolicyFile().Raw(context.Background())
	assert.NoError(t, err)
	assert.EqualValues(t, string(huJSONACL), acl)
	assert.EqualValues(t, http.MethodGet, server.Method)
	assert.EqualValues(t, "application/hujson", server.Header.Get("Accept"))
	assert.EqualValues(t, "/api/v2/tailnet/example.com/acl", server.Path)
}
