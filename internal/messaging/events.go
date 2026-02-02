package messaging

import "time"

// OrderStatus represents order status
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusPaid      OrderStatus = "paid"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// ============================================
// MAJOR LIFECYCLE EVENTS (Enterprise-grade)
// ============================================

// OrderCreatedEvent represents order creation event
type OrderCreatedEvent struct {
	OrderID       string    `json:"order_id"`
	UserID        string    `json:"user_id"`
	TotalAmount   float64   `json:"total_amount"`
	ItemCount     int       `json:"item_count"`
	PaymentMethod string    `json:"payment_method"`
	CreatedAt     time.Time `json:"created_at"`
}

// OrderPaidEvent represents payment confirmation event
type OrderPaidEvent struct {
	OrderID        string    `json:"order_id"`
	UserID         string    `json:"user_id"`
	PaidAmount     float64   `json:"paid_amount"`
	PaymentMethod  string    `json:"payment_method"`
	PaymentGateway string    `json:"payment_gateway"` // e.g., "midtrans", "xendit"
	TransactionID  string    `json:"transaction_id"`  // Payment gateway transaction ID
	PaymentProof   string    `json:"payment_proof"`   // URL to receipt/proof
	OriginalAmount float64   `json:"original_amount"` // Before discount
	DiscountAmount float64   `json:"discount_amount"` // Promo/voucher discount
	PaidAt         time.Time `json:"paid_at"`
}

// OrderShippedEvent represents order shipment event
type OrderShippedEvent struct {
	OrderID          string    `json:"order_id"`
	UserID           string    `json:"user_id"`
	TrackingNumber   string    `json:"tracking_number"`
	Courier          string    `json:"courier"`        // e.g., "JNE", "SiCepat"
	WarehouseID      string    `json:"warehouse_id"`   // Which warehouse shipped
	PackageWeight    float64   `json:"package_weight"` // In kg
	ShippingCost     float64   `json:"shipping_cost"`
	EstimatedArrival time.Time `json:"estimated_arrival"`
	ShippedAt        time.Time `json:"shipped_at"`
}

// OrderDeliveredEvent represents successful delivery event
type OrderDeliveredEvent struct {
	OrderID       string    `json:"order_id"`
	UserID        string    `json:"user_id"`
	ReceiverName  string    `json:"receiver_name"`  // Who received the package
	DeliveryProof string    `json:"delivery_proof"` // Signature/photo URL
	DeliveryNotes string    `json:"delivery_notes"` // Courier notes
	DeliveredAt   time.Time `json:"delivered_at"`
	ShippedAt     time.Time `json:"shipped_at"` // For delivery duration calculation
}

// OrderCancelledEvent represents order cancellation event
type OrderCancelledEvent struct {
	OrderID         string    `json:"order_id"`
	UserID          string    `json:"user_id"`
	CancelledBy     string    `json:"cancelled_by"`    // "user", "admin", "system"
	CancelReason    string    `json:"cancel_reason"`   // User-provided reason
	CancelCategory  string    `json:"cancel_category"` // E.g., "out_of_stock", "user_request", "payment_failed"
	RefundAmount    float64   `json:"refund_amount"`
	RefundMethod    string    `json:"refund_method"`    // "original_payment", "wallet", "manual"
	CancellationFee float64   `json:"cancellation_fee"` // Penalty if late cancellation
	OriginalStatus  string    `json:"original_status"`  // Status before cancellation
	CancelledAt     time.Time `json:"cancelled_at"`
}

// OrderRefundedEvent represents refund processing event
type OrderRefundedEvent struct {
	OrderID         string    `json:"order_id"`
	UserID          string    `json:"user_id"`
	RefundAmount    float64   `json:"refund_amount"`
	RefundMethod    string    `json:"refund_method"`    // "original_payment", "wallet"
	RefundReason    string    `json:"refund_reason"`    // "cancellation", "return", "dispute"
	RefundReference string    `json:"refund_reference"` // Payment gateway refund ID
	ProcessedBy     string    `json:"processed_by"`     // Admin who processed
	RefundedAt      time.Time `json:"refunded_at"`
	ExpectedCredit  time.Time `json:"expected_credit"` // When funds will arrive
}

// ============================================
// MINOR STATUS UPDATE EVENT (Internal Tracking)
// ============================================

// OrderStatusChangedEvent for minor status transitions that don't need specific events
// E.g., "pending" → "processing", "processing" → "ready_to_ship", etc.
type OrderStatusChangedEvent struct {
	OrderID      string      `json:"order_id"`
	UserID       string      `json:"user_id"`
	Email        string      `json:"email"`
	OldStatus    OrderStatus `json:"old_status"`
	NewStatus    OrderStatus `json:"new_status"`
	TotalAmount  float64     `json:"total_amount"`
	ProductCount int         `json:"product_count"`
	ChangedAt    time.Time   `json:"changed_at"`
}
