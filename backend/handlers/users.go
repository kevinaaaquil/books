package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kevinaaaquil/books/backend/middleware"
	"github.com/kevinaaaquil/books/backend/models"
	"github.com/kevinaaaquil/books/backend/store"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

type UsersHandler struct {
	DB *store.DB
}

type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type CreateUserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type UserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	CreatedAt string `json:"createdAt"`
}

type UpdateUserRequest struct {
	Email    *string `json:"email"`
	Password *string `json:"password"`
	Role     *string `json:"role"`
}

func roleValid(role string) bool {
	for _, r := range models.ValidRoles {
		if r == role {
			return true
		}
	}
	return false
}

// CreateUser creates a new user. Only admin can call. Role must be viewer, editor, or write_only (not admin).
func (h *UsersHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		http.Error(w, `{"error":"email and password required"}`, http.StatusBadRequest)
		return
	}
	role := strings.TrimSpace(strings.ToLower(req.Role))
	if role == "" {
		role = models.RoleViewer
	}
	if role == models.RoleAdmin {
		http.Error(w, `{"error":"cannot create admin user via API"}`, http.StatusBadRequest)
		return
	}
	if !roleValid(role) {
		http.Error(w, `{"error":"invalid role; use viewer, editor, or write_only"}`, http.StatusBadRequest)
		return
	}
	existing, err := h.DB.UserByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, `{"error":"failed to create user"}`, http.StatusInternalServerError)
		return
	}
	if existing != nil {
		http.Error(w, `{"error":"email already in use"}`, http.StatusConflict)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, `{"error":"failed to create user"}`, http.StatusInternalServerError)
		return
	}
	user := &models.User{
		Email:     req.Email,
		Password:  string(hash),
		Role:      role,
		CreatedAt: time.Now(),
	}
	id, err := h.DB.CreateUser(r.Context(), user)
	if err != nil {
		http.Error(w, `{"error":"failed to create user"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreateUserResponse{
		ID:    id.Hex(),
		Email: user.Email,
		Role:  user.Role,
	})
}

func userToResponse(u *models.User) UserResponse {
	return UserResponse{
		ID:        u.ID.Hex(),
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
	}
}

// ListUsers returns all users (admin only). Password is omitted via json:"-".
func (h *UsersHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	users, err := h.DB.ListUsers(r.Context())
	if err != nil {
		http.Error(w, `{"error":"failed to list users"}`, http.StatusInternalServerError)
		return
	}
	out := make([]UserResponse, 0, len(users))
	for i := range users {
		out = append(out, userToResponse(&users[i]))
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// UpdateUser updates a user by ID (admin only). Body: { "email"?, "password"?, "role"? }
func (h *UsersHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid user id"}`, http.StatusBadRequest)
		return
	}
	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}
	user, err := h.DB.UserByID(r.Context(), id)
	if err != nil || user == nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}
	var newEmail *string
	if req.Email != nil {
		e := strings.TrimSpace(strings.ToLower(*req.Email))
		if e == "" {
			http.Error(w, `{"error":"email cannot be empty"}`, http.StatusBadRequest)
			return
		}
		existing, _ := h.DB.UserByEmail(r.Context(), e)
		if existing != nil && existing.ID != id {
			http.Error(w, `{"error":"email already in use"}`, http.StatusConflict)
			return
		}
		newEmail = &e
	}
	var newHash *string
	if req.Password != nil && *req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, `{"error":"failed to update user"}`, http.StatusInternalServerError)
			return
		}
		s := string(hash)
		newHash = &s
	}
	var newRole *string
	if req.Role != nil {
		r := strings.TrimSpace(strings.ToLower(*req.Role))
		if !roleValid(r) {
			http.Error(w, `{"error":"invalid role"}`, http.StatusBadRequest)
			return
		}
		// Only allow setting admin via update if needed; for simplicity we allow it for admin caller
		newRole = &r
	}
	if err := h.DB.UpdateUser(r.Context(), id, newEmail, newHash, newRole); err != nil {
		http.Error(w, `{"error":"failed to update user"}`, http.StatusInternalServerError)
		return
	}
	user, _ = h.DB.UserByID(r.Context(), id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userToResponse(user))
}

// DeleteUser deletes a user by ID (admin only). Prevents deleting self.
func (h *UsersHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid user id"}`, http.StatusBadRequest)
		return
	}
	currentID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	if currentID == id {
		http.Error(w, `{"error":"cannot delete your own account"}`, http.StatusBadRequest)
		return
	}
	user, err := h.DB.UserByID(r.Context(), id)
	if err != nil || user == nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}
	if user.Role == models.RoleAdmin {
		count, err := h.DB.AdminsCount(r.Context())
		if err != nil {
			http.Error(w, `{"error":"failed to delete user"}`, http.StatusInternalServerError)
			return
		}
		if count <= 1 {
			http.Error(w, `{"error":"cannot delete the last admin user"}`, http.StatusBadRequest)
			return
		}
	}
	if err := h.DB.DeleteUser(r.Context(), id); err != nil {
		http.Error(w, `{"error":"failed to delete user"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
