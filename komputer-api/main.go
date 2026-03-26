package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

func main() {
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
	redisQueue := os.Getenv("REDIS_QUEUE")
	if redisQueue == "" {
		redisQueue = "komputer-events"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	k8s, err := NewK8sClient(namespace)
	if err != nil {
		log.Fatalf("failed to create k8s client: %v", err)
	}
	log.Println("kubernetes client initialized")

	StartRedisWorker(ctx, RedisWorkerConfig{
		Address:  redisAddr,
		Password: redisPassword,
		DB:       0,
		Queue:    redisQueue,
	})
	log.Println("redis worker started")

	r := gin.Default()
	SetupRoutes(r, k8s)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		cancel()
	}()

	log.Printf("listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
