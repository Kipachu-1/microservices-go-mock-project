// internal/productservice/repository/product_repository.go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"microservices-project/internal/productservice/model"
	"time"

	"github.com/google/uuid"
)

var ErrProductNotFound = errors.New("product not found")
var ErrInsufficientStock = errors.New("insufficient stock")


type ProductRepositoryInterface interface {
	CreateProduct(ctx context.Context, product *model.Product) (*model.Product, error)
	GetProductByID(ctx context.Context, id string) (*model.Product, error)
	ListProducts(ctx context.Context, limit int, offset int) ([]*model.Product, error) // Simple limit/offset for now
	UpdateProduct(ctx context.Context, product *model.Product) (*model.Product, error)
	DeleteProduct(ctx context.Context, id string) error
	UpdateStock(ctx context.Context, productID string, quantityChange int32) (*model.Product, error)
}

type ProductRepository struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) CreateProduct(ctx context.Context, product *model.Product) (*model.Product, error) {
	product.ID = uuid.New().String()
	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()

	query := `INSERT INTO products (id, name, description, price, stock_quantity, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7)
	          RETURNING created_at, updated_at` // ID is client-generated

	err := r.db.QueryRowContext(ctx, query,
		product.ID, product.Name, product.Description, product.Price, product.StockQuantity, product.CreatedAt, product.UpdatedAt,
	).Scan(&product.CreatedAt, &product.UpdatedAt)

	if err != nil {
		log.Printf("Error creating product in DB: %v", err)
		return nil, err
	}
	return product, nil
}

func (r *ProductRepository) GetProductByID(ctx context.Context, id string) (*model.Product, error) {
	product := &model.Product{}
	query := `SELECT id, name, description, price, stock_quantity, created_at, updated_at
	          FROM products WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&product.ID, &product.Name, &product.Description, &product.Price,
		&product.StockQuantity, &product.CreatedAt, &product.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrProductNotFound
		}
		log.Printf("Error getting product by ID from DB: %v", err)
		return nil, err
	}
	return product, nil
}

func (r *ProductRepository) ListProducts(ctx context.Context, limit int, offset int) ([]*model.Product, error) {
	query := `SELECT id, name, description, price, stock_quantity, created_at, updated_at
	          FROM products ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		log.Printf("Error listing products from DB: %v", err)
		return nil, err
	}
	defer rows.Close()

	products := []*model.Product{}
	for rows.Next() {
		product := &model.Product{}
		if err := rows.Scan(
			&product.ID, &product.Name, &product.Description, &product.Price,
			&product.StockQuantity, &product.CreatedAt, &product.UpdatedAt,
		); err != nil {
			log.Printf("Error scanning product row: %v", err)
			return nil, err // Or collect errors and continue
		}
		products = append(products, product)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating product rows: %v", err)
		return nil, err
	}
	return products, nil
}

func (r *ProductRepository) UpdateProduct(ctx context.Context, product *model.Product) (*model.Product, error) {
	product.UpdatedAt = time.Now()
	query := `UPDATE products
	          SET name = $1, description = $2, price = $3, stock_quantity = $4, updated_at = $5
	          WHERE id = $6
	          RETURNING created_at` // So we have all fields populated

	err := r.db.QueryRowContext(ctx, query,
		product.Name, product.Description, product.Price, product.StockQuantity, product.UpdatedAt, product.ID,
	).Scan(&product.CreatedAt) // Scan CreatedAt to keep the model consistent

	if err != nil {
		if err == sql.ErrNoRows { // If RETURNING yields no row, it means ID didn't match
			return nil, ErrProductNotFound
		}
		log.Printf("Error updating product in DB: %v", err)
		return nil, err
	}
	// We need to re-fetch or fill CreatedAt. The RETURNING helps here.
	// If we didn't return CreatedAt, we'd have to do a GetProductByID or assume it's unchanged.
	return product, nil
}

func (r *ProductRepository) DeleteProduct(ctx context.Context, id string) error {
	query := `DELETE FROM products WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		log.Printf("Error deleting product from DB: %v", err)
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected after delete: %v", err)
		return err
	}
	if rowsAffected == 0 {
		return ErrProductNotFound
	}
	return nil
}

// UpdateStock adjusts the stock quantity for a product.
// It uses a transaction to ensure atomicity and checks for sufficient stock if decreasing.
func (r *ProductRepository) UpdateStock(ctx context.Context, productID string, quantityChange int32) (*model.Product, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	// Get current stock and details (lock the row for update)
	currentProduct := &model.Product{}
	querySelect := `SELECT id, name, description, price, stock_quantity, created_at, updated_at
	                 FROM products WHERE id = $1 FOR UPDATE`
	err = tx.QueryRowContext(ctx, querySelect, productID).Scan(
    &currentProduct.ID, &currentProduct.Name, &currentProduct.Description, &currentProduct.Price,
    &currentProduct.StockQuantity, &currentProduct.CreatedAt, &currentProduct.UpdatedAt,
)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to get product for stock update: %w", err)
	}

	newStock := currentProduct.StockQuantity + quantityChange
	if newStock < 0 {
		return nil, ErrInsufficientStock
	}

	currentProduct.StockQuantity = newStock
	currentProduct.UpdatedAt = time.Now()

	queryUpdate := `UPDATE products SET stock_quantity = $1, updated_at = $2 WHERE id = $3`
	_, err = tx.ExecContext(ctx, queryUpdate, currentProduct.StockQuantity, currentProduct.UpdatedAt, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to update stock: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return currentProduct, nil
}