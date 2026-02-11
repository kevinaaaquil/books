package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	mail "github.com/go-mail/mail/v2"
	"github.com/kevinaaaquil/books/backend/middleware"
	"github.com/kevinaaaquil/books/backend/models"
	"github.com/kevinaaaquil/books/backend/service"
	"github.com/kevinaaaquil/books/backend/store"
	"github.com/kevinaaaquil/books/backend/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const iCloudSMTPHost = "smtp.mail.me.com"
const iCloudSMTPPort = 587

type BooksHandler struct {
	DB     *store.DB
	S3     *service.S3Service
	EncKey []byte // 32 bytes for decrypting Kindle app password; nil = not set
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
	role := middleware.RoleFromContext(r.Context())
	var books []models.Book
	var err error
	if role == models.RoleGuest {
		books, err = h.DB.BooksVisibleToGuest(r.Context())
	} else {
		books, err = h.DB.AllBooks(r.Context())
	}
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
	role := middleware.RoleFromContext(r.Context())
	if role == models.RoleGuest && !book.ViewByGuest {
		http.Error(w, `{"error":"book not found"}`, http.StatusNotFound)
		return
	}
	setCoverURLIfExtracted(book)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(book)
}

// setCoverURLIfExtracted sets book.CoverURL / ThumbnailURL when an extracted cover is stored, and always sets ExtractedCoverURL when CoverS3Key is set so the frontend can toggle.
func setCoverURLIfExtracted(book *models.Book) {
	if book.CoverS3Key == "" {
		return
	}
	extractedURL := "/api/books/" + book.ID.Hex() + "/cover"
	book.ExtractedCoverURL = extractedURL
	if book.CoverURL == "" {
		book.CoverURL = extractedURL
	}
	if book.ThumbnailURL == "" {
		book.ThumbnailURL = extractedURL
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
	role := middleware.RoleFromContext(r.Context())
	if role == models.RoleGuest && !book.ViewByGuest {
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
	isbn := strings.ReplaceAll(strings.TrimSpace(req.ISBN), "-", "")
	if isbn == "" {
		isbn = strings.ReplaceAll(strings.TrimSpace(book.ISBN), "-", "")
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

type PatchViewByGuestRequest struct {
	ViewByGuest bool `json:"viewByGuest"`
}

// PatchViewByGuest sets whether a book is visible to guests (admin only).
func (h *BooksHandler) PatchViewByGuest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPut {
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
	var req PatchViewByGuestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}
	if err := h.DB.UpdateBookViewByGuest(r.Context(), id, req.ViewByGuest); err != nil {
		http.Error(w, `{"error":"book not found"}`, http.StatusNotFound)
		return
	}
	book, _ := h.DB.BookByID(r.Context(), id)
	setCoverURLIfExtracted(book)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(book)
}

// SendToKindleResponse is returned on 400 when Kindle config is not set up.
type SendToKindleErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// SendToKindle sends the book file to the user's Kindle email using their Kindle config (iCloud SMTP).
func (h *BooksHandler) SendToKindle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
	book, err := h.DB.BookByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"book not found"}`, http.StatusNotFound)
		return
	}
	role := middleware.RoleFromContext(r.Context())
	if role == models.RoleGuest && !book.ViewByGuest {
		http.Error(w, `{"error":"book not found"}`, http.StatusNotFound)
		return
	}
	cfg, err := h.DB.GetEmailConfig(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"failed to load Kindle config"}`, http.StatusInternalServerError)
		return
	}
	if cfg == nil || cfg.KindleMail == "" || cfg.SenderMail == "" || cfg.AppSpecificPassword == "" || cfg.ICloudMail == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SendToKindleErrorResponse{
			Error: "Kindle config required. Set up your Kindle email in Kindle setup.",
			Code:  "KINDLE_CONFIG_REQUIRED",
		})
		return
	}
	appPassword := cfg.AppSpecificPassword
	if len(h.EncKey) == 32 && appPassword != "" {
		dec, err := utils.Decrypt(appPassword, h.EncKey)
		if err != nil {
			log.Printf("send-to-kindle: decrypt app password: %v", err)
			http.Error(w, `{"error":"failed to use Kindle config"}`, http.StatusInternalServerError)
			return
		}
		appPassword = dec
	}
	if h.S3 == nil {
		http.Error(w, `{"error":"download not configured"}`, http.StatusServiceUnavailable)
		return
	}
	body, _, err := h.S3.GetObject(r.Context(), book.S3Key)
	if err != nil {
		http.Error(w, `{"error":"failed to load book file"}`, http.StatusInternalServerError)
		return
	}
	defer body.Close()

	m := mail.NewMessage()
	m.SetHeader("From", cfg.SenderMail)
	m.SetHeader("To", cfg.KindleMail)
	m.SetHeader("Subject", book.Title)
	m.SetBody("text/plain", "Sent from Books. Attachment: "+book.OriginalName)
	m.AttachReader(book.OriginalName, body)

	d := mail.NewDialer(iCloudSMTPHost, iCloudSMTPPort, cfg.ICloudMail, appPassword)
	d.StartTLSPolicy = mail.MandatoryStartTLS
	if err := d.DialAndSend(m); err != nil {
		log.Printf("send-to-kindle: %v", err)
		http.Error(w, `{"error":"failed to send to Kindle: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	emailLog := &models.EmailLog{
		BookID:    id,
		FileTitle: book.Title,
		ToEmail:   cfg.KindleMail,
		UserID:    userID,
		UserEmail: middleware.EmailFromContext(r.Context()),
		SentAt:    time.Now(),
	}
	if err := h.DB.InsertEmailLog(r.Context(), emailLog); err != nil {
		log.Printf("send-to-kindle: failed to insert email log: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Sent to Kindle", "kindleMail": cfg.KindleMail})
}
