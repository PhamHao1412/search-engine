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
	"gorm.io/gorm"
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
	//seedDatabase(dbConn)

	// 3. Connect Redis & OpenSearch Clients
	redisClient, err := redis.Connect(cfg.RedisHost+":"+cfg.RedisPort, cfg.RedisPassword)
	if err != nil {
		log.Fatalf("failed to connect redis: %v", err)
	}

	// Clear Redis cache on startup
	log.Println("Clearing Redis cache on startup...")
	flushCtx, flushCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := redisClient.FlushDB(flushCtx).Err(); err != nil {
		log.Printf("Warning: failed to clear Redis cache: %v", err)
	} else {
		log.Println("Successfully cleared Redis cache on startup.")
	}
	flushCancel()

	productCache := redis.NewRedisCache(redisClient)

	opensearchClient, err := opensearch.Connect(cfg.OpenSearchURL)
	if err != nil {
		log.Fatalf("failed to connect opensearch: %v", err)
	}
	productIndexer := opensearch.NewOpenSearchIndexer(
		opensearchClient,
		cfg.RankingFeaturedBoost,
		cfg.RankingInventoryDecay,
	)

	// Initialize GORM-based AnalyticsRepository
	analyticsRepo := repository.NewAnalyticsRepository(dbConn)

	searchRepo := repository.NewSearchRepository(dbConn)
	translator := translate.NewTranslationService()
	tagGenerator := ai.NewTagGenerator(cfg.OpenAIAPIKey, cfg.OpenAIModel)
	syncSvc := service.NewSyncService(searchRepo, productIndexer, productCache, translator, tagGenerator)

	analyzer, ok := tagGenerator.(service.KeywordAnalyzer)
	if !ok {
		log.Fatalf("failed to cast tagGenerator to KeywordAnalyzer")
	}
	aiSvc := service.NewAISuggestionService(searchRepo, analyzer)
	analyticsSvc := service.NewAnalyticsService(analyticsRepo)
	assistantSvc := service.NewAssistantService(searchRepo, cfg.OpenAIAPIKey, cfg.OpenAIModel)

	searchSvc := service.NewSearchService(productIndexer, productCache, analyticsRepo, searchRepo)
	searchHandler := v1.NewSearchHandler(searchSvc, syncSvc)
	adminHandler := v1.NewAdminHandler(searchRepo, productCache, aiSvc, analyticsSvc, assistantSvc)

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

	route.V1Router(r, searchHandler, adminHandler)

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

func seedDatabase(db *gorm.DB) {
	log.Println("[Seeding] Checking categories and products category_id...")

	// 1. Get tenants
	var tenants []struct {
		ID   string
		Name string
	}
	if err := db.Table("product_svc.tenants").Find(&tenants).Error; err != nil {
		log.Printf("[Seeding] Failed to fetch tenants: %v", err)
		return
	}

	if len(tenants) == 0 {
		log.Println("[Seeding] No tenants found.")
		return
	}

	// 3. Categories
	categories := []struct {
		ID       string
		TenantID string
		Name     string
	}{
		{TenantID: "d3b07384-d113-4956-a5db-251d50c18d01", Name: "Bàn phím"},
		{TenantID: "d3b07384-d113-4956-a5db-251d50c18d01", Name: "Chuột máy tính"},
		{TenantID: "d3b07384-d113-4956-a5db-251d50c18d01", Name: "Tai nghe"},
		{TenantID: "d3b07384-d113-4956-a5db-251d50c18d01", Name: "Thiết bị gia dụng"},
	}
	categories = append(categories, struct {
		ID       string
		TenantID string
		Name     string
	}{ID: "9c2ea5b2-2974-4b5c-897b-ea2f7d3cf2a5", TenantID: "25f2c7e4-92d7-4efb-a710-7bf77bc479a2", Name: "Mỹ phẩm"})

	for _, cat := range categories {
		var count int64
		db.Table("product_svc.categories").Where("id = ?", cat.ID).Count(&count)
		if count == 0 {
			now := time.Now()
			record := map[string]interface{}{
				"tenant_id":  cat.TenantID,
				"name":       cat.Name,
				"created_at": now,
				"updated_at": now,
			}
			if err := db.Table("product_svc.categories").Create(&record).Error; err != nil {
				log.Printf("[Seeding] Failed to create category %s: %v", cat.Name, err)
			} else {
				log.Printf("[Seeding] Created category: %s", cat.Name)
			}
		}
	}

	// 3. Products category_id update
	var products []struct {
		ID   string
		Name string
	}
	if err := db.Table("product_svc.products").Find(&products).Error; err != nil {
		log.Printf("[Seeding] Failed to fetch products: %v", err)
		return
	}

	log.Println("[Seeding] Database seeding check completed.")
}
