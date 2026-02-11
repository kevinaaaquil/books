package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/kevinaaaquil/books/backend/middleware"
	"github.com/kevinaaaquil/books/backend/models"
	"github.com/kevinaaaquil/books/backend/store"
	"github.com/kevinaaaquil/books/backend/utils"
)

type EmailConfigHandler struct {
	DB    *store.DB
	EncKey []byte // 32 bytes for AES-256; nil means store/return app password in plaintext (not recommended)
}

type EmailConfigResponse struct {
	AppSpecificPassword string `json:"appSpecificPassword"`
	ICloudMail          string `json:"icloudMail"`
	SenderMail          string `json:"senderMail"`
	KindleMail          string `json:"kindleMail"`
}

type SaveEmailConfigRequest struct {
	AppSpecificPassword string `json:"appSpecificPassword"`
	ICloudMail          string `json:"icloudMail"`
	SenderMail          string `json:"senderMail"`
	KindleMail          string `json:"kindleMail"`
}

// Get returns the current user's Kindle config. Password is decrypted when EncKey is set.
func (h *EmailConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	cfg, err := h.DB.GetEmailConfig(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"failed to load Kindle config"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if cfg == nil {
		json.NewEncoder(w).Encode(EmailConfigResponse{})
		return
	}
	password := cfg.AppSpecificPassword
	if len(h.EncKey) == 32 && password != "" {
		dec, err := utils.Decrypt(password, h.EncKey)
		if err != nil {
			log.Printf("kindle config: decrypt app password: %v", err)
			password = ""
		} else {
			password = dec
		}
	}
	json.NewEncoder(w).Encode(EmailConfigResponse{
		AppSpecificPassword: password,
		ICloudMail:          cfg.ICloudMail,
		SenderMail:          cfg.SenderMail,
		KindleMail:          cfg.KindleMail,
	})
}

// Save creates or updates the current user's Kindle config. App-specific password is encrypted at rest when EncKey is set.
func (h *EmailConfigHandler) Save(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	var req SaveEmailConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}
	passwordToStore := req.AppSpecificPassword
	if len(h.EncKey) == 32 && passwordToStore != "" {
		enc, err := utils.Encrypt([]byte(passwordToStore), h.EncKey)
		if err != nil {
			log.Printf("kindle config: encrypt app password: %v", err)
			http.Error(w, `{"error":"failed to encrypt password"}`, http.StatusInternalServerError)
			return
		}
		passwordToStore = enc
	}
	cfg := &models.EmailConfig{
		AppSpecificPassword: passwordToStore,
		ICloudMail:          req.ICloudMail,
		SenderMail:          req.SenderMail,
		KindleMail:          req.KindleMail,
	}
	if err := h.DB.UpsertEmailConfig(r.Context(), userID, cfg); err != nil {
		http.Error(w, `{"error":"failed to save Kindle config"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(EmailConfigResponse{
		AppSpecificPassword: req.AppSpecificPassword,
		ICloudMail:          cfg.ICloudMail,
		SenderMail:          cfg.SenderMail,
		KindleMail:          cfg.KindleMail,
	})
}
