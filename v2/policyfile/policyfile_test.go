package policyfile

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tailscale/hujson"
	"github.com/tailscale/tailscale-client-go/v2"
	"github.com/tailscale/tailscale-client-go/v2/internal/testsupport"
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
		Expected      ACL
		UnmarshalFunc func(data []byte, v interface{}) error
	}{
		{
			Name:          "It should handle JSON ACLs",
			ACLContent:    jsonACL,
			UnmarshalFunc: json.Unmarshal,
			Expected: ACL{
				ACLs: []ACLEntry{
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
				DERPMap: (*ACLDERPMap)(nil),
				Tests: []ACLTest{
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
				SSH: []ACLSSH{
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
						CheckPeriod: tailscale.Duration(time.Hour * 20),
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
			Expected: ACL{
				ACLs: []ACLEntry{
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
				DERPMap: (*ACLDERPMap)(nil),
				SSH: []ACLSSH{
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
						CheckPeriod: tailscale.Duration(time.Hour * 20),
					},
				},
				Tests: []ACLTest{
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
			var actual ACL

			assert.NoError(t, tc.UnmarshalFunc(tc.ACLContent, &actual))
			assert.EqualValues(t, tc.Expected, actual)
		})
	}
}

func TestClient_SetACL(t *testing.T) {
	t.Parallel()

	client, server := testsupport.NewTestHarness(t)
	server.ResponseCode = http.StatusOK
	expectedACL := ACL{
		ACLs: []ACLEntry{
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
		Tests: []ACLTest{
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

	assert.NoError(t, With(client).SetACL(context.Background(), expectedACL))
	assert.Equal(t, http.MethodPost, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/acl", server.Path)
	assert.Equal(t, "", server.Header.Get("If-Match"))
	assert.EqualValues(t, "application/json", server.Header.Get("Content-Type"))

	var actualACL ACL
	assert.NoError(t, json.Unmarshal(server.Body.Bytes(), &actualACL))
	assert.EqualValues(t, expectedACL, actualACL)
}

func TestClient_SetACL_HuJSON(t *testing.T) {
	t.Parallel()

	client, server := testsupport.NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	assert.NoError(t, With(client).SetACL(context.Background(), string(huJSONACL)))
	assert.Equal(t, http.MethodPost, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/acl", server.Path)
	assert.Equal(t, "", server.Header.Get("If-Match"))
	assert.EqualValues(t, "application/hujson", server.Header.Get("Content-Type"))
	assert.EqualValues(t, string(huJSONACL), server.Body.String())
}

func TestClient_SetACLWithETag(t *testing.T) {
	t.Parallel()

	client, server := testsupport.NewTestHarness(t)
	server.ResponseCode = http.StatusOK
	expectedACL := ACL{
		ACLs: []ACLEntry{
			{
				Action: "accept",
				Ports:  []string{"*:*"},
				Users:  []string{"*"},
			},
		},
	}

	assert.NoError(t, With(client).SetACL(context.Background(), expectedACL, WithETag("test-etag")))
	assert.Equal(t, http.MethodPost, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/acl", server.Path)
	assert.Equal(t, `"test-etag"`, server.Header.Get("If-Match"))

	var actualACL ACL
	assert.NoError(t, json.Unmarshal(server.Body.Bytes(), &actualACL))
	assert.EqualValues(t, expectedACL, actualACL)
}

func TestClient_ACL(t *testing.T) {
	t.Parallel()

	client, server := testsupport.NewTestHarness(t)

	server.ResponseCode = http.StatusOK
	server.ResponseBody = &ACL{
		ACLs: []ACLEntry{
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
		Tests: []ACLTest{
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

	acl, err := With(client).ACL(context.Background())
	assert.NoError(t, err)
	assert.EqualValues(t, acl, server.ResponseBody)
	assert.EqualValues(t, http.MethodGet, server.Method)
	assert.EqualValues(t, "application/json", server.Header.Get("Accept"))
	assert.EqualValues(t, "/api/v2/tailnet/example.com/acl", server.Path)
}

func TestClient_RawACL(t *testing.T) {
	t.Parallel()

	client, server := testsupport.NewTestHarness(t)

	server.ResponseCode = http.StatusOK
	server.ResponseBody = huJSONACL

	acl, err := With(client).RawACL(context.Background())
	assert.NoError(t, err)
	assert.EqualValues(t, string(huJSONACL), acl)
	assert.EqualValues(t, http.MethodGet, server.Method)
	assert.EqualValues(t, "application/hujson", server.Header.Get("Accept"))
	assert.EqualValues(t, "/api/v2/tailnet/example.com/acl", server.Path)
}

func TestClient_ValidateACL(t *testing.T) {
	t.Parallel()

	client, server := testsupport.NewTestHarness(t)

	acl := ACL{
		ACLs: []ACLEntry{
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
			"example-host-1": "10.0.0.0/8",
			"example-host-2": "10.0.0.1",
		},
		Groups: map[string][]string{
			"group:example": {
				"user1@example.com",
				"user2@example.com",
			},
		},
		Tests: []ACLTest{
			{
				User:  "user1@example.com",
				Allow: []string{"example-host-1:22", "example-host-2:80"},
				Deny:  []string{"exapmle-host-2:100"},
			},
			{
				User:  "user2@example.com",
				Allow: []string{"100.64.0.1:22"},
			},
		},
	}

	server.ResponseCode = http.StatusOK
	server.ResponseBody = acl

	err := With(client).ValidateACL(context.Background(), acl)
	assert.NoError(t, err)
	assert.EqualValues(t, server.ResponseBody, acl)
	assert.EqualValues(t, http.MethodPost, server.Method)
	assert.EqualValues(t, "application/json", server.Header.Get("Content-Type"))
	assert.EqualValues(t, "/api/v2/tailnet/example.com/acl/validate", server.Path)

	tests := []struct {
		name         string
		responseCode int
		responseBody any
		wantErr      string
	}{
		{
			name:         "403_response",
			responseCode: 403,
			responseBody: tailscale.APIError{Message: "access denied"},
			wantErr:      "access denied",
		},
		{
			name:         "200_response_with_error",
			responseCode: 200,
			responseBody: tailscale.APIError{Message: "validation failed"},
			wantErr:      "validation failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server.ResponseCode = tt.responseCode
			server.ResponseBody = tt.responseBody
			err := With(client).ValidateACL(context.Background(), acl)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestClient_ValidateACL_HuJSON(t *testing.T) {
	t.Parallel()

	client, server := testsupport.NewTestHarness(t)

	server.ResponseCode = http.StatusOK
	server.ResponseBody = huJSONACL

	err := With(client).ValidateACL(context.Background(), string(huJSONACL))
	assert.NoError(t, err)
	assert.EqualValues(t, server.ResponseBody, huJSONACL)
	assert.EqualValues(t, http.MethodPost, server.Method)
	assert.EqualValues(t, "application/hujson", server.Header.Get("Content-Type"))
	assert.EqualValues(t, "/api/v2/tailnet/example.com/acl/validate", server.Path)
}
