package tailscale

import (
	"context"
	"fmt"
	"net/http"
)

const (
	ContactAccount  ContactType = "account"
	ContactSupport  ContactType = "support"
	ContactSecurity ContactType = "security"
)

type (
	// ContactType defines the type of contact.
	ContactType string

	// Contacts type defines the object returned when retrieving contacts.
	Contacts struct {
		Account  Contact `json:"account"`
		Support  Contact `json:"support"`
		Security Contact `json:"security"`
	}

	// Contact type defines the structure of an individual contact for the tailnet.
	Contact struct {
		Email string `json:"email"`
		// FallbackEmail is the email used when Email has not been verified.
		FallbackEmail string `json:"fallbackEmail,omitempty"`
		// NeedsVerification is true if Email needs to be verified.
		NeedsVerification bool `json:"needsVerification"`
	}

	// UpdateContactRequest type defines the structure of a request to update a Contact.
	UpdateContactRequest struct {
		Email *string `json:"email,omitempty"`
	}
)

type ContactsResource struct {
	*Client
}

// Get retieves the contact information for a tailnet.
func (c *ContactsResource) Get(ctx context.Context) (*Contacts, error) {
	const uriFmt = "/api/v2/tailnet/%s/contacts"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnetPathEscaped))
	if err != nil {
		return nil, err
	}

	var contacts Contacts
	return &contacts, c.performRequest(req, &contacts)
}

// Update updates the email for the specified ContactType within the tailnet.
// If the email address changes, the system will send a verification email to confirm the change.
func (c *ContactsResource) Update(ctx context.Context, contactType ContactType, contact UpdateContactRequest) error {
	const uriFmt = "/api/v2/tailnet/%s/contacts/%s"

	req, err := c.buildRequest(ctx, http.MethodPatch, fmt.Sprintf(uriFmt, c.tailnetPathEscaped, contactType), requestBody(contact))
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}
