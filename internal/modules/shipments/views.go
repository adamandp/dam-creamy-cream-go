package shipments

import (
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
)

// Prisma DateTime JSON uses UTC ISO timestamps with exactly three
// millisecond digits. Go's default time.Time JSON omits trailing zeroes.
const prismaTimeLayout = "2006-01-02T15:04:05.000Z"

func prismaTime(t time.Time) string { return t.UTC().Format(prismaTimeLayout) }

func nullablePrismaTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	text := prismaTime(*t)
	return &text
}

type shipmentView struct {
	ID             string  `json:"id"`
	OrderID        string  `json:"orderId"`
	Courier        string  `json:"courier"`
	Service        string  `json:"service"`
	Cost           float64 `json:"cost"`
	Estimated      string  `json:"estimated"`
	DeliveredAt    *string `json:"deliveredAt"`
	Description    *string `json:"description"`
	TrackingNumber *string `json:"trackingNumber"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
}

func toShipmentView(m domain.Shipment) shipmentView {
	return shipmentView{
		ID: m.ID.ID, OrderID: m.OrderID, Courier: m.Courier,
		Service: m.Service, Cost: m.Cost, Estimated: prismaTime(m.Estimated),
		DeliveredAt: nullablePrismaTime(m.DeliveredAt),
		Description: m.Description, TrackingNumber: m.TrackingNumber,
		Status: m.Status, CreatedAt: prismaTime(m.CreatedAt),
		UpdatedAt: prismaTime(m.UpdatedAt),
	}
}
