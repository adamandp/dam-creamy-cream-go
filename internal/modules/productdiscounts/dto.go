package productdiscounts

type CreateInput struct {
	ProductID  string `json:"productId"`
	DiscountID string `json:"discountId"`
}
