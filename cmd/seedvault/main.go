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
	"path/filepath"
	"syscall"
	"time"

	"seed-vault-release/internal/application"
	"seed-vault-release/internal/httpui"
	"seed-vault-release/internal/repository"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Printf("seedvault: %v", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := parseConfig(args)
	if err != nil {
		return err
	}
	if cfg.selfcheck {
		temp, err := os.MkdirTemp("", "seedvault-selfcheck-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(temp)
		cfg.dataDir = temp
	}
	store, err := repository.Open(filepath.Join(cfg.dataDir, "seedvault.db"))
	if err != nil {
		return err
	}
	defer store.Close()
	app := application.New(store)
	ui := httpui.New(app)
	listener, err := net.Listen("tcp", cfg.addr)
	if err != nil {
		return fmt.Errorf("监听%s: %w", cfg.addr, err)
	}
	server := &http.Server{Handler: ui.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 45 * time.Second}
	done := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		done <- err
	}()
	if cfg.selfcheck {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		checkErr := runSelfcheck(ctx, "http://"+listener.Addr().String())
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer shutdownCancel()
		shutdownErr := server.Shutdown(shutdownCtx)
		serveErr := <-done
		if checkErr != nil {
			return checkErr
		}
		if shutdownErr != nil {
			return shutdownErr
		}
		if serveErr != nil {
			return serveErr
		}
		fmt.Println("selfcheck通过：完整入藏、异常整改、审核放行及凭据校验流程成功")
		return nil
	}
	log.Printf("种质入藏放行工作台监听于 http://%s", listener.Addr().String())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	select {
	case err := <-done:
		return err
	case <-signals:
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		return err
	}
	return <-done
}
