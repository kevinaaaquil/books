package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kevinaaaquil/books/backend/middleware"
	"github.com/kevinaaaquil/books/backend/models"
	"github.com/kevinaaaquil/books/backend/service"
	"github.com/kevinaaaquil/books/backend/store"
	"github.com/kevinaaaquil/books/backend/utils"
)

// downloadImage fetches an image from url with a timeout. Returns body, Content-Type, and error.
func downloadImage(url string, timeout time.Duration) ([]byte, string, error) {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("cover URL returned %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/jpeg"
	}
	return body, ct, nil
}

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
	ID          string `json:"id"`
	Title       string `json:"title,omitempty"`
	NoISBNFound bool   `json:"noISBNFound,omitempty"` // true when EPUB had no ISBN so metadata was not fetched
}

func (h *UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := middleware.UserIDFromContext(r.Context()); !ok {
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

	s3Prefix := "books/"
	contentType := contentTypePDF
	format := "pdf"
	if ext == ".epub" || strings.HasPrefix(partContentType, "application/epub+zip") {
		contentType = contentTypeEPUB
		format = "epub"
	}

	uploadedBy := middleware.EmailFromContext(r.Context())
	fileNameTitle := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))

	var noISBNFound bool
	var bookKey string
	var bookKeyErr error
	var meta *service.BookMetadata
	var coverS3Key string
	var wg sync.WaitGroup

	// Run book S3 upload in parallel with metadata and cover work so total time ≈ max(book upload, metadata, cover).
	wg.Add(1)
	go func() {
		defer wg.Done()
		k, e := h.S3.Upload(r.Context(), s3Prefix, header.Filename, bytes.NewReader(fileBytes), contentType)
		bookKey, bookKeyErr = k, e
	}()

	if format == "epub" {
		wg.Add(2)

		go func() {
			defer wg.Done()
			isbn, err := utils.ExtractISBNFromMultipartFile(bytes.NewReader(fileBytes))
			if err != nil || isbn == "" {
				return
			}
			m, err := service.FetchMetadataByISBN(isbn)
			if err != nil {
				return
			}
			meta = m
		}()

		go func() {
			defer wg.Done()
			coverBytes, coverContentType, err := utils.ExtractCoverFromEPUBBytes(fileBytes)
			if err != nil || len(coverBytes) == 0 {
				return
			}
			coverExt := ".jpg"
			if strings.Contains(coverContentType, "png") {
				coverExt = ".png"
			}
			key, err := h.S3.Upload(r.Context(), "books/covers/", "cover"+coverExt, bytes.NewReader(coverBytes), coverContentType)
			if err != nil {
				return
			}
			coverS3Key = key
		}()
	}

	wg.Wait()

	if bookKeyErr != nil {
		http.Error(w, `{"error":"failed to upload to storage"}`, http.StatusInternalServerError)
		return
	}

	book := &models.Book{
		Format:          format,
		S3Key:           bookKey,
		OriginalName:    header.Filename,
		UploadedByEmail: uploadedBy,
		CreatedAt:       time.Now(),
		Title:           fileNameTitle,
	}

	if format == "epub" {
		if meta != nil {
			if meta.Title != "" {
				book.Title = meta.Title
			}
			book.Authors = meta.Authors
			book.Publisher = meta.Publisher
			book.PublishDate = meta.PublishDate
			book.ISBN = meta.ISBN
			book.PageCount = meta.PageCount
			book.CoverURL = meta.CoverURL
			book.ThumbnailURL = meta.ThumbnailURL
			book.Edition = meta.Edition
			book.Preface = meta.Preface
			book.Category = meta.Category
			book.Categories = meta.Categories
			book.RatingAverage = meta.RatingAverage
			book.RatingCount = meta.RatingCount
		} else {
			noISBNFound = true
		}
		if coverS3Key != "" {
			book.CoverS3Key = coverS3Key
		} else if meta != nil && meta.CoverURL != "" {
			// Store API cover in S3 so we don't depend on slow/unreliable external URLs when displaying.
			if imgBytes, contentType, err := downloadImage(meta.CoverURL, 10*time.Second); err == nil && len(imgBytes) > 0 {
				ext := ".jpg"
				if strings.Contains(contentType, "png") {
					ext = ".png"
				}
				if apiCoverKey, err := h.S3.Upload(r.Context(), "books/covers/", "cover"+ext, bytes.NewReader(imgBytes), contentType); err == nil {
					book.CoverS3Key = apiCoverKey
				}
			}
		}
	}

	id, err := h.DB.InsertBook(r.Context(), book)
	if err != nil {
		http.Error(w, `{"error":"failed to save book record"}`, http.StatusInternalServerError)
		return
	}
	book.ID = id

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(UploadResponse{ID: id.Hex(), Title: book.Title, NoISBNFound: noISBNFound})
}
