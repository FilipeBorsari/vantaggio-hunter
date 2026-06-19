package main

import (
	"context"
	"encoding/base64"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/vantaggio/prospect-api/internal/admin"
	"github.com/vantaggio/prospect-api/internal/analytics"
	authpkg "github.com/vantaggio/prospect-api/internal/auth"
	"github.com/vantaggio/prospect-api/internal/companies"
	"github.com/vantaggio/prospect-api/internal/credits"
	"github.com/vantaggio/prospect-api/internal/exports"
	"github.com/vantaggio/prospect-api/internal/ia"
	"github.com/vantaggio/prospect-api/internal/invitations"
	"github.com/vantaggio/prospect-api/internal/orgadmin"
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

	creditsRepo := credits.NewPostgresRepository(pool)
	creditsSvc := credits.NewService(creditsRepo)
	creditsHandler := credits.NewHandler(creditsSvc)

	adminRepo := admin.NewPostgresRepository(pool)
	adminSvc := admin.NewService(adminRepo, creditsSvc)
	adminHandler := admin.NewHandler(adminSvc)

	companiesRepo := companies.NewPostgresRepository(pool)
	companiesSvc := companies.NewService(companiesRepo)
	companiesHandler := companies.NewHandler(companiesSvc, creditsSvc)

	searchesRepo := searches.NewPostgresRepository(pool)
	searchesSvc := searches.NewService(searchesRepo)
	searchesHandler := searches.NewHandler(searchesSvc, redisClient)

	analyticsRepo := analytics.NewPostgresRepository(pool)
	analyticsSvc := analytics.NewService(analyticsRepo)
	analyticsHandler := analytics.NewHandler(analyticsSvc)

	encKey := loadEncryptionKey()
	exportsRepo := exports.NewPostgresRepository(pool)
	exportsSvc := exports.NewService(exportsRepo, creditsSvc, encKey)
	exportsHandler := exports.NewHandler(exportsSvc, redisClient)

	iaProvider := ia.NewProvider(ia.Config{
		Provider:  os.Getenv("AI_PROVIDER"),
		ChatModel: os.Getenv("AI_CHAT_MODEL"),
	})
	iaRepo := ia.NewPostgresRepository(pool)
	iaSvc := ia.NewService(iaRepo, creditsSvc, iaProvider)
	iaHandler := ia.NewHandler(iaSvc)

	orgadminRepo := orgadmin.NewPostgresRepository(pool)
	orgadminSvc := orgadmin.NewService(orgadminRepo, creditsSvc)
	orgadminHandler := orgadmin.NewHandler(orgadminSvc)

	invitationsRepo := invitations.NewPostgresRepository(pool)
	invitationsSvc := invitations.NewService(invitationsRepo, authSvc)
	invitationsHandler := invitations.NewHandler(invitationsSvc)

	// Start ETL job
	go analytics.StartETLJob(ctx, analyticsSvc)

	// Start search workers
	workerCount := workerConcurrency()
	worker := searches.NewWorker(searchesRepo, redisClient, creditsSvc)
	for i := 0; i < workerCount; i++ {
		go worker.Run(ctx)
	}
	slog.Info("search workers started", "count", workerCount)

	// Start export worker
	exportWorker := exports.NewWorker(exportsRepo, redisClient, creditsSvc, companiesRepo, encKey)
	go exportWorker.Run(ctx)
	slog.Info("export worker started")

	r := chi.NewRouter()
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(apimiddleware.CORS)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Post("/auth/login", authHandler.Login)
	r.Post("/auth/refresh", authHandler.Refresh)

	// Public: invitation accept flow
	r.Get("/invitations/{token}", invitationsHandler.ValidateToken)
	r.Post("/invitations/{token}/accept", invitationsHandler.Accept)

	r.Group(func(r chi.Router) {
		r.Use(authpkg.Authenticate)

		r.Post("/auth/logout", authHandler.Logout)

		r.Get("/companies", companiesHandler.List)
		r.Get("/companies/{cnpj}", companiesHandler.GetByCNPJ)

		r.Get("/cnaes", searchesHandler.SearchCNAEs)

		r.Post("/searches", searchesHandler.Create)
		r.Post("/searches/estimate", searchesHandler.Estimate)
		r.Get("/searches", searchesHandler.List)
		r.Get("/searches/{id}", searchesHandler.GetByID)

		r.Get("/credits/balance", creditsHandler.GetBalance)
		r.Get("/credits/transactions", creditsHandler.ListTransactions)

		r.Get("/analytics/kpis", analyticsHandler.GetKPIs)
		r.Get("/analytics/daily-consumption", analyticsHandler.GetDailyConsumption)
		r.Get("/analytics/top-cnaes", analyticsHandler.GetTopCNAEs)
		r.Get("/analytics/funnel", analyticsHandler.GetFunnel)

		r.Post("/ia/qualify/{cnpj}", iaHandler.Qualify)
		r.Get("/ia/qualifications", iaHandler.ListQualifications)

		r.Get("/crm/integrations", exportsHandler.GetIntegration)
		r.Post("/crm/integrations", exportsHandler.CreateIntegration)
		r.Post("/exports", exportsHandler.CreateExport)
		r.Get("/exports", exportsHandler.ListExports)
		r.Get("/exports/{id}", exportsHandler.GetExport)

		// Seller: own profile and searches
		r.Get("/me/profile", orgadminHandler.GetProfile)
		r.Patch("/me/profile", orgadminHandler.UpdateProfile)
		r.Get("/me/searches", orgadminHandler.ListSellerSearches)

		// Org admin routes
		r.Group(func(r chi.Router) {
			r.Use(authpkg.RequireOrgAdmin())
			r.Get("/org/users", orgadminHandler.ListUsers)
			r.Patch("/org/users/{userId}", orgadminHandler.PatchUser)
			r.Delete("/org/users/{userId}", orgadminHandler.DeleteUser)
			r.Get("/org/users/{userId}/history", orgadminHandler.GetUserHistory)
			r.Post("/org/invitations", orgadminHandler.CreateInvitation)
			r.Get("/org/invitations", orgadminHandler.ListInvitations)
			r.Delete("/org/invitations/{invitationId}", orgadminHandler.DeleteInvitation)
			r.Get("/org/costs", orgadminHandler.GetOrgCosts)
			r.Get("/org/credits", orgadminHandler.GetOrgCredits)
		})

		// Super admin routes
		r.Group(func(r chi.Router) {
			r.Use(authpkg.RequireSuperAdmin())
			r.Get("/admin/plans", adminHandler.ListPlans)
			r.Get("/admin/organizations", adminHandler.ListOrgs)
			r.Post("/admin/organizations", adminHandler.CreateOrg)
			r.Get("/admin/organizations/{orgId}", adminHandler.GetOrgDetail)
			r.Patch("/admin/organizations/{orgId}", adminHandler.PatchOrg)
			r.Post("/admin/organizations/{orgId}/users", adminHandler.CreateUser)
			r.Post("/admin/organizations/{orgId}/credits", adminHandler.AddOrgCredits)
			r.Post("/admin/organizations/{orgId}/impersonate", adminHandler.Impersonate)
			r.Get("/admin/dashboard", adminHandler.GetDashboard)
			// Legacy compat routes
			r.Post("/admin/organizations/{id}/users", adminHandler.CreateUser)
			r.Patch("/admin/organizations/{id}/users/{userId}", adminHandler.SetUserActive)
			r.Post("/admin/credits/add", creditsHandler.AdminAddCredits)
			r.Post("/internal/etl/run", analyticsHandler.TriggerETL)
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

func loadEncryptionKey() []byte {
	raw := os.Getenv("ENCRYPTION_KEY")
	if raw == "" {
		log.Fatal("ENCRYPTION_KEY env var is required")
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		log.Fatalf("decode ENCRYPTION_KEY: %v", err)
	}
	if len(key) != 32 {
		log.Fatalf("ENCRYPTION_KEY must decode to exactly 32 bytes, got %d", len(key))
	}
	return key
}
