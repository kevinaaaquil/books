package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Kevin-Aaaquil/books/backend/middleware"
	"github.com/Kevin-Aaaquil/books/backend/service"
	"github.com/Kevin-Aaaquil/books/backend/store"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BooksHandler struct {
	DB *store.DB
	S3 *service.S3Service
}

func (h *BooksHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	books, err := h.DB.BooksByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"failed to list books"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(books)
}

func (h *BooksHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid book id"}`, http.StatusBadRequest)
		return
	}
	book, err := h.DB.BookByID(r.Context(), id, userID)
	if err != nil {
		http.Error(w, `{"error":"book not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(book)
}

type DownloadResponse struct {
	URL string `json:"url"`
}

func (h *BooksHandler) Download(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid book id"}`, http.StatusBadRequest)
		return
	}
	book, err := h.DB.BookByID(r.Context(), id, userID)
	if err != nil {
		http.Error(w, `{"error":"book not found"}`, http.StatusNotFound)
		return
	}
	if h.S3 == nil {
		http.Error(w, `{"error":"download not configured"}`, http.StatusServiceUnavailable)
		return
	}
	url, err := h.S3.PresignedGetURL(r.Context(), book.S3Key, 15*time.Minute)
	if err != nil {
		http.Error(w, `{"error":"failed to generate download url"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DownloadResponse{URL: url})
}
