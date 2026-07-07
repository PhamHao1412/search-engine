package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"search-service/internal/app"
	"search-service/internal/infrastructure/ai"
	"search-service/internal/infrastructure/kafka"
	"search-service/internal/infrastructure/opensearch"
	"search-service/internal/infrastructure/postgres"
	"search-service/internal/infrastructure/redis"
	"search-service/internal/infrastructure/translate"
	"search-service/internal/repository"
	"search-service/internal/service"

	"github.com/robfig/cron/v3"
)

func main() {
	log.Println("Starting Search Consumer Worker (workerd)...")

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

	// 3. Connect Redis & OpenSearch & External Services
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

	productIndexer.EnsureIndex(context.Background())

	translator := translate.NewTranslationService()
	tagGenerator := ai.NewTagGenerator(cfg.OpenAIAPIKey, cfg.OpenAIModel)

	// 4. Init Repositories & Services
	searchRepo := repository.NewSearchRepository(dbConn)
	analyticsRepo := repository.NewAnalyticsRepository(dbConn)

	syncSvc := service.NewSyncService(searchRepo, productIndexer, productCache, translator, tagGenerator)
	analyticsSvc := service.NewAnalyticsService(analyticsRepo)

	analyzer, ok := tagGenerator.(service.KeywordAnalyzer)
	if !ok {
		log.Fatalf("failed to cast tagGenerator to KeywordAnalyzer")
	}
	aiSvc := service.NewAISuggestionService(searchRepo, analyzer)
	_ = aiSvc
	_ = analyticsSvc

	// 5. Connect Kafka Consumer (Reader) and DLQ Publisher (Writer)
	topic := "product-ingestion-events"
	dlqTopic := "product-ingestion-events-dlq"
	groupID := "search-indexer-group"
	kafkaReader := kafka.InitReader([]string{cfg.KafkaBrokers}, topic, groupID)
	defer kafkaReader.Close()

	kafkaDLQWriter := kafka.InitWriter([]string{cfg.KafkaBrokers}, dlqTopic)
	defer kafkaDLQWriter.Close()

	consumer := kafka.NewProductEventConsumer(kafkaReader, kafkaDLQWriter, syncSvc)

	// 6. Graceful Shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down Search Consumer Worker...")
		cancel()
	}()

	// 7. Start Reprocessor Cron Job
	cronSched := os.Getenv("REPROCESS_CRON")
	if cronSched == "" {
		cronSched = "*/1 * * * *"
	}

	c := cron.New()
	_, err = c.AddFunc(cronSched, func() {
		log.Printf("[CronJob] Starting ReprocessFailedJobs (schedule: %s)...\n", cronSched)
		if err := syncSvc.ReprocessFailedJobs(ctx); err != nil {
			log.Printf("[CronJob] Error executing ReprocessFailedJobs: %v\n", err)
		}
	})
	if err != nil {
		log.Fatalf("failed to schedule reprocessor cron job: %v", err)
	}

	// 8. Start Analytics Aggregator Cron Job
	analyticsCronSched := os.Getenv("ANALYTICS_CRON")
	if analyticsCronSched == "" {
		analyticsCronSched = "0 * * * *" // Hourly
	}
	_, err = c.AddFunc(analyticsCronSched, func() {
		log.Printf("[CronJob] Starting AggregateAnalytics (schedule: %s)...\n", analyticsCronSched)
		// Aggregate for today
		if err := analyticsSvc.AggregateAnalytics(ctx, time.Now()); err != nil {
			log.Printf("[CronJob] Error executing AggregateAnalytics for today: %v\n", err)
		}
		// Aggregate for yesterday
		if err := analyticsSvc.AggregateAnalytics(ctx, time.Now().AddDate(0, 0, -1)); err != nil {
			log.Printf("[CronJob] Error executing AggregateAnalytics for yesterday: %v\n", err)
		}
	})
	if err != nil {
		log.Fatalf("failed to schedule analytics cron job: %v", err)
	}

	// 9. Start Log Retention Cleanup Cron Job
	cleanupCronSched := os.Getenv("CLEANUP_CRON")
	if cleanupCronSched == "" {
		cleanupCronSched = "0 2 * * *" // Daily at 2 AM
	}
	_, err = c.AddFunc(cleanupCronSched, func() {
		log.Printf("[CronJob] Starting DeleteOldRawLogs (schedule: %s)...\n", cleanupCronSched)
		rowsDeleted, err := analyticsSvc.DeleteOldRawLogs(ctx, 90)
		if err != nil {
			log.Printf("[CronJob] Error executing DeleteOldRawLogs: %v\n", err)
		} else {
			log.Printf("[CronJob] DeleteOldRawLogs completed successfully. Cleaned up %d raw log entries.\n", rowsDeleted)
		}
	})
	if err != nil {
		log.Fatalf("failed to schedule cleanup cron job: %v", err)
	}

	//aiCronSched := os.Getenv("AI_SUGGESTION_CRON")
	//if aiCronSched == "" {
	//	aiCronSched = "*/10 * * * *"
	//}
	//_, err = c.AddFunc(aiCronSched, func() {
	//	log.Printf("[CronJob] Starting GenerateAISuggestions (schedule: %s)...\n", aiCronSched)
	//	if err := aiSvc.GenerateAISuggestions(ctx); err != nil {
	//		log.Printf("[CronJob] Error executing GenerateAISuggestions: %v\n", err)
	//	}
	//})
	//if err != nil {
	//	log.Fatalf("failed to schedule AI suggestion cron job: %v", err)
	//}
	c.Start()
	defer c.Stop()

	// Start consuming loop
	consumer.Start(ctx)
}
