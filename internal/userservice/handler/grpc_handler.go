// internal/userservice/handler/grpc_handler.go
package handler

import (
	"context"
	"log"
	"microservices-project/internal/userservice/service" // We'll create this soon
	userpb "microservices-project/protos/userpb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// UserGRPCServer implements the gRPC UserServiceServer interface
type UserGRPCServer struct {
	userpb.UnimplementedUserServiceServer
	userService service.UserServiceInterface // Dependency on the service layer
}

// NewUserGRPCServer creates a new UserGRPCServer
func NewUserGRPCServer(userService service.UserServiceInterface) *UserGRPCServer {
	return &UserGRPCServer{
		userService: userService,
	}
}

// CreateUser handles the gRPC request to create a new user
func (s *UserGRPCServer) CreateUser(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.CreateUserResponse, error) {
	log.Printf("gRPC CreateUser request received for username: %s, email: %s", req.Username, req.Email)

	// Basic validation (more can be added)
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return nil, status.Errorf(codes.InvalidArgument, "username, email, and password are required")
	}

	// Call the service layer to create the user
	domainUser, err := s.userService.CreateUser(ctx, req.Username, req.Email, req.Password)
	if err != nil {
		// TODO: Map service errors to gRPC status codes more granularly
		log.Printf("Error creating user: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	// Convert domain model to protobuf response
	return &userpb.CreateUserResponse{
		User: &userpb.User{
			Id:        domainUser.ID,
			Username:  domainUser.Username,
			Email:     domainUser.Email,
			CreatedAt: timestamppb.New(domainUser.CreatedAt),
			UpdatedAt: timestamppb.New(domainUser.UpdatedAt),
		},
	}, nil
}

// GetUser handles the gRPC request to retrieve a user
func (s *UserGRPCServer) GetUser(ctx context.Context, req *userpb.GetUserRequest) (*userpb.GetUserResponse, error) {
	log.Printf("gRPC GetUser request received for ID: %s", req.UserId)

	if req.UserId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user_id is required")
	}

	domainUser, err := s.userService.GetUserByID(ctx, req.UserId)
	if err != nil {
		// TODO: Map service errors (e.g., NotFound) to appropriate gRPC codes
		log.Printf("Error getting user: %v", err)
		if err == service.ErrUserNotFound { // Assuming ErrUserNotFound is defined in service package
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}

	return &userpb.GetUserResponse{
		User: &userpb.User{
			Id:        domainUser.ID,
			Username:  domainUser.Username,
			Email:     domainUser.Email,
			CreatedAt: timestamppb.New(domainUser.CreatedAt),
			UpdatedAt: timestamppb.New(domainUser.UpdatedAt),
		},
	}, nil
}

// LoginUser (Stub for now)
func (s *UserGRPCServer) LoginUser(ctx context.Context, req *userpb.LoginRequest) (*userpb.LoginResponse, error) {
	log.Printf("Received LoginUser request for email: %s", req.Email)
	// TODO: Implement actual login logic by calling s.userService.LoginUser(...)
	return nil, status.Errorf(codes.Unimplemented, "method LoginUser not implemented")
}