package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(newServer("test", prometheus.NewRegistry()))
	t.Cleanup(srv.Close)
	return srv
}

func get(t *testing.T, url string) (int, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func post(t *testing.T, url, body string) (int, string, http.Header) {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b), resp.Header
}

func TestInfo(t *testing.T) {
	srv := newTestServer(t)
	code, body := get(t, srv.URL+"/")
	if code != http.StatusOK || !strings.Contains(body, `"servicio":"svc-pedidos"`) || !strings.Contains(body, `"entorno":"test"`) {
		t.Fatalf("respuesta inesperada: %d %s", code, body)
	}
}

func TestListPedidos(t *testing.T) {
	srv := newTestServer(t)
	_, body := get(t, srv.URL+"/api/pedidos")
	var all []Pedido
	if err := json.Unmarshal([]byte(body), &all); err != nil || len(all) != 3 {
		t.Fatalf("se esperaban 3 pedidos: %s (%v)", body, err)
	}
	_, body = get(t, srv.URL+"/api/pedidos?estado=pendiente")
	var pendientes []Pedido
	if err := json.Unmarshal([]byte(body), &pendientes); err != nil || len(pendientes) != 1 {
		t.Fatalf("se esperaba 1 pedido pendiente: %s (%v)", body, err)
	}
}

func TestGetPedido(t *testing.T) {
	srv := newTestServer(t)
	if code, body := get(t, srv.URL+"/api/pedidos/PED-0002"); code != http.StatusOK || !strings.Contains(body, `"estado":"enviado"`) {
		t.Fatalf("respuesta inesperada: %d %s", code, body)
	}
	if code, _ := get(t, srv.URL+"/api/pedidos/NO-EXISTE"); code != http.StatusNotFound {
		t.Fatalf("se esperaba 404, llegó %d", code)
	}
}

func TestCreatePedido(t *testing.T) {
	srv := newTestServer(t)
	code, body, hdr := post(t, srv.URL+"/api/pedidos", `{"cliente":"cliente-009","lineas":[{"productoId":"P-002","cantidad":3}]}`)
	if code != http.StatusCreated || hdr.Get("Location") != "/api/pedidos/PED-0004" || !strings.Contains(body, `"estado":"pendiente"`) {
		t.Fatalf("respuesta inesperada: %d %s %v", code, body, hdr)
	}
	if code, _ := get(t, srv.URL+"/api/pedidos/PED-0004"); code != http.StatusOK {
		t.Fatalf("el pedido creado no se encuentra: %d", code)
	}
}

func TestCreatePedidoInvalido(t *testing.T) {
	srv := newTestServer(t)
	for _, body := range []string{
		`no es json`,
		`{"cliente":"","lineas":[{"productoId":"P-001","cantidad":1}]}`,
		`{"cliente":"c","lineas":[]}`,
		`{"cliente":"c","lineas":[{"productoId":"P-001","cantidad":0}]}`,
		`{"cliente":"c","lineas":[{"productoId":"P-001","cantidad":1}],"extra":true}`,
	} {
		if code, _, _ := post(t, srv.URL+"/api/pedidos", body); code != http.StatusBadRequest {
			t.Errorf("%s: se esperaba 400, llegó %d", body, code)
		}
	}
}

func TestSondas(t *testing.T) {
	srv := newTestServer(t)
	for _, path := range []string{"/healthz", "/readyz"} {
		if code, _ := get(t, srv.URL+path); code != http.StatusOK {
			t.Fatalf("%s devolvió %d", path, code)
		}
	}
}

func TestMetricas(t *testing.T) {
	srv := newTestServer(t)
	get(t, srv.URL+"/api/pedidos")
	post(t, srv.URL+"/api/pedidos", `{}`)
	_, body := get(t, srv.URL+"/metrics")
	for _, want := range []string{
		`http_requests_total{code="200",handler="pedidos_list",method="get"} 1`,
		`http_requests_total{code="400",handler="pedidos_create",method="post"} 1`,
		`app_build_info{environment="test",service="svc-pedidos",version="dev"} 1`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("falta en /metrics: %s", want)
		}
	}
}
