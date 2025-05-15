
This is critical for anyone (including you later) to get the project running.

- **Prerequisites:**
    
    - Go (specify version, e.g., 1.21+)
        
    - Docker and Docker Compose (specify versions if important)
        
    - protoc (Protocol Buffer Compiler)
        
    - Make (optional, if you add a Makefile)
        
- **Project Structure Overview (brief).**
    
- **Initial Setup:**
    
    - git clone <repository-url>
        
    - cd microservices-project
        
    - How to install Go dependencies: go mod tidy or go get ./...
        
    - How to generate gRPC and Protobuf code:
        
        ```
        # From project root
        protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative protos/*.proto
        ```
        
        content_copydownload
        
        Use code [with caution](https://support.google.com/legal/answer/13505487).Bash
        
- **Database Setup:**
    
    - Instructions for setting up PostgreSQL (if not using Docker for DB initially).
        
    - SQL DDL statements to create tables (or mention they are in database_setup.sql).
        
    - How to configure database connection strings (environment variables).
        
- **Running Services Locally (Without Docker):**
    
    - List environment variables needed for each service (e.g., DB_HOST, DB_USER, USER_SERVICE_GRPC_ADDR, etc.).
        
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
        
        Use code [with caution](https://support.google.com/legal/answer/13505487).Bash
        
- **Running Services with Docker Compose (Recommended):**
    
    - Explain the docker-compose.yml file.
        
    - Command: docker-compose up --build
        
    - Command to stop: docker-compose down
        
- **Running Tests:**
    
    - go test ./...
        
    - go test -race ./... (for race detection)
        
    - go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out (for coverage)
        
- **Running Benchmarks:**
    
    - go test -bench=. ./...
        
- **Accessing APIs:**
    
    - List default HTTP ports for each service (e.g., UserService: 8081, ProductService: 8082, OrderService: 8083).
        
    - List default gRPC ports (e.g., UserService: 50051, etc.).
        
    - Provide a few `curl` examples or Postman collection link.

### `curl` Examples:

Assuming services are running and accessible on `localhost` with default HTTP ports:

**UserService (HTTP Port: 8081 by default)**

*   **Create User:**

    ```bash
    curl -X POST -H "Content-Type: application/json" -d '{
      "username": "testuser",
      "email": "test@example.com",
      "password": "password123"
    }' http://localhost:8081/users
    ```

*   **Get User (replace `:userId` with an actual ID):**

    ```bash
    curl http://localhost:8081/users/:userId
    ```

*   **Login User:**

    ```bash
    curl -X POST -H "Content-Type: application/json" -d '{
      "email": "test@example.com",
      "password": "password123"
    }' http://localhost:8081/login
    ```

**ProductService (HTTP Port: 8082 by default)**

*   **Create Product:**

    ```bash
    curl -X POST -H "Content-Type: application/json" -d '{
      "name": "Awesome Laptop",
      "description": "A very powerful laptop",
      "price": 1299.99,
      "stock_quantity": 50
    }' http://localhost:8082/products
    ```

*   **Get Product (replace `:productId` with an actual ID):**

    ```bash
    curl http://localhost:8082/products/:productId
    ```

*   **List Products:**

    ```bash
    curl http://localhost:8082/products
    ```

**OrderService (HTTP Port: 8083 by default)**

*   **Create Order (replace `userId` and `productId` with actual IDs):**

    ```bash
    curl -X POST -H "Content-Type: application/json" -d '{
      "user_id": "some-user-id",
      "items": [
        {
          "product_id": "some-product-id",
          "quantity": 2
        },
        {
          "product_id": "another-product-id",
          "quantity": 1
        }
      ]
    }' http://localhost:8083/orders
    ```

*   **Get Order (replace `:orderId` with an actual ID):**

    ```bash
    curl http://localhost:8083/orders/:orderId
    ```

*   **List User Orders (replace `:userId` with an actual ID):**

    ```bash
    curl http://localhost:8083/users/:userId/orders
    ```

### `gcurl` Examples:

Make sure you have `gcurl` installed (`go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest`).

Proto files are assumed to be in a `protos` directory relative to where you run `grpcurl`, or use the `-proto` flag to specify the path to each `.proto` file or a directory containing them.

The gRPC ports are (by default):
*   UserService: `localhost:50051`
*   ProductService: `localhost:50052`
*   OrderService: `localhost:50053`

**UserService (gRPC Port: 50051)**

*   **List Methods:**

    ```bash
    grpcurl -plaintext localhost:50051 list
    grpcurl -plaintext localhost:50051 list user.UserService
    ```

*   **Create User:**

    ```bash
    grpcurl -plaintext -d '{
      "username": "testgrpcurluser",
      "email": "grpcurl@example.com",
      "password": "securepassword"
    }' localhost:50051 user.UserService/CreateUser
    ```

*   **Get User (replace `some-user-id` with an actual ID from CreateUser response or DB):**

    ```bash
    grpcurl -plaintext -d '{
      "user_id": "some-user-id"
    }' localhost:50051 user.UserService/GetUser
    ```

*   **Login User:**

    ```bash
    grpcurl -plaintext -d '{
      "email": "grpcurl@example.com",
      "password": "securepassword"
    }' localhost:50051 user.UserService/LoginUser
    ```

**ProductService (gRPC Port: 50052)**

*   **List Methods:**

    ```bash
    grpcurl -plaintext localhost:50052 list product.ProductService
    ```

*   **Create Product:**

    ```bash
    grpcurl -plaintext -d '{
      "name": "gRPC Widget",
      "description": "A widget created via gRPC",
      "price": 19.99,
      "stock_quantity": 100
    }' localhost:50052 product.ProductService/CreateProduct
    ```

*   **Get Product (replace `some-product-id` with an actual ID):**

    ```bash
    grpcurl -plaintext -d '{
      "product_id": "some-product-id"
    }' localhost:50052 product.ProductService/GetProduct
    ```

*   **List Products:**

    ```bash
    grpcurl -plaintext localhost:50052 product.ProductService/ListProducts
    # With pagination
    grpcurl -plaintext -d '{"page_size": 5}' localhost:50052 product.ProductService/ListProducts
    ```

**OrderService (gRPC Port: 50053)**

*   **List Methods:**

    ```bash
    grpcurl -plaintext localhost:50053 list order.OrderService
    ```

*   **Create Order (replace `userId` and `productId` with actual IDs):**

    ```bash
    grpcurl -plaintext -d '{
      "user_id": "some-user-id",
      "items": [
        {
          "product_id": "some-product-id",
          "quantity": 1
        }
      ]
    }' localhost:50053 order.OrderService/CreateOrder
    ```

*   **Get Order (replace `some-order-id` with an actual ID):**

    ```bash
    grpcurl -plaintext -d '{
      "order_id": "some-order-id"
    }' localhost:50053 order.OrderService/GetOrder
    ```

*   **List User Orders (replace `some-user-id` with an actual ID):**

    ```bash
    grpcurl -plaintext -d '{
      "user_id": "some-user-id"
    }' localhost:50053 order.OrderService/ListUserOrders
    ```

**Note on `grpcurl` with all protos in one directory:**
If all your `.proto` files (`user.proto`, `product.proto`, `order.proto`) are in the `./protos` directory, you can simplify the `grpcurl` commands by adding `-proto protos/*.proto` or by navigating into the `protos` directory and running `grpcurl` from there (then you might not need `-import-path` or `-proto` flags if your `go_package` options are set up to allow generation from that relative path, but explicitly providing proto paths is often more robust).

Example using `-proto` flag from project root:
```bash
grpcurl -proto protos/user.proto -proto protos/product.proto -proto protos/order.proto -plaintext -d '{
  "user_id": "some-user-id"
}' localhost:50053 order.OrderService/ListUserOrders
```
Or, if `grpcurl` can discover all necessary types from the entry-point proto (e.g., `order.proto` for `OrderService`):
```bash
grpcurl -proto protos/order.proto -plaintext -d '{
  "user_id": "some-user-id"
}' localhost:50053 order.OrderService/ListUserOrders
```
However, for services that depend on types from other proto files (like `google/protobuf/timestamp.proto`), you might also need `-import-path` to point to where `grpcurl` can find these standard protos if they are not in the default include paths. Often, `grpcurl` finds standard types automatically.