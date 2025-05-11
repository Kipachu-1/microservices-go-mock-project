// internal/productservice/handler/http_handler.go
package handler

import (
	"errors"
	"log"
	"microservices-project/internal/productservice/service"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type ProductHTTPHandler struct {
	productService service.ProductServiceInterface
}

func NewProductHTTPHandler(productService service.ProductServiceInterface) *ProductHTTPHandler {
	return &ProductHTTPHandler{
		productService: productService,
	}
}

func (h *ProductHTTPHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Post("/products", h.createProduct)
	r.Get("/products/{productID}", h.getProduct)
	r.Get("/products", h.listProducts)
	r.Put("/products/{productID}", h.updateProduct)
	r.Delete("/products/{productID}", h.deleteProduct)
	// UpdateStock is likely internal via gRPC, but could be exposed for admin if needed
	// r.Patch("/products/{productID}/stock", h.updateStock) // Example for PATCH to update stock

	return r
}

// --- DTOs for HTTP ---
type ProductHTTPRequest struct {
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	Price         float64 `json:"price"`
	StockQuantity int32   `json:"stock_quantity"`
}

func (p *ProductHTTPRequest) Bind(r *http.Request) error {
	if p.Name == "" {
		return errors.New("product name is required")
	}
	if p.Price < 0 {
		return errors.New("product price cannot be negative")
	}
	if p.StockQuantity < 0 {
		return errors.New("stock quantity cannot be negative")
	}
	return nil
}

// Re-use model.Product for response, or create a specific HTTP response DTO
// For simplicity, we can marshal model.Product directly or adapt it.
// Let's use a function to convert model.Product to a map for render.JSON if needed,
// or rely on struct tags if model.Product is suitable.
// `render.JSON` works well with struct tags directly.

func (h *ProductHTTPHandler) createProduct(w http.ResponseWriter, r *http.Request) {
	data := &ProductHTTPRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	log.Printf("HTTP CreateProduct request: Name=%s", data.Name)
	product, err := h.productService.CreateProduct(r.Context(), data.Name, data.Description, data.Price, data.StockQuantity)
	if err != nil {
		log.Printf("Error creating product via HTTP: %v", err)
		if errors.Is(err, service.ErrInvalidProductData) {
			render.Status(r, http.StatusBadRequest)
		} else {
			render.Status(r, http.StatusInternalServerError)
		}
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, product) // model.Product has json tags
}

func (h *ProductHTTPHandler) getProduct(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "productID")
	log.Printf("HTTP GetProduct request: ID=%s", productID)

	product, err := h.productService.GetProductByID(r.Context(), productID)
	if err != nil {
		log.Printf("Error getting product via HTTP: %v", err)
		if errors.Is(err, service.ErrProductNotFound) {
			render.Status(r, http.StatusNotFound)
		} else if errors.Is(err, service.ErrInvalidProductData) {
			render.Status(r, http.StatusBadRequest)
		} else {
			render.Status(r, http.StatusInternalServerError)
		}
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, product)
}

func (h *ProductHTTPHandler) listProducts(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("pageSize")

	page, _ := strconv.Atoi(pageStr)
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}

	log.Printf("HTTP ListProducts request: Page=%d, PageSize=%d", page, pageSize)
	products, err := h.productService.ListProducts(r.Context(), page, pageSize)
	if err != nil {
		log.Printf("Error listing products via HTTP: %v", err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to list products"})
		return
	}
	// TODO: Could wrap this in a response struct with pagination info
	render.Status(r, http.StatusOK)
	render.JSON(w, r, products)
}

func (h *ProductHTTPHandler) updateProduct(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "productID")
	data := &ProductHTTPRequest{} // Use the same DTO for update payload
	if err := render.Bind(r, data); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	log.Printf("HTTP UpdateProduct request: ID=%s, Name=%s", productID, data.Name)
	product, err := h.productService.UpdateProduct(r.Context(), productID, data.Name, data.Description, data.Price, data.StockQuantity)
	if err != nil {
		log.Printf("Error updating product via HTTP: %v", err)
		if errors.Is(err, service.ErrProductNotFound) {
			render.Status(r, http.StatusNotFound)
		} else if errors.Is(err, service.ErrInvalidProductData) {
			render.Status(r, http.StatusBadRequest)
		} else {
			render.Status(r, http.StatusInternalServerError)
		}
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, product)
}

func (h *ProductHTTPHandler) deleteProduct(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "productID")
	log.Printf("HTTP DeleteProduct request: ID=%s", productID)

	err := h.productService.DeleteProduct(r.Context(), productID)
	if err != nil {
		log.Printf("Error deleting product via HTTP: %v", err)
		if errors.Is(err, service.ErrProductNotFound) {
			render.Status(r, http.StatusNotFound)
		} else if errors.Is(err, service.ErrInvalidProductData) {
			render.Status(r, http.StatusBadRequest)
		} else {
			render.Status(r, http.StatusInternalServerError)
		}
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}
	render.Status(r, http.StatusOK) // Or http.StatusNoContent (204)
	render.JSON(w, r, map[string]string{"message": "Product deleted successfully"})
}