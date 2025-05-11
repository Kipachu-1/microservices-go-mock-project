// internal/productservice/service/product_service.go
package service

import (
	"context"
	"errors"
	"log"
	"microservices-project/internal/productservice/model"
	"microservices-project/internal/productservice/repository"
)

// Custom errors
var (
	ErrProductNotFound   = repository.ErrProductNotFound // Propagate
	ErrInvalidProductData = errors.New("invalid product data")
	ErrInsufficientStock = repository.ErrInsufficientStock
)

type ProductServiceInterface interface {
	CreateProduct(ctx context.Context, name, description string, price float64, stockQuantity int32) (*model.Product, error)
	GetProductByID(ctx context.Context, id string) (*model.Product, error)
	ListProducts(ctx context.Context, page, pageSize int) ([]*model.Product, error) // Using page/pageSize for simplicity
	UpdateProduct(ctx context.Context, id, name, description string, price float64, stockQuantity int32) (*model.Product, error)
	DeleteProduct(ctx context.Context, id string) error
	UpdateStock(ctx context.Context, productID string, quantityChange int32) (*model.Product, error)
}

type ProductService struct {
	repo repository.ProductRepositoryInterface
}

func NewProductService(repo repository.ProductRepositoryInterface) *ProductService {
	return &ProductService{repo: repo}
}

func (s *ProductService) CreateProduct(ctx context.Context, name, description string, price float64, stockQuantity int32) (*model.Product, error) {
	if name == "" || price < 0 || stockQuantity < 0 {
		return nil, ErrInvalidProductData
	}
	product := &model.Product{
		Name:          name,
		Description:   description,
		Price:         price,
		StockQuantity: stockQuantity,
	}
	return s.repo.CreateProduct(ctx, product)
}

func (s *ProductService) GetProductByID(ctx context.Context, id string) (*model.Product, error) {
	if id == "" {
		return nil, ErrInvalidProductData // Or a specific error for bad ID format
	}
	return s.repo.GetProductByID(ctx, id)
}

func (s *ProductService) ListProducts(ctx context.Context, page, pageSize int) ([]*model.Product, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 { // Max page size
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	return s.repo.ListProducts(ctx, pageSize, offset)
}

func (s *ProductService) UpdateProduct(ctx context.Context, id, name, description string, price float64, stockQuantity int32) (*model.Product, error) {
	if id == "" || name == "" || price < 0 || stockQuantity < 0 {
		return nil, ErrInvalidProductData
	}
	// First, get the existing product to ensure it exists
	existingProduct, err := s.repo.GetProductByID(ctx, id)
	if err != nil {
		return nil, err // Handles ErrProductNotFound
	}

	// Update fields
	existingProduct.Name = name
	existingProduct.Description = description
	existingProduct.Price = price
	existingProduct.StockQuantity = stockQuantity
	// existingProduct.UpdatedAt will be set by repository

	return s.repo.UpdateProduct(ctx, existingProduct)
}

func (s *ProductService) DeleteProduct(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidProductData
	}
	return s.repo.DeleteProduct(ctx, id)
}

func (s *ProductService) UpdateStock(ctx context.Context, productID string, quantityChange int32) (*model.Product, error) {
	if productID == "" {
		return nil, ErrInvalidProductData
	}
	log.Printf("Service: Attempting to update stock for product %s by %d", productID, quantityChange)
	updatedProduct, err := s.repo.UpdateStock(ctx, productID, quantityChange)
	if err != nil {
		log.Printf("Service: Error updating stock for product %s: %v", productID, err)
		return nil, err
	}
	log.Printf("Service: Stock updated successfully for product %s. New stock: %d", productID, updatedProduct.StockQuantity)
	return updatedProduct, nil
}