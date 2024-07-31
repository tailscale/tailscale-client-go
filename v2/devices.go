package tailscale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type DevicesResource struct {
	*Client
}

type (
	DeviceRoutes struct {
		Advertised []string `json:"advertisedRoutes"`
		Enabled    []string `json:"enabledRoutes"`
	}
)

// SetSubnetRoutes sets which subnet routes are enabled to be routed by a device by replacing the existing list
// of subnet routes with the supplied routes. Routes can be enabled without a device advertising them (e.g. for preauth).
func (dr *DevicesResource) SetSubnetRoutes(ctx context.Context, deviceID string, routes []string) error {
	const uriFmt = "/api/v2/device/%s/routes"

	req, err := dr.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, deviceID), requestBody(map[string][]string{
		"routes": routes,
	}))
	if err != nil {
		return err
	}

	return dr.performRequest(req, nil)
}

// SubnetRoutes Retrieves the list of subnet routes that a device is advertising, as well as those that are
// enabled for it. Enabled routes are not necessarily advertised (e.g. for pre-enabling), and likewise, advertised
// routes are not necessarily enabled.
func (dr *DevicesResource) SubnetRoutes(ctx context.Context, deviceID string) (*DeviceRoutes, error) {
	const uriFmt = "/api/v2/device/%s/routes"

	req, err := dr.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, deviceID))
	if err != nil {
		return nil, err
	}

	var resp DeviceRoutes
	if err = dr.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// Time wraps a time and allows for unmarshalling timestamps that represent an empty time as an empty string (e.g "")
// this is used by the tailscale API when it returns devices that have no created date, such as its hello service.
type Time struct {
	time.Time
}

// MarshalJSON is an implementation of json.Marshal.
func (t Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Time)
}

// UnmarshalJSON unmarshals the content of data as a time.Time, a blank string will keep the time at its zero value.
func (t *Time) UnmarshalJSON(data []byte) error {
	if string(data) == `""` {
		return nil
	}

	if err := json.Unmarshal(data, &t.Time); err != nil {
		return err
	}

	return nil
}

type Device struct {
	Addresses                 []string `json:"addresses"`
	Name                      string   `json:"name"`
	ID                        string   `json:"id"`
	Authorized                bool     `json:"authorized"`
	User                      string   `json:"user"`
	Tags                      []string `json:"tags"`
	KeyExpiryDisabled         bool     `json:"keyExpiryDisabled"`
	BlocksIncomingConnections bool     `json:"blocksIncomingConnections"`
	ClientVersion             string   `json:"clientVersion"`
	Created                   Time     `json:"created"`
	Expires                   Time     `json:"expires"`
	Hostname                  string   `json:"hostname"`
	IsExternal                bool     `json:"isExternal"`
	LastSeen                  Time     `json:"lastSeen"`
	MachineKey                string   `json:"machineKey"`
	NodeKey                   string   `json:"nodeKey"`
	OS                        string   `json:"os"`
	UpdateAvailable           bool     `json:"updateAvailable"`
}

// List lists the devices in a tailnet.
func (dr *DevicesResource) List(ctx context.Context) ([]Device, error) {
	const uriFmt = "/api/v2/tailnet/%s/devices"

	req, err := dr.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, dr.tailnetPathEscaped))
	if err != nil {
		return nil, err
	}

	resp := make(map[string][]Device)
	if err = dr.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return resp["devices"], nil
}

// SetAuthorized marks the specified device as authorized or not.
func (dr *DevicesResource) SetAuthorized(ctx context.Context, deviceID string, authorized bool) error {
	const uriFmt = "/api/v2/device/%s/authorized"

	req, err := dr.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, deviceID), requestBody(map[string]bool{
		"authorized": authorized,
	}))
	if err != nil {
		return err
	}

	return dr.performRequest(req, nil)
}

// Delete deletes the device given its deviceID.
func (dr *DevicesResource) Delete(ctx context.Context, deviceID string) error {
	const uriFmt = "/api/v2/device/%s"
	req, err := dr.buildRequest(ctx, http.MethodDelete, fmt.Sprintf(uriFmt, deviceID))
	if err != nil {
		return err
	}

	return dr.performRequest(req, nil)
}

// SetTags updates the tags of a target device.
func (dr *DevicesResource) SetTags(ctx context.Context, deviceID string, tags []string) error {
	const uriFmt = "/api/v2/device/%s/tags"

	req, err := dr.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, deviceID), requestBody(map[string][]string{
		"tags": tags,
	}))
	if err != nil {
		return err
	}

	return dr.performRequest(req, nil)
}

type (
	// DeviceKey type represents the properties of the key of an individual device within
	// the tailnet.
	DeviceKey struct {
		KeyExpiryDisabled bool `json:"keyExpiryDisabled"` // Whether or not this device's key will ever expire.
	}
)

// SetKey updates the properties of a device's key.
func (dr *DevicesResource) SetKey(ctx context.Context, deviceID string, key DeviceKey) error {
	const uriFmt = "/api/v2/device/%s/key"

	req, err := dr.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, deviceID), requestBody(key))
	if err != nil {
		return err
	}

	return dr.performRequest(req, nil)
}

// SetDeviceIPv4Address sets the Tailscale IPv4 address of the device.
func (dr *DevicesResource) SetDeviceIPv4Address(ctx context.Context, deviceID string, ipv4Address string) error {
	const uriFmt = "/api/v2/device/%s/ip"

	req, err := dr.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, deviceID), requestBody(map[string]string{
		"ipv4": ipv4Address,
	}))
	if err != nil {
		return err
	}

	return dr.performRequest(req, nil)
}
