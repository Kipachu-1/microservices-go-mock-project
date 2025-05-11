// internal/orderservice/handler/http_handler.go
package handler

import (
	"errors"
	"log"
	"microservices-project/internal/orderservice/model"
	"microservices-project/internal/orderservice/service"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type OrderHTTPHandler struct {
	orderService service.OrderServiceInterface
}

func NewOrderHTTPHandler(orderService service.OrderServiceInterface) *OrderHTTPHandler {
	return &OrderHTTPHandler{
		orderService: orderService,
	}
}

func (h *OrderHTTPHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Post("/orders", h.createOrder)                // Create a new order
	r.Get("/orders/{orderID}", h.getOrder)          // Get a specific order
	r.Get("/users/{userID}/orders", h.listUserOrders) // List orders for a specific user

	return r
}

// --- DTOs for HTTP ---
type CreateOrderHTTPRequestItem struct {
	ProductID string `json:"product_id"`
	Quantity  int32  `json:"quantity"`
}

type CreateOrderHTTPRequest struct {
	UserID string                     `json:"user_id"`
	Items  []CreateOrderHTTPRequestItem `json:"items"`
}

func (req *CreateOrderHTTPRequest) Bind(r *http.Request) error {
	if req.UserID == "" {
		return errors.New("user_id is required")
	}
	if len(req.Items) == 0 {
		return errors.New("at least one item is required in the order")
	}
	for _, item := range req.Items {
		if item.ProductID == "" {
			return errors.New("product_id is required for all items")
		}
		if item.Quantity <= 0 {
			return errors.New("quantity must be positive for all items")
		}
	}
	return nil
}

// We can use model.Order directly for responses as it has JSON tags.

func (h *OrderHTTPHandler) createOrder(w http.ResponseWriter, r *http.Request) {
	data := &CreateOrderHTTPRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	log.Printf("HTTP CreateOrder request for UserID: %s", data.UserID)

	domainItems := make([]model.OrderItem, len(data.Items))
	for i, item := range data.Items {
		domainItems[i] = model.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}

	createdOrder, err := h.orderService.CreateOrder(r.Context(), data.UserID, domainItems)
	if err != nil {
		log.Printf("Error creating order via HTTP: %v", err)
		// More granular error mapping
		if errors.Is(err, service.ErrInvalidOrderData) || errors.Is(err, service.ErrUserValidationFailed) {
			render.Status(r, http.StatusBadRequest) // Or specific codes like 404 for user not found
		} else if errors.Is(err, service.ErrProductFetchFailed) || errors.Is(err, service.ErrInsufficientStockForOrder) {
			render.Status(r, http.StatusConflict) // 409 Conflict if resource unavailable/insufficient
		} else if errors.Is(err, service.ErrProductStockUpdateFailed) {
			render.Status(r, http.StatusInternalServerError) // Or 409 if considered a business rule conflict
		} else {
			render.Status(r, http.StatusInternalServerError)
		}
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, createdOrder)
}

func (h *OrderHTTPHandler) getOrder(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "orderID")
	log.Printf("HTTP GetOrder request for OrderID: %s", orderID)

	if orderID == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "order_id is required"})
		return
	}

	order, err := h.orderService.GetOrderByID(r.Context(), orderID)
	if err != nil {
		log.Printf("Error getting order via HTTP: %v", err)
		if errors.Is(err, service.ErrOrderNotFound) {
			render.Status(r, http.StatusNotFound)
		} else {
			render.Status(r, http.StatusInternalServerError)
		}
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, order)
}

func (h *OrderHTTPHandler) listUserOrders(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "user_id is required"})
		return
	}

	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("pageSize")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	log.Printf("HTTP ListUserOrders request for UserID: %s, Page: %d, PageSize: %d", userID, page, pageSize)

	orders, err := h.orderService.ListUserOrders(r.Context(), userID, page, pageSize)
	if err != nil {
		log.Printf("Error listing user orders via HTTP: %v", err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to list user orders"})
		return
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, orders)
}