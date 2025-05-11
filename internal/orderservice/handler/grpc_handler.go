// internal/orderservice/handler/grpc_handler.go
package handler

import (
	"context"
	"errors"
	"log"
	"microservices-project/internal/orderservice/model"
	"microservices-project/internal/orderservice/service"
	orderpb "microservices-project/protos/orderpb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrderGRPCServer struct {
	orderpb.UnimplementedOrderServiceServer
	orderService service.OrderServiceInterface
}

func NewOrderGRPCServer(orderService service.OrderServiceInterface) *OrderGRPCServer {
	return &OrderGRPCServer{
		orderService: orderService,
	}
}

func (s *OrderGRPCServer) CreateOrder(ctx context.Context, req *orderpb.CreateOrderRequest) (*orderpb.CreateOrderResponse, error) {
	log.Printf("gRPC CreateOrder request for UserID: %s, Items: %d", req.UserId, len(req.Items))

	if req.UserId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user_id is required")
	}
	if len(req.Items) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "at least one item is required")
	}

	domainItems := make([]model.OrderItem, len(req.Items))
	for i, item := range req.Items {
		if item.ProductId == "" || item.Quantity <= 0 {
			return nil, status.Errorf(codes.InvalidArgument, "valid product_id and quantity > 0 are required for all items")
		}
		domainItems[i] = model.OrderItem{
			ProductID: item.ProductId,
			Quantity:  item.Quantity,
			// PriceAtPurchase will be filled by the service layer
		}
	}

	createdOrder, err := s.orderService.CreateOrder(ctx, req.UserId, domainItems)
	if err != nil {
		log.Printf("Error creating order via gRPC: %v", err)
		// Map service errors to gRPC status codes
		if errors.Is(err, service.ErrInvalidOrderData) {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, service.ErrUserValidationFailed) {
			return nil, status.Errorf(codes.FailedPrecondition, err.Error()) // Or NotFound if user not found
		}
		if errors.Is(err, service.ErrProductFetchFailed) {
			// This could be NotFound if product not found, or Internal for other fetch issues
			return nil, status.Errorf(codes.FailedPrecondition, err.Error())
		}
		if errors.Is(err, service.ErrInsufficientStockForOrder) {
			return nil, status.Errorf(codes.FailedPrecondition, err.Error()) // A resource (stock) was insufficient
		}
		if errors.Is(err, service.ErrProductStockUpdateFailed) {
			return nil, status.Errorf(codes.Aborted, err.Error()) // Indicates an operation was aborted, often due to concurrency issues
		}
		return nil, status.Errorf(codes.Internal, "failed to create order: %v", err)
	}

	return &orderpb.CreateOrderResponse{Order: toProtoOrder(createdOrder)}, nil
}

func (s *OrderGRPCServer) GetOrder(ctx context.Context, req *orderpb.GetOrderRequest) (*orderpb.GetOrderResponse, error) {
	log.Printf("gRPC GetOrder request for OrderID: %s", req.OrderId)
	if req.OrderId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "order_id is required")
	}

	domainOrder, err := s.orderService.GetOrderByID(ctx, req.OrderId)
	if err != nil {
		log.Printf("Error getting order via gRPC: %v", err)
		if errors.Is(err, service.ErrOrderNotFound) {
			return nil, status.Errorf(codes.NotFound, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to get order: %v", err)
	}
	return &orderpb.GetOrderResponse{Order: toProtoOrder(domainOrder)}, nil
}

func (s *OrderGRPCServer) ListUserOrders(ctx context.Context, req *orderpb.ListUserOrdersRequest) (*orderpb.ListUserOrdersResponse, error) {
	log.Printf("gRPC ListUserOrders request for UserID: %s", req.UserId)
	if req.UserId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user_id is required")
	}

	page := 1 // Default
	// Basic pagination, ignoring page_token for now
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}

	domainOrders, err := s.orderService.ListUserOrders(ctx, req.UserId, page, pageSize)
	if err != nil {
		log.Printf("Error listing user orders via gRPC: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to list user orders: %v", err)
	}

	protoOrders := []*orderpb.Order{}
	for _, o := range domainOrders {
		protoOrders = append(protoOrders, toProtoOrder(o))
	}
	// Not implementing next_page_token yet
	return &orderpb.ListUserOrdersResponse{Orders: protoOrders, NextPageToken: ""}, nil
}


// Helper to convert domain model.Order to orderpb.Order
func toProtoOrder(o *model.Order) *orderpb.Order {
	if o == nil {
		return nil
	}
	items := make([]*orderpb.OrderItem, len(o.Items))
	for i, item := range o.Items {
		items[i] = &orderpb.OrderItem{
			ProductId:       item.ProductID,
			Quantity:        item.Quantity,
			PriceAtPurchase: item.PriceAtPurchase,
		}
	}
	return &orderpb.Order{
		Id:          o.ID,
		UserId:      o.UserID,
		Items:       items,
		TotalAmount: o.TotalAmount,
		Status:      string(o.Status),
		CreatedAt:   timestamppb.New(o.CreatedAt),
		UpdatedAt:   timestamppb.New(o.UpdatedAt),
	}
}