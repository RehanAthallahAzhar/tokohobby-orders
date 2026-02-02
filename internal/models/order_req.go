package models

import (
	"github.com/google/uuid"
)

type OrderReq struct {
	TotalPrice           float64 `json:"total_price" validate:"required"`
	ShippingAddress      string  `json:"shipping_address" validate:"required"`
	ShippingMethod       string  `json:"shipping_method" validate:"required"`
	PaymentMethod        string  `json:"payment_method" validate:"required"`
	ShippingTrackingCode string  `json:"shipping_tracking_code" validate:"required"`
	PaymentGatewayID     string  `json:"payment_gateway_id" validate:"required"`
}

type OrderDetailReq struct {
	Order OrderReq       `json:"order"`
	Items []OrderItemReq `json:"items"`
}

type UpdateOrderStatusReq struct {
	Status        string `json:"status" validate:"required"`
	CurrentStatus string `json:"current_status" validate:"required"`
}

// MarkOrderAsPaidReq request for marking order as paid
type MarkOrderAsPaidReq struct {
	OrderID        uuid.UUID `json:"order_id" validate:"required"`
	PaymentMethod  string    `json:"payment_method" validate:"required"`
	PaymentGateway string    `json:"payment_gateway" validate:"required"` // e.g., "midtrans", "xendit"
	TransactionID  string    `json:"transaction_id" validate:"required"`  // Payment gateway transaction ID
	PaymentProof   string    `json:"payment_proof"`                       // URL to receipt/proof (optional)
	PaidAmount     float64   `json:"paid_amount" validate:"required,min=0"`
	OriginalAmount float64   `json:"original_amount"` // Before discount
	DiscountAmount float64   `json:"discount_amount"` // Promo/voucher discount
}

// MarkOrderAsShippedReq request for marking order as shipped
type MarkOrderAsShippedReq struct {
	OrderID          uuid.UUID `json:"order_id" validate:"required"`
	TrackingNumber   string    `json:"tracking_number" validate:"required"`
	Courier          string    `json:"courier" validate:"required"` // e.g., "JNE", "SiCepat"
	WarehouseID      string    `json:"warehouse_id"`                // Which warehouse shipped (optional)
	PackageWeight    float64   `json:"package_weight"`              // In kg
	ShippingCost     float64   `json:"shipping_cost"`
	EstimatedArrival string    `json:"estimated_arrival"` // ISO date string
}

// MarkOrderAsDeliveredReq request for marking order as delivered
type MarkOrderAsDeliveredReq struct {
	OrderID       uuid.UUID `json:"order_id" validate:"required"`
	ReceiverName  string    `json:"receiver_name" validate:"required"` // Who received the package
	DeliveryProof string    `json:"delivery_proof"`                    // Signature/photo URL
	DeliveryNotes string    `json:"delivery_notes"`                    // Courier notes
}
