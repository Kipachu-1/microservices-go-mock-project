// internal/orderservice/repository/order_repository.go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"microservices-project/internal/orderservice/model"
	"time"

	"github.com/google/uuid"
)

var ErrOrderNotFound = errors.New("order not found")

type OrderRepositoryInterface interface {
	CreateOrder(ctx context.Context, order *model.Order) (*model.Order, error)
	GetOrderByID(ctx context.Context, id string) (*model.Order, error)
	ListOrdersByUserID(ctx context.Context, userID string, limit int, offset int) ([]*model.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID string, status model.OrderStatus) (*model.Order, error)
}

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) CreateOrder(ctx context.Context, order *model.Order) (*model.Order, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	order.ID = uuid.New().String()
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()
	if order.Status == "" {
		order.Status = model.StatusPending // Default status
	}

	orderQuery := `INSERT INTO orders (id, user_id, total_amount, status, created_at, updated_at)
	               VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = tx.ExecContext(ctx, orderQuery, order.ID, order.UserID, order.TotalAmount, order.Status, order.CreatedAt, order.UpdatedAt)
	if err != nil {
		log.Printf("Error inserting order into DB: %v", err)
		return nil, err
	}

	itemQuery := `INSERT INTO order_items (id, order_id, product_id, quantity, price_at_purchase, created_at)
	              VALUES ($1, $2, $3, $4, $5, $6)`
	for i := range order.Items {
		order.Items[i].ID = uuid.New().String()
		order.Items[i].OrderID = order.ID
		order.Items[i].CreatedAt = time.Now() // Or use order.CreatedAt

		_, err = tx.ExecContext(ctx, itemQuery,
			order.Items[i].ID, order.Items[i].OrderID, order.Items[i].ProductID,
			order.Items[i].Quantity, order.Items[i].PriceAtPurchase, order.Items[i].CreatedAt,
		)
		if err != nil {
			log.Printf("Error inserting order item into DB: %v", err)
			return nil, err // This will trigger rollback
		}
	}

	if err = tx.Commit(); err != nil {
		log.Printf("Error committing order transaction: %v", err)
		return nil, err
	}
	return order, nil
}

func (r *OrderRepository) GetOrderByID(ctx context.Context, id string) (*model.Order, error) {
	order := &model.Order{}
	queryOrder := `SELECT id, user_id, total_amount, status, created_at, updated_at
	               FROM orders WHERE id = $1`
	err := r.db.QueryRowContext(ctx, queryOrder, id).Scan(
		&order.ID, &order.UserID, &order.TotalAmount, &order.Status, &order.CreatedAt, &order.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrOrderNotFound
		}
		log.Printf("Error getting order by ID from DB: %v", err)
		return nil, err
	}

	// Fetch order items
	queryItems := `SELECT id, product_id, quantity, price_at_purchase, created_at
	               FROM order_items WHERE order_id = $1 ORDER BY created_at ASC`
	rows, err := r.db.QueryContext(ctx, queryItems, id)
	if err != nil {
		log.Printf("Error fetching order items for order %s: %v", id, err)
		return nil, err // Or return order without items if partial data is acceptable
	}
	defer rows.Close()

	items := []model.OrderItem{}
	for rows.Next() {
		item := model.OrderItem{OrderID: order.ID}
		if err := rows.Scan(&item.ID, &item.ProductID, &item.Quantity, &item.PriceAtPurchase, &item.CreatedAt); err != nil {
			log.Printf("Error scanning order item: %v", err)
			return nil, err
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating order item rows: %v", err)
		return nil, err
	}
	order.Items = items

	return order, nil
}

func (r *OrderRepository) ListOrdersByUserID(ctx context.Context, userID string, limit int, offset int) ([]*model.Order, error) {
	query := `SELECT id, user_id, total_amount, status, created_at, updated_at
	          FROM orders WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		log.Printf("Error listing orders by user ID from DB: %v", err)
		return nil, err
	}
	defer rows.Close()

	orders := []*model.Order{}
	for rows.Next() {
		order := &model.Order{}
		if err := rows.Scan(
			&order.ID, &order.UserID, &order.TotalAmount, &order.Status, &order.CreatedAt, &order.UpdatedAt,
		); err != nil {
			log.Printf("Error scanning order row: %v", err)
			return nil, err
		}
		// Optionally fetch items for each order here, or do it on demand (N+1 problem if not careful)
		// For a list view, often items are not fully loaded immediately.
		// For this example, we'll skip loading items for the list view to keep it simpler.
		// If items are needed, call GetOrderByID or a specialized function.
		orders = append(orders, order)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating order rows: %v", err)
		return nil, err
	}
	return orders, nil
}

func (r *OrderRepository) UpdateOrderStatus(ctx context.Context, orderID string, status model.OrderStatus) (*model.Order, error) {
	updatedAt := time.Now()
	query := `UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3 RETURNING user_id, total_amount, created_at`

	order := &model.Order{ID: orderID, Status: status, UpdatedAt: updatedAt}
	err := r.db.QueryRowContext(ctx, query, status, updatedAt, orderID).Scan(
		&order.UserID, &order.TotalAmount, &order.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrOrderNotFound
		}
		log.Printf("Error updating order status in DB: %v", err)
		return nil, err
	}
	// To return the full order with items, you'd call GetOrderByID here
	// For now, returning the partially filled order (without items)
	return order, nil
}