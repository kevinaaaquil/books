package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/kevinaaaquil/books/backend/config"
	"github.com/kevinaaaquil/books/backend/handlers"
	"github.com/kevinaaaquil/books/backend/middleware"
	"github.com/kevinaaaquil/books/backend/models"
	"github.com/kevinaaaquil/books/backend/service"
	"github.com/kevinaaaquil/books/backend/store"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	_ = godotenv.Load()
	config.ValidateEnv()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config:", err)
	}

	ctx := context.Background()
	db, err := store.NewMongoDB(ctx, cfg.MongoURI, cfg.DBName)
	if err != nil {
		log.Fatal("mongodb:", err)
	}
	defer func() {
		if err := db.Disconnect(context.Background()); err != nil {
			log.Println("mongodb disconnect:", err)
		}
	}()

	if err := db.EnsureEmailConfigIndex(ctx); err != nil {
		log.Fatal("email_config index:", err)
	}

	// If users collection is empty, create admin user from env (once); after that only MongoDB is used for login.
	if err := seedBootstrapUser(ctx, db, cfg.AuthEmail, cfg.AuthPass); err != nil {
		log.Fatal("bootstrap user:", err)
	}
	// Ensure at least one guest user exists for "View as guest" on login page.
	if err := seedGuestUser(ctx, db); err != nil {
		log.Fatal("seed guest user:", err)
	}

	var s3Service *service.S3Service
	if cfg.S3Bucket != "" {
		s3Service, err = service.NewS3Service(ctx, cfg.S3Bucket, cfg.S3Region, cfg.S3AccessKeyID, cfg.S3SecretKey)
		if err != nil {
			log.Fatal("s3:", err)
		}
	} else {
		log.Println("warning: AWS_S3_BUCKET not set; uploads will fail")
	}
	if len(cfg.EmailConfigEncryptionKey) != 32 {
		log.Println("warning: Kindle app-specific password will be stored in plaintext (set KINDLE_CONFIG_ENCRYPTION_KEY with: openssl rand -base64 32)")
	}

	authHandler := &handlers.AuthHandler{DB: db, JWTSecret: cfg.JWTSecret}
	uploadHandler := &handlers.UploadHandler{
		DB:       db,
		S3:       s3Service,
		MaxBytes: cfg.MaxUploadMB * 1024 * 1024,
	}
	booksHandler := &handlers.BooksHandler{DB: db, S3: s3Service, EncKey: cfg.EmailConfigEncryptionKey}
	usersHandler := &handlers.UsersHandler{DB: db}
	emailConfigHandler := &handlers.EmailConfigHandler{DB: db, EncKey: cfg.EmailConfigEncryptionKey}

	r := chi.NewRouter()
	r.Use(middleware.AllowAll())
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"welcome to books."}`))
	})
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

		r.Route("/api", func(r chi.Router) {
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/guest", authHandler.LoginAsGuest)
		r.Get("/books/{id}/cover", booksHandler.Cover) // public so <img src> works without auth
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret))
			r.Get("/me", usersHandler.GetMe)
			r.Patch("/me/preferences", usersHandler.PatchMePreferences)
			// Read: admin, editor, viewer, guest (guests see only books with viewByGuest)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAnyRole("admin", "editor", "viewer", "guest"))
				r.Get("/books", booksHandler.List)
				r.Get("/books/{id}", booksHandler.Get)
				r.Get("/books/{id}/download", booksHandler.Download)
				r.Post("/books/{id}/send-to-kindle", booksHandler.SendToKindle)
			})
			// Write (upload): admin, editor
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAnyRole("admin", "editor"))
				r.Post("/upload", uploadHandler.Upload)
			})
			// Refresh metadata: admin, editor
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAnyRole("admin", "editor"))
				r.Post("/books/{id}/refresh-metadata", booksHandler.RefreshMetadata)
			})
			// Delete books: admin only
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAdmin)
				r.Delete("/books/{id}", booksHandler.Delete)
			})
			// Toggle view-by-guest (demo visibility): admin only
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAdmin)
				r.Patch("/books/{id}/view-by-guest", booksHandler.PatchViewByGuest)
				r.Put("/books/{id}/view-by-guest", booksHandler.PatchViewByGuest)
			})
			// Manage users: admin only
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAdmin)
				r.Get("/users", usersHandler.ListUsers)
				r.Post("/users", usersHandler.CreateUser)
				r.Patch("/users/{id}", usersHandler.UpdateUser)
				r.Delete("/users/{id}", usersHandler.DeleteUser)
			})
			// Kindle config (per user): any authenticated user
			r.Get("/email-config", emailConfigHandler.Get)
			r.Put("/email-config", emailConfigHandler.Save)
			r.Patch("/email-config", emailConfigHandler.Save)
		})
	})

	server := &http.Server{Addr: ":" + cfg.Port, Handler: r}
	go func() {
		log.Println("server listening on :" + cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Println("shutdown:", err)
	}
}

func seedBootstrapUser(ctx context.Context, db *store.DB, email, password string) error {
	count, err := db.UsersCount(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user := &models.User{
		Email:     email,
		Password:  string(hash),
		Role:      models.RoleAdmin,
		CreatedAt: time.Now(),
	}
	_, err = db.CreateUser(ctx, user)
	if err != nil {
		return err
	}
	log.Println("created bootstrap admin user from env (users collection was empty)")
	return nil
}

const guestUserEmail = "guest@guest.local"

func seedGuestUser(ctx context.Context, db *store.DB) error {
	existing, err := db.UserByRole(ctx, models.RoleGuest)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("guest"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user := &models.User{
		Email:     guestUserEmail,
		Password:  string(hash),
		Role:      models.RoleGuest,
		CreatedAt: time.Now(),
	}
	_, err = db.CreateUser(ctx, user)
	if err != nil {
		return err
	}
	log.Println("created bootstrap guest user for View as guest")
	return nil
}
