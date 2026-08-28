package main

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ch1kulya/kappalib/internal/api"
	"github.com/ch1kulya/kappalib/internal/auth"
	"github.com/ch1kulya/kappalib/internal/data"
	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/kappalib/internal/templates"
	"github.com/ch1kulya/kappalib/internal/web"

	"github.com/ch1kulya/logger"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	esbuild "github.com/evanw/esbuild/pkg/api"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/joho/godotenv/autoload"
)

//go:embed docs.html
var docsHTML string

var requiredEnvVars = []string{
	"DATABASE_URL",
	"PHARE_TOKEN",
	"PHARE_ID",
	"TURNSTILE_COMMENTS_SITE_KEY",
	"TURNSTILE_COMMENTS_SECRET",
	"SMARTCAPTCHA_SITE_KEY",
	"SMARTCAPTCHA_SECRET",
	"TELEGRAM_BOT_TOKEN",
	"TELEGRAM_CHAT_ID",
	"TELEGRAM_WEBHOOK_SECRET",
	"GITHUB_TOKEN",
	"S3_ENDPOINT",
	"S3_BUCKET",
	"S3_ACCESS_KEY",
	"S3_SECRET_KEY",
	"S3_USE_SSL",
	"S3_PUBLIC_URL",
	"INDEX_NOW_KEY",
	"AUTH_SECRET",
	"AUTH_URL",
	"AUTH_SECURE",
	"ALLOWED_ORIGIN",
}

func buildAssets() error {
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
			"process.env.TURNSTILE_COMMENTS_SITE_KEY": fmt.Sprintf("\"%s\"", os.Getenv("TURNSTILE_COMMENTS_SITE_KEY")),
			"process.env.SMARTCAPTCHA_SITE_KEY":       fmt.Sprintf("\"%s\"", os.Getenv("SMARTCAPTCHA_SITE_KEY")),
			"process.env.S3_PUBLIC_URL":               fmt.Sprintf("\"%s\"", os.Getenv("S3_PUBLIC_URL")),
		},
		External: []string{"/assets/fonts/*"},
		Engines: []esbuild.Engine{
			{Name: esbuild.EngineChrome, Version: "100"},
		},
	})

	if len(result.Errors) > 0 {
		var errMsgs []string
		for _, e := range result.Errors {
			errMsgs = append(errMsgs, e.Text)
		}
		return fmt.Errorf("build errors: %s", strings.Join(errMsgs, "; "))
	}

	for _, warn := range result.Warnings {
		logger.Warn("Build warning: %s", warn.Text)
	}

	logger.Info("Assets built successfully")
	return nil
}

func validateEnv() error {
	var missingVars []string
	for _, key := range requiredEnvVars {
		value := os.Getenv(key)
		if value == "" {
			missingVars = append(missingVars, key)
		}
	}
	if len(missingVars) > 0 {
		return fmt.Errorf("missing required env variables: %s", strings.Join(missingVars, ", "))
	}
	return nil
}

func init() {
	if os.Getenv("SHOW_USER_AGENT") == "true" {
		logger.SetShowUserAgent(true)
	}
	logger.Info("Initializing application...")

	if err := validateEnv(); err != nil {
		logger.Fatal("Configuration error: %v", err)
	}

	if err := templates.Init(); err != nil {
		logger.Fatal("Failed to initialize templates: %v", err)
	}
	logger.Info("Templates initialized")

	if err := database.RunMigrations(os.Getenv("DATABASE_URL"), "file://migrations"); err != nil {
		logger.Fatal("Database migrations failed: %v", err)
	}

	if err := database.Init(); err != nil {
		logger.Fatal("Database initialization failed: %v", err)
	}

	if err := buildAssets(); err != nil {
		logger.Fatal("Assets build failed: %v", err)
	}
}

