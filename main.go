package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 加载配置
	if err := LoadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 设置路由
	router, limiter := SetupRouter()

	// 创建 HTTP 服务器
	addr := ":" + GetConfig().Port
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// 启动服务器
	go func() {
		log.Printf("服务器启动，端口: %s", GetConfig().Port)
		log.Printf("Dashboard: http://localhost:%s/", GetConfig().Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("正在关闭服务器...")

	// 关闭限流器后台清理 goroutine
	limiter.Stop()

	// 优雅关闭
	if err := srv.Close(); err != nil {
		log.Fatalf("服务器关闭失败: %v", err)
	}

	log.Println("服务器已关闭")
}
