package tsclient

import (
	"context"
	"net/http"
	"time"
)

type KeysResource struct {
	*Client
}

type (
	// KeyCapabilities type describes the capabilities of an authentication key.
	KeyCapabilities struct {
		Devices struct {
			Create struct {
				Reusable      bool     `json:"reusable"`
				Ephemeral     bool     `json:"ephemeral"`
				Tags          []string `json:"tags"`
				Preauthorized bool     `json:"preauthorized"`
			} `json:"create"`
		} `json:"devices"`
	}

	// CreateKeyRequest type describes the definition of an authentication key to create.
	CreateKeyRequest struct {
		Capabilities  KeyCapabilities `json:"capabilities"`
		ExpirySeconds int64           `json:"expirySeconds"`
		Description   string          `json:"description"`
	}

	// Key type describes an authentication key within the tailnet.
	Key struct {
		ID           string          `json:"id"`
		Key          string          `json:"key"`
		Description  string          `json:"description"`
		Created      time.Time       `json:"created"`
		Expires      time.Time       `json:"expires"`
		Revoked      time.Time       `json:"revoked"`
		Invalid      bool            `json:"invalid"`
		Capabilities KeyCapabilities `json:"capabilities"`
	}
)

// Create creates a new authentication key. Returns the generated key if successful.
func (kr *KeysResource) Create(ctx context.Context, ckr CreateKeyRequest) (*Key, error) {
	req, err := kr.buildRequest(ctx, http.MethodPost, kr.buildTailnetURL("keys"), requestBody(ckr))
	if err != nil {
		return nil, err
	}

	var key Key
	return &key, kr.do(req, &key)
}

// Get returns all information on a key whose identifier matches the one provided. This will not return the
// authentication key itself, just the metadata.
func (kr *KeysResource) Get(ctx context.Context, id string) (*Key, error) {
	req, err := kr.buildRequest(ctx, http.MethodGet, kr.buildTailnetURL("keys", id))
	if err != nil {
		return nil, err
	}

	var key Key
	return &key, kr.do(req, &key)
}

// List returns all keys within the tailnet. The only fields set for each key will be its identifier. The keys returned
// are relative to the user that owns the API key used to authenticate the client.
func (kr *KeysResource) List(ctx context.Context) ([]Key, error) {
	req, err := kr.buildRequest(ctx, http.MethodGet, kr.buildTailnetURL("keys"))
	if err != nil {
		return nil, err
	}

	resp := make(map[string][]Key)
	if err = kr.do(req, &resp); err != nil {
		return nil, err
	}

	return resp["keys"], nil
}

// Delete removes an authentication key from the tailnet.
func (kr *KeysResource) Delete(ctx context.Context, id string) error {
	req, err := kr.buildRequest(ctx, http.MethodDelete, kr.buildTailnetURL("keys", id))
	if err != nil {
		return err
	}

	return kr.do(req, nil)
}
