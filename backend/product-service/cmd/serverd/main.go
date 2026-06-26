package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"product-service/cmd/serverd/route"
	"product-service/internal/app"
	v1 "product-service/internal/handler/rest/v1"
	"product-service/internal/infrastructure/kafka"
	"product-service/internal/infrastructure/postgres"
	"product-service/internal/repository"
	"product-service/internal/service"
)

func main() {
	log.Println("Starting Product Ingestion Service...")

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

	// 3. Init Infrastructure Kafka Publisher
	topic := "product-ingestion-events"
	kafka.CheckConnection(cfg.KafkaBrokers)
	kafkaWriter := kafka.InitWriter([]string{cfg.KafkaBrokers}, topic)
	kafkaPublisher := kafka.NewKafkaPublisher(kafkaWriter)
	defer kafkaPublisher.Close()

	// 4. Init Repositories & Services & Handlers
	productRepo := repository.NewProductRepository(dbConn)
	productSvc := service.NewProductService(productRepo, kafkaPublisher)
	productHandler := v1.NewProductHandler(productSvc)

	// 5. Setup Gin Router
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	r.Use(gin.Recovery())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "UP",
			"service": "product-service",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	route.V1Router(r, productHandler)

	// 6. Start HTTP Server with Graceful Shutdown
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Printf("Product API Server running on port %s\n", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down Product Service...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Product Service exited gracefully")
}
