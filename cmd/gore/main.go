package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/user/gore/internal/config"
	gorelog "github.com/user/gore/internal/log"
	"github.com/user/gore/internal/server"
)

func main() {
	configPath := flag.String("c", "gore.yaml", "config file path")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		gorelog.Errorf("failed to load config: %v", err)
		os.Exit(1)
	}

	gorelog.Init(&gorelog.Config{
		Level:  cfg.Modules.ErrorLog.GetLevel(),
		Output: cfg.Modules.ErrorLog.GetOutput(),
	})

	srv := server.New(cfg)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-quit
		gorelog.Infof("shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Stop(ctx)
	}()

	if err := srv.Start(); err != nil {
		gorelog.Errorf("server error: %v", err)
		os.Exit(1)
	}
}
