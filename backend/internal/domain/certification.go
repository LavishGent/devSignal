package domain

import "devsignal/internal/validation"

// Certification is one tracked professional certification.
type Certification struct {
	BaseRecord
	Name          string `json:"name"`
	Issuer        string `json:"issuer"`
	Status        string `json:"status"`
	EarnedOn      Date   `json:"earnedOn"`
	ExpiresOn     Date   `json:"expiresOn"`
	CredentialID  string `json:"credentialId"`
	CredentialURL string `json:"credentialUrl"`
	Notes         string `json:"notes"`
}

// Validate implements Validator.
func (c *Certification) Validate() validation.Errors {
	e := validation.New()
	validation.Required(e, "name", c.Name)
	validation.MaxLen(e, "name", c.Name, 300)
	validation.Required(e, "issuer", c.Issuer)
	validation.MaxLen(e, "issuer", c.Issuer, 200)
	validation.Required(e, "status", c.Status)
	validation.DateOrder(e, "expiresOn", c.EarnedOn.Time(), c.ExpiresOn.Time(), "earnedOn")
	validation.MaxLen(e, "credentialId", c.CredentialID, 200)
	validation.WebURL(e, "credentialUrl", c.CredentialURL)
	validation.MaxLen(e, "credentialUrl", c.CredentialURL, 2000)
	validation.MaxLen(e, "notes", c.Notes, 5000)
	if c.Status == "earned" && c.EarnedOn.IsZero() {
		e.Add("earnedOn", "is required when status is earned")
	}
	return e
}

// EnumFields implements Enumerated; the allowed values come from the lookup
// table at request time.
func (c *Certification) EnumFields() []EnumField {
	return []EnumField{
		{Field: "status", Kind: KindCertificationStatus, Value: c.Status},
	}
}
