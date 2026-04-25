package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/komputer-ai/komputer-api/docs"
)

// @title komputer.ai API
// @version 1.0
// @description API-first platform for running persistent Claude AI agents on Kubernetes.
// @description Designed to be driven by external systems — create agents, send tasks, and stream real-time results via REST + WebSocket.

// @host localhost:8080
// @BasePath /api/v1

func main() {
	InitLogger()
	defer Logger.Sync()

	log.Println("komputer-api starting...")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}
	redisAddr := os.Getenv("REDIS_ADDRESS")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisStreamPrefix := os.Getenv("REDIS_STREAM_PREFIX")
	if redisStreamPrefix == "" {
		redisStreamPrefix = "komputer-events"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	k8s, err := NewK8sClient(namespace)
	if err != nil {
		log.Fatalf("failed to create k8s client: %v", err)
	}
	log.Println("kubernetes client initialized")

	hostname := os.Getenv("CONSUMER_NAME")
	if hostname == "" {
		hostname, _ = os.Hostname()
	}
	if hostname == "" {
		hostname = fmt.Sprintf("worker-%d", os.Getpid())
	}

	rdbForHub := redis.NewClient(&redis.Options{Addr: redisAddr, Password: redisPassword, DB: 0})
	hub := NewHub(rdbForHub, hostname)

	rw := StartRedisWorker(ctx, RedisWorkerConfig{
		Address:      redisAddr,
		Password:     redisPassword,
		DB:           0,
		StreamPrefix: redisStreamPrefix,
		ConsumerName: hostname,
	}, k8s, hub)
	log.Println("redis worker started")

	r := gin.Default()

	// CORS middleware — allow all origins (UI may run on a different host/port)
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	SetupRoutes(r, k8s, hub, rw)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		srv.Shutdown(shutdownCtx)
	}()

	log.Printf("listening on :%s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
