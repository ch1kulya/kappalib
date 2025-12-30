package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ch1kulya/kappalib/assets/templates"
	"github.com/ch1kulya/kappalib/internal/api"
	"github.com/ch1kulya/kappalib/internal/data"
	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/kappalib/internal/web"

	"github.com/ch1kulya/logger"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	esbuild "github.com/evanw/esbuild/pkg/api"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/joho/godotenv/autoload"
)

//go:embed docs.html
var docsHTML string

func runMigrations() {
	logger.Info("Starting database migrations...")
	databaseURL := os.Getenv("DATABASE_URL")

	m, err := migrate.New("file://migrations", databaseURL)
	if err != nil {
		logger.Error("Migration initialization failed: %v", err)
		os.Exit(1)
	}

	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			logger.Warn("Migration source close error: %v", srcErr)
		}
		if dbErr != nil {
			logger.Warn("Migration db close error: %v", dbErr)
		}
	}()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Info("Already up to date.")
			return
		}
		logger.Error("Migration failed: %v", err)
		os.Exit(1)
	}

	logger.Info("Migrations applied successfully")
}

func buildAssets() {
	logger.Info("Building assets...")

	apiUrl := "/api"

	result := esbuild.Build(esbuild.BuildOptions{
		EntryPoints: []string{
			"./assets/src/app.ts",
			"./assets/src/styles/main.css",
		},
		Outdir:            "./assets/static/dist",
		Bundle:            true,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		Sourcemap:         esbuild.SourceMapLinked,
		Write:             true,
		Platform:          esbuild.PlatformBrowser,
		Target:            esbuild.ES2020,
		Format:            esbuild.FormatESModule,
		TreeShaking:       esbuild.TreeShakingTrue,
		Define: map[string]string{
			"process.env.API_URL":                     fmt.Sprintf("\"%s\"", apiUrl),
			"process.env.TURNSTILE_SITE_KEY":          fmt.Sprintf("\"%s\"", os.Getenv("TURNSTILE_SITE_KEY")),
			"process.env.TURNSTILE_COMMENTS_SITE_KEY": fmt.Sprintf("\"%s\"", os.Getenv("TURNSTILE_COMMENTS_SITE_KEY")),
			"process.env.S3_ENDPOINT":                 fmt.Sprintf("\"%s\"", os.Getenv("S3_ENDPOINT")),
			"process.env.S3_BUCKET":                   fmt.Sprintf("\"%s\"", os.Getenv("S3_BUCKET")),
			"process.env.S3_USE_SSL":                  fmt.Sprintf("\"%s\"", os.Getenv("S3_USE_SSL")),
		},
	})

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			logger.Error("Build error: %s", e.Text)
		}
		return
	}

	if len(result.Warnings) > 0 {
		for _, warn := range result.Warnings {
			logger.Warn("Build warning: %s", warn.Text)
		}
	}

	logger.Info("Assets built successfully")
}

