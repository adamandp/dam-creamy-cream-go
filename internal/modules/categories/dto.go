package categories

type CreateInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"isActive"`
}

type UpdateInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"isActive"`
}
