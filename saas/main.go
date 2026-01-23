package main

import (
	"log"
	"time"

	"github.com/Elk-123/gotix/engine"
	"github.com/Elk-123/gotix/saas/internal/handler"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func main() {
	// 1. 初始化 Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// 2. 初始化 Engine
	coreEngine, err := engine.NewEngine(
		engine.WithRedis(rdb),
		engine.WithLockTTL(5*time.Minute),
	)
	if err != nil {
		log.Fatalf("Failed to init engine: %v", err)
	}

	// 3. 初始化 Handler
	ticketHandler := handler.NewTicketHandler(coreEngine)

	// 4. 启动 Web 服务
	r := gin.Default()

	api := r.Group("/api")
	{
		api.POST("/book", ticketHandler.Book)
	}

	log.Println("🚀 GoTix SaaS is running on :8080")
	r.Run(":8080")
}
