// internal/productservice/model/product.go
package model

import "time"

// Product represents the domain model for a product.
type Product struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Price          float64   `json:"price"` // Use float64 for consistency with proto, handle precision carefully
	StockQuantity  int32     `json:"stock_quantity"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}