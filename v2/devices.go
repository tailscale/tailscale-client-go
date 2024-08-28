package tsclient

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// TailnetDevicesResource provides access to https://tailscale.com/api#tag/devices.
type TailnetDevicesResource struct {
	*TailnetClient
}

type (
	DeviceRoutes struct {
		Advertised []string `json:"advertisedRoutes"`
		Enabled    []string `json:"enabledRoutes"`
	}
)

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

// Get gets the [Device] identified by deviceID.
func (dr *TailnetDevicesResource) Get(ctx context.Context, deviceID string) (*Device, error) {
	req, err := dr.buildRequest(ctx, http.MethodGet, dr.buildURL("device", deviceID))
	if err != nil {
		return nil, err
	}

	var result Device
	return &result, dr.do(req, &result)
}

// List lists every [Device] in the tailnet.
func (dr *TailnetDevicesResource) List(ctx context.Context) ([]Device, error) {
	req, err := dr.buildRequest(ctx, http.MethodGet, dr.buildTailnetURL("devices"))
	if err != nil {
		return nil, err
	}

	m := make(map[string][]Device)
	err = dr.do(req, &m)
	if err != nil {
		return nil, err
	}

	return m["devices"], nil
}

// SetAuthorized marks the specified device as authorized or not.
func (dr *TailnetDevicesResource) SetAuthorized(ctx context.Context, deviceID string, authorized bool) error {
	req, err := dr.buildRequest(ctx, http.MethodPost, dr.buildURL("device", deviceID, "authorized"), requestBody(map[string]bool{
		"authorized": authorized,
	}))
	if err != nil {
		return err
	}

	return dr.do(req, nil)
}

// Delete deletes the device identified by deviceID.
func (dr *TailnetDevicesResource) Delete(ctx context.Context, deviceID string) error {
	req, err := dr.buildRequest(ctx, http.MethodDelete, dr.buildURL("device", deviceID))
	if err != nil {
		return err
	}

	return dr.do(req, nil)
}

// SetTags updates the tags of the device identified by deviceID.
func (dr *TailnetDevicesResource) SetTags(ctx context.Context, deviceID string, tags []string) error {
	req, err := dr.buildRequest(ctx, http.MethodPost, dr.buildURL("device", deviceID, "tags"), requestBody(map[string][]string{
		"tags": tags,
	}))
	if err != nil {
		return err
	}

	return dr.do(req, nil)
}

type (
	// DeviceKey type represents the properties of the key of an individual device within
	// the tailnet.
	DeviceKey struct {
		KeyExpiryDisabled bool `json:"keyExpiryDisabled"` // Whether or not this device's key will ever expire.
	}
)

// SetKey updates the properties of a device's key.
func (dr *TailnetDevicesResource) SetKey(ctx context.Context, deviceID string, key DeviceKey) error {
	req, err := dr.buildRequest(ctx, http.MethodPost, dr.buildURL("device", deviceID, "key"), requestBody(key))
	if err != nil {
		return err
	}

	return dr.do(req, nil)
}

// SetDeviceIPv4Address sets the Tailscale IPv4 address of the device.
func (dr *TailnetDevicesResource) SetIPv4Address(ctx context.Context, deviceID string, ipv4Address string) error {
	req, err := dr.buildRequest(ctx, http.MethodPost, dr.buildURL("device", deviceID, "ip"), requestBody(map[string]string{
		"ipv4": ipv4Address,
	}))
	if err != nil {
		return err
	}

	return dr.do(req, nil)
}

// SetSubnetRoutes sets which subnet routes are enabled to be routed by a device by replacing the existing list
// of subnet routes with the supplied routes. Routes can be enabled without a device advertising them (e.g. for preauth).
func (dr *TailnetDevicesResource) SetSubnetRoutes(ctx context.Context, deviceID string, routes []string) error {
	req, err := dr.buildRequest(ctx, http.MethodPost, dr.buildURL("device", deviceID, "routes"), requestBody(map[string][]string{
		"routes": routes,
	}))
	if err != nil {
		return err
	}

	return dr.do(req, nil)
}

// SubnetRoutes Retrieves the list of subnet routes that a device is advertising, as well as those that are
// enabled for it. Enabled routes are not necessarily advertised (e.g. for pre-enabling), and likewise, advertised
// routes are not necessarily enabled.
func (dr *TailnetDevicesResource) SubnetRoutes(ctx context.Context, deviceID string) (*DeviceRoutes, error) {
	req, err := dr.buildRequest(ctx, http.MethodGet, dr.buildURL("device", deviceID, "routes"))
	if err != nil {
		return nil, err
	}

	var result DeviceRoutes
	return &result, dr.do(req, &result)
}