func main() {
	logger.Info("Initializing application...")

	if err := templates.Init(); err != nil {
		logger.Error("Failed to initialize templates: %v", err)
		os.Exit(1)
	}
	logger.Info("Templates initialized")

	runMigrations()

	if err := database.Init(); err != nil {
		logger.Error("Database initialization failed: %v", err)
		os.Exit(1)
	}
	defer database.Close()

	buildAssets()

	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(logger.Middleware)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(middleware.Timeout(60 * time.Second))

	webRateLimiter := web.NewRateLimiter()
	r.Use(web.WwwRedirect)
	r.Use(web.SecurityHeadersMiddleware)
	r.Use(web.RateLimitMiddleware(webRateLimiter))

	h := web.NewHandler()
	r.NotFound(h.NotFound)

	fileServer := http.FileServer(http.Dir("./assets/static"))
	r.Handle("/assets/*", http.StripPrefix("/assets", web.StaticCacheMiddleware(fileServer)))

	r.Get("/robots.txt", h.RobotsTxt)
	r.Get("/sitemap.xml", h.Sitemap)
	r.Get("/", h.Home)
	r.Get("/dmca", h.StaticPage("dmca", "DMCA"))
	r.Get("/privacy", h.StaticPage("privacy", "Политика конфиденциальности"))
	r.Get("/copyright", h.StaticPage("copyright", "Правообладателям"))
	r.Get("/license", h.StaticPage("license", "Лицензия MIT"))
	r.Get("/{id}", h.Novel)
	r.Get("/{id}/chapter/{chapterId}", h.Chapter)
	r.Get("/status", h.GetStatus)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	apiRateLimiter := api.NewRateLimiter()
	r.Route("/api", func(r chi.Router) {
		r.Use(api.CorsMiddleware)
		r.Use(api.RateLimitMiddleware(apiRateLimiter))
		r.Use(api.CacheMiddleware)

		config := huma.DefaultConfig("kappalib", "stable")
		config.Info.Description = "Public API for accessing kappalib services."
		config.DocsPath = ""
		config.Servers = []*huma.Server{{URL: "/api"}}

		humaApi := humachi.New(r, config)

		r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(docsHTML))
		})

		huma.Register(humaApi, huma.Operation{
			OperationID: "get-status",
			Method:      http.MethodGet,
			Path:        "/",
			Summary:     "API Status",
		}, api.HandleStatus)

		huma.Register(humaApi, huma.Operation{
			OperationID: "get-novels",
			Method:      http.MethodGet,
			Path:        "/novels",
			Summary:     "List novels",
		}, api.HandleGetNovels)

		huma.Register(humaApi, huma.Operation{
			OperationID: "get-sitemap",
			Method:      http.MethodGet,
			Path:        "/novels/sitemap-data",
			Summary:     "Get sitemap data",
		}, api.HandleGetSitemapData)

		huma.Register(humaApi, huma.Operation{
			OperationID: "search-novels",
			Method:      http.MethodGet,
			Path:        "/novels/search",
			Summary:     "Search novels",
		}, api.HandleSearchNovels)

		huma.Register(humaApi, huma.Operation{
			OperationID: "get-novel",
			Method:      http.MethodGet,
			Path:        "/novels/{id}",
			Summary:     "Get novel by ID",
		}, api.HandleGetNovel)

		huma.Register(humaApi, huma.Operation{
			OperationID: "get-chapters",
			Method:      http.MethodGet,
			Path:        "/novels/{id}/chapters",
			Summary:     "List chapters for novel",
		}, api.HandleGetChaptersList)

		huma.Register(humaApi, huma.Operation{
			OperationID: "get-chapter",
			Method:      http.MethodGet,
			Path:        "/chapters/{id}",
			Summary:     "Get chapter by ID",
		}, api.HandleGetChapter)

		huma.Register(humaApi, huma.Operation{
			OperationID: "create-profile",
			Method:      http.MethodPost,
			Path:        "/profile",
			Summary:     "Create user profile",
		}, api.HandleCreateProfile)

		huma.Register(humaApi, huma.Operation{
			OperationID: "get-profile",
			Method:      http.MethodGet,
			Path:        "/profile/{id}",
			Summary:     "Get user profile",
		}, api.HandleGetProfile)

		huma.Register(humaApi, huma.Operation{
			OperationID: "delete-profile",
			Method:      http.MethodDelete,
			Path:        "/profile/{id}",
			Summary:     "Delete user profile",
		}, api.HandleDeleteProfile)

		huma.Register(humaApi, huma.Operation{
			OperationID: "generate-sync-code",
			Method:      http.MethodPost,
			Path:        "/profile/{id}/sync-code",
			Summary:     "Generate sync code",
		}, api.HandleGenerateSyncCode)

		huma.Register(humaApi, huma.Operation{
			OperationID: "login-profile",
			Method:      http.MethodPost,
			Path:        "/profile/login",
			Summary:     "Login with sync code",
		}, api.HandleLogin)

		huma.Register(humaApi, huma.Operation{
			OperationID: "sync-cookies",
			Method:      http.MethodPost,
			Path:        "/profile/sync-cookies",
			Summary:     "Sync cookies",
		}, api.HandleSyncCookies)
		huma.Register(humaApi, huma.Operation{
			OperationID: "get-comments",
			Method:      http.MethodGet,
			Path:        "/chapters/{chapterId}/comments",
			Summary:     "Get chapter comments",
		}, api.HandleGetComments)

		huma.Register(humaApi, huma.Operation{
			OperationID: "create-comment",
			Method:      http.MethodPost,
			Path:        "/chapters/{chapterId}/comments",
			Summary:     "Create comment",
		}, api.HandleCreateComment)

		huma.Register(humaApi, huma.Operation{
			OperationID: "telegram-webhook",
			Method:      http.MethodPost,
			Path:        "/webhook/telegram",
			Summary:     "Telegram webhook",
		}, api.HandleTelegramWebhook)

		huma.Register(humaApi, huma.Operation{
			OperationID: "update-display-name",
			Method:      http.MethodPatch,
			Path:        "/profile/{id}/name",
			Summary:     "Update display name",
		}, api.HandleUpdateDisplayName)

		huma.Register(humaApi, huma.Operation{
			OperationID: "upload-avatar",
			Method:      http.MethodPost,
			Path:        "/profile/{id}/avatar",
			Summary:     "Upload avatar",
		}, api.HandleUploadAvatar)
	})

	go func() {
		logger.Info("Warming up sitemap cache...")
		if _, err := data.GetSitemapData(context.Background()); err != nil {
			logger.Warn("Failed to warm up sitemap: %v", err)
		} else {
			logger.Info("Sitemap cache warmed up.")
		}
	}()

	var port string = "8080"

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("Server listening on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to listen: %s", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited properly")
}
