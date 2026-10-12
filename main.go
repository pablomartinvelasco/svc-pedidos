// svc-pedidos: pedidos de una tienda ficticia. Servicio de ejemplo de la IDP
// del TFG: API REST mínima con datos sintéticos, sondas de salud y métricas de Prometheus.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const serviceName = "svc-pedidos"

// version se fija al compilar: -ldflags "-X main.version=<sha del commit>".
var version = "dev"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", serviceName, "version", version)
	slog.SetDefault(logger)

	addr := ":" + getenv("PORT", "8080")
	env := getenv("APP_ENV", "local")

	srv := &http.Server{
		Addr:              addr,
		Handler:           newServer(env, prometheus.NewRegistry()),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("servidor iniciado", "addr", addr, "environment", env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("error del servidor", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	// Kubernetes envía SIGTERM al parar el pod: se terminan las peticiones en curso.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("cierre con errores", "error", err)
		os.Exit(1)
	}
	slog.Info("servidor detenido")
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
