package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Kevin-Aaaquil/books/backend/config"
	"github.com/Kevin-Aaaquil/books/backend/handlers"
	"github.com/Kevin-Aaaquil/books/backend/middleware"
	"github.com/Kevin-Aaaquil/books/backend/service"
	"github.com/Kevin-Aaaquil/books/backend/store"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

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

	var s3Service *service.S3Service
	if cfg.S3Bucket != "" {
		s3Service, err = service.NewS3Service(ctx, cfg.S3Bucket, cfg.S3Region, cfg.S3AccessKeyID, cfg.S3SecretKey)
		if err != nil {
			log.Fatal("s3:", err)
		}
	} else {
		log.Println("warning: AWS_S3_BUCKET not set; uploads will fail")
	}

	authHandler := &handlers.AuthHandler{
		DB:           db,
		JWTSecret:    cfg.JWTSecret,
		DefaultEmail: cfg.AuthEmail,
		DefaultPass:  cfg.AuthPass,
	}
	uploadHandler := &handlers.UploadHandler{
		DB:       db,
		S3:       s3Service,
		MaxBytes: cfg.MaxUploadMB * 1024 * 1024,
	}
	booksHandler := &handlers.BooksHandler{DB: db, S3: s3Service}

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
		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret))
			r.Post("/upload", uploadHandler.Upload)
			r.Get("/books", booksHandler.List)
			r.Get("/books/{id}", booksHandler.Get)
			r.Get("/books/{id}/download", booksHandler.Download)
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
