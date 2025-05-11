// cmd/orderservice/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"microservices-project/internal/database"
	orderHandler "microservices-project/internal/orderservice/handler"
	orderRepo "microservices-project/internal/orderservice/repository"
	orderService "microservices-project/internal/orderservice/service"
	"microservices-project/pkg/grpcclient" // Our gRPC client helper
	orderpb "microservices-project/protos/orderpb"
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
	defaultGRPCPort         = "50053"
	defaultHTTPPort         = "8083"
	defaultUserServiceAddr   = "localhost:50051" // Address of UserService gRPC
	defaultProductServiceAddr = "localhost:50052" // Address of ProductService gRPC
)

func main() {
	log.Println("Starting Order Service...")

	// --- Database Connection ---
	if err := database.ConnectDB(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	// defer database.CloseDB() // Will be closed by OS or last service if sharing connection

	// --- gRPC Client Connections ---
	userServiceAddr := os.Getenv("USER_SERVICE_GRPC_ADDR")
	if userServiceAddr == "" {
		userServiceAddr = defaultUserServiceAddr
	}
	productServiceAddr := os.Getenv("PRODUCT_SERVICE_GRPC_ADDR")
	if productServiceAddr == "" {
		productServiceAddr = defaultProductServiceAddr
	}

	userSvcClient, userConn, err := grpcclient.NewUserServiceClient(userServiceAddr)
	if err != nil {
		log.Fatalf("Failed to connect to UserService: %v", err)
	}
	defer userConn.Close()

	productSvcClient, productConn, err := grpcclient.NewProductServiceClient(productServiceAddr)
	if err != nil {
		log.Fatalf("Failed to connect to ProductService: %v", err)
	}
	defer productConn.Close()

	// --- Initialize Layers ---
	ordRepository := orderRepo.NewOrderRepository(database.DB)
	ordSvc := orderService.NewOrderService(ordRepository, userSvcClient, productSvcClient)
	grpcOrderServer := orderHandler.NewOrderGRPCServer(ordSvc)
	httpOrderHandler := orderHandler.NewOrderHTTPHandler(ordSvc)

	// Configuration for OrderService ports
	grpcPort := os.Getenv("ORDER_GRPC_PORT")
	if grpcPort == "" {
		grpcPort = defaultGRPCPort
	}
	httpPort := os.Getenv("ORDER_HTTP_PORT")
	if httpPort == "" {
		httpPort = defaultHTTPPort
	}

	// --- Start gRPC Server ---
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatalf("Failed to listen for Order gRPC: %v", err)
	}
	grpcServer := grpc.NewServer()
	orderpb.RegisterOrderServiceServer(grpcServer, grpcOrderServer)
	reflection.Register(grpcServer)
	go func() {
		log.Printf("Order gRPC server listening on :%s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve Order gRPC: %v", err)
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
		fmt.Fprintln(w, "OrderService is healthy")
	})
	r.Mount("/api/v1", httpOrderHandler.Routes())

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", httpPort),
		Handler: r,
	}
	go func() {
		log.Printf("Order HTTP server listening on :%s", httpPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to serve Order HTTP: %v", err)
		}
	}()

	// --- Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Order Service shutting down servers...")

	grpcServer.GracefulStop()
	log.Println("Order gRPC server gracefully stopped.")

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := httpServer.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Order HTTP server shutdown failed: %v", err)
	}
	log.Println("Order HTTP server gracefully stopped.")
	log.Println("Order Service shut down.")
}