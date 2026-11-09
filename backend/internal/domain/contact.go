package domain

import "devsignal/internal/validation"

// Contact is one professional or recruiter contact.
type Contact struct {
	BaseRecord
	Name            string `json:"name"`
	Company         string `json:"company"`
	Role            string `json:"role"`
	Relationship    string `json:"relationship"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	LinkedinURL     string `json:"linkedinUrl"`
	LastContactedOn Date   `json:"lastContactedOn"`
	NextFollowupOn  Date   `json:"nextFollowupOn"`
	Notes           string `json:"notes"`
}

// Validate implements Validator.
func (c *Contact) Validate() validation.Errors {
	e := validation.New()
	validation.Required(e, "name", c.Name)
	validation.MaxLen(e, "name", c.Name, 200)
	validation.MaxLen(e, "company", c.Company, 200)
	validation.MaxLen(e, "role", c.Role, 200)
	validation.Required(e, "relationship", c.Relationship)
	validation.Email(e, "email", c.Email)
	validation.MaxLen(e, "email", c.Email, 254)
	validation.MaxLen(e, "phone", c.Phone, 50)
	validation.WebURL(e, "linkedinUrl", c.LinkedinURL)
	validation.MaxLen(e, "linkedinUrl", c.LinkedinURL, 2000)
	validation.MaxLen(e, "notes", c.Notes, 5000)
	return e
}

// EnumFields implements Enumerated; the allowed values come from the lookup
// table at request time.
func (c *Contact) EnumFields() []EnumField {
	return []EnumField{
		{Field: "relationship", Kind: KindContactRelationship, Value: c.Relationship},
	}
}
