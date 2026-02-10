package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/kevinaaaquil/books/backend/middleware"
	"github.com/kevinaaaquil/books/backend/models"
	"github.com/kevinaaaquil/books/backend/store"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	DB        *store.DB
	JWTSecret string
	// Predefined credentials (from config); used if no user exists yet
	DefaultEmail string
	DefaultPass  string
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	Email string `json:"email"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		http.Error(w, `{"error":"email and password required"}`, http.StatusBadRequest)
		return
	}

	user, err := h.DB.UserByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, `{"error":"login failed"}`, http.StatusInternalServerError)
		return
	}
	// If no user in DB, accept predefined credentials and optionally seed user (for "predefined" mode)
	if user == nil {
		if req.Email != h.DefaultEmail || req.Password != h.DefaultPass {
			http.Error(w, `{"error":"invalid email or password"}`, http.StatusUnauthorized)
			return
		}
		// Seed user so we have a valid ID for tokens (optional: could use a special "default" user id)
		user, err = h.ensureDefaultUser(r)
		if err != nil {
			http.Error(w, `{"error":"login failed"}`, http.StatusInternalServerError)
			return
		}
	} else {
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			http.Error(w, `{"error":"invalid email or password"}`, http.StatusUnauthorized)
			return
		}
	}

	token, err := h.createToken(user.ID.Hex(), user.Email)
	if err != nil {
		http.Error(w, `{"error":"could not create token"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{Token: token, Email: user.Email})
}

func (h *AuthHandler) ensureDefaultUser(r *http.Request) (*models.User, error) {
	// Check again in case of race
	user, err := h.DB.UserByEmail(r.Context(), h.DefaultEmail)
	if err != nil {
		return nil, err
	}
	if user != nil {
		return user, nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(h.DefaultPass), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	newUser := &models.User{
		Email:     h.DefaultEmail,
		Password:  string(hash),
		CreatedAt: time.Now(),
	}
	id, err := h.DB.CreateUser(r.Context(), newUser)
	if err != nil {
		return nil, err
	}
	newUser.ID = id
	return newUser, nil
}

func (h *AuthHandler) createToken(userID, email string) (string, error) {
	claims := &middleware.Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour * 7)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.JWTSecret))
}
