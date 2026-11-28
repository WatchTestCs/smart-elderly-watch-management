// Command server starts the native Suining Go API and background workers.
package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"suining-care/internal/config"
	"suining-care/internal/handlers"
	"suining-care/internal/observability"
	"suining-care/internal/runtuning"
	"suining-care/internal/session"
)

func main() { os.Exit(run()) }

func run() int {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if tuning := runtuning.ApplyOptIn(); tuning.Parallelism > 0 || tuning.GoMemoryLimit > 0 {
		slog.Info("runtime_cgroup_tuning", "gomaxprocs", tuning.Parallelism, "go_memory_limit_bytes", tuning.GoMemoryLimit)
	}
	cfg := config.Load()
	// 配置错误不会随等待消失，必须在这里就说清楚；否则 watchDatabase 会在第一轮
	// 校验后静默退出，而端口、就绪文件、/healthz 都已经成功。
	cfgErr := cfg.Validate()
	if cfgErr != nil {
		log.Printf("CONFIG INVALID: %v", cfgErr)
		log.Print("业务接口将保持 503；配置错误不会随重试恢复，修正环境变量后需重启进程")
	}
	addr, readyFile, parentPid := applyDesktopEnv(cfg.Addr)
	cfg.Addr = addr
	ctx := context.Background()

	sessions := session.NewStore(cfg.SessionTTL)
	shutdownRequests := make(chan struct{})
	var requestShutdownOnce sync.Once
	requestShutdown := func() {
		requestShutdownOnce.Do(func() { close(shutdownRequests) })
	}
	service := &databaseService{configErr: cfgErr, desktopToken: cfg.DesktopToken, desktopExit: requestShutdown}
	mux := http.NewServeMux()
	handlers.MountWeb(mux)
	localHandlers := handlers.New(nil, sessions, cfg)
	localHandlers.OnDesktopExit = requestShutdown
	localHandlers.MountDesktopExit(mux)
	localHandlers.MountLocalLogout(mux)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	srv := &http.Server{
		Handler:           observability.Default.HTTP(service.middleware(mux)),
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
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	go func() {
		log.Printf("suining-care listening on %s", ln.Addr())
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve: %v", err)
		}
	}()

	// Desktop sidecar: publish readiness and watch the owning process.
	writeReadyFile(readyFile, actualPort)
	watchParent(parentPid, requestShutdown)

	select {
	case <-quit:
	case <-shutdownRequests:
	}
	service.beginDrain()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if !shutdownHTTP(shutdownCtx, srv) {
		// Active handlers may still own database resources after Close.
		log.Fatal("HTTP drain aborted; skipping graceful database cleanup")
	}
	if !cleanupDatabase(sessions.Close, cancelDatabase, databaseDone, databaseCleanupTimeout) {
		log.Printf("database cleanup exceeded %s; exiting with best-effort cleanup", databaseCleanupTimeout)
		return 1
	}
	return 0
}
