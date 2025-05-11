package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Internal packages
	"microservices-project/internal/database"
	userHandler "microservices-project/internal/userservice/handler"
	userRepo "microservices-project/internal/userservice/repository"
	userService "microservices-project/internal/userservice/service"

	// Protobuf
	userpb "microservices-project/protos/userpb"

	// gRPC
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	// Chi router
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware" // For useful middlewares
)

const (
	defaultGRPCPort = "50051"
	defaultHTTPPort = "8081"
)

func main() {
	log.Println("Starting User Service...")

	// --- Database Connection ---
	if err := database.ConnectDB(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.CloseDB()

	// --- Initialize Layers (Dependency Injection) ---
	userRepository := userRepo.NewUserRepository(database.DB)
	usrSvc := userService.NewUserService(userRepository) // 'usrSvc' to avoid conflict with package name
	grpcUserServer := userHandler.NewUserGRPCServer(usrSvc)
	httpUserHandler := userHandler.NewUserHTTPHandler(usrSvc) // Initialize HTTP handler

	// Configuration
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = defaultGRPCPort
	}
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = defaultHTTPPort
	}

	// --- Start gRPC Server ---
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatalf("Failed to listen for gRPC: %v", err)
	}
	grpcServer := grpc.NewServer()
	userpb.RegisterUserServiceServer(grpcServer, grpcUserServer)
	reflection.Register(grpcServer)
	go func() {
		log.Printf("gRPC server listening on :%s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// --- Start HTTP Server ---
	// Main router
	r := chi.NewRouter()

	// Middlewares
	r.Use(middleware.RequestID) // Injects a request ID into the context
	r.Use(middleware.RealIP)    // Sets X-Forwarded-For
	r.Use(middleware.Logger)    // Logs the start and end of each request with latency
	r.Use(middleware.Recoverer) // Recovers from panics and returns a 500 error

	// Health check
	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "UserService is healthy")
	})

	// Mount user specific routes
	r.Mount("/api/v1", httpUserHandler.Routes()) // Prefix with /api/v1

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", httpPort),
		Handler: r, // Use chi router
	}

	go func() {
		log.Printf("HTTP server listening on :%s", httpPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to serve HTTP: %v", err)
		}
	}()

	// --- Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down servers...")

	grpcServer.GracefulStop()
	log.Println("gRPC server gracefully stopped.")

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := httpServer.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("HTTP server shutdown failed: %v", err)
	}
	log.Println("HTTP server gracefully stopped.")
	log.Println("User Service shut down.")
}