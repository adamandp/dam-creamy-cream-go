package domain

import (
	"encoding/json"
	"time"
)

// Prisma serializes DateTime values in UTC with exactly three fractional
// digits. Go's time.Time MarshalJSON omits trailing zeroes, so raw GORM model
// responses otherwise differ from the NestJS JSON contract.
const prismaTimestampFormat = "2006-01-02T15:04:05.000Z"

func prismaTimestamp(t time.Time) string {
	return t.UTC().Format(prismaTimestampFormat)
}

func marshalStamped(value any, stamp Stamp, extra map[string]any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	for name, value := range map[string]any{
		"createdAt": prismaTimestamp(stamp.CreatedAt),
		"updatedAt": prismaTimestamp(stamp.UpdatedAt),
	} {
		fields[name], err = json.Marshal(value)
		if err != nil {
			return nil, err
		}
	}
	for name, value := range extra {
		fields[name], err = json.Marshal(value)
		if err != nil {
			return nil, err
		}
	}
	return json.Marshal(fields)
}

func (m Role) MarshalJSON() ([]byte, error) {
	type raw Role
	return marshalStamped(raw(m), m.Stamp, nil)
}

func (m User) MarshalJSON() ([]byte, error) {
	type raw User
	var extra map[string]any
	if m.BirthDate != nil {
		extra = map[string]any{"birthDate": prismaTimestamp(*m.BirthDate)}
	}
	return marshalStamped(raw(m), m.Stamp, extra)
}

func (m UserToken) MarshalJSON() ([]byte, error) {
	type raw UserToken
	return marshalStamped(raw(m), m.Stamp, map[string]any{"expiredAt": prismaTimestamp(m.ExpiredAt)})
}

func (m Address) MarshalJSON() ([]byte, error) {
	type raw Address
	return marshalStamped(raw(m), m.Stamp, nil)
}

func (m Category) MarshalJSON() ([]byte, error) {
	type raw Category
	return marshalStamped(raw(m), m.Stamp, nil)
}

func (m Product) MarshalJSON() ([]byte, error) {
	type raw Product
	return marshalStamped(raw(m), m.Stamp, nil)
}

func (m Inventory) MarshalJSON() ([]byte, error) {
	type raw Inventory
	return marshalStamped(raw(m), m.Stamp, nil)
}

func (m Cart) MarshalJSON() ([]byte, error) {
	type raw Cart
	return marshalStamped(raw(m), m.Stamp, nil)
}

func (m CartItem) MarshalJSON() ([]byte, error) {
	type raw CartItem
	return marshalStamped(raw(m), m.Stamp, nil)
}

func (m Order) MarshalJSON() ([]byte, error) {
	type raw Order
	return marshalStamped(raw(m), m.Stamp, nil)
}

func (m OrderItem) MarshalJSON() ([]byte, error) {
	type raw OrderItem
	return marshalStamped(raw(m), m.Stamp, nil)
}

func (m Payment) MarshalJSON() ([]byte, error) {
	type raw Payment
	return marshalStamped(raw(m), m.Stamp, map[string]any{
		"expiryDate": nullablePrismaTimestamp(m.ExpiryDate),
		"paidAt":     nullablePrismaTimestamp(m.PaidAt),
	})
}

func (m Shipment) MarshalJSON() ([]byte, error) {
	type raw Shipment
	return marshalStamped(raw(m), m.Stamp, map[string]any{
		"estimated":   prismaTimestamp(m.Estimated),
		"deliveredAt": nullablePrismaTimestamp(m.DeliveredAt),
	})
}

func nullablePrismaTimestamp(t *time.Time) any {
	if t == nil {
		return nil
	}
	return prismaTimestamp(*t)
}

func (m Review) MarshalJSON() ([]byte, error) {
	type raw Review
	return marshalStamped(raw(m), m.Stamp, nil)
}

func (m Discount) MarshalJSON() ([]byte, error) {
	type raw Discount
	return marshalStamped(raw(m), m.Stamp, map[string]any{
		"startDate": prismaTimestamp(m.StartDate),
		"endDate":   prismaTimestamp(m.EndDate),
	})
}

func (m ProductDiscount) MarshalJSON() ([]byte, error) {
	type raw ProductDiscount
	return marshalStamped(raw(m), m.Stamp, nil)
}

func (m ProductInformation) MarshalJSON() ([]byte, error) {
	type raw ProductInformation
	return marshalStamped(raw(m), m.Stamp, nil)
}

func (m Voucher) MarshalJSON() ([]byte, error) {
	type raw Voucher
	return marshalStamped(raw(m), m.Stamp, map[string]any{
		"startDate": prismaTimestamp(m.StartDate),
		"endDate":   prismaTimestamp(m.EndDate),
	})
}

func (m UserVoucher) MarshalJSON() ([]byte, error) {
	type raw UserVoucher
	return marshalStamped(raw(m), m.Stamp, map[string]any{
		"claimedAt": prismaTimestamp(m.ClaimedAt),
		"expiredAt": prismaTimestamp(m.ExpiredAt),
	})
}

func (m Notification) MarshalJSON() ([]byte, error) {
	type raw Notification
	return marshalStamped(raw(m), m.Stamp, nil)
}

func (m UserNotification) MarshalJSON() ([]byte, error) {
	type raw UserNotification
	return marshalStamped(raw(m), m.Stamp, nil)
}
