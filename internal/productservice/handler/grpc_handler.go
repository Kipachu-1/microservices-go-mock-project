// internal/productservice/handler/grpc_handler.go
package handler

import (
	"context"
	"log"
	"microservices-project/internal/productservice/service"
	"microservices-project/internal/productservice/model"

	productpb "microservices-project/protos/productpb"


	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ProductGRPCServer struct {
	productpb.UnimplementedProductServiceServer
	productService service.ProductServiceInterface
}

func NewProductGRPCServer(productService service.ProductServiceInterface) *ProductGRPCServer {
	return &ProductGRPCServer{
		productService: productService,
	}
}

func (s *ProductGRPCServer) CreateProduct(ctx context.Context, req *productpb.CreateProductRequest) (*productpb.CreateProductResponse, error) {
	log.Printf("gRPC CreateProduct request: Name=%s, Price=%.2f", req.Name, req.Price)
	domainProduct, err := s.productService.CreateProduct(ctx, req.Name, req.Description, req.Price, req.StockQuantity)
	if err != nil {
		log.Printf("Error creating product via gRPC: %v", err)
		if err == service.ErrInvalidProductData {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to create product: %v", err)
	}
	return &productpb.CreateProductResponse{Product: toProtoProduct(domainProduct)}, nil
}

func (s *ProductGRPCServer) GetProduct(ctx context.Context, req *productpb.GetProductRequest) (*productpb.GetProductResponse, error) {
	log.Printf("gRPC GetProduct request: ID=%s", req.ProductId)
	domainProduct, err := s.productService.GetProductByID(ctx, req.ProductId)
	if err != nil {
		log.Printf("Error getting product via gRPC: %v", err)
		if err == service.ErrProductNotFound {
			return nil, status.Errorf(codes.NotFound, err.Error())
		}
		if err == service.ErrInvalidProductData { // e.g. empty ID
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to get product: %v", err)
	}
	return &productpb.GetProductResponse{Product: toProtoProduct(domainProduct)}, nil
}

func (s *ProductGRPCServer) ListProducts(ctx context.Context, req *productpb.ListProductsRequest) (*productpb.ListProductsResponse, error) {
	log.Printf("gRPC ListProducts request: PageSize=%d, PageToken=%s", req.PageSize, req.PageToken)
	// Simple pagination for now, ignoring page_token from proto for this implementation phase
	// Page token based pagination is more complex. We'll use page number derived from token if it were numeric.
	// For now, let's assume page_size is the limit and page_token might imply an offset or page number.
	// Let's treat PageSize as limit and map PageToken to page number if it's a simple integer string.
	
	page := 1 // Default page
	// if req.PageToken != "" {
	// 	// A proper implementation would decode the page_token
	// }
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 10 // Default page size
	}

	domainProducts, err := s.productService.ListProducts(ctx, page, pageSize) // Using page 1 for now
	if err != nil {
		log.Printf("Error listing products via gRPC: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to list products: %v", err)
	}

	protoProducts := []*productpb.Product{}
	for _, p := range domainProducts {
		protoProducts = append(protoProducts, toProtoProduct(p))
	}
	// For simplicity, not implementing next_page_token yet
	return &productpb.ListProductsResponse{Products: protoProducts, NextPageToken: ""}, nil
}


func (s *ProductGRPCServer) UpdateProduct(ctx context.Context, req *productpb.UpdateProductRequest) (*productpb.UpdateProductResponse, error) {
	log.Printf("gRPC UpdateProduct request: ID=%s, Name=%s", req.ProductId, req.Name)
	domainProduct, err := s.productService.UpdateProduct(ctx, req.ProductId, req.Name, req.Description, req.Price, req.StockQuantity)
	if err != nil {
		log.Printf("Error updating product via gRPC: %v", err)
		if err == service.ErrProductNotFound {
			return nil, status.Errorf(codes.NotFound, err.Error())
		}
		if err == service.ErrInvalidProductData {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to update product: %v", err)
	}
	return &productpb.UpdateProductResponse{Product: toProtoProduct(domainProduct)}, nil
}


func (s *ProductGRPCServer) DeleteProduct(ctx context.Context, req *productpb.DeleteProductRequest) (*productpb.DeleteProductResponse, error) {
	log.Printf("gRPC DeleteProduct request: ID=%s", req.ProductId)
	err := s.productService.DeleteProduct(ctx, req.ProductId)
	if err != nil {
		log.Printf("Error deleting product via gRPC: %v", err)
		if err == service.ErrProductNotFound {
			return nil, status.Errorf(codes.NotFound, err.Error())
		}
		if err == service.ErrInvalidProductData {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to delete product: %v", err)
	}
	return &productpb.DeleteProductResponse{Message: "Product deleted successfully"}, nil
}

func (s *ProductGRPCServer) UpdateStock(ctx context.Context, req *productpb.UpdateStockRequest) (*productpb.UpdateStockResponse, error) {
	log.Printf("gRPC UpdateStock request: ProductID=%s, QuantityChange=%d", req.ProductId, req.QuantityChange)
	updatedProduct, err := s.productService.UpdateStock(ctx, req.ProductId, req.QuantityChange)
	if err != nil {
		log.Printf("Error updating stock via gRPC: %v", err)
		if err == service.ErrProductNotFound {
			return nil, status.Errorf(codes.NotFound, "product not found for stock update")
		}
		if err == service.ErrInsufficientStock {
			return nil, status.Errorf(codes.FailedPrecondition, "insufficient stock")
		}
		if err == service.ErrInvalidProductData {
			return nil, status.Errorf(codes.InvalidArgument, "invalid product ID for stock update")
		}
		return nil, status.Errorf(codes.Internal, "failed to update stock: %v", err)
	}
	return &productpb.UpdateStockResponse{Product: toProtoProduct(updatedProduct)}, nil
}


// Helper to convert domain model.Product to productpb.Product
func toProtoProduct(p *model.Product) *productpb.Product {
	if p == nil {
		return nil
	}
	return &productpb.Product{
		Id:             p.ID,
		Name:           p.Name,
		Description:    p.Description,
		Price:          p.Price,
		StockQuantity:  p.StockQuantity,
		CreatedAt:      timestamppb.New(p.CreatedAt),
		UpdatedAt:      timestamppb.New(p.UpdatedAt),
	}
}