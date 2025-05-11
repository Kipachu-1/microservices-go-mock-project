// cmd/productservice/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"microservices-project/internal/database" // Shared database package
	productHandler "microservices-project/internal/productservice/handler"
	productRepo "microservices-project/internal/productservice/repository"
	productService "microservices-project/internal/productservice/service"
	productpb "microservices-project/protos/productpb"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	defaultGRPCPort = "50052" // Different port from UserService
	defaultHTTPPort = "8082" // Different port from UserService
)

func main() {
	log.Println("Starting Product Service...")

	// --- Database Connection ---
	// IMPORTANT: Ensure your DB environment variables (DB_HOST, DB_USER, etc.) are set
	// This service will connect to the SAME database instance as UserService for this project,
	// but will use its own 'products' table.
	if err := database.ConnectDB(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.CloseDB() // This will be closed by the last service shutting down, or handled by OS

	// --- Initialize Layers (Dependency Injection) ---
	prodRepository := productRepo.NewProductRepository(database.DB)
	prodSvc := productService.NewProductService(prodRepository)
	grpcProductServer := productHandler.NewProductGRPCServer(prodSvc)
	httpProductHandler := productHandler.NewProductHTTPHandler(prodSvc)

	// Configuration
	grpcPort := os.Getenv("PRODUCT_GRPC_PORT")
	if grpcPort == "" {
		grpcPort = defaultGRPCPort
	}
	httpPort := os.Getenv("PRODUCT_HTTP_PORT")
	if httpPort == "" {
		httpPort = defaultHTTPPort
	}

	// --- Start gRPC Server ---
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatalf("Failed to listen for gRPC: %v", err)
	}
	grpcServer := grpc.NewServer()
	productpb.RegisterProductServiceServer(grpcServer, grpcProductServer)
	reflection.Register(grpcServer)
	go func() {
		log.Printf("Product gRPC server listening on :%s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve Product gRPC: %v", err)
		}
	}()

	// --- Start HTTP Server ---
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ProductService is healthy")
	})
	r.Mount("/api/v1", httpProductHandler.Routes()) // Will define Routes() in http handler

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", httpPort),
		Handler: r,
	}
	go func() {
		log.Printf("Product HTTP server listening on :%s", httpPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to serve Product HTTP: %v", err)
		}
	}()

	// --- Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Product Service shutting down servers...")

	grpcServer.GracefulStop()
	log.Println("Product gRPC server gracefully stopped.")

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := httpServer.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Product HTTP server shutdown failed: %v", err)
	}
	log.Println("Product HTTP server gracefully stopped.")
	log.Println("Product Service shut down.")
}