func main() {
	defer database.Close()

	authURL := os.Getenv("AUTH_URL")
	userStore := auth.NewUserStore()
	authService := auth.NewService(auth.Config{
		Secret: os.Getenv("AUTH_SECRET"),
		URL:    authURL,
		Secure: os.Getenv("AUTH_SECURE") == "true",
	}, userStore)

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
	r.Get(fmt.Sprintf("/%s.txt", os.Getenv("INDEX_NOW_KEY")), h.IndexNowKey)
	r.Get("/", h.Home)
	r.Get("/catalog", h.Catalog)
	r.Get("/history", h.History)
	r.Get("/comments", h.MyComments)
	r.Get("/bookmarks", h.Bookmarks)
	r.Get("/updates", h.Updates)
	r.Get("/dmca", h.StaticPage("dmca", "DMCA"))
	r.Get("/privacy", h.StaticPage("privacy", "Политика конфиденциальности"))
	r.Get("/copyright", h.StaticPage("copyright", "Правообладателям"))
	r.Get("/license", h.StaticPage("license", "Лицензия MIT"))
	r.Get("/terms", h.StaticPage("terms", "Пользовательское соглашение"))
	r.Get("/markdown", h.StaticPage("markdown", "Шпаргалка по форматированию"))
	r.Get("/{id}", h.Novel)
	r.Get("/{id}/chapter/{chapterId}", h.Chapter)
	r.Get("/status", h.GetStatus)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r.Route("/auth", func(r chi.Router) {
		r.Use(middleware.RealIP)
		r.Use(web.RateLimitMiddleware(webRateLimiter))
		r.Use(web.SecurityHeadersMiddleware)
		r.Mount("/", authService.AuthRoutes)
	})

	apiRateLimiter := api.NewRateLimiter()
	r.Route("/api", func(r chi.Router) {
		r.Use(api.CorsMiddleware)
		r.Use(api.RateLimitMiddleware(apiRateLimiter))
		r.Use(api.CacheMiddleware)
		r.Use(authService.Middleware())
		r.Use(auth.BridgeMiddleware)

		config := huma.DefaultConfig("kappalib", "stable")
		config.Info.Description = "Public API for accessing kappalib services."
		config.DocsPath = ""
		config.Servers = []*huma.Server{{URL: "/api"}}
		config.OpenAPI.OpenAPI = "3.0.3"

		humaApi := humachi.New(r, config)

		humaApi.OpenAPI().Components.SecuritySchemes = map[string]*huma.SecurityScheme{
			"sessionCookie": {
				Type: "apiKey",
				In:   "cookie",
				Name: "kpl_session",
			},
		}

		r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(docsHTML))
		})

		huma.Register(humaApi, huma.Operation{
			OperationID: "get-status",
			Method:      http.MethodGet,
			Path:        "/",
			Summary:     "API Status",
			Tags:        []string{"System"},
		}, api.HandleStatus)

		huma.Register(humaApi, huma.Operation{
			OperationID: "get-novels",
			Method:      http.MethodGet,
			Path:        "/novels",
			Summary:     "List novels",
			Tags:        []string{"Novels"},
		}, api.HandleGetNovels)

		huma.Register(humaApi, huma.Operation{
			OperationID: "get-sitemap",
			Method:      http.MethodGet,
			Path:        "/novels/sitemap-data",
			Summary:     "Get sitemap data",
			Tags:        []string{"Novels"},
		}, api.HandleGetSitemapData)

		huma.Register(humaApi, huma.Operation{
			OperationID: "search-novels",
			Method:      http.MethodGet,
			Path:        "/novels/search",
			Summary:     "Search novels",
			Tags:        []string{"Novels"},
		}, api.HandleSearchNovels)

		huma.Register(humaApi, huma.Operation{
			OperationID: "get-novel",
			Method:      http.MethodGet,
			Path:        "/novels/{id}",
			Summary:     "Get novel by ID",
			Tags:        []string{"Novels"},
		}, api.HandleGetNovel)

		huma.Register(humaApi, huma.Operation{
			OperationID: "get-novels-batch",
			Method:      http.MethodPost,
			Path:        "/novels/batch",
			Summary:     "Get multiple novels by IDs",
			Tags:        []string{"Novels"},
		}, api.HandleGetNovelsBatch)

		huma.Register(humaApi, huma.Operation{
			OperationID: "get-chapters",
			Method:      http.MethodGet,
			Path:        "/novels/{id}/chapters",
			Summary:     "List chapters for novel",
			Tags:        []string{"Novels"},
		}, api.HandleGetChaptersList)

		huma.Register(humaApi, huma.Operation{
			OperationID: "get-chapter",
			Method:      http.MethodGet,
			Path:        "/chapters/{id}",
			Summary:     "Get chapter by ID",
			Tags:        []string{"Chapters"},
		}, api.HandleGetChapter)

		huma.Register(humaApi, huma.Operation{
			OperationID: "get-profile",
			Method:      http.MethodGet,
			Path:        "/profile/{id}",
			Summary:     "Get user profile",
			Tags:        []string{"Profile"},
		}, api.HandleGetProfile)

		huma.Register(humaApi, huma.Operation{
			OperationID: "get-comments",
			Method:      http.MethodGet,
			Path:        "/chapters/{chapterId}/comments",
			Summary:     "Get chapter comments",
			Tags:        []string{"Chapters"},
		}, api.HandleGetComments)

		huma.Register(humaApi, huma.Operation{
			OperationID: "telegram-webhook",
			Method:      http.MethodPost,
			Path:        "/webhook/telegram",
			Summary:     "Telegram webhook",
			Tags:        []string{"Webhook"},
		}, api.HandleTelegramWebhook)

		huma.Register(humaApi, huma.Operation{
			OperationID: "get-current-user",
			Method:      http.MethodGet,
			Path:        "/profile/me",
			Summary:     "Get current authenticated user",
			Security:    []map[string][]string{{"sessionCookie": {}}},
			Tags:        []string{"Profile"},
		}, api.HandleGetCurrentUser)

		huma.Register(humaApi, huma.Operation{
			OperationID: "delete-profile",
			Method:      http.MethodDelete,
			Path:        "/profile/{id}",
			Summary:     "Delete user profile",
			Security:    []map[string][]string{{"sessionCookie": {}}},
			Tags:        []string{"Profile"},
		}, api.HandleDeleteProfile)

		huma.Register(humaApi, huma.Operation{
			OperationID: "logout",
			Method:      http.MethodPost,
			Path:        "/profile/logout",
			Summary:     "Logout and clear session cookie",
			Security:    []map[string][]string{{"sessionCookie": {}}},
			Tags:        []string{"Profile"},
		}, api.HandleLogout)

		huma.Register(humaApi, huma.Operation{
			OperationID: "sync-cookies",
			Method:      http.MethodPost,
			Path:        "/profile/sync-cookies",
			Summary:     "Sync cookies",
			Security:    []map[string][]string{{"sessionCookie": {}}},
			Tags:        []string{"Profile"},
		}, api.HandleSyncCookies)

		huma.Register(humaApi, huma.Operation{
			OperationID: "create-comment",
			Method:      http.MethodPost,
			Path:        "/chapters/{chapterId}/comments",
			Summary:     "Create comment",
			Security:    []map[string][]string{{"sessionCookie": {}}},
			Tags:        []string{"Chapters"},
		}, api.HandleCreateComment)

		huma.Register(humaApi, huma.Operation{
			OperationID: "update-display-name",
			Method:      http.MethodPatch,
			Path:        "/profile/{id}/name",
			Summary:     "Update display name",
			Security:    []map[string][]string{{"sessionCookie": {}}},
			Tags:        []string{"Profile"},
		}, api.HandleUpdateDisplayName)

		huma.Register(humaApi, huma.Operation{
			OperationID:  "upload-avatar",
			Method:       http.MethodPost,
			Path:         "/profile/{id}/avatar",
			Summary:      "Upload avatar",
			Security:     []map[string][]string{{"sessionCookie": {}}},
			MaxBodyBytes: 2 << 20,
			Tags:         []string{"Profile"},
		}, api.HandleUploadAvatar)

		huma.Register(humaApi, huma.Operation{
			OperationID:  "upload-comment-image",
			Method:       http.MethodPost,
			Path:         "/comments/image",
			Summary:      "Upload comment image",
			Security:     []map[string][]string{{"sessionCookie": {}}},
			MaxBodyBytes: 8 << 20,
			Tags:         []string{"Comments"},
		}, api.HandleUploadCommentImage)

		huma.Register(humaApi, huma.Operation{
			OperationID: "vote-comment",
			Method:      http.MethodPost,
			Path:        "/comments/{commentId}/vote",
			Summary:     "Vote on comment",
			Security:    []map[string][]string{{"sessionCookie": {}}},
			Tags:        []string{"Comments"},
		}, api.HandleVoteComment)

		huma.Register(humaApi, huma.Operation{
			OperationID: "get-user-bookmarks",
			Method:      http.MethodGet,
			Path:        "/profile/me/bookmarks",
			Summary:     "Get current user's bookmarks",
			Security:    []map[string][]string{{"sessionCookie": {}}},
			Tags:        []string{"Bookmarks"},
		}, api.HandleGetUserBookmarks)

		huma.Register(humaApi, huma.Operation{
			OperationID: "add-bookmark",
			Method:      http.MethodPost,
			Path:        "/bookmarks",
			Summary:     "Add a bookmark",
			Security:    []map[string][]string{{"sessionCookie": {}}},
			Tags:        []string{"Bookmarks"},
		}, api.HandleAddBookmark)

		huma.Register(humaApi, huma.Operation{
			OperationID: "delete-bookmark",
			Method:      http.MethodDelete,
			Path:        "/bookmarks/{id}",
			Summary:     "Delete a bookmark",
			Security:    []map[string][]string{{"sessionCookie": {}}},
			Tags:        []string{"Bookmarks"},
		}, api.HandleDeleteBookmark)

		huma.Register(humaApi, huma.Operation{
			OperationID: "update-bookmark",
			Method:      http.MethodPatch,
			Path:        "/bookmarks/{id}",
			Summary:     "Update a bookmark",
			Security:    []map[string][]string{{"sessionCookie": {}}},
			Tags:        []string{"Bookmarks"},
		}, api.HandleUpdateBookmark)

		huma.Register(humaApi, huma.Operation{
			OperationID: "delete-bookmark-category",
			Method:      http.MethodDelete,
			Path:        "/bookmarks/category/{name}",
			Summary:     "Delete a bookmark category",
			Security:    []map[string][]string{{"sessionCookie": {}}},
			Tags:        []string{"Bookmarks"},
		}, api.HandleDeleteBookmarkCategory)

		huma.Register(humaApi, huma.Operation{
			OperationID: "rename-bookmark-category",
			Method:      http.MethodPatch,
			Path:        "/bookmarks/category/{name}",
			Summary:     "Rename a bookmark category",
			Security:    []map[string][]string{{"sessionCookie": {}}},
			Tags:        []string{"Bookmarks"},
		}, api.HandleRenameBookmarkCategory)

		huma.Register(humaApi, huma.Operation{
			OperationID: "get-user-comments",
			Method:      http.MethodGet,
			Path:        "/profile/me/comments",
			Summary:     "Get current user's comments",
			Security:    []map[string][]string{{"sessionCookie": {}}},
			Tags:        []string{"Profile"},
		}, api.HandleGetUserComments)

		huma.Register(humaApi, huma.Operation{
			OperationID: "delete-comment",
			Method:      http.MethodDelete,
			Path:        "/comments/{commentId}",
			Summary:     "Delete own comment",
			Security:    []map[string][]string{{"sessionCookie": {}}},
			Tags:        []string{"Comments"},
		}, api.HandleDeleteComment)

		huma.Register(humaApi, huma.Operation{
			OperationID: "edit-comment",
			Method:      http.MethodPatch,
			Path:        "/comments/{commentId}",
			Summary:     "Edit own comment",
			Security:    []map[string][]string{{"sessionCookie": {}}},
			Tags:        []string{"Comments"},
		}, api.HandleEditComment)

		huma.Register(humaApi, huma.Operation{
			OperationID: "create-comment-answer",
			Method:      http.MethodPost,
			Path:        "/comments/{commentId}/answers",
			Summary:     "Create answer to comment",
			Security:    []map[string][]string{{"sessionCookie": {}}},
			Tags:        []string{"Comments"},
		}, api.HandleCreateCommentAnswer)

		huma.Register(humaApi, huma.Operation{
			OperationID: "edit-comment-answer",
			Method:      http.MethodPatch,
			Path:        "/comment-answers/{answerId}",
			Summary:     "Edit own comment answer",
			Security:    []map[string][]string{{"sessionCookie": {}}},
			Tags:        []string{"Comments"},
		}, api.HandleEditCommentAnswer)

		huma.Register(humaApi, huma.Operation{
			OperationID: "delete-comment-answer",
			Method:      http.MethodDelete,
			Path:        "/comment-answers/{answerId}",
			Summary:     "Delete own comment answer",
			Security:    []map[string][]string{{"sessionCookie": {}}},
			Tags:        []string{"Comments"},
		}, api.HandleDeleteCommentAnswer)

		huma.Register(humaApi, huma.Operation{
			OperationID: "get-comment-stats",
			Method:      http.MethodGet,
			Path:        "/profile/me/comment-stats",
			Summary:     "Get current user comment stats",
			Security:    []map[string][]string{{"sessionCookie": {}}},
			Tags:        []string{"Profile"},
		}, api.HandleGetCommentStats)

		huma.Register(humaApi, huma.Operation{
			OperationID: "record-time",
			Method:      http.MethodPost,
			Path:        "/stats/time",
			Summary:     "Record active user time",
			Security:    []map[string][]string{{"sessionCookie": {}}},
			Tags:        []string{"Stats"},
		}, api.HandleRecordTime)
	})

	go func() {
		logger.Info("Warming up sitemap cache...")
		if _, err := data.GetSitemapData(context.Background()); err != nil {
			logger.Warn("Failed to warm up sitemap: %v", err)
		} else {
			logger.Info("Sitemap cache warmed up.")
		}
	}()

	data.TimeTracker.Start(60 * time.Second)

	port := "1666"

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
			logger.Fatal("Server failed to listen: %s", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown: %v", err)
	}

	flushCtx, flushCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer flushCancel()

	if err := data.TimeTracker.Stop(flushCtx); err != nil {
		logger.Error("Failed to flush time tracker on shutdown: %v", err)
	}

	logger.Info("Server exited properly")
}
