package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/vantaggio/prospect-api/internal/admin"
	authpkg "github.com/vantaggio/prospect-api/internal/auth"
	"github.com/vantaggio/prospect-api/internal/companies"
	"github.com/vantaggio/prospect-api/pkg/db"
	"github.com/vantaggio/prospect-api/pkg/httputil"
	apimiddleware "github.com/vantaggio/prospect-api/pkg/middleware"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	ctx := context.Background()

	pool, err := db.NewPool(ctx)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	authRepo := authpkg.NewPostgresRepository(pool)
	authSvc := authpkg.NewService(authRepo)
	authHandler := authpkg.NewHandler(authSvc)

	adminRepo := admin.NewPostgresRepository(pool)
	adminSvc := admin.NewService(adminRepo)
	adminHandler := admin.NewHandler(adminSvc)

	companiesRepo := companies.NewPostgresRepository(pool)
	companiesSvc := companies.NewService(companiesRepo)
	companiesHandler := companies.NewHandler(companiesSvc)

	r := chi.NewRouter()
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(apimiddleware.CORS)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Post("/auth/login", authHandler.Login)
	r.Post("/auth/refresh", authHandler.Refresh)

	r.Group(func(r chi.Router) {
		r.Use(authpkg.Authenticate)

		r.Post("/auth/logout", authHandler.Logout)

		r.Get("/companies", companiesHandler.List)
		r.Get("/companies/{cnpj}", companiesHandler.GetByCNPJ)

		r.Group(func(r chi.Router) {
			r.Use(authpkg.RequireRole("admin"))
			r.Get("/admin/plans", adminHandler.ListPlans)
			r.Get("/admin/organizations", adminHandler.ListOrgs)
			r.Post("/admin/organizations", adminHandler.CreateOrg)
			r.Post("/admin/organizations/{id}/users", adminHandler.CreateUser)
			r.Patch("/admin/organizations/{id}/users/{userId}", adminHandler.SetUserActive)
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	slog.Info("server starting", "port", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
