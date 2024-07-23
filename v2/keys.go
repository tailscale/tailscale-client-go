package tailscale

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

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

	// CreateKeyOption type is a function that is used to modify a CreateKeyRequest.
	CreateKeyOption func(c *CreateKeyRequest) error

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

// WithKeyExpiry sets how long the key is valid for.
func WithKeyExpiry(e time.Duration) CreateKeyOption {
	return func(c *CreateKeyRequest) error {
		c.ExpirySeconds = int64(e.Seconds())
		return nil
	}
}

// WithKeyDescription sets the description for the key.
func WithKeyDescription(desc string) CreateKeyOption {
	return func(c *CreateKeyRequest) error {
		c.Description = desc
		return nil
	}
}

// CreateKey creates a new authentication key with the capabilities selected via the KeyCapabilities type. Returns
// the generated key if successful.
func (c *Client) CreateKey(ctx context.Context, capabilities KeyCapabilities, opts ...CreateKeyOption) (Key, error) {
	const uriFmt = "/api/v2/tailnet/%s/keys"

	ckr := &CreateKeyRequest{
		Capabilities: capabilities,
	}

	for _, opt := range opts {
		if err := opt(ckr); err != nil {
			return Key{}, err
		}
	}

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnet), requestBody(ckr))
	if err != nil {
		return Key{}, err
	}

	var key Key
	return key, c.performRequest(req, &key)
}

// GetKey returns all information on a key whose identifier matches the one provided. This will not return the
// authentication key itself, just the metadata.
func (c *Client) GetKey(ctx context.Context, id string) (Key, error) {
	const uriFmt = "/api/v2/tailnet/%s/keys/%s"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnet, id))
	if err != nil {
		return Key{}, err
	}

	var key Key
	return key, c.performRequest(req, &key)
}

// Keys returns all keys within the tailnet. The only fields set for each key will be its identifier. The keys returned
// are relative to the user that owns the API key used to authenticate the client.
func (c *Client) Keys(ctx context.Context) ([]Key, error) {
	const uriFmt = "/api/v2/tailnet/%s/keys"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnet))
	if err != nil {
		return nil, err
	}

	resp := make(map[string][]Key)
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return resp["keys"], nil
}

// DeleteKey removes an authentication key from the tailnet.
func (c *Client) DeleteKey(ctx context.Context, id string) error {
	const uriFmt = "/api/v2/tailnet/%s/keys/%s"

	req, err := c.buildRequest(ctx, http.MethodDelete, fmt.Sprintf(uriFmt, c.tailnet, id))
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}
