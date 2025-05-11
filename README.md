
This is critical for anyone (including you later) to get the project running.

- **Prerequisites:**
    
    - Go (specify version, e.g., 1.21+)
        
    - Docker and Docker Compose (specify versions if important)
        
    - protoc (Protocol Buffer Compiler)
        
    - Make (optional, if you add a Makefile)
        
- **Project Structure Overview (brief).**
    
- **Initial Setup:**
    
    - git clone <repository-url>
        
    - cd microservices-project
        
    - How to install Go dependencies: go mod tidy or go get ./...
        
    - How to generate gRPC and Protobuf code:
        
        ```
        # From project root
        protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative protos/*.proto
        ```
        
        content_copydownload
        
        Use code [with caution](https://support.google.com/legal/answer/13505487).Bash
        
- **Database Setup:**
    
    - Instructions for setting up PostgreSQL (if not using Docker for DB initially).
        
    - SQL DDL statements to create tables (or mention they are in database_setup.sql).
        
    - How to configure database connection strings (environment variables).
        
- **Running Services Locally (Without Docker):**
    
    - List environment variables needed for each service (e.g., DB_HOST, DB_USER, USER_SERVICE_GRPC_ADDR, etc.).
        
    - Command to run each service:
        
        ```
        # Terminal 1: UserService
        export DB_USER=... # etc.
        go run cmd/userservice/main.go
        
        # Terminal 2: ProductService
        export DB_USER=... # etc.
        go run cmd/productservice/main.go
        
        # Terminal 3: OrderService
        export DB_USER=... # etc.
        export USER_SERVICE_GRPC_ADDR=localhost:50051
        export PRODUCT_SERVICE_GRPC_ADDR=localhost:50052
        go run cmd/orderservice/main.go
        ```
        
        content_copydownload
        
        Use code [with caution](https://support.google.com/legal/answer/13505487).Bash
        
- **Running Services with Docker Compose (Recommended):**
    
    - Explain the docker-compose.yml file.
        
    - Command: docker-compose up --build
        
    - Command to stop: docker-compose down
        
- **Running Tests:**
    
    - go test ./...
        
    - go test -race ./... (for race detection)
        
    - go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out (for coverage)
        
- **Running Benchmarks:**
    
    - go test -bench=. ./...
        
- **Accessing APIs:**
    
    - List default HTTP ports for each service (e.g., UserService: 8081, ProductService: 8082, OrderService: 8083).
        
    - List default gRPC ports (e.g., UserService: 50051, etc.).
        
    - Provide a few curl examples or Postman collection link.