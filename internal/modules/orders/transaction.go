package orders

import (
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"gorm.io/gorm"
)

func (s *Service) checkoutTransaction(tx *gorm.DB, body CheckoutInput) (CheckoutResult, error) {
	now := time.Now().UTC()
	cart, prepared, err := s.prepareCheckout(tx, body, now)
	if err != nil {
		return CheckoutResult{}, err
	}
	order := domain.Order{
		UserID: body.Order.UserID, AddressID: body.Order.AddressID,
		TotalPrice: prepared.totalPrice, DiscountAmount: prepared.lineDiscount,
		VoucherAmount: prepared.voucherAmount, FinalPrice: prepared.finalPrice,
		Status: "PENDING",
	}
	if err := tx.Create(&order).Error; err != nil {
		return CheckoutResult{}, err
	}
	for _, p := range prepared.items {
		item := domain.OrderItem{
			OrderID: order.ID.ID, ProductID: p.item.ProductID, DiscountID: p.item.DiscountID,
			Quantity: p.item.Quantity, Price: amount(p.product.Price),
			DiscountAmount: p.unitDiscount, FinalPrice: p.unitFinal,
		}
		if err := tx.Create(&item).Error; err != nil {
			return CheckoutResult{}, err
		}
	}
	for _, gift := range prepared.gifts {
		gift.OrderID = order.ID.ID
		if err := tx.Create(&gift).Error; err != nil {
			return CheckoutResult{}, err
		}
	}
	// Inventory updates must reuse the same locked rows. These increments are
	// done only after all stock, discount and voucher checks have passed.
	for _, productID := range sortedIDs(prepared.allQuantities) {
		qty := prepared.allQuantities[productID]
		if err := tx.Model(&domain.Inventory{}).Where("product_id = ?", productID).
			Update("reserved_quantity", gorm.Expr("reserved_quantity + ?", qty)).Error; err != nil {
			return CheckoutResult{}, err
		}
	}
	discountCount := map[string]int{}
	for _, p := range prepared.items {
		if p.item.DiscountID != nil {
			discountCount[*p.item.DiscountID] += p.item.Quantity
		}
		remaining := p.cart.Quantity - p.item.Quantity
		if remaining <= 0 {
			if err := tx.Delete(&p.cart).Error; err != nil {
				return CheckoutResult{}, err
			}
		} else {
			if err := tx.Model(&p.cart).Update("quantity", remaining).Error; err != nil {
				return CheckoutResult{}, err
			}
		}
	}
	for _, id := range sortedIDs(discountCount) {
		if err := tx.Model(&domain.Discount{}).Where("id = ?", id).
			Update("redeemed_count", gorm.Expr("redeemed_count + ?", discountCount[id])).Error; err != nil {
			return CheckoutResult{}, err
		}
	}
	for _, cv := range prepared.vouchers {
		if err := tx.Model(&cv.claim).Update("status", "USED").Error; err != nil {
			return CheckoutResult{}, err
		}
		if err := tx.Model(&domain.Voucher{}).Where("id = ?", cv.voucher.ID.ID).
			Update("redeemed_count", gorm.Expr("redeemed_count + 1")).Error; err != nil {
			return CheckoutResult{}, err
		}
		link := domain.OrderVoucher{OrderID: order.ID.ID, VoucherID: cv.voucher.ID.ID}
		if err := tx.Create(&link).Error; err != nil {
			return CheckoutResult{}, err
		}
	}
	payment := domain.Payment{
		OrderID: order.ID.ID, Method: body.Payment.Method, MethodType: body.Payment.Method,
		Amount: prepared.finalPrice, Status: "PENDING",
	}
	if err := tx.Create(&payment).Error; err != nil {
		return CheckoutResult{}, err
	}
	shipment := domain.Shipment{
		OrderID: order.ID.ID, Courier: body.Shipment.Courier, Service: body.Shipment.Service,
		Cost: prepared.shippingCost, Estimated: body.Shipment.Estimated.UTC(),
		Description: body.Shipment.Description, Status: "PENDING",
	}
	if err := tx.Create(&shipment).Error; err != nil {
		return CheckoutResult{}, err
	}
	_ = cart // held locked until the transaction commits.
	return CheckoutResult{
		OrderID: order.ID.ID, TotalPrice: prepared.totalPrice,
		DiscountAmount: prepared.lineDiscount, VoucherAmount: prepared.voucherAmount,
		ShippingCost: prepared.shippingCost, FinalPrice: prepared.finalPrice, Status: order.Status,
	}, nil
}
