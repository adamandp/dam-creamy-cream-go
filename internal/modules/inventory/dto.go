package inventory

type UpdateInput struct {
	ProductID        *string `json:"productId"`
	Quantity         *int    `json:"quantity"`
	ReservedQuantity *int    `json:"reservedQuantity"`
	IsActive         *bool   `json:"isActive"`
}
