package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kevinaaaquil/books/backend/middleware"
	"github.com/kevinaaaquil/books/backend/models"
	"github.com/kevinaaaquil/books/backend/service"
	"github.com/kevinaaaquil/books/backend/store"
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
	if _, ok := middleware.UserIDFromContext(r.Context()); !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	books, err := h.DB.AllBooks(r.Context())
	if err != nil {
		http.Error(w, `{"error":"failed to list books"}`, http.StatusInternalServerError)
		return
	}
	for i := range books {
		setCoverURLIfExtracted(&books[i])
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(books)
}

func (h *BooksHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := middleware.UserIDFromContext(r.Context()); !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid book id"}`, http.StatusBadRequest)
		return
	}
	book, err := h.DB.BookByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"book not found"}`, http.StatusNotFound)
		return
	}
	setCoverURLIfExtracted(book)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(book)
}

// setCoverURLIfExtracted sets book.CoverURL to the API cover endpoint when an extracted cover is stored (CoverS3Key), so the client can use it as main or alternate. Only sets when CoverURL is empty so API-fetched cover takes precedence.
func setCoverURLIfExtracted(book *models.Book) {
	if book.CoverS3Key == "" {
		return
	}
	if book.CoverURL == "" {
		book.CoverURL = "/api/books/" + book.ID.Hex() + "/cover"
	}
	if book.ThumbnailURL == "" {
		book.ThumbnailURL = book.CoverURL
	}
}

// Cover streams the book's extracted cover image from S3 (e.g. cover.jpeg from EPUB). GET /api/books/:id/cover (public so img src works).
func (h *BooksHandler) Cover(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid book id"}`, http.StatusBadRequest)
		return
	}
	book, err := h.DB.BookByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"book not found"}`, http.StatusNotFound)
		return
	}
	if book.CoverS3Key == "" || h.S3 == nil {
		http.Error(w, `{"error":"no cover"}`, http.StatusNotFound)
		return
	}
	body, contentType, err := h.S3.GetObject(r.Context(), book.CoverS3Key)
	if err != nil {
		http.Error(w, `{"error":"failed to load cover"}`, http.StatusInternalServerError)
		return
	}
	defer body.Close()
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	io.Copy(w, body)
}

type DownloadResponse struct {
	URL string `json:"url"`
}

func (h *BooksHandler) Download(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := middleware.UserIDFromContext(r.Context()); !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid book id"}`, http.StatusBadRequest)
		return
	}
	book, err := h.DB.BookByID(r.Context(), id)
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

func (h *BooksHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := middleware.UserIDFromContext(r.Context()); !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid book id"}`, http.StatusBadRequest)
		return
	}
	s3Key, coverS3Key, err := h.DB.DeleteBook(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"book not found"}`, http.StatusNotFound)
		return
	}
	if h.S3 != nil {
		if s3Key != "" {
			_ = h.S3.Delete(r.Context(), s3Key)
		}
		if coverS3Key != "" {
			_ = h.S3.Delete(r.Context(), coverS3Key)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

type RefreshMetadataRequest struct {
	ISBN string `json:"isbn"`
}

// RefreshMetadata refetches book metadata by ISBN and updates the book. If body.isbn is provided, uses it (overwrites book ISBN); otherwise uses book's current ISBN.
func (h *BooksHandler) RefreshMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := middleware.UserIDFromContext(r.Context()); !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid book id"}`, http.StatusBadRequest)
		return
	}
	book, err := h.DB.BookByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"book not found"}`, http.StatusNotFound)
		return
	}
	var req RefreshMetadataRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	isbn := strings.TrimSpace(req.ISBN)
	if isbn == "" {
		isbn = strings.TrimSpace(book.ISBN)
	}
	if isbn == "" {
		http.Error(w, `{"error":"no ISBN provided and book has no ISBN"}`, http.StatusBadRequest)
		return
	}
	meta, err := service.FetchMetadataByISBN(isbn)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch metadata: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}
	book.ISBN = meta.ISBN
	if meta.Title != "" {
		book.Title = meta.Title
	}
	book.Authors = meta.Authors
	book.Publisher = meta.Publisher
	book.PublishDate = meta.PublishDate
	book.PageCount = meta.PageCount
	book.CoverURL = meta.CoverURL
	book.ThumbnailURL = meta.ThumbnailURL
	book.Edition = meta.Edition
	book.Preface = meta.Preface
	book.Category = meta.Category
	book.Categories = meta.Categories
	book.RatingAverage = meta.RatingAverage
	book.RatingCount = meta.RatingCount
	if err := h.DB.UpdateBookMetadata(r.Context(), id, book); err != nil {
		http.Error(w, `{"error":"failed to update book"}`, http.StatusInternalServerError)
		return
	}
	book, _ = h.DB.BookByID(r.Context(), id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(book)
}
