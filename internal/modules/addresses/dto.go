package addresses

type CreateInput struct {
	UserID               string `json:"userId"`
	RecipientName        string `json:"recipientName"`
	RecipientPhoneNumber string `json:"recipientPhoneNumber"`
	Province             string `json:"province"`
	City                 string `json:"city"`
	Subdistrict          string `json:"subdistrict"`
	Village              string `json:"village"`
	PostalCode           string `json:"postalCode"`
	Address              string `json:"address"`
	IsPrimary            bool   `json:"isPrimary"`
}

type UpdateInput struct {
	UserID               *string `json:"userId"`
	RecipientName        *string `json:"recipientName"`
	RecipientPhoneNumber *string `json:"recipientPhoneNumber"`
	Province             *string `json:"province"`
	City                 *string `json:"city"`
	Subdistrict          *string `json:"subdistrict"`
	Village              *string `json:"village"`
	PostalCode           *string `json:"postalCode"`
	Address              *string `json:"address"`
	IsPrimary            *bool   `json:"isPrimary"`
}
