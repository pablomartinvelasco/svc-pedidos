package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Pedido de un cliente. Los datos son sintéticos (RNF-03) y viven en memoria: se pierden al
// reiniciar el pod, lo que basta para un servicio de demostración.
type Pedido struct {
	ID       string        `json:"id"`
	Cliente  string        `json:"cliente"`
	Lineas   []LineaPedido `json:"lineas"`
	Estado   string        `json:"estado"`
	CreadoEn time.Time     `json:"creadoEn"`
}

// LineaPedido referencia un producto de svc-catalogo por su identificador.
type LineaPedido struct {
	ProductoID string `json:"productoId"`
	Cantidad   int    `json:"cantidad"`
}

type store struct {
	mu      sync.RWMutex
	pedidos []Pedido
	next    int
}

func newStore() *store {
	t := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	return &store{
		pedidos: []Pedido{
			{ID: "PED-0001", Cliente: "cliente-001", Estado: "entregado", CreadoEn: t,
				Lineas: []LineaPedido{{ProductoID: "P-001", Cantidad: 2}}},
			{ID: "PED-0002", Cliente: "cliente-002", Estado: "enviado", CreadoEn: t.Add(26 * time.Hour),
				Lineas: []LineaPedido{{ProductoID: "P-003", Cantidad: 1}, {ProductoID: "P-002", Cantidad: 4}}},
			{ID: "PED-0003", Cliente: "cliente-001", Estado: "pendiente", CreadoEn: t.Add(50 * time.Hour),
				Lineas: []LineaPedido{{ProductoID: "P-004", Cantidad: 1}}},
		},
		next: 4,
	}
}

func registerAPI(mux *http.ServeMux, instrument func(string, http.HandlerFunc) http.Handler) {
	s := newStore()
	mux.Handle("GET /api/pedidos", instrument("pedidos_list", s.list))
	mux.Handle("GET /api/pedidos/{id}", instrument("pedidos_get", s.get))
	mux.Handle("POST /api/pedidos", instrument("pedidos_create", s.create))
}

// list devuelve los pedidos; ?estado= filtra por estado.
func (s *store) list(w http.ResponseWriter, r *http.Request) {
	estado := r.URL.Query().Get("estado")
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Pedido, 0, len(s.pedidos))
	for _, p := range s.pedidos {
		if estado == "" || p.Estado == estado {
			out = append(out, p)
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *store) get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.pedidos {
		if p.ID == id {
			writeJSON(w, http.StatusOK, p)
			return
		}
	}
	writeError(w, http.StatusNotFound, "pedido no encontrado: "+id)
}

// create registra un pedido nuevo en estado "pendiente".
func (s *store) create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Cliente string        `json:"cliente"`
		Lineas  []LineaPedido `json:"lineas"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo JSON no válido")
		return
	}
	if in.Cliente == "" || len(in.Lineas) == 0 {
		writeError(w, http.StatusBadRequest, "se requieren cliente y al menos una línea")
		return
	}
	for _, l := range in.Lineas {
		if l.ProductoID == "" || l.Cantidad <= 0 {
			writeError(w, http.StatusBadRequest, "cada línea necesita productoId y cantidad positiva")
			return
		}
	}

	s.mu.Lock()
	p := Pedido{
		ID:       fmt.Sprintf("PED-%04d", s.next),
		Cliente:  in.Cliente,
		Lineas:   in.Lineas,
		Estado:   "pendiente",
		CreadoEn: time.Now().UTC(),
	}
	s.next++
	s.pedidos = append(s.pedidos, p)
	s.mu.Unlock()

	w.Header().Set("Location", "/api/pedidos/"+p.ID)
	writeJSON(w, http.StatusCreated, p)
}
