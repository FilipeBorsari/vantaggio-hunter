package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/vantaggio/prospect-api/internal/admin"
	authpkg "github.com/vantaggio/prospect-api/internal/auth"
	"github.com/vantaggio/prospect-api/internal/companies"
	"github.com/vantaggio/prospect-api/internal/credits"
	"github.com/vantaggio/prospect-api/internal/searches"
	"github.com/vantaggio/prospect-api/pkg/db"
	"github.com/vantaggio/prospect-api/pkg/httputil"
	apimiddleware "github.com/vantaggio/prospect-api/pkg/middleware"
	redispkg "github.com/vantaggio/prospect-api/pkg/redis"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.NewPool(ctx)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	redisClient, err := redispkg.NewClient()
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer redisClient.Close()

	authRepo := authpkg.NewPostgresRepository(pool)
	authSvc := authpkg.NewService(authRepo)
	authHandler := authpkg.NewHandler(authSvc)

	adminRepo := admin.NewPostgresRepository(pool)
	adminSvc := admin.NewService(adminRepo)
	adminHandler := admin.NewHandler(adminSvc)

	creditsRepo := credits.NewPostgresRepository(pool)
	creditsSvc := credits.NewService(creditsRepo)
	creditsHandler := credits.NewHandler(creditsSvc)

	companiesRepo := companies.NewPostgresRepository(pool)
	companiesSvc := companies.NewService(companiesRepo)
	companiesHandler := companies.NewHandler(companiesSvc, creditsSvc)

	searchesRepo := searches.NewPostgresRepository(pool)
	searchesSvc := searches.NewService(searchesRepo)
	searchesHandler := searches.NewHandler(searchesSvc, redisClient)

	// Start search workers
	workerCount := workerConcurrency()
	worker := searches.NewWorker(searchesRepo, redisClient, creditsSvc)
	for i := 0; i < workerCount; i++ {
		go worker.Run(ctx)
	}
	slog.Info("search workers started", "count", workerCount)

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

		r.Get("/cnaes", searchesHandler.SearchCNAEs)

		r.Post("/searches", searchesHandler.Create)
		r.Get("/searches", searchesHandler.List)
		r.Get("/searches/{id}", searchesHandler.GetByID)

		r.Get("/credits/balance", creditsHandler.GetBalance)
		r.Get("/credits/transactions", creditsHandler.ListTransactions)

		r.Group(func(r chi.Router) {
			r.Use(authpkg.RequireRole("admin"))
			r.Get("/admin/plans", adminHandler.ListPlans)
			r.Get("/admin/organizations", adminHandler.ListOrgs)
			r.Post("/admin/organizations", adminHandler.CreateOrg)
			r.Post("/admin/organizations/{id}/users", adminHandler.CreateUser)
			r.Patch("/admin/organizations/{id}/users/{userId}", adminHandler.SetUserActive)
			r.Post("/admin/credits/add", creditsHandler.AdminAddCredits)
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

func workerConcurrency() int {
	n, err := strconv.Atoi(os.Getenv("SEARCH_WORKERS"))
	if err != nil || n < 1 {
		return 2
	}
	return n
}
