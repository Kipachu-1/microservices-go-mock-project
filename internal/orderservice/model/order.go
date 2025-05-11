// internal/orderservice/model/order.go
package model

import "time"

type OrderStatus string

const (
	StatusPending    OrderStatus = "PENDING"
	StatusProcessing OrderStatus = "PROCESSING"
	StatusCompleted  OrderStatus = "COMPLETED"
	StatusCancelled  OrderStatus = "CANCELLED"
)

type OrderItem struct {
	ID              string    `json:"id"` // Internal ID for the order item row
	OrderID         string    `json:"-"`  // Foreign key to Order
	ProductID       string    `json:"product_id"`
	Quantity        int32     `json:"quantity"`
	PriceAtPurchase float64   `json:"price_at_purchase"`
	CreatedAt       time.Time `json:"created_at,omitempty"`
}

type Order struct {
	ID          string      `json:"id"`
	UserID      string      `json:"user_id"`
	Items       []OrderItem `json:"items"` // For returning items with order
	TotalAmount float64     `json:"total_amount"`
	Status      OrderStatus `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}