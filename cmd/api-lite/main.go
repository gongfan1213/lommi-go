package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/blueplan/loomi-go/internal/loomi/api_lite"
	"github.com/blueplan/loomi-go/internal/loomi/config"
	contextx "github.com/blueplan/loomi-go/internal/loomi/context"
	"github.com/blueplan/loomi-go/internal/loomi/database"
	logx "github.com/blueplan/loomi-go/internal/loomi/log"
	"github.com/blueplan/loomi-go/internal/loomi/pool"
	"github.com/blueplan/loomi-go/internal/loomi/utils"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()
	logger, err := logx.NewWithFileRotation(cfg.App.LogLevel, "./logs/blueplan-research.log_json")
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	ctx = contextx.WithRequireID(ctx, "api-lite-boot")
	logger.Info(ctx, "api-lite starting...")

	// 依赖：仅 Redis/AccessCounter；不注入 orchestrator，chat 将返回 503
	redisMgr := pool.NewInmem()
	access := utils.NewAccessCounter(logger, redisMgr)
	// 注入持久化（与主入口一致）
	x1, x2, x3, x4 := database.NewInmem()
	persist := database.NewPersistenceManager(x1, x2, x3, x4)
	srv := api_lite.New(logger, access, cfg, redisMgr).WithPersistence(persist)

	addr := fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port)
	go func() {
		if err := srv.Start(addr); err != nil {
			log.Fatalf("http server: %v", err)
		}
	}()

	logger.Info(ctx, "api-lite ready", logx.KV("addr", addr), logx.KV("time", time.Now().Format(time.RFC3339)))

	// 阻塞运行
	select {}
}
