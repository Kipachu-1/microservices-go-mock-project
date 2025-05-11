// internal/orderservice/service/order_service.go
package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"microservices-project/internal/orderservice/model"
	"microservices-project/internal/orderservice/repository"
	productpb "microservices-project/protos/productpb" // Product service proto
	userpb "microservices-project/protos/userpb"       // User service proto
	"sync"                                              // For concurrent product fetches

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrOrderNotFound         = repository.ErrOrderNotFound
	ErrInvalidOrderData      = errors.New("invalid order data")
	ErrUserValidationFailed  = errors.New("user validation failed")
	ErrProductFetchFailed    = errors.New("failed to fetch product details")
	ErrProductStockUpdateFailed = errors.New("failed to update product stock")
	ErrInsufficientStockForOrder = errors.New("insufficient stock for one or more items in the order")
)

type OrderServiceInterface interface {
	CreateOrder(ctx context.Context, userID string, items []model.OrderItem) (*model.Order, error)
	GetOrderByID(ctx context.Context, id string) (*model.Order, error)
	ListUserOrders(ctx context.Context, userID string, page, pageSize int) ([]*model.Order, error)
}

type OrderService struct {
	repo                repository.OrderRepositoryInterface
	userServiceClient   userpb.UserServiceClient     // gRPC client for UserService
	productServiceClient productpb.ProductServiceClient // gRPC client for ProductService
}

