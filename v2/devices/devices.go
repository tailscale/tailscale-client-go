package devices

import (
	"context"
	"fmt"
	"net/http"

	"github.com/tailscale/tailscale-client-go/v2"
)

type (
	DeviceRoutes struct {
		Advertised []string `json:"advertisedRoutes"`
		Enabled    []string `json:"enabledRoutes"`
	}

	Device struct {
		Addresses                 []string       `json:"addresses"`
		Name                      string         `json:"name"`
		ID                        string         `json:"id"`
		Authorized                bool           `json:"authorized"`
		User                      string         `json:"user"`
		Tags                      []string       `json:"tags"`
		KeyExpiryDisabled         bool           `json:"keyExpiryDisabled"`
		BlocksIncomingConnections bool           `json:"blocksIncomingConnections"`
		ClientVersion             string         `json:"clientVersion"`
		Created                   tailscale.Time `json:"created"`
		Expires                   tailscale.Time `json:"expires"`
		Hostname                  string         `json:"hostname"`
		IsExternal                bool           `json:"isExternal"`
		LastSeen                  tailscale.Time `json:"lastSeen"`
		MachineKey                string         `json:"machineKey"`
		NodeKey                   string         `json:"nodeKey"`
		OS                        string         `json:"os"`
		UpdateAvailable           bool           `json:"updateAvailable"`
	}
)

type Client struct {
	tailscale.Client
}

func With(c tailscale.Client) *Client {
	return &Client{c}
}

// SetDeviceSubnetRoutes sets which subnet routes are enabled to be routed by a device by replacing the existing list
// of subnet routes with the supplied routes. Routes can be enabled without a device advertising them (e.g. for preauth).
func (c *Client) SetDeviceSubnetRoutes(ctx context.Context, deviceID string, routes []string) error {
	const uriFmt = "/api/v2/device/%s/routes"

	req, err := c.BuildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, deviceID), tailscale.RequestBody(map[string][]string{
		"routes": routes,
	}))
	if err != nil {
		return err
	}

	return c.PerformRequest(req, nil)
}

// DeviceSubnetRoutes Retrieves the list of subnet routes that a device is advertising, as well as those that are
// enabled for it. Enabled routes are not necessarily advertised (e.g. for pre-enabling), and likewise, advertised
// routes are not necessarily enabled.
func (c *Client) DeviceSubnetRoutes(ctx context.Context, deviceID string) (*DeviceRoutes, error) {
	const uriFmt = "/api/v2/device/%s/routes"

	req, err := c.BuildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, deviceID))
	if err != nil {
		return nil, err
	}

	var resp DeviceRoutes
	if err = c.PerformRequest(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// Devices lists the devices in a tailnet.
func (c *Client) Devices(ctx context.Context) ([]Device, error) {
	const uriFmt = "/api/v2/tailnet/%s/devices"

	req, err := c.BuildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.Tailnet()))
	if err != nil {
		return nil, err
	}

	resp := make(map[string][]Device)
	if err = c.PerformRequest(req, &resp); err != nil {
		return nil, err
	}

	return resp["devices"], nil
}

// AuthorizeDevice marks the specified device identifier as authorized to join the tailnet.
func (c *Client) AuthorizeDevice(ctx context.Context, deviceID string) error {
	return c.SetDeviceAuthorized(ctx, deviceID, true)
}

// SetDeviceAuthorized marks the specified device as authorized or not.
func (c *Client) SetDeviceAuthorized(ctx context.Context, deviceID string, authorized bool) error {
	const uriFmt = "/api/v2/device/%s/authorized"

	req, err := c.BuildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, deviceID), tailscale.RequestBody(map[string]bool{
		"authorized": authorized,
	}))
	if err != nil {
		return err
	}

	return c.PerformRequest(req, nil)
}

// DeleteDevice deletes the device given its deviceID.
func (c *Client) DeleteDevice(ctx context.Context, deviceID string) error {
	const uriFmt = "/api/v2/device/%s"
	req, err := c.BuildRequest(ctx, http.MethodDelete, fmt.Sprintf(uriFmt, deviceID))
	if err != nil {
		return err
	}

	return c.PerformRequest(req, nil)
}

// SetDeviceTags updates the tags of a target device.
func (c *Client) SetDeviceTags(ctx context.Context, deviceID string, tags []string) error {
	const uriFmt = "/api/v2/device/%s/tags"

	req, err := c.BuildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, deviceID), tailscale.RequestBody(map[string][]string{
		"tags": tags,
	}))
	if err != nil {
		return err
	}

	return c.PerformRequest(req, nil)
}

type (
	// DeviceKey type represents the properties of the key of an individual device within
	// the tailnet.
	DeviceKey struct {
		KeyExpiryDisabled bool `json:"keyExpiryDisabled"` // Whether or not this device's key will ever expire.
	}
)

// SetDeviceKey updates the properties of a device's key.
func (c *Client) SetDeviceKey(ctx context.Context, deviceID string, key DeviceKey) error {
	const uriFmt = "/api/v2/device/%s/key"

	req, err := c.BuildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, deviceID), tailscale.RequestBody(key))
	if err != nil {
		return err
	}

	return c.PerformRequest(req, nil)
}

// SetDeviceIPv4Address sets the Tailscale IPv4 address of the device.
func (c *Client) SetDeviceIPv4Address(ctx context.Context, deviceID string, ipv4Address string) error {
	const uriFmt = "/api/v2/device/%s/ip"

	req, err := c.BuildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, deviceID), tailscale.RequestBody(map[string]string{
		"ipv4": ipv4Address,
	}))
	if err != nil {
		return err
	}

	return c.PerformRequest(req, nil)
}
