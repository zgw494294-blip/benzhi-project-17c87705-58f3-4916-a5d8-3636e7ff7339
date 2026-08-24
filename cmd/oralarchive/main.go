package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		log.Printf("oralarchive: %v", err)
		os.Exit(1)
	}
}
func run() error {
	cfg, err := parseConfig()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dataDir := cfg.dataDir
	if cfg.selfcheck {
		dataDir, err = os.MkdirTemp("", "oralarchive-selfcheck-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(dataDir)
	}
	application, err := buildApp(ctx, dataDir)
	if err != nil {
		return err
	}
	defer application.Close()
	listener, err := net.Listen("tcp", cfg.addr)
	if err != nil {
		return fmt.Errorf("监听 %s: %w", cfg.addr, err)
	}
	server := &http.Server{Handler: application.handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second}
	serveErr := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
		close(serveErr)
	}()
	if cfg.selfcheck {
		checkCtx, checkCancel := context.WithTimeout(ctx, 25*time.Second)
		defer checkCancel()
		err = runSelfcheck(checkCtx, "http://"+listener.Addr().String())
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer shutdownCancel()
		_ = server.Shutdown(shutdownCtx)
		if err != nil {
			return err
		}
		fmt.Println("自检通过：完整受控发布流程已签发并验证")
		return nil
	}
	log.Printf("口述史发布治理台监听于 http://%s", listener.Addr())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-signals:
		log.Printf("收到 %s，正在关闭", sig)
	case err := <-serveErr:
		if err != nil {
			return err
		}
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer shutdownCancel()
	return server.Shutdown(shutdownCtx)
}
