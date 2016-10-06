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
			code := http.StatusBadRequest
			log.Printf("%s: %s %s: %d", r.RemoteAddr, r.Method, r.URL, code)
			http.Error(w, err.Error(), code)
			return
		}
		v, err := s.Sum(req.A, req.B)
		if err != nil {
			code := http.StatusInternalServerError
			log.Printf("%s: %s %s: %d", r.RemoteAddr, r.Method, r.URL, code)
			http.Error(w, err.Error(), code)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]int{"v": v})
		log.Printf("%s: %s %s: %d", r.RemoteAddr, r.Method, r.URL, 200)

	case "/concat":
		var req struct {
			A string `json:"a"`
			B string `json:"b"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			code := http.StatusBadRequest
			log.Printf("%s: %s %s: %d", r.RemoteAddr, r.Method, r.URL, code)
			http.Error(w, err.Error(), code)
			return
		}
		v, err := s.Concat(req.A, req.B)
		if err != nil {
			code := http.StatusInternalServerError
			log.Printf("%s: %s %s: %d", r.RemoteAddr, r.Method, r.URL, code)
			http.Error(w, err.Error(), code)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]string{"v": v})
		log.Printf("%s: %s %s: %d", r.RemoteAddr, r.Method, r.URL, 200)

	default:
		log.Printf("%s: %s %s: %d", r.RemoteAddr, r.Method, r.URL, 400)
		http.NotFound(w, r)
	}
}

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()
	log.Printf("listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, basicService{}))
}
