package orders

import (
	"fmt"
	"sort"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/pricing"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type purchase struct {
	item         CheckoutItem
	cart         domain.CartItem
	product      domain.Product
	discount     *domain.Discount
	unitFinal    float64
	unitDiscount float64
}
type claimVoucher struct {
	voucher domain.Voucher
	claim   domain.UserVoucher
}
type checkoutData struct {
	items         []purchase
	vouchers      []claimVoucher
	gifts         []domain.OrderItem
	allQuantities map[string]int
	totalPrice    float64
	lineDiscount  float64
	voucherAmount float64
	shippingCost  float64
	finalPrice    float64
}

// Every checkout locks the owner's cart row first, then cart items, discounts,
// voucher rows, and inventory in stable order. A failed step rolls everything
// back, including voucher usage and inventory reservations.
func (s *Service) prepareCheckout(tx *gorm.DB, body CheckoutInput, now time.Time) (domain.Cart, checkoutData, error) {
	data := checkoutData{allQuantities: map[string]int{}, shippingCost: amount(body.Shipment.Cost)}
	var user domain.User
	if err := tx.First(&user, "id = ?", body.Order.UserID).Error; err != nil {
		return domain.Cart{}, data, err
	}
	if !user.IsActive {
		return domain.Cart{}, data, unavailable("User account inactive")
	}
	var address domain.Address
	if err := tx.Where("id = ? AND user_id = ?", body.Order.AddressID, user.ID.ID).First(&address).Error; err != nil {
		return domain.Cart{}, data, unavailable("Delivery address must belong to the purchaser")
	}
	var cart domain.Cart
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", user.ID.ID).First(&cart).Error; err != nil {
		return cart, data, err
	}

	orderItems := append([]CheckoutItem{}, body.OrderItems...)
	sort.Slice(orderItems, func(i, j int) bool { return orderItems[i].ProductID < orderItems[j].ProductID })
	var discountQuantities = map[string]int{}
	var discounts = map[string]domain.Discount{}
	data.items = make([]purchase, 0, len(orderItems))
	for _, item := range orderItems {
		var row domain.CartItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("cart_id = ? AND product_id = ?", cart.ID.ID, item.ProductID).First(&row).Error; err != nil {
			return cart, data, unavailable("Requested product is missing from cart")
		}
		if row.Quantity < item.Quantity {
			return cart, data, unavailable("Requested quantity exceeds cart quantity")
		}
		var product domain.Product
		if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).First(&product, "id = ?", item.ProductID).Error; err != nil {
			return cart, data, err
		}
		if !product.IsActive {
			return cart, data, unavailable("Product is inactive")
		}
		if !saneMoney(product.Price) {
			return cart, data, unavailable("Invalid product price")
		}
		p := purchase{item: item, cart: row, product: product, unitFinal: product.Price}
		if item.DiscountID != nil {
			// An ID alone does not entitle a customer to use an unrelated/inactive discount.
			var association domain.ProductDiscount
			if err := tx.Where("product_id = ? AND discount_id = ? AND is_active = true", item.ProductID, *item.DiscountID).
				First(&association).Error; err != nil {
				return cart, data, unavailable("Discount is not active for this product")
			}
			discountQuantities[*item.DiscountID] += item.Quantity
		}
		data.items = append(data.items, p)
		data.allQuantities[item.ProductID] += item.Quantity
	}
	// Each shared discount is locked once and quota is checked using the sum of
	// all lines using it, not one line at a time.
	discountIDs := sortedIDs(discountQuantities)
	for _, id := range discountIDs {
		var discount domain.Discount
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&discount, "id = ?", id).Error; err != nil {
			return cart, data, err
		}
		if now.Before(discount.StartDate) || now.After(discount.EndDate) {
			return cart, data, unavailable("Discount is outside its validity period")
		}
		qty := discountQuantities[id]
		if discount.MaxApplied > 0 && qty > discount.MaxApplied {
			return cart, data, unavailable("Discount per-order limit exceeded")
		}
		if discount.MaxRedeemed <= 0 || discount.RedeemedCount+qty > discount.MaxRedeemed {
			return cart, data, unavailable("Discount redemption limit exceeded")
		}
		discounts[id] = discount
	}
	for i := range data.items {
		p := &data.items[i]
		if p.item.DiscountID != nil {
			disc := discounts[*p.item.DiscountID]
			p.discount = &disc
			switch disc.DiscountType {
			case "PERCENTAGE", "FIXED", "PRICE":
				if disc.Value == nil || !saneMoney(*disc.Value) {
					return cart, data, unavailable("Invalid discount value")
				}
				// Shared with Catalog, Detail and Cart; PRICE is an override,
				// not the amount to subtract from the original product price.
				if discounted := pricing.DiscountPrice(p.product.Price, disc.DiscountType, disc.Value); discounted != nil {
					p.unitFinal = *discounted
				}
			case "FREE_ITEM":
				if err := addGift(data.allQuantities, &data.gifts, disc.ProductID, disc.Quantity); err != nil {
					return cart, data, err
				}
			default:
				return cart, data, unavailable("Unsupported discount type")
			}
		}
		p.unitDiscount = amount(p.product.Price - p.unitFinal)
		data.totalPrice += p.product.Price * float64(p.item.Quantity)
		data.lineDiscount += p.unitDiscount * float64(p.item.Quantity)
	}
	data.totalPrice = amount(data.totalPrice)
	data.lineDiscount = amount(data.lineDiscount)
	if !saneMoney(data.totalPrice) || !saneMoney(data.lineDiscount) {
		return cart, data, unavailable("Invalid order total")
	}
	if err := s.prepareVouchers(tx, body, now, &data); err != nil {
		return cart, data, err
	}
	if err := lockInventory(tx, data.allQuantities); err != nil {
		return cart, data, err
	}
	return cart, data, nil
}
func sortedIDs(m map[string]int) []string {
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
func addGift(quantities map[string]int, gifts *[]domain.OrderItem, productID *string, qty *int) error {
	if productID == nil || qty == nil || *qty < 1 || *qty > 100 {
		return unavailable("Invalid free-item reward configuration")
	}
	quantities[*productID] += *qty
	*gifts = append(*gifts, domain.OrderItem{ProductID: *productID, Quantity: *qty, Price: 0, FinalPrice: 0, DiscountAmount: 0})
	return nil
}
func lockInventory(tx *gorm.DB, required map[string]int) error {
	for _, id := range sortedIDs(required) {
		var product domain.Product
		if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).First(&product, "id = ?", id).Error; err != nil {
			return err
		}
		if !product.IsActive {
			return unavailable("Gift or ordered product is inactive")
		}
		var stock domain.Inventory
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("product_id = ?", id).First(&stock).Error; err != nil {
			return unavailable("Inventory not found")
		}
		if !stock.IsActive || stock.Quantity < 0 || stock.ReservedQuantity < 0 || stock.Quantity-stock.ReservedQuantity < required[id] {
			return unavailable(fmt.Sprintf("Insufficient available stock for product %s", id))
		}
	}
	return nil
}
