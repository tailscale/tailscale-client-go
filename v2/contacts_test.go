package tailscale_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tailscale/tailscale-client-go/v2"
)

func TestClient_Contacts(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	expectedContacts := &tailscale.Contacts{
		Account: tailscale.Contact{
			Email:             "test@example.com",
			FallbackEmail:     "test2@example.com",
			NeedsVerification: false,
		},
		Support: tailscale.Contact{
			Email:             "test3@example.com",
			NeedsVerification: false,
		},
		Security: tailscale.Contact{
			Email:             "test4@example.com",
			FallbackEmail:     "test5@example.com",
			NeedsVerification: true,
		},
	}
	server.ResponseBody = expectedContacts

	actualContacts, err := client.Contacts.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, http.MethodGet, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/contacts", server.Path)
	assert.Equal(t, expectedContacts, actualContacts)
}

func TestClient_UpdateContact(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK
	server.ResponseBody = nil

	email := "new@example.com"
	updateRequest := tailscale.UpdateContactRequest{
		Email: &email,
	}
	err := client.Contacts.Update(context.Background(), tailscale.ContactAccount, updateRequest)
	assert.NoError(t, err)
	assert.Equal(t, http.MethodPatch, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/contacts/account", server.Path)
	var receivedRequest tailscale.UpdateContactRequest
	err = json.Unmarshal(server.Body.Bytes(), &receivedRequest)
	assert.NoError(t, err)
	assert.EqualValues(t, updateRequest, receivedRequest)
}
