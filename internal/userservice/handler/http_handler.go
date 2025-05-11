// internal/userservice/handler/http_handler.go
package handler

import (
	"errors"
	"log"
	"microservices-project/internal/userservice/model"
	"microservices-project/internal/userservice/service"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// UserHTTPHandler handles HTTP requests for the User service.
type UserHTTPHandler struct {
	userService service.UserServiceInterface
}

// NewUserHTTPHandler creates a new UserHTTPHandler.
func NewUserHTTPHandler(userService service.UserServiceInterface) *UserHTTPHandler {
	return &UserHTTPHandler{
		userService: userService,
	}
}

// Routes sets up the routing for user HTTP endpoints.
func (h *UserHTTPHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Use(render.SetContentType(render.ContentTypeJSON)) // Set content-type headers as JSON

	r.Post("/users/register", h.createUser)
	r.Get("/users/{userID}", h.getUser)
	r.Post("/users/login", h.loginUser)
	// Add other routes like PUT /users/{userID}, DELETE /users/{userID} as needed

	return r
}

// --- DTOs for HTTP requests/responses ---
// We can define specific DTOs or reuse/adapt protobuf messages if simple enough.
// For clarity, let's define specific DTOs here.

type CreateUserHTTPRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Bind is a method on the request payload to satisfy the render.Binder interface.
// It can be used for request-time validation or processing.
func (c *CreateUserHTTPRequest) Bind(r *http.Request) error {
	if c.Username == "" || c.Email == "" || c.Password == "" {
		return errors.New("username, email, and password are required")
	}
	// Add more specific validation (e.g., email format, password strength)
	return nil
}

type UserHTTPResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"` // Consider RFC3339 format
	UpdatedAt string `json:"updated_at"`
}

func NewUserHTTPResponse(user *model.User) *UserHTTPResponse {
	return &UserHTTPResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Format(http.TimeFormat), // Or time.RFC3339
		UpdatedAt: user.UpdatedAt.Format(http.TimeFormat), // Or time.RFC3339
	}
}

type LoginHTTPRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (l *LoginHTTPRequest) Bind(r *http.Request) error {
	if l.Email == "" || l.Password == "" {
		return errors.New("email and password are required")
	}
	return nil
}

type LoginHTTPResponse struct {
	Token string            `json:"token"`
	User  *UserHTTPResponse `json:"user"`
}


// --- HTTP Handler Methods ---

// createUser handles POST /users/register
func (h *UserHTTPHandler) createUser(w http.ResponseWriter, r *http.Request) {
	data := &CreateUserHTTPRequest{}
	if err := render.Bind(r, data); err != nil {
		log.Printf("Bad request for createUser: %v", err)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	log.Printf("HTTP CreateUser request received for username: %s, email: %s", data.Username, data.Email)

	createdUser, err := h.userService.CreateUser(r.Context(), data.Username, data.Email, data.Password)
	if err != nil {
		log.Printf("Error creating user via HTTP: %v", err)
		if errors.Is(err, service.ErrUserAlreadyExists) {
			render.Status(r, http.StatusConflict) // 409 Conflict
			render.JSON(w, r, map[string]string{"error": err.Error()})
		} else {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": "Failed to create user"})
		}
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, NewUserHTTPResponse(createdUser))
}

// getUser handles GET /users/{userID}
func (h *UserHTTPHandler) getUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "User ID is required"})
		return
	}

	log.Printf("HTTP GetUser request received for ID: %s", userID)

	user, err := h.userService.GetUserByID(r.Context(), userID)
	if err != nil {
		log.Printf("Error getting user via HTTP: %v", err)
		if errors.Is(err, service.ErrUserNotFound) {
			render.Status(r, http.StatusNotFound) // 404 Not Found
			render.JSON(w, r, map[string]string{"error": err.Error()})
		} else {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": "Failed to retrieve user"})
		}
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, NewUserHTTPResponse(user))
}


// loginUser handles POST /users/login
func (h *UserHTTPHandler) loginUser(w http.ResponseWriter, r *http.Request) {
	data := &LoginHTTPRequest{}
	if err := render.Bind(r, data); err != nil {
		log.Printf("Bad request for loginUser: %v", err)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	log.Printf("HTTP LoginUser request received for email: %s", data.Email)

	user, token, err := h.userService.LoginUser(r.Context(), data.Email, data.Password)
	if err != nil {
		log.Printf("Error during login via HTTP: %v", err)
		if errors.Is(err, service.ErrInvalidCredentials) || errors.Is(err, service.ErrUserNotFound) {
			render.Status(r, http.StatusUnauthorized) // 401 Unauthorized
			render.JSON(w, r, map[string]string{"error": "Invalid email or password"})
		} else {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": "Login failed"})
		}
		return
	}

	response := LoginHTTPResponse{
		Token: token,
		User:  NewUserHTTPResponse(user),
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, response)
}