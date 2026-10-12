package main

import (
	"encoding/json"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// newServer construye el router con las rutas de la API, las sondas y /metrics.
// Cada ruta de la API se instrumenta con su nombre como etiqueta "handler".
func newServer(env string, reg *prometheus.Registry) http.Handler {
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	requests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Peticiones HTTP atendidas, por ruta, método y código de respuesta.",
	}, []string{"handler", "method", "code"})
	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duración de las peticiones HTTP, por ruta, método y código de respuesta.",
		Buckets: prometheus.DefBuckets,
	}, []string{"handler", "method", "code"})
	buildInfo := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "app_build_info",
		Help: "Versión desplegada del servicio y entorno (valor constante 1).",
	}, []string{"service", "version", "environment"})
	reg.MustRegister(requests, duration, buildInfo)
	buildInfo.WithLabelValues(serviceName, version, env).Set(1)

	instrument := func(name string, h http.HandlerFunc) http.Handler {
		labels := prometheus.Labels{"handler": name}
		return promhttp.InstrumentHandlerDuration(duration.MustCurryWith(labels),
			promhttp.InstrumentHandlerCounter(requests.MustCurryWith(labels), h))
	}

	mux := http.NewServeMux()
	mux.Handle("GET /{$}", instrument("info", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"servicio": serviceName, "version": version, "entorno": env})
	}))
	registerAPI(mux, instrument)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	mux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
