package tsclient

import (
	"context"
	"net/http"
	"time"
)

const (
	UserTypeMember UserType = "member"
	UserTypeShared UserType = "shared"
)

const (
	UserRoleOwner        UserRole = "owner"
	UserRoleMember       UserRole = "member"
	UserRoleAdmin        UserRole = "admin"
	UserRoleITAdmin      UserRole = "it-admin"
	UserRoleNetworkAdmin UserRole = "network-admin"
	UserRoleBillingAdmin UserRole = "billing-admin"
	UserRoleAuditor      UserRole = "auditor"
)

const (
	UserStatusActive           UserStatus = "active"
	UserStatusIdle             UserStatus = "idle"
	UserStatusSuspended        UserStatus = "suspended"
	UserStatusNeedsApproval    UserStatus = "needs-approval"
	UserStatusOverBillingLimit UserStatus = "over-billing-limit"
)

type (
	// UserType is the type of relation this user has to the tailnet associated with the request.
	UserType string

	// UserRole is the role of the user.
	UserRole string

	// UserStatus is the status of the user.
	UserStatus string

	// User is a representation of a user within a tailnet.
	User struct {
		ID                 string     `json:"id"`
		DisplayName        string     `json:"displayName"`
		LoginName          string     `json:"loginName"`
		ProfilePicURL      string     `json:"profilePicUrl"`
		TailnetID          string     `json:"tailnetId"`
		Created            time.Time  `json:"created"`
		Type               UserType   `json:"type"`
		Role               UserRole   `json:"role"`
		Status             UserStatus `json:"status"`
		DeviceCount        int        `json:"deviceCount"`
		LastSeen           time.Time  `json:"lastSeen"`
		CurrentlyConnected bool       `json:"currentlyConnected"`
	}
)

type UsersResource struct {
	*Client
}

// List lists all [User]s of a tailnet. If userType and/or role are provided,
// the list of users will be filtered by those.
func (ur *UsersResource) List(ctx context.Context, userType *UserType, role *UserRole) ([]User, error) {
	u := ur.buildTailnetURL("users")
	q := u.Query()
	if userType != nil {
		q.Add("type", string(*userType))
	}
	if role != nil {
		q.Add("role", string(*role))
	}
	u.RawQuery = q.Encode()

	req, err := ur.buildRequest(ctx, http.MethodGet, u)
	if err != nil {
		return nil, err
	}

	resp := make(map[string][]User)
	if err = ur.do(req, &resp); err != nil {
		return nil, err
	}

	return resp["users"], nil
}

// Get retrieves the [User] identified by the given id.
func (ur *UsersResource) Get(ctx context.Context, id string) (*User, error) {
	req, err := ur.buildRequest(ctx, http.MethodGet, ur.buildURL("users", id))
	if err != nil {
		return nil, err
	}

	var resp User
	return &resp, ur.do(req, &resp)
}
