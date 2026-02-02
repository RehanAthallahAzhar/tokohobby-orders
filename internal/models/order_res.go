package models

import (
	"time"
)

type OrderDetailsResponse struct {
	Order OrderResponse  `json:"order"`
	Items []OrderItemRes `json:"items"`
}

type OrderResponse struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	Status          string    `json:"status"`
	TotalPrice      float64   `json:"total_price"`
	ShippingAddress string    `json:"shipping_address"`
	CreatedAt       time.Time `json:"created_at"`
}
