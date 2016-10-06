package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
)

// Service describes a service that adds things together.
type Service interface {
	Sum(a, b int) (int, error)
	Concat(a, b string) (string, error)
}

// basicService implements Service.
type basicService struct{}

func (s basicService) Sum(a, b int) (int, error) {
	return a + b, nil
}

func (s basicService) Concat(a, b string) (string, error) {
	return a + b, nil
}

func (s basicService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/sum":
		var req struct {
			A int `json:"a"`
			B int `json:"b"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		v, err := s.Sum(req.A, req.B)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]int{"v": v})

	case "/concat":
		var req struct {
			A string `json:"a"`
			B string `json:"b"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		v, err := s.Concat(req.A, req.B)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]string{"v": v})

	default:
		http.NotFound(w, r)
	}
}

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()
	log.Printf("listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, basicService{}))
}
