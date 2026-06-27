package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"search-service/cmd/serverd/route"
	"search-service/internal/app"
	v1 "search-service/internal/handler/rest/v1"
	"search-service/internal/infrastructure/ai"
	"search-service/internal/infrastructure/opensearch"
	"search-service/internal/infrastructure/postgres"
	"search-service/internal/infrastructure/redis"
	"search-service/internal/infrastructure/translate"
	"search-service/internal/repository"
	"search-service/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("Starting Search API Service...")

	// 1. Load config
	cfg, err := app.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 2. Connect DB
	dsn := "host=" + cfg.DBHost +
		" user=" + cfg.DBUser +
		" password=" + cfg.DBPassword +
		" dbname=" + cfg.DBName +
		" port=" + cfg.DBPort +
		" sslmode=" + cfg.DBSSLMode +
		" search_path=" + cfg.DBSchema +
		" TimeZone=Asia/Ho_Chi_Minh"

	dbConn, err := postgres.Connect(dsn, cfg.Env == "development")
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// 3. Connect Redis & OpenSearch Clients
	redisClient, err := redis.Connect(cfg.RedisHost+":"+cfg.RedisPort, cfg.RedisPassword)
	if err != nil {
		log.Fatalf("failed to connect redis: %v", err)
	}
	productCache := redis.NewRedisCache(redisClient)

	opensearchClient, err := opensearch.Connect(cfg.OpenSearchURL)
	if err != nil {
		log.Fatalf("failed to connect opensearch: %v", err)
	}
	productIndexer := opensearch.NewOpenSearchIndexer(opensearchClient)

	// Ensure OpenSearch index and mappings exist
	productIndexer.EnsureIndex(context.Background())

	// Initialize GORM-based AnalyticsRepository
	analyticsRepo := repository.NewAnalyticsRepository(dbConn)

	searchRepo := repository.NewSearchRepository(dbConn)
	translator := translate.NewTranslationService()
	tagGenerator := ai.NewTagGenerator(cfg.OpenAIAPIKey, cfg.OpenAIModel)
	syncSvc := service.NewSyncService(searchRepo, productIndexer, productCache, translator, tagGenerator)

	searchSvc := service.NewSearchService(productIndexer, productCache, analyticsRepo)
	searchHandler := v1.NewSearchHandler(searchSvc, syncSvc)

	log.Printf("Initialized backend services: Database: %T, RedisCache: %T, OpenSearchIndexer: %T, AnalyticsRepository: %T, SyncService: %T\n", searchRepo, productCache, productIndexer, analyticsRepo, syncSvc)

	// 4. Setup Gin Router
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	r.Use(gin.Recovery())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "UP",
			"service": "search-service",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	route.V1Router(r, searchHandler)

	// 5. Start HTTP Server with Graceful Shutdown
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Printf("Search API Server running on port %s\n", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down Search API Service...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Search API Service exited gracefully")
}
