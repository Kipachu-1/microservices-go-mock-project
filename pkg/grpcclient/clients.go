// pkg/grpcclient/clients.go
package grpcclient

import (
	"log"
	userpb "microservices-project/protos/userpb"
	productpb "microservices-project/protos/productpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure" // For non-TLS connection
)

// NewUserServiceClient creates a new gRPC client for the UserService.
func NewUserServiceClient(userServiceAddr string) (userpb.UserServiceClient, *grpc.ClientConn, error) {
	conn, err := grpc.Dial(userServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Printf("Failed to connect to UserService at %s: %v", userServiceAddr, err)
		return nil, nil, err
	}
	log.Printf("Successfully connected to UserService at %s", userServiceAddr)
	client := userpb.NewUserServiceClient(conn)
	return client, conn, nil
}

// NewProductServiceClient creates a new gRPC client for the ProductService.
func NewProductServiceClient(productServiceAddr string) (productpb.ProductServiceClient, *grpc.ClientConn, error) {
	conn, err := grpc.Dial(productServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Printf("Failed to connect to ProductService at %s: %v", productServiceAddr, err)
		return nil, nil, err
	}
	log.Printf("Successfully connected to ProductService at %s", productServiceAddr)
	client := productpb.NewProductServiceClient(conn)
	return client, conn, nil
}