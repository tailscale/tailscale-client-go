package tailscale

import (
	"context"
	"net/http"
)

type ContactsResource struct {
	*Client
}

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

// Contacts retieves the contact information for a tailnet.
func (cr *ContactsResource) Get(ctx context.Context) (*Contacts, error) {
	req, err := cr.buildRequest(ctx, http.MethodGet, cr.buildTailnetURL("contacts"))
	if err != nil {
		return nil, err
	}

	var contacts Contacts
	return &contacts, cr.do(req, &contacts)
}

// UpdateContact updates the email for the specified ContactType within the tailnet.
// If the email address changes, the system will send a verification email to confirm the change.
func (cr *ContactsResource) Update(ctx context.Context, contactType ContactType, contact UpdateContactRequest) error {
	req, err := cr.buildRequest(ctx, http.MethodPatch, cr.buildTailnetURL("contacts", contactType), requestBody(contact))
	if err != nil {
		return err
	}

	return cr.do(req, nil)
}
