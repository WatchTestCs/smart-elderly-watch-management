// Command server 启动 careold-go 后端：HTTP API（复刻 preview 分支 careold）。
package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"careold-go/internal/config"
	"careold-go/internal/handlers"
	"careold-go/internal/session"
)

func main() {
	cfg := config.Load()
	addr, imageDir, readyFile, parentPid := applyDesktopEnv(cfg.Addr, cfg.ImageDir)
	cfg.Addr, cfg.ImageDir = addr, imageDir
	ctx := context.Background()

	sessions := session.NewStore(cfg.SessionTTL)
	service := &databaseService{}
	mux := http.NewServeMux()
	handlers.MountWeb(mux)
	localHandlers := handlers.New(nil, sessions, cfg)
	localHandlers.MountDesktopExit(mux)
	localHandlers.MountLocalLogout(mux)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	srv := &http.Server{
		Handler:           service.middleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      35 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		log.Fatalf("listen %s: %v", cfg.Addr, err)
	}
	actualPort := ln.Addr().(*net.TCPAddr).Port
	databaseCtx, cancelDatabase := context.WithCancel(ctx)
	databaseDone := make(chan struct{})
	go func() {
		defer close(databaseDone)
		watchDatabase(databaseCtx, cfg, sessions, service)
	}()
	defer func() { cancelDatabase(); <-databaseDone }()

	done := make(chan struct{})
	var shutdownOnce sync.Once
	shutdown := func() {
		shutdownOnce.Do(func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutdownCtx)
			close(done)
		})
	}

	go func() {
		log.Printf("careold-go listening on %s", ln.Addr())
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve: %v", err)
		}
	}()

	// 桌面端内嵌模式：写就绪文件 + 看护父进程（对齐 Java DesktopReady）。
	writeReadyFile(readyFile, actualPort)
	watchParent(parentPid, shutdown)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)
	select {
	case <-quit:
		shutdown()
	case <-done:
	}
}
