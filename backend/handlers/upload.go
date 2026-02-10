package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/kevinaaaquil/books/backend/middleware"
	"github.com/kevinaaaquil/books/backend/models"
	"github.com/kevinaaaquil/books/backend/service"
	"github.com/kevinaaaquil/books/backend/store"
	"github.com/kevinaaaquil/books/backend/utils"
)

const (
	contentTypeEPUB = "application/epub+zip"
	contentTypePDF  = "application/pdf"
)

type UploadHandler struct {
	DB        *store.DB
	S3        *service.S3Service
	MaxBytes  int64
}

type UploadResponse struct {
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
}

func (h *UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	if h.MaxBytes > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, h.MaxBytes)
	}
	if err := r.ParseMultipartForm(h.MaxBytes); err != nil {
		http.Error(w, `{"error":"failed to parse multipart form"}`, http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error":"missing file"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	if h.S3 == nil {
		http.Error(w, `{"error":"upload not configured (missing S3)"}`, http.StatusServiceUnavailable)
		return
	}
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(header.Filename)))
	partContentType := header.Header.Get("Content-Type")
	allowedByExt := ext == ".epub" || ext == ".pdf"
	allowedByMime := strings.HasPrefix(partContentType, "application/epub+zip") || strings.HasPrefix(partContentType, "application/pdf")
	if !allowedByExt && !allowedByMime {
		http.Error(w, `{"error":"only epub and pdf are allowed"}`, http.StatusBadRequest)
		return
	}

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, `{"error":"failed to read file"}`, http.StatusInternalServerError)
		return
	}

	s3Prefix := userID.Hex() + "/"
	contentType := contentTypePDF
	format := "pdf"
	if ext == ".epub" || strings.HasPrefix(partContentType, "application/epub+zip") {
		contentType = contentTypeEPUB
		format = "epub"
	}

	key, err := h.S3.Upload(r.Context(), s3Prefix, header.Filename, bytes.NewReader(fileBytes), contentType)
	if err != nil {
		http.Error(w, `{"error":"failed to upload to storage"}`, http.StatusInternalServerError)
		return
	}

	book := &models.Book{
		UserID:       userID,
		Format:       format,
		S3Key:        key,
		OriginalName: header.Filename,
		CreatedAt:    time.Now(),
	}

	if format == "epub" {
		isbn, err := utils.ExtractISBNFromMultipartFile(bytes.NewReader(fileBytes))
		if err == nil && isbn != "" {
			meta, err := service.FetchMetadataByISBN(isbn)
			if err == nil {
				book.Title = meta.Title
				book.Authors = meta.Authors
				book.Publisher = meta.Publisher
				book.PublishDate = meta.PublishDate
				book.ISBN = meta.ISBN
				book.PageCount = meta.PageCount
				book.CoverURL = meta.CoverURL
				book.Edition = meta.Edition
			}
			if book.Title == "" {
				book.Title = strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
			}
		} else {
			book.Title = strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
		}
	} else {
		book.Title = strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	}

	id, err := h.DB.InsertBook(r.Context(), book)
	if err != nil {
		http.Error(w, `{"error":"failed to save book record"}`, http.StatusInternalServerError)
		return
	}
	book.ID = id

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(UploadResponse{ID: id.Hex(), Title: book.Title})
}