func NewOrderService(
	repo repository.OrderRepositoryInterface,
	userClient userpb.UserServiceClient,
	productClient productpb.ProductServiceClient,
) *OrderService {
	return &OrderService{
		repo:                repo,
		userServiceClient:   userClient,
		productServiceClient: productClient,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, userID string, requestedItems []model.OrderItem) (*model.Order, error) {
	if userID == "" || len(requestedItems) == 0 {
		return nil, ErrInvalidOrderData
	}

	// 1. Validate User
	_, err := s.userServiceClient.GetUser(ctx, &userpb.GetUserRequest{UserId: userID})
	if err != nil {
		log.Printf("Error validating user %s: %v", userID, err)
		// Check gRPC status code
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.NotFound {
			return nil, fmt.Errorf("%w: user %s not found", ErrUserValidationFailed, userID)
		}
		return nil, fmt.Errorf("%w: %v", ErrUserValidationFailed, err)
	}
	log.Printf("User %s validated successfully.", userID)


	// 2. Fetch product details, check stock, and calculate total amount concurrently
	var totalAmount float64
	var processedItems []model.OrderItem
	var wg sync.WaitGroup
	mu := sync.Mutex{} // To protect shared variables (totalAmount, processedItems, and any error flags)
	var firstError error // To capture the first error encountered in goroutines

	productStockUpdates := make(map[string]int32) // productID -> quantity to deduct

	for _, item := range requestedItems {
		if item.Quantity <= 0 {
			return nil, fmt.Errorf("%w: quantity for product %s must be positive", ErrInvalidOrderData, item.ProductID)
		}
		wg.Add(1)
		go func(currentItem model.OrderItem) {
			defer wg.Done()

			// Check if an error already occurred in another goroutine
			mu.Lock()
			if firstError != nil {
				mu.Unlock()
				return
			}
			mu.Unlock()

			// Get Product Details
			productResp, err := s.productServiceClient.GetProduct(ctx, &productpb.GetProductRequest{ProductId: currentItem.ProductID})
			if err != nil {
				log.Printf("Error fetching product %s: %v", currentItem.ProductID, err)
				mu.Lock()
				if firstError == nil {
					st, ok := status.FromError(err)
					if ok && st.Code() == codes.NotFound {
						firstError = fmt.Errorf("%w: product %s not found", ErrProductFetchFailed, currentItem.ProductID)
					} else {
						firstError = fmt.Errorf("%w: %v", ErrProductFetchFailed, err)
					}
				}
				mu.Unlock()
				return
			}
			product := productResp.GetProduct()
			log.Printf("Fetched product %s: Price=%.2f, Stock=%d", product.Id, product.Price, product.StockQuantity)

			// Check Stock
			if product.StockQuantity < currentItem.Quantity {
				log.Printf("Insufficient stock for product %s: requested %d, available %d", product.Id, currentItem.Quantity, product.StockQuantity)
				mu.Lock()
				if firstError == nil {
					firstError = fmt.Errorf("%w: product %s (requested %d, available %d)", ErrInsufficientStockForOrder, product.Id, currentItem.Quantity, product.StockQuantity)
				}
				mu.Unlock()
				return
			}

			mu.Lock()
			totalAmount += product.Price * float64(currentItem.Quantity)
			processedItems = append(processedItems, model.OrderItem{
				ProductID:       product.Id,
				Quantity:        currentItem.Quantity,
				PriceAtPurchase: product.Price,
			})
			productStockUpdates[product.Id] -= currentItem.Quantity // Negative for deduction
			mu.Unlock()

		}(item)
	}
	wg.Wait()

	if firstError != nil {
		return nil, firstError
	}

	if len(processedItems) != len(requestedItems) {
		// This case should ideally be caught by firstError, but as a safeguard
		return nil, errors.New("failed to process all items in the order")
	}

	// 3. (Important) Update stock for all products in a "transactional" manner (best effort here)
	// In a real system, you might use a Saga pattern or a distributed transaction coordinator
	// For this project, we'll update stock one by one. If one fails, we should ideally roll back previous stock updates.
	// The ProductRepository's UpdateStock now uses a DB transaction for a single product.
	var updatedProducts []*productpb.Product
	for prodID, qtyChange := range productStockUpdates {
		updateStockReq := &productpb.UpdateStockRequest{
			ProductId: prodID,
			QuantityChange: qtyChange,
		}
		log.Printf("Attempting to update stock for product %s by %d", prodID, qtyChange)
		resp, err := s.productServiceClient.UpdateStock(ctx, updateStockReq)
		if err != nil {
			log.Printf("Failed to update stock for product %s: %v", prodID, err)
			// Attempt to revert previous stock updates (complex, requires careful implementation)
			// For now, we'll just return an error.
			// This is a critical point for data consistency in microservices.
			// For now, let's just log and potentially mark the order as "NEEDS_ATTENTION" or similar.
			return nil, fmt.Errorf("%w for product %s: %v. Order creation aborted. Manual stock correction might be needed.", ErrProductStockUpdateFailed, prodID, err)
		}
		log.Printf("Stock updated successfully for product %s. New stock: %d", prodID, resp.GetProduct().GetStockQuantity())
		updatedProducts = append(updatedProducts, resp.GetProduct())
	}


	// 4. Create Order in DB
	order := &model.Order{
		UserID:      userID,
		Items:       processedItems,
		TotalAmount: totalAmount,
		Status:      model.StatusPending, // Or model.StatusProcessing if payment is next
	}

	createdOrder, err := s.repo.CreateOrder(ctx, order)
	if err != nil {
		log.Printf("Error creating order in repository: %v", err)
		// If order creation fails after stock update, this is also a point of inconsistency.
		// A compensating transaction (Saga) would add stock back.
		return nil, err
	}

	log.Printf("Order %s created successfully for user %s.", createdOrder.ID, userID)
	return createdOrder, nil
}


func (s *OrderService) GetOrderByID(ctx context.Context, id string) (*model.Order, error) {
	if id == "" {
		return nil, ErrInvalidOrderData
	}
	return s.repo.GetOrderByID(ctx, id)
}

func (s *OrderService) ListUserOrders(ctx context.Context, userID string, page, pageSize int) ([]*model.Order, error) {
	if userID == "" {
		return nil, ErrInvalidOrderData
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	return s.repo.ListOrdersByUserID(ctx, userID, pageSize, offset)
}